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

// Config represents the configuration for Microsoft Graph client
type Config struct {
	ClientID      string
	ClientSecret  string
	TenantID      string
	LoginEndpoint string
	Scopes        []string
}

// Client represents the base Microsoft Graph client
type Client struct {
	clientID      string
	clientSecret  string
	tenantID      string
	loginEndpoint string
	scopes        []string
	graphClient   *msgraph.GraphServiceClient
}

// NewClient creates a new Microsoft Graph client with service credentials (client credentials flow)
func NewClient(config Config) (*Client, error) {
	// Set default scopes if not provided
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"Notes.Read",
			"Notes.Read.All",
			"User.Read",
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

	return &Client{
		clientID:      config.ClientID,
		clientSecret:  config.ClientSecret,
		tenantID:      config.TenantID,
		loginEndpoint: config.LoginEndpoint,
		scopes:        scopes,
		graphClient:   graphClient,
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
		graphClient: graphClient,
		scopes:      scopes,
		// Note: clientID, clientSecret, tenantID, loginEndpoint are not needed for token-based auth
	}, nil
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
