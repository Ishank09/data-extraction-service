package pipelinehandler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ishank09/data-extraction-service/internal/types"
	"github.com/ishank09/data-extraction-service/pkg/api/v1/msgraphhandler"
	"github.com/ishank09/data-extraction-service/pkg/api/v1/statichandler"
	"github.com/ishank09/data-extraction-service/pkg/mongodb"
	"github.com/ishank09/data-extraction-service/pkg/msgraph"
	"github.com/ishank09/data-extraction-service/pkg/static"
)

// Handler handles ETL pipeline operations from multiple sources
type Handler struct {
	staticHandler   *statichandler.Handler
	msgraphHandler  *msgraphhandler.Handler
	documentService *mongodb.DocumentService
}

// Config represents the configuration for the pipeline handler
type Config struct {
	MSGraphConfig   *msgraph.Config          `json:"msgraph_config,omitempty"`
	UserID          string                   `json:"user_id,omitempty"` // Required for application flow when accessing user data
	DocumentService *mongodb.DocumentService `json:"document_service,omitempty"`
}

// New creates a new pipeline handler
func New(config *Config) (*Handler, error) {
	handler := &Handler{
		staticHandler: statichandler.New(),
	}

	// Set document service if provided
	if config != nil && config.DocumentService != nil {
		handler.documentService = config.DocumentService
	}

	// Initialize msgraph handler if config is provided
	if config != nil && config.MSGraphConfig != nil {
		msgraphConfig := &msgraphhandler.Config{
			MSGraphConfig: config.MSGraphConfig,
			UserID:        config.UserID, // Pass user ID for application flow
		}

		msgraphHandler, err := msgraphhandler.New(msgraphConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create msgraph handler: %w", err)
		}
		handler.msgraphHandler = msgraphHandler
	}

	return handler, nil
}

// NewWithMSGraphClient creates a new handler with an existing msgraph client
func NewWithMSGraphClient(msgraphClient msgraph.Interface) *Handler {
	return &Handler{
		staticHandler:  statichandler.New(),
		msgraphHandler: msgraphhandler.NewWithClient(msgraphClient),
	}
}

// NewWithDocumentService creates a new handler with document service for testing
func NewWithDocumentService(documentService *mongodb.DocumentService) *Handler {
	return &Handler{
		staticHandler:   statichandler.New(),
		documentService: documentService,
	}
}

// extractStaticData retrieves data from static handler
func (h *Handler) extractStaticData(ctx context.Context) (*types.DocumentCollection, error) {
	staticClient := static.NewClient()
	return staticClient.GetAllDataAsJSON(ctx)
}

// extractMsgraphData retrieves data from msgraph handler
func (h *Handler) extractMsgraphData(ctx context.Context) (*types.DocumentCollection, error) {
	if h.msgraphHandler == nil || !h.msgraphHandler.IsConfigured() {
		return nil, fmt.Errorf("msgraph handler not configured")
	}

	return h.msgraphHandler.GetDocuments(ctx)
}

// extractMsgraphDataWithToken retrieves data using an access token
func (h *Handler) extractMsgraphDataWithToken(ctx context.Context, token string) (*types.DocumentCollection, error) {
	tempHandler, err := msgraphhandler.NewWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create msgraph client with token: %w", err)
	}

	return tempHandler.GetDocuments(ctx)
}

// mergeDataCollections merges data from different sources into a single collection
func (h *Handler) mergeDataCollections(staticData, msgraphData *types.DocumentCollection) *types.DocumentCollection {
	masterCollection := types.NewDocumentCollection("etl_pipeline")

	// Add static data
	if staticData != nil {
		for _, doc := range staticData.Documents {
			masterCollection.AddDocument(doc)
		}
	}

	// Add msgraph data
	if msgraphData != nil {
		for _, doc := range msgraphData.Documents {
			masterCollection.AddDocument(doc)
		}
	}

	return masterCollection
}

// storeDocuments stores documents to MongoDB if document service is available
func (h *Handler) storeDocuments(ctx context.Context, collection *types.DocumentCollection) (*mongodb.StoreCollectionResult, error) {
	if h.documentService == nil {
		return nil, nil // No error if document service is not configured
	}

	return h.documentService.StoreDocumentCollection(ctx, collection)
}

// ExtractAllData returns data from all available sources and stores to MongoDB
func (h *Handler) ExtractAllData(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract static data
	staticData, err := h.extractStaticData(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to extract static data",
			"details": err.Error(),
		})
		return
	}

	// Extract msgraph data
	var msgraphData *types.DocumentCollection

	// Check for Authorization header with Bearer token
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		// Extract token from Authorization header
		token := strings.TrimPrefix(authHeader, "Bearer ")

		msgraphData, err = h.extractMsgraphDataWithToken(ctx, token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Failed to extract msgraph data with provided token",
				"details": err.Error(),
			})
			return
		}
	} else if h.msgraphHandler != nil && h.msgraphHandler.IsConfigured() {
		// Use configured msgraph handler
		msgraphData, err = h.extractMsgraphData(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to extract msgraph data",
				"details": err.Error(),
			})
			return
		}
	}

	// Merge data from both sources
	mergedCollection := h.mergeDataCollections(staticData, msgraphData)

	// Store documents to MongoDB
	var storeResult *mongodb.StoreCollectionResult
	if h.documentService != nil {
		storeResult, err = h.storeDocuments(ctx, mergedCollection)
		if err != nil {
			// Log the error but don't fail the request
			// The user still gets their processed data even if storage fails
			c.Header("X-Storage-Warning", fmt.Sprintf("Failed to store documents: %v", err))
		}
	}

	// Prepare response
	response := gin.H{
		"source":         mergedCollection.Source,
		"fetched_at":     mergedCollection.FetchedAt,
		"schema_version": mergedCollection.SchemaVersion,
		"documents":      mergedCollection.Documents,
		"document_count": len(mergedCollection.Documents),
	}

	// Add storage information if available
	if storeResult != nil {
		response["storage"] = gin.H{
			"stored":           true,
			"collection_id":    storeResult.CollectionID,
			"stored_documents": storeResult.DocumentCount,
		}
	} else if h.documentService != nil {
		response["storage"] = gin.H{
			"stored": false,
			"error":  "Failed to store documents",
		}
	} else {
		response["storage"] = gin.H{
			"stored": false,
			"reason": "Document storage not configured",
		}
	}

	c.JSON(http.StatusOK, response)
}

// ExtractDataBySource returns data from a specific source and stores to MongoDB
func (h *Handler) ExtractDataBySource(c *gin.Context) {
	source := c.Param("source")
	ctx := c.Request.Context()

	var collection *types.DocumentCollection
	var err error

	switch strings.ToLower(source) {
	case "static":
		collection, err = h.extractStaticData(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to extract static data",
				"details": err.Error(),
			})
			return
		}

	case "msgraph", "onenote":
		// Check for Authorization header with Bearer token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			// Extract token from Authorization header
			token := strings.TrimPrefix(authHeader, "Bearer ")

			collection, err = h.extractMsgraphDataWithToken(ctx, token)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Failed to extract msgraph data with provided token",
					"details": err.Error(),
				})
				return
			}
		} else {
			// Fall back to configured handler
			if h.msgraphHandler == nil || !h.msgraphHandler.IsConfigured() {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "Microsoft Graph client not configured and no access token provided",
					"message": "Either configure the service with client credentials or provide an Authorization header with Bearer token",
				})
				return
			}

			collection, err = h.extractMsgraphData(ctx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to extract msgraph data",
					"details": err.Error(),
				})
				return
			}
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Invalid source",
			"supported_sources": []string{"static", "msgraph", "onenote"},
		})
		return
	}

	// Store documents to MongoDB
	var storeResult *mongodb.StoreCollectionResult
	if h.documentService != nil {
		storeResult, err = h.storeDocuments(ctx, collection)
		if err != nil {
			// Log the error but don't fail the request
			c.Header("X-Storage-Warning", fmt.Sprintf("Failed to store documents: %v", err))
		}
	}

	// Prepare response
	response := gin.H{
		"source":         collection.Source,
		"fetched_at":     collection.FetchedAt,
		"schema_version": collection.SchemaVersion,
		"documents":      collection.Documents,
		"document_count": len(collection.Documents),
	}

	// Add storage information if available
	if storeResult != nil {
		response["storage"] = gin.H{
			"stored":           true,
			"collection_id":    storeResult.CollectionID,
			"stored_documents": storeResult.DocumentCount,
		}
	} else if h.documentService != nil {
		response["storage"] = gin.H{
			"stored": false,
			"error":  "Failed to store documents",
		}
	} else {
		response["storage"] = gin.H{
			"stored": false,
			"reason": "Document storage not configured",
		}
	}

	c.JSON(http.StatusOK, response)
}

// ExtractDataByType returns data filtered by type from static source and stores to MongoDB
func (h *Handler) ExtractDataByType(c *gin.Context) {
	fileType := c.Param("type")
	ctx := c.Request.Context()

	staticClient := static.NewClient()
	documents, err := staticClient.GetFilesByType(ctx, fileType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to extract data by type",
			"details": err.Error(),
		})
		return
	}

	// Create a collection for the filtered data
	collection := types.NewDocumentCollection(fmt.Sprintf("static_%s", fileType))
	for _, doc := range documents {
		collection.AddDocument(doc)
	}

	// Store documents to MongoDB
	var storeResult *mongodb.StoreCollectionResult
	if h.documentService != nil {
		storeResult, err = h.storeDocuments(ctx, collection)
		if err != nil {
			// Log the error but don't fail the request
			c.Header("X-Storage-Warning", fmt.Sprintf("Failed to store documents: %v", err))
		}
	}

	// Prepare response
	response := gin.H{
		"source":         collection.Source,
		"fetched_at":     collection.FetchedAt,
		"schema_version": collection.SchemaVersion,
		"documents":      collection.Documents,
		"document_count": len(collection.Documents),
		"type_filter":    fileType,
	}

	// Add storage information if available
	if storeResult != nil {
		response["storage"] = gin.H{
			"stored":           true,
			"collection_id":    storeResult.CollectionID,
			"stored_documents": storeResult.DocumentCount,
		}
	} else if h.documentService != nil {
		response["storage"] = gin.H{
			"stored": false,
			"error":  "Failed to store documents",
		}
	} else {
		response["storage"] = gin.H{
			"stored": false,
			"reason": "Document storage not configured",
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetSources returns information about available data sources
func (h *Handler) GetSources(c *gin.Context) {
	staticClient := static.NewClient()
	sources := []map[string]interface{}{
		{
			"name":        "static",
			"description": "Static files embedded in the application",
			"types":       staticClient.GetSupportedFileTypes(),
			"available":   true,
		},
	}

	// Add msgraph source if available
	if h.msgraphHandler != nil && h.msgraphHandler.IsConfigured() {
		sources = append(sources, map[string]interface{}{
			"name":        "msgraph",
			"description": "Microsoft Graph OneNote data",
			"types":       []string{"onenote"},
			"available":   true,
		})
	} else {
		sources = append(sources, map[string]interface{}{
			"name":        "msgraph",
			"description": "Microsoft Graph OneNote data",
			"types":       []string{"onenote"},
			"available":   false,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"sources":       sources,
		"total_sources": len(sources),
	})
}

// GetHealth returns the health status of the handler and its components
func (h *Handler) GetHealth(c *gin.Context) {
	health := gin.H{
		"status": "healthy",
		"components": gin.H{
			"static_handler": "healthy",
		},
	}

	// Check msgraph handler availability
	if h.msgraphHandler != nil && h.msgraphHandler.IsConfigured() {
		health["components"].(gin.H)["msgraph_handler"] = "healthy"
	} else {
		health["components"].(gin.H)["msgraph_handler"] = "not_configured"
	}

	c.JSON(http.StatusOK, health)
}
