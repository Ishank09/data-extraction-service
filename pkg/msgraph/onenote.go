package msgraph

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"sync"
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

// SectionJob represents a section processing job
type SectionJob struct {
	NotebookID    string
	Section       msgraphmodels.OnenoteSectionable
	SectionIndex  int
	TotalSections int
}

// SectionResult represents the result of section processing
type SectionResult struct {
	SectionID string
	Pages     []msgraphmodels.OnenotePageable
	Error     error
}

// ContentJob represents a content fetching job
type ContentJob struct {
	PageID     string
	PageTitle  string
	PageIndex  int
	TotalPages int
}

// ContentResult represents the result of content fetching
type ContentResult struct {
	PageID  string
	Content []byte
	Error   error
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

	// Fetch raw OneNote data using concurrent implementation
	rawData, err := c.fetchOneNoteRawDataConcurrent(ctx)
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
// LAYER 3: Data Source - Concurrent Raw Data Fetching
// ============================================================================

// fetchOneNoteRawDataConcurrent fetches all raw OneNote data from the API using concurrent workers
// This implementation significantly improves performance by parallelizing API calls
func (c *Client) fetchOneNoteRawDataConcurrent(ctx context.Context) (*OneNoteRawData, error) {
	return c.fetchOneNoteRawDataConcurrentWithConfig(ctx, c.oneNoteConcurrency)
}

// fetchOneNoteRawDataConcurrentWithConfig fetches OneNote data with custom concurrency configuration
func (c *Client) fetchOneNoteRawDataConcurrentWithConfig(ctx context.Context, config ConcurrencyConfig) (*OneNoteRawData, error) {
	log.Printf("🚀 Starting concurrent OneNote data fetching process...")
	log.Printf("⚙️  Concurrency config: %d section workers, %d content workers", config.MaxSectionWorkers, config.MaxContentWorkers)

	rawData := &OneNoteRawData{
		Sections: make(map[string][]msgraphmodels.OnenoteSectionable),
		Pages:    make(map[string][]msgraphmodels.OnenotePageable),
		Content:  make(map[string][]byte),
	}

	// Use mutex to protect shared data structures
	var dataMutex sync.RWMutex

	// Step 1: Fetch all notebooks (sequential as it's typically few items)
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
		return rawData, nil
	}

	rawData.Notebooks = notebooks.GetValue()
	log.Printf("✅ Found %d notebooks", len(rawData.Notebooks))

	// Step 2: Fetch all sections (sequential as it's a single API call)
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
		return rawData, nil
	}

	if allSections == nil || allSections.GetValue() == nil {
		log.Printf("⚠️  No sections found")
		return rawData, nil
	}

	// Group sections by notebook ID
	for _, section := range allSections.GetValue() {
		parentNotebook := section.GetParentNotebook()
		if parentNotebook != nil {
			notebookID := getStringValue(parentNotebook.GetId())
			if notebookID != "" {
				if rawData.Sections[notebookID] == nil {
					rawData.Sections[notebookID] = []msgraphmodels.OnenoteSectionable{}
				}
				rawData.Sections[notebookID] = append(rawData.Sections[notebookID], section)
			}
		}
	}

	log.Printf("✅ Found %d sections grouped by notebook", len(allSections.GetValue()))

	// Step 3: Concurrent page fetching for each section
	log.Printf("🔍 Starting concurrent page fetching for sections...")

	// Collect all section jobs
	var sectionJobs []SectionJob
	for notebookID, sections := range rawData.Sections {
		for i, section := range sections {
			sectionJobs = append(sectionJobs, SectionJob{
				NotebookID:    notebookID,
				Section:       section,
				SectionIndex:  i + 1,
				TotalSections: len(sections),
			})
		}
	}

	if len(sectionJobs) == 0 {
		log.Printf("⚠️  No sections to process")
		return rawData, nil
	}

	// Create channels for section worker pool
	sectionJobChan := make(chan SectionJob, len(sectionJobs))
	sectionResultChan := make(chan SectionResult, len(sectionJobs))

	// Start section workers
	var sectionWG sync.WaitGroup
	for i := 0; i < config.MaxSectionWorkers; i++ {
		sectionWG.Add(1)
		go c.sectionWorker(ctx, &sectionWG, sectionJobChan, sectionResultChan)
	}

	// Send section jobs
	for _, job := range sectionJobs {
		sectionJobChan <- job
	}
	close(sectionJobChan)

	// Wait for all section workers to complete
	go func() {
		sectionWG.Wait()
		close(sectionResultChan)
	}()

	// Collect section results
	var sectionErrors []error
	totalPages := 0
	for result := range sectionResultChan {
		if result.Error != nil {
			sectionErrors = append(sectionErrors, result.Error)
			log.Printf("❌ Section processing error: %v", result.Error)
			continue
		}

		dataMutex.Lock()
		rawData.Pages[result.SectionID] = result.Pages
		dataMutex.Unlock()

		totalPages += len(result.Pages)
		log.Printf("✅ Section %s: Found %d pages", result.SectionID, len(result.Pages))
	}

	log.Printf("📊 Concurrent page fetching completed: %d total pages found", totalPages)

	// Step 4: Concurrent content fetching for all pages
	log.Printf("🔍 Starting concurrent content fetching for pages...")

	// Collect all content jobs
	var contentJobs []ContentJob
	pageIndex := 0
	dataMutex.RLock()
	for _, pages := range rawData.Pages {
		for _, page := range pages {
			pageID := getStringValue(page.GetId())
			pageTitle := getStringValue(page.GetTitle())
			if pageID != "" {
				contentJobs = append(contentJobs, ContentJob{
					PageID:     pageID,
					PageTitle:  pageTitle,
					PageIndex:  pageIndex + 1,
					TotalPages: totalPages,
				})
				pageIndex++
			}
		}
	}
	dataMutex.RUnlock()

	if len(contentJobs) == 0 {
		log.Printf("⚠️  No pages to fetch content for")
		return rawData, nil
	}

	// Create channels for content worker pool
	contentJobChan := make(chan ContentJob, len(contentJobs))
	contentResultChan := make(chan ContentResult, len(contentJobs))

	// Start content workers
	var contentWG sync.WaitGroup
	for i := 0; i < config.MaxContentWorkers; i++ {
		contentWG.Add(1)
		go c.contentWorker(ctx, &contentWG, contentJobChan, contentResultChan)
	}

	// Send content jobs
	for _, job := range contentJobs {
		contentJobChan <- job
	}
	close(contentJobChan)

	// Wait for all content workers to complete
	go func() {
		contentWG.Wait()
		close(contentResultChan)
	}()

	// Collect content results
	var contentErrors []error
	successfulContent := 0
	for result := range contentResultChan {
		if result.Error != nil {
			contentErrors = append(contentErrors, result.Error)
			log.Printf("❌ Content fetching error for page %s: %v", result.PageID, result.Error)
			continue
		}

		dataMutex.Lock()
		rawData.Content[result.PageID] = result.Content
		dataMutex.Unlock()

		successfulContent++
	}

	log.Printf("📊 Concurrent content fetching completed: %d/%d pages successful", successfulContent, len(contentJobs))

	// Log any errors but don't fail the entire operation
	if len(sectionErrors) > 0 {
		log.Printf("⚠️  Section errors encountered: %d", len(sectionErrors))
	}
	if len(contentErrors) > 0 {
		log.Printf("⚠️  Content errors encountered: %d", len(contentErrors))
	}

	// Final summary
	dataMutex.RLock()
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
	dataMutex.RUnlock()

	log.Printf("🎉 Concurrent OneNote data fetching completed!")
	log.Printf("📊 Final summary:")
	log.Printf("  📚 Notebooks: %d", totalNotebooks)
	log.Printf("  📂 Sections: %d", totalSectionsFound)
	log.Printf("  📄 Pages: %d", totalPagesFound)
	log.Printf("  📝 Content retrieved: %d", totalContentFound)
	log.Printf("  ⚡ Performance: Used %d section workers, %d content workers", config.MaxSectionWorkers, config.MaxContentWorkers)

	return rawData, nil
}

// sectionWorker processes section jobs concurrently
func (c *Client) sectionWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan SectionJob, results chan<- SectionResult) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			results <- SectionResult{Error: ctx.Err()}
			return
		default:
		}

		sectionID := getStringValue(job.Section.GetId())
		sectionName := getStringValue(job.Section.GetDisplayName())

		if sectionID == "" {
			results <- SectionResult{
				SectionID: sectionID,
				Error:     fmt.Errorf("section %s has no ID", sectionName),
			}
			continue
		}

		log.Printf("  🔍 Worker fetching pages for section '%s' (ID: %s)...", sectionName, sectionID)

		var pages msgraphmodels.OnenotePageCollectionResponseable
		var err error

		if c.IsDelegatedAuth() {
			pages, err = c.graphClient.Me().Onenote().Sections().ByOnenoteSectionId(sectionID).Pages().Get(ctx, nil)
		} else {
			userID := c.GetUserID()
			pages, err = c.graphClient.Users().ByUserId(userID).Onenote().Sections().ByOnenoteSectionId(sectionID).Pages().Get(ctx, nil)
		}

		if err != nil {
			results <- SectionResult{
				SectionID: sectionID,
				Error:     fmt.Errorf("failed to fetch pages for section %s: %w", sectionName, err),
			}
			continue
		}

		var pageList []msgraphmodels.OnenotePageable
		if pages != nil && pages.GetValue() != nil {
			pageList = pages.GetValue()
		}

		results <- SectionResult{
			SectionID: sectionID,
			Pages:     pageList,
			Error:     nil,
		}
	}
}

// contentWorker processes content jobs concurrently
func (c *Client) contentWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan ContentJob, results chan<- ContentResult) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			results <- ContentResult{Error: ctx.Err()}
			return
		default:
		}

		log.Printf("  🔍 Worker fetching content for page '%s' (ID: %s)...", job.PageTitle, job.PageID)

		var content []byte
		var err error

		if c.IsDelegatedAuth() {
			content, err = c.graphClient.Me().Onenote().Pages().ByOnenotePageId(job.PageID).Content().Get(ctx, nil)
		} else {
			userID := c.GetUserID()
			content, err = c.graphClient.Users().ByUserId(userID).Onenote().Pages().ByOnenotePageId(job.PageID).Content().Get(ctx, nil)
		}

		if err != nil {
			results <- ContentResult{
				PageID: job.PageID,
				Error:  fmt.Errorf("failed to fetch content for page %s: %w", job.PageTitle, err),
			}
			continue
		}

		results <- ContentResult{
			PageID:  job.PageID,
			Content: content,
			Error:   nil,
		}
	}
}

// ============================================================================
// Legacy Sequential Implementation (kept for fallback)
// ============================================================================

// fetchOneNoteRawData fetches all raw OneNote data from the API using flattened endpoints
// This approach works better with personal Microsoft accounts
// DEPRECATED: Use fetchOneNoteRawDataConcurrent for better performance
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
