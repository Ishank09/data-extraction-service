package types

import (
	"time"
)

// DocumentCollection represents a collection of documents from any source
type DocumentCollection struct {
	Source        string     `json:"source"`
	FetchedAt     time.Time  `json:"fetched_at"`
	SchemaVersion string     `json:"schema_version"`
	Documents     []Document `json:"documents"`
}

// Document represents a single document from any source
type Document struct {
	ID                   string                 `json:"id"`
	Source               string                 `json:"source"`   // "onenote", "slack", "teams", "sharepoint", etc.
	Type                 string                 `json:"type"`     // "note", "chat", "file", "page", etc.
	Title                string                 `json:"title"`    // Generic title (was "Filename")
	Location             string                 `json:"location"` // Generic location (was "OriginalPath")
	CreatedAt            time.Time              `json:"created_at"`
	FetchedAt            time.Time              `json:"fetched_at"`
	VersionHash          string                 `json:"version_hash"`
	Language             string                 `json:"language"`
	TextChunkingStrategy string                 `json:"text_chunking_strategy"`
	Content              string                 `json:"content"`
	Metadata             map[string]interface{} `json:"metadata"` // Source-specific metadata
}

// NewDocumentCollection creates a new document collection
func NewDocumentCollection(source string) *DocumentCollection {
	return &DocumentCollection{
		Source:        source,
		FetchedAt:     time.Now(),
		SchemaVersion: "v1",
		Documents:     make([]Document, 0),
	}
}

// AddDocument adds a document to the collection
func (dc *DocumentCollection) AddDocument(doc Document) {
	dc.Documents = append(dc.Documents, doc)
}

// GetDocumentCount returns the number of documents in the collection
func (dc *DocumentCollection) GetDocumentCount() int {
	return len(dc.Documents)
}

// GetDocumentByID finds a document by its ID
func (dc *DocumentCollection) GetDocumentByID(id string) (*Document, bool) {
	for _, doc := range dc.Documents {
		if doc.ID == id {
			return &doc, true
		}
	}
	return nil, false
}

// GetDocumentsByLocation finds documents by their location (supports prefix matching)
func (dc *DocumentCollection) GetDocumentsByLocation(locationPrefix string) []Document {
	var matches []Document
	for _, doc := range dc.Documents {
		if len(doc.Location) >= len(locationPrefix) &&
			doc.Location[:len(locationPrefix)] == locationPrefix {
			matches = append(matches, doc)
		}
	}
	return matches
}

// GetDocumentsBySource finds documents by their source
func (dc *DocumentCollection) GetDocumentsBySource(source string) []Document {
	var matches []Document
	for _, doc := range dc.Documents {
		if doc.Source == source {
			matches = append(matches, doc)
		}
	}
	return matches
}

// GetDocumentsByType finds documents by their type
func (dc *DocumentCollection) GetDocumentsByType(docType string) []Document {
	var matches []Document
	for _, doc := range dc.Documents {
		if doc.Type == docType {
			matches = append(matches, doc)
		}
	}
	return matches
}
