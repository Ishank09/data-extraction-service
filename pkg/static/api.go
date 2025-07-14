package static

import (
	"context"
	"fmt"

	"github.com/ishank09/data-extraction-service/internal/types"
	"github.com/ishank09/data-extraction-service/pkg/static/csv"
	"github.com/ishank09/data-extraction-service/pkg/static/html"
	"github.com/ishank09/data-extraction-service/pkg/static/json"
	"github.com/ishank09/data-extraction-service/pkg/static/pdf"
	"github.com/ishank09/data-extraction-service/pkg/static/txt"
	"github.com/ishank09/data-extraction-service/pkg/static/xml"
)

// FileProcessor interface for all file type processors
type FileProcessor interface {
	GetDocuments(ctx context.Context) ([]types.Document, error)
	ListFiles(ctx context.Context) ([]string, error)
}

// Client handles static file operations
type Client struct {
	csvProcessor  *csv.Processor
	jsonProcessor *json.Processor
	txtProcessor  *txt.Processor
	pdfProcessor  *pdf.Processor
	xmlProcessor  *xml.Processor
	htmlProcessor *html.Processor
}

// NewClient creates a new static file client
func NewClient() *Client {
	return &Client{
		csvProcessor:  csv.NewProcessor(),
		jsonProcessor: json.NewProcessor(),
		txtProcessor:  txt.NewProcessor(),
		pdfProcessor:  pdf.NewProcessor(),
		xmlProcessor:  xml.NewProcessor(),
		htmlProcessor: html.NewProcessor(),
	}
}

// GetAllDataAsJSON returns all embedded files as JSON documents
func (c *Client) GetAllDataAsJSON(ctx context.Context) (*types.DocumentCollection, error) {
	collection := types.NewDocumentCollection("static_files")

	// Get documents from all processors
	processors := []FileProcessor{
		c.csvProcessor,
		c.jsonProcessor,
		c.txtProcessor,
		c.pdfProcessor,
		c.xmlProcessor,
		c.htmlProcessor,
	}

	for _, processor := range processors {
		docs, err := processor.GetDocuments(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get documents: %w", err)
		}

		for _, doc := range docs {
			collection.AddDocument(doc)
		}
	}

	return collection, nil
}

// GetFilesByType returns documents for a specific file type
func (c *Client) GetFilesByType(ctx context.Context, fileType string) ([]types.Document, error) {
	var processor FileProcessor

	switch fileType {
	case "csv":
		processor = c.csvProcessor
	case "json":
		processor = c.jsonProcessor
	case "txt":
		processor = c.txtProcessor
	case "pdf":
		processor = c.pdfProcessor
	case "xml":
		processor = c.xmlProcessor
	case "html":
		processor = c.htmlProcessor
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}

	return processor.GetDocuments(ctx)
}

// ListFilesByType returns filenames for a specific file type
func (c *Client) ListFilesByType(ctx context.Context, fileType string) ([]string, error) {
	var processor FileProcessor

	switch fileType {
	case "csv":
		processor = c.csvProcessor
	case "json":
		processor = c.jsonProcessor
	case "txt":
		processor = c.txtProcessor
	case "pdf":
		processor = c.pdfProcessor
	case "xml":
		processor = c.xmlProcessor
	case "html":
		processor = c.htmlProcessor
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}

	return processor.ListFiles(ctx)
}

// GetSupportedFileTypes returns list of supported file types
func (c *Client) GetSupportedFileTypes() []string {
	return []string{"csv", "json", "txt", "pdf", "xml", "html"}
}
