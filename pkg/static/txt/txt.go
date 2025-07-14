package txt

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
var txtFiles embed.FS

// Processor handles TXT file processing
type Processor struct{}

// NewProcessor creates a new TXT processor
func NewProcessor() *Processor {
	return &Processor{}
}

// GetDocuments returns all TXT files as documents
func (p *Processor) GetDocuments(ctx context.Context) ([]types.Document, error) {
	var documents []types.Document

	err := fs.WalkDir(txtFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".txt") {
			return nil
		}

		content, err := txtFiles.ReadFile(path)
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
		return nil, fmt.Errorf("failed to walk TXT files: %w", err)
	}

	return documents, nil
}

// ListFiles returns list of all TXT filenames
func (p *Processor) ListFiles(ctx context.Context) ([]string, error) {
	var files []string

	err := fs.WalkDir(txtFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".txt") {
			return nil
		}

		files = append(files, filepath.Base(path))
		return nil
	})

	return files, err
}

// processFile converts a TXT file to a document using utils functions
func (p *Processor) processFile(filePath string, content []byte) (*types.Document, error) {
	filename := filepath.Base(filePath)

	// Use utils function for consistent processing
	contentJSON, err := utils.BytesToJSON(content)
	if err != nil {
		return nil, fmt.Errorf("failed to convert content to JSON: %w", err)
	}

	return &types.Document{
		ID:        fmt.Sprintf("txt_%s_%d", strings.TrimSuffix(filename, ".txt"), time.Now().UnixNano()),
		Type:      "txt",
		Title:     filename,
		Content:   string(content),
		Source:    "embedded",
		Location:  filePath,
		CreatedAt: time.Now(),
		FetchedAt: time.Now(),
		Metadata: map[string]interface{}{
			"filename":      filename,
			"file_type":     "txt",
			"file_size":     len(content),
			"embedded_path": filePath,
			"parsed_data":   contentJSON,
		},
	}, nil
}
