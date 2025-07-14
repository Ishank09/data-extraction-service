package msgraph

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with all fields",
			config: Config{
				ClientID:      "test-client-id",
				ClientSecret:  "test-client-secret",
				TenantID:      "test-tenant-id",
				LoginEndpoint: "https://login.microsoftonline.com",
				Scopes:        []string{"Notes.Read", "User.Read"},
			},
			expectError: false,
		},
		{
			name: "valid config with default scopes",
			config: Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				TenantID:     "test-tenant-id",
			},
			expectError: false,
		},
		{
			name: "missing client ID",
			config: Config{
				ClientSecret: "test-client-secret",
				TenantID:     "test-tenant-id",
			},
			expectError: false, // Azure SDK accepts empty strings, validation happens during API calls
		},
		{
			name: "missing client secret",
			config: Config{
				ClientID: "test-client-id",
				TenantID: "test-tenant-id",
			},
			expectError: true,
			errorMsg:    "failed to create credentials",
		},
		{
			name: "missing tenant ID",
			config: Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			expectError: true,
			errorMsg:    "failed to create credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				if tt.errorMsg != "" && err != nil {
					if err.Error()[:len(tt.errorMsg)] != tt.errorMsg {
						t.Errorf("Expected error message to start with '%s', got '%s'", tt.errorMsg, err.Error())
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}

			if client == nil {
				t.Errorf("Expected non-nil client for %s", tt.name)
				return
			}

			// Verify client fields
			if client.clientID != tt.config.ClientID {
				t.Errorf("Expected clientID '%s', got '%s'", tt.config.ClientID, client.clientID)
			}
			if client.clientSecret != tt.config.ClientSecret {
				t.Errorf("Expected clientSecret '%s', got '%s'", tt.config.ClientSecret, client.clientSecret)
			}
			if client.tenantID != tt.config.TenantID {
				t.Errorf("Expected tenantID '%s', got '%s'", tt.config.TenantID, client.tenantID)
			}

			// Verify scopes
			expectedScopes := tt.config.Scopes
			if len(expectedScopes) == 0 {
				expectedScopes = []string{"Notes.Read", "Notes.Read.All", "User.Read"}
			}
			if len(client.scopes) != len(expectedScopes) {
				t.Errorf("Expected %d scopes, got %d", len(expectedScopes), len(client.scopes))
			}

			// Verify graph client is created
			if client.graphClient == nil {
				t.Error("Expected non-nil graphClient")
			}
		})
	}
}

func TestNewClientWithToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid token",
			token:       "valid-access-token",
			expectError: false,
		},
		{
			name:        "empty token",
			token:       "",
			expectError: true,
			errorMsg:    "access token cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClientWithToken(tt.token)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				if tt.errorMsg != "" && err != nil {
					if err.Error() != tt.errorMsg {
						t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}

			if client == nil {
				t.Errorf("Expected non-nil client for %s", tt.name)
				return
			}

			// Verify scopes are set correctly
			expectedScopes := []string{"Notes.Read", "Notes.Read.All", "User.Read"}
			if len(client.scopes) != len(expectedScopes) {
				t.Errorf("Expected %d scopes, got %d", len(expectedScopes), len(client.scopes))
			}

			// Verify graph client is created
			if client.graphClient == nil {
				t.Error("Expected non-nil graphClient")
			}

			// Verify other fields are empty (not needed for token-based auth)
			if client.clientID != "" {
				t.Errorf("Expected empty clientID for token-based auth, got '%s'", client.clientID)
			}
			if client.clientSecret != "" {
				t.Errorf("Expected empty clientSecret for token-based auth, got '%s'", client.clientSecret)
			}
			if client.tenantID != "" {
				t.Errorf("Expected empty tenantID for token-based auth, got '%s'", client.tenantID)
			}
		})
	}
}

func TestClient_GetGraphClient(t *testing.T) {
	// Create a valid client
	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TenantID:     "test-tenant-id",
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test GetGraphClient
	graphClient := client.GetGraphClient()
	if graphClient == nil {
		t.Error("Expected non-nil graph client")
	}

	// Verify it's the same instance
	if graphClient != client.graphClient {
		t.Error("GetGraphClient should return the same instance as stored in client")
	}
}

func TestStaticTokenCredential_GetToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "valid token",
			token: "test-access-token",
		},
		{
			name:  "empty token",
			token: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credential := &staticTokenCredential{token: tt.token}
			ctx := context.Background()
			options := policy.TokenRequestOptions{}

			accessToken, err := credential.GetToken(ctx, options)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if accessToken.Token != tt.token {
				t.Errorf("Expected token '%s', got '%s'", tt.token, accessToken.Token)
			}

			// Verify token expires in about 1 hour
			expectedExpiry := time.Now().Add(time.Hour)
			timeDiff := accessToken.ExpiresOn.Sub(expectedExpiry)
			if timeDiff < -time.Minute || timeDiff > time.Minute {
				t.Errorf("Token expiry should be about 1 hour from now, got %v", accessToken.ExpiresOn)
			}
		})
	}
}

func TestConfig_DefaultScopes(t *testing.T) {
	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TenantID:     "test-tenant-id",
		// No scopes provided
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	expectedScopes := []string{"Notes.Read", "Notes.Read.All", "User.Read"}
	if len(client.scopes) != len(expectedScopes) {
		t.Errorf("Expected %d default scopes, got %d", len(expectedScopes), len(client.scopes))
	}

	for i, expectedScope := range expectedScopes {
		if client.scopes[i] != expectedScope {
			t.Errorf("Expected scope[%d] to be '%s', got '%s'", i, expectedScope, client.scopes[i])
		}
	}
}

func TestConfig_CustomScopes(t *testing.T) {
	customScopes := []string{"Notes.Read", "User.Read", "Custom.Scope"}
	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TenantID:     "test-tenant-id",
		Scopes:       customScopes,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if len(client.scopes) != len(customScopes) {
		t.Errorf("Expected %d custom scopes, got %d", len(customScopes), len(client.scopes))
	}

	for i, expectedScope := range customScopes {
		if client.scopes[i] != expectedScope {
			t.Errorf("Expected scope[%d] to be '%s', got '%s'", i, expectedScope, client.scopes[i])
		}
	}
}
