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

// fetchOneNoteRawData fetches all raw OneNote data from the API using flattened endpoints
// This approach works better with personal Microsoft accounts
func (c *Client) fetchOneNoteRawData(ctx context.Context) (*OneNoteRawData, error) {
	log.Printf("🚀 Starting OneNote data fetching process...")

	rawData := &OneNoteRawData{
		Sections: make(map[string][]msgraphmodels.OnenoteSectionable),
		Pages:    make(map[string][]msgraphmodels.OnenotePageable),
		Content:  make(map[string][]byte),
	}

	// Step 1: Fetch all notebooks using flattened endpoint
	log.Printf("🔍 Fetching OneNote notebooks...")
	var notebooks msgraphmodels.NotebookCollectionResponseable
	var err error

	if c.IsDelegatedAuth() {
		notebooks, err = c.graphClient.Me().Onenote().Notebooks().Get(ctx, nil)
	} else {
		userID := c.GetUserID()
		if userID == "" {
			return nil, fmt.Errorf("user ID is required for application authentication flow")
		}
		notebooks, err = c.graphClient.Users().ByUserId(userID).Onenote().Notebooks().Get(ctx, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch notebooks: %w", err)
	}

	if notebooks == nil || notebooks.GetValue() == nil {
		log.Printf("⚠️  No notebooks found")
		return rawData, nil // No notebooks found
	}

	rawData.Notebooks = notebooks.GetValue()
	log.Printf("✅ Found %d notebooks", len(rawData.Notebooks))

	for i, notebook := range rawData.Notebooks {
		notebookName := getStringValue(notebook.GetDisplayName())
		notebookID := getStringValue(notebook.GetId())
		log.Printf("  📚 Notebook %d: '%s' (ID: %s)", i+1, notebookName, notebookID)
	}

	// Step 2: Fetch all sections using flattened endpoint (instead of hierarchical)
	log.Printf("🔍 Fetching OneNote sections...")
	var allSections msgraphmodels.OnenoteSectionCollectionResponseable
	if c.IsDelegatedAuth() {
		allSections, err = c.graphClient.Me().Onenote().Sections().Get(ctx, nil)
	} else {
		userID := c.GetUserID()
		allSections, err = c.graphClient.Users().ByUserId(userID).Onenote().Sections().Get(ctx, nil)
	}

	if err != nil {
		log.Printf("❌ Failed to fetch sections: %v", err)
	} else if allSections != nil && allSections.GetValue() != nil {
		log.Printf("✅ Found %d sections", len(allSections.GetValue()))

		// Group sections by notebook ID
		for i, section := range allSections.GetValue() {
			sectionName := getStringValue(section.GetDisplayName())
			sectionID := getStringValue(section.GetId())

			// Get the parent notebook ID from the section
			parentNotebook := section.GetParentNotebook()
			if parentNotebook != nil {
				notebookID := getStringValue(parentNotebook.GetId())
				parentNotebookName := getStringValue(parentNotebook.GetDisplayName())

				log.Printf("  📂 Section %d: '%s' (ID: %s) in notebook '%s'", i+1, sectionName, sectionID, parentNotebookName)

				if notebookID != "" {
					if rawData.Sections[notebookID] == nil {
						rawData.Sections[notebookID] = []msgraphmodels.OnenoteSectionable{}
					}
					rawData.Sections[notebookID] = append(rawData.Sections[notebookID], section)
				}
			} else {
				log.Printf("  📂 Section %d: '%s' (ID: %s) - no parent notebook found", i+1, sectionName, sectionID)
			}
		}

		log.Printf("📊 Sections grouped by notebook:")
		for notebookID, sections := range rawData.Sections {
			log.Printf("  📚 Notebook %s has %d sections", notebookID, len(sections))
		}
	} else {
		log.Printf("⚠️  No sections found")
	}

	// Step 3: Fetch pages for each section individually (to avoid API limits)
	log.Printf("🔍 Fetching pages for each section...")
	totalSections := 0
	totalPages := 0

	for notebookID, sections := range rawData.Sections {
		log.Printf("📚 Processing notebook %s (%d sections)...", notebookID, len(sections))

		for sectionIndex, section := range sections {
			sectionID := getStringValue(section.GetId())
			sectionName := getStringValue(section.GetDisplayName())

			if sectionID == "" {
				log.Printf("  ⚠️  Section %d: '%s' - no ID found, skipping", sectionIndex+1, sectionName)
				continue
			}

			log.Printf("  🔍 Section %d/%d: Fetching pages for '%s' (ID: %s)...", sectionIndex+1, len(sections), sectionName, sectionID)
			totalSections++

			var pages msgraphmodels.OnenotePageCollectionResponseable
			if c.IsDelegatedAuth() {
				pages, err = c.graphClient.Me().Onenote().Sections().ByOnenoteSectionId(sectionID).Pages().Get(ctx, nil)
			} else {
				userID := c.GetUserID()
				pages, err = c.graphClient.Users().ByUserId(userID).Onenote().Sections().ByOnenoteSectionId(sectionID).Pages().Get(ctx, nil)
			}

			if err != nil {
				log.Printf("  ❌ Failed to fetch pages for section '%s' (ID: %s): %v", sectionName, sectionID, err)
				continue
			}

			if pages != nil && pages.GetValue() != nil {
				pageCount := len(pages.GetValue())
				totalPages += pageCount
				rawData.Pages[sectionID] = pages.GetValue()
				log.Printf("  ✅ Section '%s': Found %d pages", sectionName, pageCount)

				for i, page := range pages.GetValue() {
					pageTitle := getStringValue(page.GetTitle())
					pageID := getStringValue(page.GetId())
					log.Printf("    📄 Page %d: '%s' (ID: %s)", i+1, pageTitle, pageID)
				}
			} else {
				log.Printf("  ⚠️  Section '%s': No pages found", sectionName)
			}
		}
	}

	log.Printf("📊 Page fetching summary: %d sections processed, %d total pages found", totalSections, totalPages)

	// Step 4: Fetch content for each page
	log.Printf("🔍 Fetching content for each page...")
	totalContentPages := 0
	successfulContentPages := 0

	for sectionID, pages := range rawData.Pages {
		if len(pages) == 0 {
			continue
		}

		log.Printf("📂 Fetching content for %d pages in section %s...", len(pages), sectionID)

		for pageIndex, page := range pages {
			pageID := getStringValue(page.GetId())
			pageTitle := getStringValue(page.GetTitle())

			if pageID == "" {
				log.Printf("  ⚠️  Page %d: '%s' - no ID found, skipping", pageIndex+1, pageTitle)
				continue
			}

			log.Printf("  🔍 Page %d/%d: Fetching content for '%s' (ID: %s)...", pageIndex+1, len(pages), pageTitle, pageID)
			totalContentPages++

			var content []byte
			if c.IsDelegatedAuth() {
				content, err = c.graphClient.Me().Onenote().Pages().ByOnenotePageId(pageID).Content().Get(ctx, nil)
			} else {
				userID := c.GetUserID()
				content, err = c.graphClient.Users().ByUserId(userID).Onenote().Pages().ByOnenotePageId(pageID).Content().Get(ctx, nil)
			}

			if err != nil {
				log.Printf("  ❌ Failed to fetch content for page '%s' (ID: %s): %v", pageTitle, pageID, err)
				continue
			}

			rawData.Content[pageID] = content
			successfulContentPages++
			contentSize := len(content)
			log.Printf("  ✅ Page '%s': Got content (%d bytes)", pageTitle, contentSize)
		}
	}

	log.Printf("📊 Content fetching summary: %d/%d pages successfully retrieved", successfulContentPages, totalContentPages)

	// Final summary
	totalNotebooks := len(rawData.Notebooks)
	totalSectionsFound := 0
	totalPagesFound := 0
	totalContentFound := len(rawData.Content)

	for _, sections := range rawData.Sections {
		totalSectionsFound += len(sections)
	}
	for _, pages := range rawData.Pages {
		totalPagesFound += len(pages)
	}

	log.Printf("🎉 OneNote data fetching completed!")
	log.Printf("📊 Final summary:")
	log.Printf("  📚 Notebooks: %d", totalNotebooks)
	log.Printf("  📂 Sections: %d", totalSectionsFound)
	log.Printf("  📄 Pages: %d", totalPagesFound)
	log.Printf("  📝 Content retrieved: %d", totalContentFound)

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
