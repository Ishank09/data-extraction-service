package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/ishank09/data-extraction-service/internal/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	DocumentsCollectionName           = "documents"
	DocumentCollectionsCollectionName = "document_collections"
)

// DocumentService handles document operations with MongoDB
type DocumentService struct {
	client Interface
}

// NewDocumentService creates a new document service
func NewDocumentService(client Interface) *DocumentService {
	return &DocumentService{
		client: client,
	}
}

// StoredDocument represents a document stored in MongoDB
type StoredDocument struct {
	ID                   primitive.ObjectID     `bson:"_id,omitempty" json:"_id,omitempty"`
	DocumentID           string                 `bson:"document_id" json:"document_id"`
	Source               string                 `bson:"source" json:"source"`
	Type                 string                 `bson:"type" json:"type"`
	Title                string                 `bson:"title" json:"title"`
	Location             string                 `bson:"location" json:"location"`
	CreatedAt            time.Time              `bson:"created_at" json:"created_at"`
	FetchedAt            time.Time              `bson:"fetched_at" json:"fetched_at"`
	StoredAt             time.Time              `bson:"stored_at" json:"stored_at"`
	VersionHash          string                 `bson:"version_hash" json:"version_hash"`
	Language             string                 `bson:"language" json:"language"`
	TextChunkingStrategy string                 `bson:"text_chunking_strategy" json:"text_chunking_strategy"`
	Content              string                 `bson:"content" json:"content"`
	Metadata             map[string]interface{} `bson:"metadata" json:"metadata"`
}

// StoredDocumentCollection represents a document collection stored in MongoDB
type StoredDocumentCollection struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Source        string             `bson:"source" json:"source"`
	FetchedAt     time.Time          `bson:"fetched_at" json:"fetched_at"`
	StoredAt      time.Time          `bson:"stored_at" json:"stored_at"`
	SchemaVersion string             `bson:"schema_version" json:"schema_version"`
	DocumentCount int                `bson:"document_count" json:"document_count"`
	DocumentIDs   []string           `bson:"document_ids" json:"document_ids"`
}

// StoreDocumentCollection stores a complete document collection in MongoDB
func (ds *DocumentService) StoreDocumentCollection(ctx context.Context, collection *types.DocumentCollection) (*StoreCollectionResult, error) {
	if collection == nil {
		return nil, fmt.Errorf("collection cannot be nil")
	}

	var documentIDs []string
	var storedDocuments []interface{}

	// Convert and prepare documents for storage
	for _, doc := range collection.Documents {
		storedDoc := &StoredDocument{
			DocumentID:           doc.ID,
			Source:               doc.Source,
			Type:                 doc.Type,
			Title:                doc.Title,
			Location:             doc.Location,
			CreatedAt:            doc.CreatedAt,
			FetchedAt:            doc.FetchedAt,
			StoredAt:             time.Now(),
			VersionHash:          doc.VersionHash,
			Language:             doc.Language,
			TextChunkingStrategy: doc.TextChunkingStrategy,
			Content:              doc.Content,
			Metadata:             doc.Metadata,
		}
		storedDocuments = append(storedDocuments, storedDoc)
		documentIDs = append(documentIDs, doc.ID)
	}

	// Insert documents if any exist
	var insertedDocumentIDs []interface{}
	if len(storedDocuments) > 0 {
		result, err := ds.client.InsertMany(ctx, DocumentsCollectionName, storedDocuments)
		if err != nil {
			return nil, fmt.Errorf("failed to store documents: %w", err)
		}
		insertedDocumentIDs = result.InsertedIDs
	}

	// Store collection metadata
	storedCollection := &StoredDocumentCollection{
		Source:        collection.Source,
		FetchedAt:     collection.FetchedAt,
		StoredAt:      time.Now(),
		SchemaVersion: collection.SchemaVersion,
		DocumentCount: len(collection.Documents),
		DocumentIDs:   documentIDs,
	}

	collectionResult, err := ds.client.InsertOne(ctx, DocumentCollectionsCollectionName, storedCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to store collection metadata: %w", err)
	}

	return &StoreCollectionResult{
		CollectionID:        collectionResult.InsertedID,
		InsertedDocumentIDs: insertedDocumentIDs,
		DocumentCount:       len(collection.Documents),
	}, nil
}

// GetDocuments retrieves documents from MongoDB with optional filtering
func (ds *DocumentService) GetDocuments(ctx context.Context, filter DocumentFilter) ([]StoredDocument, error) {
	mongoFilter := bson.M{}

	if filter.Source != "" {
		mongoFilter["source"] = filter.Source
	}
	if filter.Type != "" {
		mongoFilter["type"] = filter.Type
	}
	if filter.Title != "" {
		mongoFilter["title"] = bson.M{"$regex": filter.Title, "$options": "i"}
	}
	if !filter.FetchedAfter.IsZero() {
		mongoFilter["fetched_at"] = bson.M{"$gte": filter.FetchedAfter}
	}
	if !filter.FetchedBefore.IsZero() {
		if existing, ok := mongoFilter["fetched_at"]; ok {
			mongoFilter["fetched_at"] = bson.M{"$gte": existing.(bson.M)["$gte"], "$lte": filter.FetchedBefore}
		} else {
			mongoFilter["fetched_at"] = bson.M{"$lte": filter.FetchedBefore}
		}
	}

	opts := options.Find()
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	if filter.Skip > 0 {
		opts.SetSkip(int64(filter.Skip))
	}

	// Sort by fetched_at descending by default
	opts.SetSort(bson.M{"fetched_at": -1})

	cursor, err := ds.client.Find(ctx, DocumentsCollectionName, mongoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []StoredDocument
	for cursor.Next(ctx) {
		var doc StoredDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode document: %w", err)
		}
		documents = append(documents, doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return documents, nil
}

// GetDocumentCollections retrieves document collection metadata
func (ds *DocumentService) GetDocumentCollections(ctx context.Context, filter CollectionFilter) ([]StoredDocumentCollection, error) {
	mongoFilter := bson.M{}

	if filter.Source != "" {
		mongoFilter["source"] = filter.Source
	}
	if !filter.FetchedAfter.IsZero() {
		mongoFilter["fetched_at"] = bson.M{"$gte": filter.FetchedAfter}
	}
	if !filter.FetchedBefore.IsZero() {
		if existing, ok := mongoFilter["fetched_at"]; ok {
			mongoFilter["fetched_at"] = bson.M{"$gte": existing.(bson.M)["$gte"], "$lte": filter.FetchedBefore}
		} else {
			mongoFilter["fetched_at"] = bson.M{"$lte": filter.FetchedBefore}
		}
	}

	opts := options.Find()
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	if filter.Skip > 0 {
		opts.SetSkip(int64(filter.Skip))
	}

	// Sort by fetched_at descending by default
	opts.SetSort(bson.M{"fetched_at": -1})

	cursor, err := ds.client.Find(ctx, DocumentCollectionsCollectionName, mongoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to find document collections: %w", err)
	}
	defer cursor.Close(ctx)

	var collections []StoredDocumentCollection
	for cursor.Next(ctx) {
		var collection StoredDocumentCollection
		if err := cursor.Decode(&collection); err != nil {
			return nil, fmt.Errorf("failed to decode collection: %w", err)
		}
		collections = append(collections, collection)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return collections, nil
}

// GetDocumentStats returns statistics about stored documents
func (ds *DocumentService) GetDocumentStats(ctx context.Context) (*DocumentStats, error) {
	// Count total documents
	totalDocs, err := ds.client.CountDocuments(ctx, DocumentsCollectionName, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	// Count total collections
	totalCollections, err := ds.client.CountDocuments(ctx, DocumentCollectionsCollectionName, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count collections: %w", err)
	}

	// Get documents by source
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":   "$source",
				"count": bson.M{"$sum": 1},
			},
		},
	}

	// For aggregation, we need to use the raw MongoDB database
	db := ds.client.Database(ds.client.GetConfig().MongoDB.Database)
	cursor, err := db.Collection(DocumentsCollectionName).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate documents by source: %w", err)
	}
	defer cursor.Close(ctx)

	documentsBySources := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode aggregation result: %w", err)
		}
		documentsBySources[result.ID] = result.Count
	}

	return &DocumentStats{
		TotalDocuments:    totalDocs,
		TotalCollections:  totalCollections,
		DocumentsBySource: documentsBySources,
	}, nil
}

// DeleteOldDocuments deletes documents older than the specified duration
func (ds *DocumentService) DeleteOldDocuments(ctx context.Context, olderThan time.Duration) (*DeleteResult, error) {
	cutoffTime := time.Now().Add(-olderThan)
	filter := bson.M{"fetched_at": bson.M{"$lt": cutoffTime}}

	result, err := ds.client.DeleteMany(ctx, DocumentsCollectionName, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old documents: %w", err)
	}

	return result, nil
}

// Filter types
type DocumentFilter struct {
	Source        string    `json:"source,omitempty"`
	Type          string    `json:"type,omitempty"`
	Title         string    `json:"title,omitempty"` // Supports regex search
	FetchedAfter  time.Time `json:"fetched_after,omitempty"`
	FetchedBefore time.Time `json:"fetched_before,omitempty"`
	Limit         int       `json:"limit,omitempty"`
	Skip          int       `json:"skip,omitempty"`
}

type CollectionFilter struct {
	Source        string    `json:"source,omitempty"`
	FetchedAfter  time.Time `json:"fetched_after,omitempty"`
	FetchedBefore time.Time `json:"fetched_before,omitempty"`
	Limit         int       `json:"limit,omitempty"`
	Skip          int       `json:"skip,omitempty"`
}

// Result types
type StoreCollectionResult struct {
	CollectionID        interface{}   `json:"collection_id"`
	InsertedDocumentIDs []interface{} `json:"inserted_document_ids"`
	DocumentCount       int           `json:"document_count"`
}

type DocumentStats struct {
	TotalDocuments    int64            `json:"total_documents"`
	TotalCollections  int64            `json:"total_collections"`
	DocumentsBySource map[string]int64 `json:"documents_by_source"`
}
