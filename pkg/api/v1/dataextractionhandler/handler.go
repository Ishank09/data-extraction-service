package dataextractionhandler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ishank09/data-extraction-service/internal/types"
	"github.com/ishank09/data-extraction-service/pkg/api/v1/msgraphhandler"
	"github.com/ishank09/data-extraction-service/pkg/api/v1/statichandler"
	"github.com/ishank09/data-extraction-service/pkg/msgraph"
	"github.com/ishank09/data-extraction-service/pkg/static"
)

// Handler handles data extraction from multiple sources
type Handler struct {
	staticHandler  *statichandler.Handler
	msgraphHandler *msgraphhandler.Handler
}

// Config represents the configuration for the data extraction handler
type Config struct {
	MSGraphConfig *msgraph.Config `json:"msgraph_config,omitempty"`
	UserID        string          `json:"user_id,omitempty"` // Required for application flow when accessing user data
}

// New creates a new data extraction handler
func New(config *Config) (*Handler, error) {
	handler := &Handler{
		staticHandler: statichandler.New(),
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

// getStaticDocuments retrieves documents from static handler
func (h *Handler) getStaticDocuments(ctx context.Context) (*types.DocumentCollection, error) {
	staticClient := static.NewClient()
	return staticClient.GetAllDataAsJSON(ctx)
}

// getMsgraphDocuments retrieves documents from msgraph handler
func (h *Handler) getMsgraphDocuments(ctx context.Context) (*types.DocumentCollection, error) {
	if h.msgraphHandler == nil || !h.msgraphHandler.IsConfigured() {
		return nil, fmt.Errorf("msgraph handler not configured")
	}

	return h.msgraphHandler.GetDocuments(ctx)
}

// getMsgraphDocumentsWithToken retrieves documents using an access token
func (h *Handler) getMsgraphDocumentsWithToken(ctx context.Context, token string) (*types.DocumentCollection, error) {
	tempHandler, err := msgraphhandler.NewWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create msgraph client with token: %w", err)
	}

	return tempHandler.GetDocuments(ctx)
}

// mergeDocuments merges documents from different sources into a single collection
func (h *Handler) mergeDocuments(staticDocs, msgraphDocs *types.DocumentCollection) *types.DocumentCollection {
	masterCollection := types.NewDocumentCollection("data_extraction_service")

	// Add static documents
	if staticDocs != nil {
		for _, doc := range staticDocs.Documents {
			masterCollection.AddDocument(doc)
		}
	}

	// Add msgraph documents
	if msgraphDocs != nil {
		for _, doc := range msgraphDocs.Documents {
			masterCollection.AddDocument(doc)
		}
	}

	return masterCollection
}

// GetAllDocuments returns documents from all available sources
func (h *Handler) GetAllDocuments(c *gin.Context) {
	ctx := c.Request.Context()

	// Get static documents
	staticDocs, err := h.getStaticDocuments(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve static documents",
			"details": err.Error(),
		})
		return
	}

	// Get msgraph documents
	var msgraphDocs *types.DocumentCollection

	// Check for Authorization header with Bearer token
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		// Extract token from Authorization header
		token := strings.TrimPrefix(authHeader, "Bearer ")

		msgraphDocs, err = h.getMsgraphDocumentsWithToken(ctx, token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Failed to retrieve msgraph documents with provided token",
				"details": err.Error(),
			})
			return
		}
	} else if h.msgraphHandler != nil && h.msgraphHandler.IsConfigured() {
		// Use configured msgraph handler
		msgraphDocs, err = h.getMsgraphDocuments(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to retrieve msgraph documents",
				"details": err.Error(),
			})
			return
		}
	}

	// Merge documents from both sources
	mergedCollection := h.mergeDocuments(staticDocs, msgraphDocs)

	c.JSON(http.StatusOK, mergedCollection)
}

// GetDocumentsBySource returns documents from a specific source
func (h *Handler) GetDocumentsBySource(c *gin.Context) {
	source := c.Param("source")
	ctx := c.Request.Context()

	switch strings.ToLower(source) {
	case "static":
		collection, err := h.getStaticDocuments(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to retrieve static documents",
				"details": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, collection)

	case "msgraph", "onenote":
		// Check for Authorization header with Bearer token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			// Extract token from Authorization header
			token := strings.TrimPrefix(authHeader, "Bearer ")

			collection, err := h.getMsgraphDocumentsWithToken(ctx, token)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Failed to retrieve msgraph documents with provided token",
					"details": err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, collection)
			return
		}

		// Fall back to configured handler
		if h.msgraphHandler == nil || !h.msgraphHandler.IsConfigured() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Microsoft Graph client not configured and no access token provided",
				"message": "Either configure the service with client credentials or provide an Authorization header with Bearer token",
			})
			return
		}

		collection, err := h.getMsgraphDocuments(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to retrieve msgraph documents",
				"details": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, collection)

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Invalid source",
			"supported_sources": []string{"static", "msgraph", "onenote"},
		})
	}
}

// GetDocumentsByType returns documents filtered by type from static source
func (h *Handler) GetDocumentsByType(c *gin.Context) {
	fileType := c.Param("type")
	ctx := c.Request.Context()

	staticClient := static.NewClient()
	documents, err := staticClient.GetFilesByType(ctx, fileType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to retrieve documents by type",
			"details": err.Error(),
		})
		return
	}

	// Create a collection for the filtered documents
	collection := types.NewDocumentCollection(fmt.Sprintf("static_%s", fileType))
	for _, doc := range documents {
		collection.AddDocument(doc)
	}

	c.JSON(http.StatusOK, collection)
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
