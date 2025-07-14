package pdf

import (
	"context"
	"strings"
	"testing"
)

func TestPDFProcessor_GetDocuments(t *testing.T) {
	processor := NewProcessor()
	ctx := context.Background()

	documents, err := processor.GetDocuments(ctx)
	if err != nil {
		t.Fatalf("GetDocuments() error = %v", err)
	}

	// Should find the Profile.pdf file
	expectedFiles := 1
	if len(documents) != expectedFiles {
		t.Errorf("Expected %d documents, got %d", expectedFiles, len(documents))
	}

	if len(documents) > 0 {
		doc := documents[0]

		// Verify basic document properties
		if doc.Type != "pdf" {
			t.Errorf("Expected document type 'pdf', got '%s'", doc.Type)
		}

		if doc.Title != "Profile.pdf" {
			t.Errorf("Expected title 'Profile.pdf', got '%s'", doc.Title)
		}

		if doc.Source != "embedded" {
			t.Errorf("Expected source 'embedded', got '%s'", doc.Source)
		}

		// Verify metadata
		if doc.Metadata["filename"] != "Profile.pdf" {
			t.Errorf("Expected filename 'Profile.pdf', got '%v'", doc.Metadata["filename"])
		}

		if doc.Metadata["file_type"] != "pdf" {
			t.Errorf("Expected file_type 'pdf', got '%v'", doc.Metadata["file_type"])
		}

		// Check if text was extracted successfully
		if doc.Content == "" {
			t.Error("Expected some content, got empty string")
		}

		// Verify successful extraction
		if extractionError, exists := doc.Metadata["extraction_error"]; exists {
			t.Errorf("Expected successful extraction, but got error: %v", extractionError)
		}

		// Verify expected metadata fields for RAG
		if doc.Metadata["filename"] == nil {
			t.Error("Expected filename in metadata")
		}

		if doc.Metadata["file_type"] == nil {
			t.Error("Expected file_type in metadata")
		}

		if doc.Metadata["word_count"] == nil {
			t.Error("Expected word_count in metadata")
		}

		if doc.Metadata["page_count"] == nil {
			t.Error("Expected page_count in metadata")
		}

		// Verify content contains expected text from the PDF
		if !containsText(doc.Content, "Contact") {
			t.Error("Expected PDF content to contain 'Contact' information")
		}

		if !containsText(doc.Content, "ishankvasania09@gmail.com") {
			t.Error("Expected PDF content to contain email address")
		}

		if !containsText(doc.Content, "Adobe") {
			t.Error("Expected PDF content to contain 'Adobe' work experience")
		}
	}
}

func TestPDFProcessor_ListFiles(t *testing.T) {
	processor := NewProcessor()
	ctx := context.Background()

	files, err := processor.ListFiles(ctx)
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}

	// Should find the Profile.pdf file
	expectedFiles := 1
	if len(files) != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, len(files))
	}

	if len(files) > 0 {
		if files[0] != "Profile.pdf" {
			t.Errorf("Expected file 'Profile.pdf', got '%s'", files[0])
		}
	}
}

func TestPDFProcessor_ExtractTextFromPDF_EmptyData(t *testing.T) {
	processor := NewProcessor()

	text, err := processor.extractTextFromPDF([]byte{})
	if err == nil {
		t.Error("Expected error for empty PDF data")
	}

	if text != "" {
		t.Errorf("Expected empty text for empty data, got '%s'", text)
	}
}

func TestPDFProcessor_ExtractTextFromPDF_InvalidData(t *testing.T) {
	processor := NewProcessor()

	invalidPDFData := []byte("This is not a PDF file")
	text, err := processor.extractTextFromPDF(invalidPDFData)
	if err == nil {
		t.Error("Expected error for invalid PDF data")
	}

	if text != "" {
		t.Errorf("Expected empty text for invalid data, got '%s'", text)
	}
}

func TestNewProcessor(t *testing.T) {
	processor := NewProcessor()
	if processor == nil {
		t.Error("NewProcessor() should not return nil")
	}
}

func TestPDFProcessor_ProcessFile_ErrorHandling(t *testing.T) {
	processor := NewProcessor()

	// Test with invalid PDF data
	invalidData := []byte("invalid pdf data")
	doc, err := processor.processFile("test.pdf", invalidData)

	if err != nil {
		t.Fatalf("processFile() should not return error for invalid PDF, got: %v", err)
	}

	if doc == nil {
		t.Fatal("processFile() should return a document even for invalid PDF")
	}

	// Verify error handling
	if doc.Type != "pdf" {
		t.Errorf("Expected document type 'pdf', got '%s'", doc.Type)
	}

	if doc.Title != "test.pdf" {
		t.Errorf("Expected title 'test.pdf', got '%s'", doc.Title)
	}

	// Should contain error information
	if _, exists := doc.Metadata["extraction_error"]; !exists {
		t.Error("Expected extraction_error in metadata for invalid PDF")
	}
}

// Helper function to check if text contains a substring
func containsText(text, substring string) bool {
	return strings.Contains(text, substring)
}
