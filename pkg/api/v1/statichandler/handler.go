package statichandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ishank09/data-extraction-service/pkg/static"
)

// Handler handles static file operations
type Handler struct {
	staticClient *static.Client
}

// New creates a new static handler
func New() *Handler {
	return &Handler{
		staticClient: static.NewClient(),
	}
}

// ExtractAllData returns all static documents
func (h *Handler) ExtractAllData(c *gin.Context) {
	ctx := c.Request.Context()

	collection, err := h.staticClient.GetAllDataAsJSON(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve static documents",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, collection)
}

// ExtractDataByType returns data filtered by type
func (h *Handler) ExtractDataByType(c *gin.Context) {
	fileType := c.Param("type")
	ctx := c.Request.Context()

	documents, err := h.staticClient.GetFilesByType(ctx, fileType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to retrieve documents by type",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documents": documents,
		"count":     len(documents),
		"type":      fileType,
	})
}

// GetSupportedTypes returns supported file types
func (h *Handler) GetSupportedTypes(c *gin.Context) {
	types := h.staticClient.GetSupportedFileTypes()
	c.JSON(http.StatusOK, gin.H{
		"supported_types": types,
		"count":           len(types),
	})
}

// GetHealth returns health status of static client
func (h *Handler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"component": "static_client",
	})
}
