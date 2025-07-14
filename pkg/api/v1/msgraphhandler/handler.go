package msgraphhandler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ishank09/data-extraction-service/internal/types"
	"github.com/ishank09/data-extraction-service/pkg/msgraph"
)

// Handler handles Microsoft Graph operations
type Handler struct {
	msgraphClient msgraph.Interface
}

// Config represents the configuration for the msgraph handler
type Config struct {
	MSGraphConfig *msgraph.Config `json:"msgraph_config,omitempty"`
}

// New creates a new msgraph handler
func New(config *Config) (*Handler, error) {
	if config == nil || config.MSGraphConfig == nil {
		return nil, nil // Return nil if no configuration provided
	}

	graphClient, err := msgraph.NewClient(*config.MSGraphConfig)
	if err != nil {
		return nil, err
	}

	return &Handler{
		msgraphClient: graphClient,
	}, nil
}

// NewWithClient creates a new msgraph handler with an existing client
func NewWithClient(client msgraph.Interface) *Handler {
	return &Handler{
		msgraphClient: client,
	}
}

// GetDocuments retrieves documents from Microsoft Graph
func (h *Handler) GetDocuments(ctx context.Context) (*types.DocumentCollection, error) {
	if h.msgraphClient == nil {
		return nil, nil
	}

	return h.msgraphClient.GetOneNoteDataAsJSON(ctx)
}

// GetAllDocuments returns all OneNote documents
func (h *Handler) GetAllDocuments(c *gin.Context) {
	if h.msgraphClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Microsoft Graph client not configured",
		})
		return
	}

	ctx := c.Request.Context()

	collection, err := h.msgraphClient.GetOneNoteDataAsJSON(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve msgraph documents",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, collection)
}

// GetHealth returns health status of msgraph client
func (h *Handler) GetHealth(c *gin.Context) {
	if h.msgraphClient == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":    "not_configured",
			"component": "msgraph_client",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"component": "msgraph_client",
	})
}

// IsConfigured returns whether the msgraph client is configured
func (h *Handler) IsConfigured() bool {
	return h.msgraphClient != nil
}
