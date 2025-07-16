package mongodb

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func TestNewClient(t *testing.T) {
	config := &Config{
		MongoDB: struct {
			URI      string
			Database string
			Username string
			Password string
		}{
			URI:      "mongodb://localhost:27017",
			Database: "test_db",
		},
		Connection: struct {
			Timeout                time.Duration
			ServerSelectionTimeout time.Duration
			MaxPoolSize            uint64
			MinPoolSize            uint64
			MaxConnIdleTime        time.Duration
		}{
			Timeout:     10 * time.Second,
			MaxPoolSize: 100,
			MinPoolSize: 5,
		},
		Security: struct {
			AuthSource string
			TLS        bool
		}{
			AuthSource: "admin",
			TLS:        false,
		},
	}

	client := NewClient(config)
	if client == nil {
		t.Error("NewClient() returned nil")
	}
	if client.GetConfig() != config {
		t.Error("Client config not set correctly")
	}
	ctx := context.Background()
	if client.IsConnected(ctx) {
		t.Error("Client should not be connected initially")
	}
}

func TestNewConfig(t *testing.T) {
	// Set test environment variables
	originalVars := make(map[string]string)
	testVars := map[string]string{
		MongoDBURIEnvVar:      "mongodb://test:27017",
		MongoDBDatabaseEnvVar: "test_db",
		MongoDBUsernameEnvVar: "testuser",
		MongoDBPasswordEnvVar: "testpass",
	}

	// Save and set test values
	for key, value := range testVars {
		originalVars[key] = os.Getenv(key)
		os.Setenv(key, value)
	}

	// Restore original values
	defer func() {
		for key, value := range originalVars {
			os.Setenv(key, value)
		}
	}()

	config := NewConfig()
	config.LoadFromEnv()

	if config.MongoDB.URI != "mongodb://test:27017" {
		t.Errorf("Expected URI 'mongodb://test:27017', got '%s'", config.MongoDB.URI)
	}
	if config.MongoDB.Database != "test_db" {
		t.Errorf("Expected Database 'test_db', got '%s'", config.MongoDB.Database)
	}
	if config.MongoDB.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", config.MongoDB.Username)
	}
	if config.MongoDB.Password != "testpass" {
		t.Errorf("Expected Password 'testpass', got '%s'", config.MongoDB.Password)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Clear environment variables
	envVars := []string{
		MongoDBURIEnvVar, MongoDBDatabaseEnvVar, MongoDBUsernameEnvVar,
		MongoDBPasswordEnvVar, MongoDBAuthSourceEnvVar,
	}

	originalValues := make(map[string]string)
	for _, envVar := range envVars {
		originalValues[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	defer func() {
		for envVar, value := range originalValues {
			os.Setenv(envVar, value)
		}
	}()

	config := NewConfig()
	if config.MongoDB.URI != "mongodb://localhost:27017" {
		t.Errorf("Expected default URI 'mongodb://localhost:27017', got '%s'", config.MongoDB.URI)
	}
	if config.MongoDB.Database != "test" {
		t.Errorf("Expected default Database 'test', got '%s'", config.MongoDB.Database)
	}
	if config.Security.AuthSource != "admin" {
		t.Errorf("Expected default AuthSource 'admin', got '%s'", config.Security.AuthSource)
	}
	if config.Connection.MaxPoolSize != 100 {
		t.Errorf("Expected default MaxPoolSize 100, got %d", config.Connection.MaxPoolSize)
	}
}

func TestClientConnectionStatus(t *testing.T) {
	config := NewConfig()
	client := NewClient(config)
	ctx := context.Background()

	// Initially not connected
	if client.IsConnected(ctx) {
		t.Error("Expected client to be disconnected initially")
	}

	// Test ping without connection should fail
	err := client.Ping(ctx)
	if err == nil {
		t.Error("Expected ping to fail when not connected")
	}

	// Test disconnect when not connected should not error
	err = client.Disconnect(ctx)
	if err != nil {
		t.Errorf("Disconnect should not error when not connected: %v", err)
	}

	// Test Database when not connected should return nil
	db := client.Database("test")
	if db != nil {
		t.Error("Database should return nil when not connected")
	}
}

func TestResultTypes(t *testing.T) {
	// Test InsertOneResult
	insertResult := &InsertOneResult{InsertedID: "test_id"}
	if insertResult.InsertedID != "test_id" {
		t.Errorf("Expected InsertedID 'test_id', got %v", insertResult.InsertedID)
	}

	// Test InsertManyResult
	insertManyResult := &InsertManyResult{InsertedIDs: []interface{}{"id1", "id2"}}
	if len(insertManyResult.InsertedIDs) != 2 {
		t.Errorf("Expected 2 InsertedIDs, got %d", len(insertManyResult.InsertedIDs))
	}

	// Test UpdateResult
	updateResult := &UpdateResult{
		MatchedCount:  1,
		ModifiedCount: 1,
		UpsertedCount: 0,
		UpsertedID:    nil,
	}
	if updateResult.MatchedCount != 1 {
		t.Errorf("Expected MatchedCount 1, got %d", updateResult.MatchedCount)
	}

	// Test DeleteResult
	deleteResult := &DeleteResult{DeletedCount: 1}
	if deleteResult.DeletedCount != 1 {
		t.Errorf("Expected DeletedCount 1, got %d", deleteResult.DeletedCount)
	}
}

func TestIndexModel(t *testing.T) {
	keys := bson.D{{Key: "email", Value: 1}}
	model := IndexModel{
		Keys:    keys,
		Options: nil,
	}

	keysDoc, ok := model.Keys.(bson.D)
	if !ok {
		t.Error("Expected keys to be bson.D")
	}
	if len(keysDoc) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keysDoc))
	}
	if keysDoc[0].Key != "email" {
		t.Errorf("Expected key 'email', got '%s'", keysDoc[0].Key)
	}
	if keysDoc[0].Value != 1 {
		t.Errorf("Expected value 1, got %v", keysDoc[0].Value)
	}
}

// Benchmark tests
func BenchmarkNewClient(b *testing.B) {
	config := NewConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := NewClient(config)
		if client == nil {
			b.Error("NewClient returned nil")
		}
	}
}

func BenchmarkNewConfig(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := NewConfig()
		if config == nil {
			b.Error("NewConfig returned nil")
		}
	}
}
