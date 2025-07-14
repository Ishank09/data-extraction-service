package msgraph

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	msgraphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"

	"github.com/ishank09/data-extraction-service/internal/types"
	"github.com/ishank09/data-extraction-service/internal/utils"
)

// OneNoteRawData represents raw data fetched from OneNote API
type OneNoteRawData struct {
	Notebooks []msgraphmodels.Notebookable
	Sections  map[string][]msgraphmodels.OnenoteSectionable
	Pages     map[string][]msgraphmodels.OnenotePageable
	Content   map[string][]byte
}

// ============================================================================
// LAYER 1: Interface Implementation - Public API
// ============================================================================

// GetOneNoteDataAsJSON implements the Interface method to get all OneNote pages as JSON array
// This is the public interface that delegates to the data combination layer
func (c *Client) GetOneNoteDataAsJSON(ctx context.Context) (*types.DocumentCollection, error) {
	return c.combineOneNoteData(ctx)
}

// ============================================================================
// LAYER 2: Business Logic - Data Combination & Orchestration
// ============================================================================

// combineOneNoteData orchestrates the data fetching and combines it into a DocumentCollection
// This layer handles the business logic of how OneNote data should be processed and combined
func (c *Client) combineOneNoteData(ctx context.Context) (*types.DocumentCollection, error) {
	// Create document collection
	collection := types.NewDocumentCollection("OneNote")

	// Fetch raw OneNote data
	rawData, err := c.fetchOneNoteRawData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OneNote data: %w", err)
	}

	// Process and combine the raw data into documents
	for _, notebook := range rawData.Notebooks {
		notebookID := getStringValue(notebook.GetId())
		sections, exists := rawData.Sections[notebookID]
		if !exists {
			log.Printf("No sections found for notebook %s", notebookID)
			continue
		}

		for _, section := range sections {
			sectionID := getStringValue(section.GetId())
			pages, exists := rawData.Pages[sectionID]
			if !exists {
				log.Printf("No pages found for section %s", sectionID)
				continue
			}

			for _, page := range pages {
				pageID := getStringValue(page.GetId())
				content, exists := rawData.Content[pageID]
				if !exists {
					log.Printf("No content found for page %s", pageID)
					continue
				}

				// Convert and process the content into a document
				doc, err := c.processPageContent(page, notebook, section, content)
				if err != nil {
					log.Printf("Error processing page %s: %v", pageID, err)
					continue
				}

				// Add document to collection
				collection.AddDocument(doc)
			}
		}
	}

	return collection, nil
}

// processPageContent converts a OneNote page and its content into a Document
func (c *Client) processPageContent(page msgraphmodels.OnenotePageable, notebook msgraphmodels.Notebookable, section msgraphmodels.OnenoteSectionable, content []byte) (types.Document, error) {
	// Convert HTML content to structured JSON format using utils function
	contentJSON, err := utils.BytesToJSON(content)
	if err != nil {
		return types.Document{}, fmt.Errorf("failed to convert content to JSON: %w", err)
	}

	// Extract text content for the document
	textContent := ""
	if jsonContent, ok := contentJSON["content"].(string); ok {
		textContent = jsonContent
	}

	// Create document location (was OriginalPath)
	location := fmt.Sprintf("OneNote/%s/%s",
		getStringValue(notebook.GetDisplayName()),
		getStringValue(section.GetDisplayName()))

	// Create version hash
	hash := sha256.Sum256([]byte(textContent))
	versionHash := fmt.Sprintf("sha256:%x", hash)

	// Create metadata with OneNote-specific information
	metadata := map[string]interface{}{
		"notebook_id":     getStringValue(notebook.GetId()),
		"notebook_name":   getStringValue(notebook.GetDisplayName()),
		"section_id":      getStringValue(section.GetId()),
		"section_name":    getStringValue(section.GetDisplayName()),
		"page_id":         getStringValue(page.GetId()),
		"content_format":  "html",
		"has_images":      contentJSON["has_images"],
		"word_count":      contentJSON["word_count"],
		"character_count": contentJSON["character_count"],
	}

	// Create and return document
	return types.Document{
		ID:                   getStringValue(page.GetId()),
		Source:               "onenote",
		Type:                 "page",
		Title:                getStringValue(page.GetTitle()),
		Location:             location,
		CreatedAt:            getTimeValue(page.GetCreatedDateTime()),
		FetchedAt:            time.Now(),
		VersionHash:          versionHash,
		Language:             "en", // Default, could be enhanced
		TextChunkingStrategy: "page_based",
		Content:              textContent,
		Metadata:             metadata,
	}, nil
}

// ============================================================================
// LAYER 3: Data Source - Raw Data Fetching
// ============================================================================

// fetchOneNoteRawData fetches all raw OneNote data from the API
// This layer is responsible for the actual API calls and data retrieval
func (c *Client) fetchOneNoteRawData(ctx context.Context) (*OneNoteRawData, error) {
	rawData := &OneNoteRawData{
		Sections: make(map[string][]msgraphmodels.OnenoteSectionable),
		Pages:    make(map[string][]msgraphmodels.OnenotePageable),
		Content:  make(map[string][]byte),
	}

	// Fetch notebooks
	notebooks, err := c.graphClient.Me().Onenote().Notebooks().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notebooks: %w", err)
	}

	if notebooks == nil || notebooks.GetValue() == nil {
		return rawData, nil // No notebooks found
	}

	rawData.Notebooks = notebooks.GetValue()

	// Fetch sections for each notebook
	for _, notebook := range rawData.Notebooks {
		notebookID := getStringValue(notebook.GetId())
		if notebookID == "" {
			continue
		}

		sections, err := c.graphClient.Me().Onenote().Notebooks().ByNotebookId(notebookID).Sections().Get(ctx, nil)
		if err != nil {
			log.Printf("Failed to fetch sections for notebook %s: %v", notebookID, err)
			continue
		}

		if sections != nil && sections.GetValue() != nil {
			rawData.Sections[notebookID] = sections.GetValue()
		}
	}

	// Fetch pages for each section
	for notebookID, sections := range rawData.Sections {
		for _, section := range sections {
			sectionID := getStringValue(section.GetId())
			if sectionID == "" {
				continue
			}

			pages, err := c.graphClient.Me().Onenote().Notebooks().ByNotebookId(notebookID).Sections().ByOnenoteSectionId(sectionID).Pages().Get(ctx, nil)
			if err != nil {
				log.Printf("Failed to fetch pages for section %s: %v", sectionID, err)
				continue
			}

			if pages != nil && pages.GetValue() != nil {
				rawData.Pages[sectionID] = pages.GetValue()
			}
		}
	}

	// Fetch content for each page
	for _, pages := range rawData.Pages {
		for _, page := range pages {
			pageID := getStringValue(page.GetId())
			if pageID == "" {
				continue
			}

			content, err := c.graphClient.Me().Onenote().Pages().ByOnenotePageId(pageID).Content().Get(ctx, nil)
			if err != nil {
				log.Printf("Failed to fetch content for page %s: %v", pageID, err)
				continue
			}

			rawData.Content[pageID] = content
		}
	}

	return rawData, nil
}

// ============================================================================
// Helper functions
// ============================================================================

func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func getTimeValue(ptr *time.Time) time.Time {
	if ptr == nil {
		return time.Now()
	}
	return *ptr
}
