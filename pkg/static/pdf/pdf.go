package pdf

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/go-fitz"
	"github.com/ishank09/data-extraction-service/internal/types"
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

// extractTextFromPDF extracts text content from PDF binary data using go-fitz
func (p *Processor) extractTextFromPDF(pdfData []byte) (string, error) {
	if len(pdfData) == 0 {
		return "", fmt.Errorf("empty PDF data")
	}

	// Create a temporary file to work with go-fitz
	tempFile, err := os.CreateTemp("", "pdf_extract_*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write PDF data to temp file
	_, err = tempFile.Write(pdfData)
	if err != nil {
		return "", fmt.Errorf("failed to write PDF data to temp file: %w", err)
	}

	// Close the file before opening with go-fitz
	tempFile.Close()

	// Open PDF with go-fitz
	doc, err := fitz.New(tempFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to open PDF with go-fitz: %w", err)
	}
	defer doc.Close()

	var textContent strings.Builder

	// Extract text from all pages
	for pageNum := 0; pageNum < doc.NumPage(); pageNum++ {
		text, err := doc.Text(pageNum)
		if err != nil {
			// Log error but continue with other pages
			continue
		}

		if text != "" {
			if textContent.Len() > 0 {
				textContent.WriteString("\n\n")
			}
			textContent.WriteString(text)
		}
	}

	extractedText := textContent.String()
	if extractedText == "" {
		return "", fmt.Errorf("no text content found in PDF")
	}

	return extractedText, nil
}

// processFile converts a PDF file to a document with proper text extraction
func (p *Processor) processFile(filePath string, content []byte) (*types.Document, error) {
	filename := filepath.Base(filePath)

	// Extract text from PDF using go-fitz
	extractedText, err := p.extractTextFromPDF(content)
	if err != nil {
		// For extraction errors, provide metadata only
		return &types.Document{
			ID:        fmt.Sprintf("pdf_%s_%d", strings.TrimSuffix(filename, ".pdf"), time.Now().UnixNano()),
			Type:      "pdf",
			Title:     filename,
			Content:   fmt.Sprintf("PDF extraction failed: %v", err),
			Source:    "embedded",
			Location:  filePath,
			CreatedAt: time.Now(),
			FetchedAt: time.Now(),
			Metadata: map[string]interface{}{
				"filename":         filename,
				"file_type":        "pdf",
				"extraction_error": err.Error(),
			},
		}, nil
	}

	return &types.Document{
		ID:        fmt.Sprintf("pdf_%s_%d", strings.TrimSuffix(filename, ".pdf"), time.Now().UnixNano()),
		Type:      "pdf",
		Title:     filename,
		Content:   extractedText,
		Source:    "embedded",
		Location:  filePath,
		CreatedAt: time.Now(),
		FetchedAt: time.Now(),
		Metadata: map[string]interface{}{
			"filename":   filename,
			"file_type":  "pdf",
			"word_count": len(strings.Fields(extractedText)),
			"page_count": p.getPageCount(content), // We'll add this helper
		},
	}, nil
}

// getPageCount returns the number of pages in a PDF
func (p *Processor) getPageCount(pdfData []byte) int {
	if len(pdfData) == 0 {
		return 0
	}

	// Create a temporary file to work with go-fitz
	tempFile, err := os.CreateTemp("", "pdf_pages_*.pdf")
	if err != nil {
		return 0
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write PDF data to temp file
	_, err = tempFile.Write(pdfData)
	if err != nil {
		return 0
	}

	// Close the file before opening with go-fitz
	tempFile.Close()

	// Open PDF with go-fitz
	doc, err := fitz.New(tempFile.Name())
	if err != nil {
		return 0
	}
	defer doc.Close()

	return doc.NumPage()
}
