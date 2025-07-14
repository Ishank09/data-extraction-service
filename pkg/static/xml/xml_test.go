package xml

import (
	"context"
	"testing"
)

func TestXMLProcessor_GetDocuments_EmptyDirectory(t *testing.T) {
	processor := NewProcessor()
	ctx := context.Background()

	documents, err := processor.GetDocuments(ctx)
	if err != nil {
		t.Fatalf("GetDocuments() error = %v", err)
	}

	// Should handle empty directory gracefully - no XML files
	expectedFiles := 0
	if len(documents) != expectedFiles {
		t.Errorf("Expected %d documents, got %d", expectedFiles, len(documents))
	}
}

func TestXMLProcessor_ListFiles_EmptyDirectory(t *testing.T) {
	processor := NewProcessor()
	ctx := context.Background()

	files, err := processor.ListFiles(ctx)
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}

	// Should handle empty directory gracefully - no XML files
	expectedFiles := 0
	if len(files) != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, len(files))
	}
}

func TestNewProcessor(t *testing.T) {
	processor := NewProcessor()
	if processor == nil {
		t.Error("NewProcessor() should not return nil")
	}
}
