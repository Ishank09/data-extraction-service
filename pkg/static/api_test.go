package static

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Error("NewClient() should not return nil")
	}
}

func TestClient_GetAllDataAsJSON(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	collection, err := client.GetAllDataAsJSON(ctx)
	if err != nil {
		t.Fatalf("GetAllDataAsJSON() error = %v", err)
	}

	if collection == nil {
		t.Fatal("GetAllDataAsJSON() should not return nil collection")
	}

	// Should have no documents since all directories are empty
	expectedDocs := 0
	actualDocs := collection.GetDocumentCount()
	if actualDocs != expectedDocs {
		t.Errorf("Expected %d documents, got %d", expectedDocs, actualDocs)
	}

	// Verify collection metadata
	if collection.Source != "static_files" {
		t.Errorf("Expected source 'static_files', got '%s'", collection.Source)
	}
	if collection.SchemaVersion != "v1" {
		t.Errorf("Expected schema version 'v1', got '%s'", collection.SchemaVersion)
	}

	// Check that all file types are empty
	fileTypes := make(map[string]int)
	for _, doc := range collection.Documents {
		fileTypes[doc.Type]++
	}

	// All file types should have 0 documents (empty directories)
	if fileTypes["json"] != 0 {
		t.Error("Should have 0 JSON documents (empty directory)")
	}
	if fileTypes["csv"] != 0 {
		t.Error("Should have 0 CSV documents (empty directory)")
	}
	if fileTypes["txt"] != 0 {
		t.Error("Should have 0 TXT documents (empty directory)")
	}
	if fileTypes["pdf"] != 0 {
		t.Error("Should have 0 PDF documents (empty directory)")
	}
	if fileTypes["html"] != 0 {
		t.Error("Should have 0 HTML documents (empty directory)")
	}
	if fileTypes["xml"] != 0 {
		t.Error("Should have 0 XML documents (empty directory)")
	}
}

func TestClient_GetFilesByType(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test JSON files (empty directory)
	jsonDocs, err := client.GetFilesByType(ctx, "json")
	if err != nil {
		t.Fatalf("GetFilesByType('json') error = %v", err)
	}
	if len(jsonDocs) != 0 {
		t.Errorf("Expected 0 JSON documents, got %d", len(jsonDocs))
	}

	// Test CSV files (empty directory)
	csvDocs, err := client.GetFilesByType(ctx, "csv")
	if err != nil {
		t.Fatalf("GetFilesByType('csv') error = %v", err)
	}
	if len(csvDocs) != 0 {
		t.Errorf("Expected 0 CSV documents, got %d", len(csvDocs))
	}

	// Test TXT files (empty directory)
	txtDocs, err := client.GetFilesByType(ctx, "txt")
	if err != nil {
		t.Fatalf("GetFilesByType('txt') error = %v", err)
	}
	if len(txtDocs) != 0 {
		t.Errorf("Expected 0 TXT documents, got %d", len(txtDocs))
	}

	// Test PDF files (empty directory)
	pdfDocs, err := client.GetFilesByType(ctx, "pdf")
	if err != nil {
		t.Fatalf("GetFilesByType('pdf') error = %v", err)
	}
	if len(pdfDocs) != 0 {
		t.Errorf("Expected 0 PDF documents, got %d", len(pdfDocs))
	}
}

func TestClient_GetFilesByType_InvalidType(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test invalid file type
	_, err := client.GetFilesByType(ctx, "invalid")
	if err == nil {
		t.Error("GetFilesByType('invalid') should return an error")
	}

	expectedError := "unsupported file type: invalid"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestClient_ListFilesByType(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test JSON files (empty directory)
	jsonFiles, err := client.ListFilesByType(ctx, "json")
	if err != nil {
		t.Fatalf("ListFilesByType('json') error = %v", err)
	}
	if len(jsonFiles) != 0 {
		t.Errorf("Expected 0 JSON files, got %d", len(jsonFiles))
	}

	// Test CSV files (empty directory)
	csvFiles, err := client.ListFilesByType(ctx, "csv")
	if err != nil {
		t.Fatalf("ListFilesByType('csv') error = %v", err)
	}
	if len(csvFiles) != 0 {
		t.Errorf("Expected 0 CSV files, got %d", len(csvFiles))
	}

	// Test PDF files (empty directory)
	pdfFiles, err := client.ListFilesByType(ctx, "pdf")
	if err != nil {
		t.Fatalf("ListFilesByType('pdf') error = %v", err)
	}
	if len(pdfFiles) != 0 {
		t.Errorf("Expected 0 PDF files, got %d", len(pdfFiles))
	}
}

func TestClient_ListFilesByType_InvalidType(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test invalid file type
	_, err := client.ListFilesByType(ctx, "invalid")
	if err == nil {
		t.Error("ListFilesByType('invalid') should return an error")
	}

	expectedError := "unsupported file type: invalid"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestClient_DocumentStructure_EmptyDirectories(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	collection, err := client.GetAllDataAsJSON(ctx)
	if err != nil {
		t.Fatalf("GetAllDataAsJSON() error = %v", err)
	}

	// Since all directories are empty, there should be no documents to test
	if len(collection.Documents) != 0 {
		t.Errorf("Expected 0 documents from empty directories, got %d", len(collection.Documents))
	}
}

func TestClient_AllSupportedFileTypes(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	supportedTypes := []string{"json", "csv", "txt", "pdf", "html", "xml"}

	for _, fileType := range supportedTypes {
		// Should not error for any supported type
		_, err := client.GetFilesByType(ctx, fileType)
		if err != nil {
			t.Errorf("GetFilesByType('%s') should not error, got: %v", fileType, err)
		}

		_, err = client.ListFilesByType(ctx, fileType)
		if err != nil {
			t.Errorf("ListFilesByType('%s') should not error, got: %v", fileType, err)
		}
	}
}
