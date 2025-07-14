package pdf

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/ishank09/data-extraction-service/internal/types"
	"github.com/ishank09/data-extraction-service/internal/utils"
)

//go:embed files/*
var pdfFiles embed.FS

// Processor handles PDF file processing
type Processor struct{}

// NewProcessor creates a new PDF processor
func NewProcessor() *Processor {
	return &Processor{}
}

// GetDocuments returns all PDF files as documents
func (p *Processor) GetDocuments(ctx context.Context) ([]types.Document, error) {
	var documents []types.Document

	err := fs.WalkDir(pdfFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".pdf") {
			return nil
		}

		content, err := pdfFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		doc, err := p.processFile(path, content)
		if err != nil {
			return fmt.Errorf("failed to process file %s: %w", path, err)
		}

		documents = append(documents, *doc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk PDF files: %w", err)
	}

	return documents, nil
}

// ListFiles returns list of all PDF filenames
func (p *Processor) ListFiles(ctx context.Context) ([]string, error) {
	var files []string

	err := fs.WalkDir(pdfFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".pdf") {
			return nil
		}

		files = append(files, filepath.Base(path))
		return nil
	})

	return files, err
}

// processFile converts a PDF file to a document using utils functions
func (p *Processor) processFile(filePath string, content []byte) (*types.Document, error) {
	filename := filepath.Base(filePath)

	// Use utils function for consistent processing
	contentJSON, err := utils.BytesToJSON(content)
	if err != nil {
		return nil, fmt.Errorf("failed to convert content to JSON: %w", err)
	}

	return &types.Document{
		ID:        fmt.Sprintf("pdf_%s_%d", strings.TrimSuffix(filename, ".pdf"), time.Now().UnixNano()),
		Type:      "pdf",
		Title:     filename,
		Content:   string(content),
		Source:    "embedded",
		Location:  filePath,
		CreatedAt: time.Now(),
		FetchedAt: time.Now(),
		Metadata: map[string]interface{}{
			"filename":      filename,
			"file_type":     "pdf",
			"file_size":     len(content),
			"embedded_path": filePath,
			"parsed_data":   contentJSON,
		},
	}, nil
}
