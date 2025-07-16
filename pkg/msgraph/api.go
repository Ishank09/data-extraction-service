package msgraph

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"

	"github.com/ishank09/data-extraction-service/internal/types"
)

// Interface defines the main interface for Microsoft Graph data extraction services
type Interface interface {
	// OneNote data extraction - returns all OneNote pages as JSON array
	GetOneNoteDataAsJSON(ctx context.Context) (*types.DocumentCollection, error)
}

// AuthType represents the type of authentication being used
type AuthType int

const (
	// AuthTypeApplication represents client credentials flow (application permissions)
	AuthTypeApplication AuthType = iota
	// AuthTypeDelegated represents delegated flow (user permissions)
	AuthTypeDelegated
)

// ConcurrencyConfig defines limits for concurrent operations
type ConcurrencyConfig struct {
	MaxSectionWorkers int // Maximum concurrent section fetchers
	MaxContentWorkers int // Maximum concurrent content fetchers
}

// DefaultConcurrencyConfig returns sensible defaults for API rate limiting
func DefaultConcurrencyConfig() ConcurrencyConfig {
	return ConcurrencyConfig{
		MaxSectionWorkers: 5,  // Conservative limit for section processing
		MaxContentWorkers: 10, // Higher limit for content fetching as it's the main bottleneck
	}
}

// Config represents the configuration for Microsoft Graph client
type Config struct {
	ClientID      string
	ClientSecret  string
	TenantID      string
	LoginEndpoint string
	Scopes        []string
	// OneNote concurrency configuration
	OneNoteConcurrency *ConcurrencyConfig
}

// Client represents the base Microsoft Graph client
type Client struct {
	clientID      string
	clientSecret  string
	tenantID      string
	loginEndpoint string
	scopes        []string
	graphClient   *msgraph.GraphServiceClient
	authType      AuthType // Track authentication type
	userID        string   // User ID for application flow
	// OneNote concurrency configuration
	oneNoteConcurrency ConcurrencyConfig
}

// NewClient creates a new Microsoft Graph client with service credentials (client credentials flow)
func NewClient(config Config) (*Client, error) {
	// Set default scopes if not provided
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			// "Notes.Read",
			// "Notes.Read.All",
			// "User.Read",
			"https://graph.microsoft.com/.default",
		}
	}

	// Create credentials
	credential, err := azidentity.NewClientSecretCredential(config.TenantID, config.ClientID, config.ClientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %w", err)
	}

	// Create graph client
	graphClient, err := msgraph.NewGraphServiceClientWithCredentials(credential, scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to create graph client: %w", err)
	}

	// Set OneNote concurrency configuration
	concurrencyConfig := DefaultConcurrencyConfig()
	if config.OneNoteConcurrency != nil {
		concurrencyConfig = *config.OneNoteConcurrency
	}

	return &Client{
		clientID:           config.ClientID,
		clientSecret:       config.ClientSecret,
		tenantID:           config.TenantID,
		loginEndpoint:      config.LoginEndpoint,
		scopes:             scopes,
		graphClient:        graphClient,
		authType:           AuthTypeApplication,
		oneNoteConcurrency: concurrencyConfig,
	}, nil
}

// NewClientWithToken creates a new Microsoft Graph client using an existing access token (from auth service)
func NewClientWithToken(accessToken string) (*Client, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token cannot be empty")
	}

	// Create token credential from the provided access token
	tokenCredential := &staticTokenCredential{token: accessToken}

	// Default scopes for token-based client
	scopes := []string{
		"Notes.Read",
		"Notes.Read.All",
		"User.Read",
	}

	// Create graph client with the token
	graphClient, err := msgraph.NewGraphServiceClientWithCredentials(tokenCredential, scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to create graph client with token: %w", err)
	}

	return &Client{
		graphClient:        graphClient,
		scopes:             scopes,
		authType:           AuthTypeDelegated,
		oneNoteConcurrency: DefaultConcurrencyConfig(), // Use default for token-based auth
		// Note: clientID, clientSecret, tenantID, loginEndpoint are not needed for token-based auth
	}, nil
}

// NewClientWithUserID creates a new Microsoft Graph client with service credentials for a specific user
func NewClientWithUserID(config Config, userID string) (*Client, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	client.userID = userID
	return client, nil
}

// IsDelegatedAuth returns true if the client uses delegated authentication flow
func (c *Client) IsDelegatedAuth() bool {
	return c.authType == AuthTypeDelegated
}

// GetUserID returns the user ID for application flow
func (c *Client) GetUserID() string {
	return c.userID
}

// GetGraphClient returns the underlying Microsoft Graph client
func (c *Client) GetGraphClient() *msgraph.GraphServiceClient {
	return c.graphClient
}

// staticTokenCredential implements the azcore.TokenCredential interface for pre-existing tokens
type staticTokenCredential struct {
	token string
}

// GetToken implements azcore.TokenCredential interface
func (s *staticTokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{
		Token:     s.token,
		ExpiresOn: time.Now().Add(time.Hour), // Assume token is valid for 1 hour
	}, nil
}
