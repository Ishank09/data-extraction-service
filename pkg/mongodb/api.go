package mongodb

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Environment variable constants
const (
	MongoDBURIEnvVar        = "MONGODB_URI"
	MongoDBDatabaseEnvVar   = "MONGODB_DATABASE"
	MongoDBUsernameEnvVar   = "MONGODB_USERNAME"
	MongoDBPasswordEnvVar   = "MONGODB_PASSWORD"
	MongoDBAuthSourceEnvVar = "MONGODB_AUTH_SOURCE"
)

// Config represents the MongoDB configuration
type Config struct {
	MongoDB struct {
		URI      string
		Database string
		Username string
		Password string
	}
	Connection struct {
		Timeout                time.Duration
		ServerSelectionTimeout time.Duration
		MaxPoolSize            uint64
		MinPoolSize            uint64
		MaxConnIdleTime        time.Duration
	}
	Security struct {
		AuthSource string
		TLS        bool
	}
}

// NewConfig creates a new MongoDB configuration with default values
func NewConfig() *Config {
	config := &Config{}
	// MongoDB connection details must be provided via environment variables
	// config.MongoDB.URI = "" // No default - must be explicitly set
	// config.MongoDB.Database = "" // No default - must be explicitly set

	// Keep reasonable technical defaults for connection settings
	config.Connection.Timeout = 10 * time.Second
	config.Connection.ServerSelectionTimeout = 30 * time.Second
	config.Connection.MaxPoolSize = 100
	config.Connection.MinPoolSize = 5
	config.Connection.MaxConnIdleTime = 30 * time.Minute
	config.Security.AuthSource = "admin"
	config.Security.TLS = false
	return config
}

// LoadFromEnv loads configuration from environment variables
func (c *Config) LoadFromEnv() {
	if uri := os.Getenv(MongoDBURIEnvVar); uri != "" {
		c.MongoDB.URI = uri
	}
	if database := os.Getenv(MongoDBDatabaseEnvVar); database != "" {
		c.MongoDB.Database = database
	}
	if username := os.Getenv(MongoDBUsernameEnvVar); username != "" {
		c.MongoDB.Username = username
	}
	if password := os.Getenv(MongoDBPasswordEnvVar); password != "" {
		c.MongoDB.Password = password
	}
	if authSource := os.Getenv(MongoDBAuthSourceEnvVar); authSource != "" {
		c.Security.AuthSource = authSource
	}
}

// Interface defines the main CRUD operations interface
type Interface interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected(ctx context.Context) bool
	Ping(ctx context.Context) error
	Database(name string) *mongo.Database
	GetConfig() *Config

	// CREATE operations
	InsertOne(ctx context.Context, collection string, document interface{}) (*InsertOneResult, error)
	InsertMany(ctx context.Context, collection string, documents []interface{}) (*InsertManyResult, error)

	// READ operations
	FindOne(ctx context.Context, collection string, filter interface{}) *SingleResult
	Find(ctx context.Context, collection string, filter interface{}) (*Cursor, error)
	CountDocuments(ctx context.Context, collection string, filter interface{}) (int64, error)

	// UPDATE operations
	UpdateOne(ctx context.Context, collection string, filter interface{}, update interface{}) (*UpdateResult, error)
	UpdateMany(ctx context.Context, collection string, filter interface{}, update interface{}) (*UpdateResult, error)
	ReplaceOne(ctx context.Context, collection string, filter interface{}, replacement interface{}) (*UpdateResult, error)

	// DELETE operations
	DeleteOne(ctx context.Context, collection string, filter interface{}) (*DeleteResult, error)
	DeleteMany(ctx context.Context, collection string, filter interface{}) (*DeleteResult, error)
}

// Result types
type InsertOneResult struct {
	InsertedID interface{}
}

type InsertManyResult struct {
	InsertedIDs []interface{}
}

type UpdateResult struct {
	MatchedCount  int64
	ModifiedCount int64
	UpsertedCount int64
	UpsertedID    interface{}
}

type DeleteResult struct {
	DeletedCount int64
}

// IndexModel represents a MongoDB index model
type IndexModel struct {
	Keys    interface{}
	Options interface{}
}

// SingleResult wraps mongo.SingleResult
type SingleResult struct {
	result *mongo.SingleResult
}

func (s *SingleResult) Decode(v interface{}) error {
	return s.result.Decode(v)
}

func (s *SingleResult) Err() error {
	return s.result.Err()
}

// Cursor wraps mongo.Cursor
type Cursor struct {
	cursor *mongo.Cursor
}

func (c *Cursor) Next(ctx context.Context) bool {
	return c.cursor.Next(ctx)
}

func (c *Cursor) Decode(val interface{}) error {
	return c.cursor.Decode(val)
}

func (c *Cursor) Err() error {
	return c.cursor.Err()
}

func (c *Cursor) Close(ctx context.Context) error {
	return c.cursor.Close(ctx)
}

func (c *Cursor) All(ctx context.Context, results interface{}) error {
	return c.cursor.All(ctx, results)
}

// Client implements the Interface
type Client struct {
	client   *mongo.Client
	database *mongo.Database
	config   *Config
}

// NewClient creates a new MongoDB client
func NewClient(config *Config) Interface {
	return &Client{
		config: config,
	}
}

// Connect establishes a connection to MongoDB
func (c *Client) Connect(ctx context.Context) error {
	clientOptions := options.Client().ApplyURI(c.config.MongoDB.URI)

	// Set timeouts
	clientOptions.SetConnectTimeout(c.config.Connection.Timeout)
	clientOptions.SetServerSelectionTimeout(c.config.Connection.ServerSelectionTimeout)
	clientOptions.SetMaxConnIdleTime(c.config.Connection.MaxConnIdleTime)
	clientOptions.SetMaxPoolSize(c.config.Connection.MaxPoolSize)
	clientOptions.SetMinPoolSize(c.config.Connection.MinPoolSize)

	// Set authentication if provided
	if c.config.MongoDB.Username != "" && c.config.MongoDB.Password != "" {
		credential := options.Credential{
			Username:   c.config.MongoDB.Username,
			Password:   c.config.MongoDB.Password,
			AuthSource: c.config.Security.AuthSource,
		}
		clientOptions.SetAuth(credential)
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	c.client = client
	c.database = client.Database(c.config.MongoDB.Database)
	return nil
}

// Disconnect closes the connection to MongoDB
func (c *Client) Disconnect(ctx context.Context) error {
	if c.client == nil {
		return nil
	}

	err := c.client.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	c.client = nil
	c.database = nil
	return nil
}

// IsConnected checks if the client is connected to MongoDB
func (c *Client) IsConnected(ctx context.Context) bool {
	if c.client == nil {
		return false
	}

	err := c.client.Ping(ctx, nil)
	return err == nil
}

// Ping tests the connection to MongoDB
func (c *Client) Ping(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("client is not connected")
	}
	return c.client.Ping(ctx, nil)
}

// Database returns the specified database
func (c *Client) Database(name string) *mongo.Database {
	if c.client == nil {
		return nil
	}
	return c.client.Database(name)
}

// GetConfig returns the configuration
func (c *Client) GetConfig() *Config {
	return c.config
}

// InsertOne inserts a single document
func (c *Client) InsertOne(ctx context.Context, collection string, document interface{}) (*InsertOneResult, error) {
	result, err := c.database.Collection(collection).InsertOne(ctx, document)
	if err != nil {
		return nil, err
	}

	return &InsertOneResult{
		InsertedID: result.InsertedID,
	}, nil
}

// InsertMany inserts multiple documents
func (c *Client) InsertMany(ctx context.Context, collection string, documents []interface{}) (*InsertManyResult, error) {
	result, err := c.database.Collection(collection).InsertMany(ctx, documents)
	if err != nil {
		return nil, err
	}

	return &InsertManyResult{
		InsertedIDs: result.InsertedIDs,
	}, nil
}

// FindOne finds a single document
func (c *Client) FindOne(ctx context.Context, collection string, filter interface{}) *SingleResult {
	result := c.database.Collection(collection).FindOne(ctx, filter)
	return &SingleResult{result: result}
}

// Find finds multiple documents
func (c *Client) Find(ctx context.Context, collection string, filter interface{}) (*Cursor, error) {
	cursor, err := c.database.Collection(collection).Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &Cursor{cursor: cursor}, nil
}

// CountDocuments counts documents
func (c *Client) CountDocuments(ctx context.Context, collection string, filter interface{}) (int64, error) {
	return c.database.Collection(collection).CountDocuments(ctx, filter)
}

// UpdateOne updates a single document
func (c *Client) UpdateOne(ctx context.Context, collection string, filter interface{}, update interface{}) (*UpdateResult, error) {
	result, err := c.database.Collection(collection).UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	return &UpdateResult{
		MatchedCount:  result.MatchedCount,
		ModifiedCount: result.ModifiedCount,
		UpsertedCount: result.UpsertedCount,
		UpsertedID:    result.UpsertedID,
	}, nil
}

// UpdateMany updates multiple documents
func (c *Client) UpdateMany(ctx context.Context, collection string, filter interface{}, update interface{}) (*UpdateResult, error) {
	result, err := c.database.Collection(collection).UpdateMany(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	return &UpdateResult{
		MatchedCount:  result.MatchedCount,
		ModifiedCount: result.ModifiedCount,
		UpsertedCount: result.UpsertedCount,
		UpsertedID:    result.UpsertedID,
	}, nil
}

// ReplaceOne replaces a single document
func (c *Client) ReplaceOne(ctx context.Context, collection string, filter interface{}, replacement interface{}) (*UpdateResult, error) {
	result, err := c.database.Collection(collection).ReplaceOne(ctx, filter, replacement)
	if err != nil {
		return nil, err
	}

	return &UpdateResult{
		MatchedCount:  result.MatchedCount,
		ModifiedCount: result.ModifiedCount,
		UpsertedCount: result.UpsertedCount,
		UpsertedID:    result.UpsertedID,
	}, nil
}

// DeleteOne deletes a single document
func (c *Client) DeleteOne(ctx context.Context, collection string, filter interface{}) (*DeleteResult, error) {
	result, err := c.database.Collection(collection).DeleteOne(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &DeleteResult{
		DeletedCount: result.DeletedCount,
	}, nil
}

// DeleteMany deletes multiple documents
func (c *Client) DeleteMany(ctx context.Context, collection string, filter interface{}) (*DeleteResult, error) {
	result, err := c.database.Collection(collection).DeleteMany(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &DeleteResult{
		DeletedCount: result.DeletedCount,
	}, nil
}
