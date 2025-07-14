package msgraph

import (
	"context"
	"testing"
	"time"

	msgraphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"

	"github.com/ishank09/data-extraction-service/internal/types"
)

// TestGetStringValue tests the helper function for handling string pointers
func TestGetStringValue(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty string",
			input:    stringPtr(""),
			expected: "",
		},
		{
			name:     "valid string",
			input:    stringPtr("test-value"),
			expected: "test-value",
		},
		{
			name:     "string with spaces",
			input:    stringPtr("  test with spaces  "),
			expected: "  test with spaces  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringValue(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestGetTimeValue tests the helper function for handling time pointers
func TestGetTimeValue(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    *time.Time
		expected time.Time
	}{
		{
			name:     "nil pointer returns current time",
			input:    nil,
			expected: time.Now(), // We'll check this is recent
		},
		{
			name:     "valid time",
			input:    &fixedTime,
			expected: fixedTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTimeValue(tt.input)

			if tt.input == nil {
				// For nil input, check that the returned time is recent (within last minute)
				if time.Since(result) > time.Minute {
					t.Errorf("Expected recent time for nil input, got %v", result)
				}
			} else {
				if !result.Equal(tt.expected) {
					t.Errorf("Expected '%v', got '%v'", tt.expected, result)
				}
			}
		})
	}
}

// TestProcessPageContent tests the page content processing logic
func TestProcessPageContent(t *testing.T) {
	// Create a mock client
	client := &Client{}

	// Create mock OneNote objects
	notebook := createMockNotebook("notebook-123", "Test Notebook")
	section := createMockSection("section-456", "Test Section")
	page := createMockPage("page-789", "Test Page", time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))

	// Test content (HTML format)
	content := []byte(`<html><body><h1>Test Page</h1><p>This is test content.</p></body></html>`)

	// Test the processPageContent function
	doc, err := client.processPageContent(page, notebook, section, content)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify document fields
	if doc.ID != "page-789" {
		t.Errorf("Expected ID 'page-789', got '%s'", doc.ID)
	}
	if doc.Source != "onenote" {
		t.Errorf("Expected source 'onenote', got '%s'", doc.Source)
	}
	if doc.Type != "page" {
		t.Errorf("Expected type 'page', got '%s'", doc.Type)
	}
	if doc.Title != "Test Page" {
		t.Errorf("Expected title 'Test Page', got '%s'", doc.Title)
	}
	if doc.Location != "OneNote/Test Notebook/Test Section" {
		t.Errorf("Expected location 'OneNote/Test Notebook/Test Section', got '%s'", doc.Location)
	}
	if doc.Language != "en" {
		t.Errorf("Expected language 'en', got '%s'", doc.Language)
	}
	if doc.TextChunkingStrategy != "page_based" {
		t.Errorf("Expected chunking strategy 'page_based', got '%s'", doc.TextChunkingStrategy)
	}

	// Verify metadata
	if doc.Metadata == nil {
		t.Fatal("Expected non-nil metadata")
	}

	expectedMetadata := map[string]interface{}{
		"notebook_id":    "notebook-123",
		"notebook_name":  "Test Notebook",
		"section_id":     "section-456",
		"section_name":   "Test Section",
		"page_id":        "page-789",
		"content_format": "html",
	}

	for key, expectedValue := range expectedMetadata {
		if actualValue, exists := doc.Metadata[key]; !exists {
			t.Errorf("Expected metadata key '%s' to exist", key)
		} else if actualValue != expectedValue {
			t.Errorf("Expected metadata[%s] = '%v', got '%v'", key, expectedValue, actualValue)
		}
	}

	// Verify timestamps
	if doc.CreatedAt.IsZero() {
		t.Error("Expected non-zero created time")
	}
	if doc.FetchedAt.IsZero() {
		t.Error("Expected non-zero fetched time")
	}

	// Verify version hash is generated
	if doc.VersionHash == "" {
		t.Error("Expected non-empty version hash")
	}
	if doc.VersionHash[:7] != "sha256:" {
		t.Errorf("Expected version hash to start with 'sha256:', got '%s'", doc.VersionHash)
	}

	// Verify content is processed (should contain extracted text)
	if doc.Content == "" {
		t.Error("Expected non-empty content")
	}
}

// TestProcessPageContentWithNilValues tests page content processing with nil values
func TestProcessPageContentWithNilValues(t *testing.T) {
	client := &Client{}

	// Create mock objects with nil values
	notebook := createMockNotebook("", "")
	section := createMockSection("", "")
	page := createMockPage("", "", time.Time{})

	content := []byte(`<html><body><p>Test content</p></body></html>`)

	doc, err := client.processPageContent(page, notebook, section, content)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify empty values are handled correctly
	if doc.ID != "" {
		t.Errorf("Expected empty ID, got '%s'", doc.ID)
	}
	if doc.Title != "" {
		t.Errorf("Expected empty title, got '%s'", doc.Title)
	}
	if doc.Location != "OneNote//" {
		t.Errorf("Expected location 'OneNote//', got '%s'", doc.Location)
	}

	// Verify metadata contains empty values
	if doc.Metadata["notebook_id"] != "" {
		t.Errorf("Expected empty notebook_id, got '%v'", doc.Metadata["notebook_id"])
	}
}

// TestCombineOneNoteDataEmptyData tests the business logic with empty data
func TestCombineOneNoteDataEmptyData(t *testing.T) {
	// Create empty raw data
	rawData := &OneNoteRawData{
		Notebooks: []msgraphmodels.Notebookable{},
		Sections:  make(map[string][]msgraphmodels.OnenoteSectionable),
		Pages:     make(map[string][]msgraphmodels.OnenotePageable),
		Content:   make(map[string][]byte),
	}

	// Mock the fetchOneNoteRawData method by creating a custom client
	mockClient := &mockClientForTesting{
		mockRawData: rawData,
	}

	// Test combineOneNoteData with empty data
	collection, err := mockClient.combineOneNoteDataForTesting(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify empty collection
	if collection == nil {
		t.Fatal("Expected non-nil collection")
	}
	if collection.Source != "OneNote" {
		t.Errorf("Expected source 'OneNote', got '%s'", collection.Source)
	}
	if collection.GetDocumentCount() != 0 {
		t.Errorf("Expected 0 documents, got %d", collection.GetDocumentCount())
	}
}

// TestOneNoteRawDataStructure tests the OneNoteRawData struct
func TestOneNoteRawDataStructure(t *testing.T) {
	rawData := &OneNoteRawData{
		Notebooks: []msgraphmodels.Notebookable{},
		Sections:  make(map[string][]msgraphmodels.OnenoteSectionable),
		Pages:     make(map[string][]msgraphmodels.OnenotePageable),
		Content:   make(map[string][]byte),
	}

	// Test that all fields are properly initialized
	if rawData.Notebooks == nil {
		t.Error("Expected non-nil Notebooks slice")
	}
	if rawData.Sections == nil {
		t.Error("Expected non-nil Sections map")
	}
	if rawData.Pages == nil {
		t.Error("Expected non-nil Pages map")
	}
	if rawData.Content == nil {
		t.Error("Expected non-nil Content map")
	}

	// Test that we can add data to the structure
	rawData.Sections["test-notebook"] = []msgraphmodels.OnenoteSectionable{}
	rawData.Pages["test-section"] = []msgraphmodels.OnenotePageable{}
	rawData.Content["test-page"] = []byte("test content")

	if len(rawData.Sections) != 1 {
		t.Errorf("Expected 1 section entry, got %d", len(rawData.Sections))
	}
	if len(rawData.Pages) != 1 {
		t.Errorf("Expected 1 page entry, got %d", len(rawData.Pages))
	}
	if len(rawData.Content) != 1 {
		t.Errorf("Expected 1 content entry, got %d", len(rawData.Content))
	}
}

// Helper functions for testing

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// createMockNotebook creates a mock notebook for testing
func createMockNotebook(id, name string) msgraphmodels.Notebookable {
	notebook := msgraphmodels.NewNotebook()
	notebook.SetId(&id)
	notebook.SetDisplayName(&name)
	return notebook
}

// createMockSection creates a mock section for testing
func createMockSection(id, name string) msgraphmodels.OnenoteSectionable {
	section := msgraphmodels.NewOnenoteSection()
	section.SetId(&id)
	section.SetDisplayName(&name)
	return section
}

// createMockPage creates a mock page for testing
func createMockPage(id, title string, createdAt time.Time) msgraphmodels.OnenotePageable {
	page := msgraphmodels.NewOnenotePage()
	page.SetId(&id)
	page.SetTitle(&title)
	if !createdAt.IsZero() {
		page.SetCreatedDateTime(&createdAt)
	}
	return page
}

// mockClientForTesting is a test helper that allows mocking of internal methods
type mockClientForTesting struct {
	*Client
	mockRawData *OneNoteRawData
	mockError   error
}

// combineOneNoteDataForTesting is a test helper that uses mock data
func (m *mockClientForTesting) combineOneNoteDataForTesting(ctx context.Context) (*types.DocumentCollection, error) {
	// Create document collection
	collection := types.NewDocumentCollection("OneNote")

	// Use mock data instead of fetching from API
	rawData := m.mockRawData
	if m.mockError != nil {
		return nil, m.mockError
	}

	// Process and combine the raw data into documents (same logic as real method)
	for _, notebook := range rawData.Notebooks {
		notebookID := getStringValue(notebook.GetId())
		sections, exists := rawData.Sections[notebookID]
		if !exists {
			continue
		}

		for _, section := range sections {
			sectionID := getStringValue(section.GetId())
			pages, exists := rawData.Pages[sectionID]
			if !exists {
				continue
			}

			for _, page := range pages {
				pageID := getStringValue(page.GetId())
				content, exists := rawData.Content[pageID]
				if !exists {
					continue
				}

				// Convert and process the content into a document
				doc, err := m.processPageContent(page, notebook, section, content)
				if err != nil {
					continue
				}

				// Add document to collection
				collection.AddDocument(doc)
			}
		}
	}

	return collection, nil
}

// processPageContent uses the real implementation for testing
func (m *mockClientForTesting) processPageContent(page msgraphmodels.OnenotePageable, notebook msgraphmodels.Notebookable, section msgraphmodels.OnenoteSectionable, content []byte) (types.Document, error) {
	// Create a real client instance for processing
	client := &Client{}
	return client.processPageContent(page, notebook, section, content)
}
