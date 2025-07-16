package documenthandler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishank09/data-extraction-service/pkg/mongodb"
)

// Handler handles document operations from MongoDB
type Handler struct {
	documentService *mongodb.DocumentService
}

// Config represents the configuration for the document handler
type Config struct {
	DocumentService *mongodb.DocumentService `json:"document_service,omitempty"`
}

// New creates a new document handler
func New(config *Config) *Handler {
	if config == nil || config.DocumentService == nil {
		return nil
	}

	return &Handler{
		documentService: config.DocumentService,
	}
}

// GetDocuments retrieves stored documents with optional filtering
func (h *Handler) GetDocuments(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters
	filter := mongodb.DocumentFilter{}

	if source := c.Query("source"); source != "" {
		filter.Source = source
	}
	if docType := c.Query("type"); docType != "" {
		filter.Type = docType
	}
	if title := c.Query("title"); title != "" {
		filter.Title = title
	}

	// Parse time filters
	if fetchedAfter := c.Query("fetched_after"); fetchedAfter != "" {
		if t, err := time.Parse(time.RFC3339, fetchedAfter); err == nil {
			filter.FetchedAfter = t
		}
	}
	if fetchedBefore := c.Query("fetched_before"); fetchedBefore != "" {
		if t, err := time.Parse(time.RFC3339, fetchedBefore); err == nil {
			filter.FetchedBefore = t
		}
	}

	// Parse pagination
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if skip := c.Query("skip"); skip != "" {
		if s, err := strconv.Atoi(skip); err == nil && s >= 0 {
			filter.Skip = s
		}
	}

	// Set default limit if not specified
	if filter.Limit == 0 {
		filter.Limit = 50 // Default to 50 documents
	}

	documents, err := h.documentService.GetDocuments(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve documents",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documents": documents,
		"count":     len(documents),
		"filter":    filter,
	})
}

// GetDocumentCollections retrieves stored document collections metadata
func (h *Handler) GetDocumentCollections(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters
	filter := mongodb.CollectionFilter{}

	if source := c.Query("source"); source != "" {
		filter.Source = source
	}

	// Parse time filters
	if fetchedAfter := c.Query("fetched_after"); fetchedAfter != "" {
		if t, err := time.Parse(time.RFC3339, fetchedAfter); err == nil {
			filter.FetchedAfter = t
		}
	}
	if fetchedBefore := c.Query("fetched_before"); fetchedBefore != "" {
		if t, err := time.Parse(time.RFC3339, fetchedBefore); err == nil {
			filter.FetchedBefore = t
		}
	}

	// Parse pagination
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if skip := c.Query("skip"); skip != "" {
		if s, err := strconv.Atoi(skip); err == nil && s >= 0 {
			filter.Skip = s
		}
	}

	// Set default limit if not specified
	if filter.Limit == 0 {
		filter.Limit = 20 // Default to 20 collections
	}

	collections, err := h.documentService.GetDocumentCollections(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve document collections",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"collections": collections,
		"count":       len(collections),
		"filter":      filter,
	})
}

// GetDocumentStats returns statistics about stored documents
func (h *Handler) GetDocumentStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats, err := h.documentService.GetDocumentStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve document statistics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// DeleteOldDocuments deletes documents older than specified duration
func (h *Handler) DeleteOldDocuments(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse duration from query parameter
	durationStr := c.Query("older_than")
	if durationStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing required parameter 'older_than'",
			"example": "?older_than=720h (for 30 days)",
		})
		return
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid duration format",
			"details": err.Error(),
			"example": "?older_than=720h (for 30 days)",
		})
		return
	}

	result, err := h.documentService.DeleteOldDocuments(ctx, duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete old documents",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Successfully deleted old documents",
		"deleted_count": result.DeletedCount,
		"older_than":    durationStr,
	})
}

// GetHealth returns health status of document handler
func (h *Handler) GetHealth(c *gin.Context) {
	if h.documentService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "not_configured",
			"component": "document_service",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"component": "document_service",
	})
}

// IsConfigured returns whether the document handler is configured
func (h *Handler) IsConfigured() bool {
	return h != nil && h.documentService != nil
}
