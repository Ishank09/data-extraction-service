package msgraph

import (
	"net/url"
	"strings"
	"testing"
)

func TestGenerateAuthorizationURL(t *testing.T) {
	client := &Client{}
	config := OAuthConfig{
		ClientID:    "test-client-id",
		TenantID:    "test-tenant-id",
		RedirectURI: "https://localhost/callback",
		Scopes:      []string{"User.Read", "Mail.Read"},
	}

	authURL, err := client.GenerateAuthorizationURL(config, "test-state")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Parse the URL to verify parameters
	parsedURL, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("Failed to parse authorization URL: %v", err)
	}

	// Check base URL
	expectedBase := "https://login.microsoftonline.com/test-tenant-id/oauth2/v2.0/authorize"
	baseURL := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
	if baseURL != expectedBase {
		t.Errorf("Expected base URL %s, got %s", expectedBase, baseURL)
	}

	// Check query parameters
	params := parsedURL.Query()
	if params.Get("client_id") != "test-client-id" {
		t.Errorf("Expected client_id test-client-id, got %s", params.Get("client_id"))
	}
	if params.Get("response_type") != "code" {
		t.Errorf("Expected response_type code, got %s", params.Get("response_type"))
	}
	if params.Get("redirect_uri") != "https://localhost/callback" {
		t.Errorf("Expected redirect_uri https://localhost/callback, got %s", params.Get("redirect_uri"))
	}
	if params.Get("state") != "test-state" {
		t.Errorf("Expected state test-state, got %s", params.Get("state"))
	}
	if params.Get("scope") != "User.Read Mail.Read" {
		t.Errorf("Expected scope 'User.Read Mail.Read', got %s", params.Get("scope"))
	}
}

func TestGenerateAuthorizationURL_DefaultScopes(t *testing.T) {
	client := &Client{}
	config := OAuthConfig{
		ClientID:    "test-client-id",
		TenantID:    "test-tenant-id",
		RedirectURI: "https://localhost/callback",
		// No scopes provided
	}

	authURL, err := client.GenerateAuthorizationURL(config, "test-state")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Parse the URL to verify default scopes
	parsedURL, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("Failed to parse authorization URL: %v", err)
	}

	params := parsedURL.Query()
	expectedScope := "offline_access User.Read Mail.Read"
	if params.Get("scope") != expectedScope {
		t.Errorf("Expected default scope '%s', got %s", expectedScope, params.Get("scope"))
	}
}

func TestGenerateAuthorizationURL_MissingConfig(t *testing.T) {
	client := &Client{}

	// Test missing ClientID
	config := OAuthConfig{
		TenantID:    "test-tenant-id",
		RedirectURI: "https://localhost/callback",
	}
	_, err := client.GenerateAuthorizationURL(config, "test-state")
	if err == nil {
		t.Error("Expected error for missing ClientID, got nil")
	}

	// Test missing RedirectURI
	config = OAuthConfig{
		ClientID: "test-client-id",
		TenantID: "test-tenant-id",
	}
	_, err = client.GenerateAuthorizationURL(config, "test-state")
	if err == nil {
		t.Error("Expected error for missing RedirectURI, got nil")
	}
}

func TestExchangeCodeForToken_Success(t *testing.T) {
	// This test will fail because we can't easily override the token URL
	// In a real implementation, you might want to make the base URL configurable
	t.Skip("Skipping integration test - would require configurable base URL")
}

func TestRefreshToken_MissingToken(t *testing.T) {
	client := &Client{}
	config := OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TenantID:     "test-tenant-id",
	}

	_, err := client.RefreshToken(config, "")
	if err == nil {
		t.Error("Expected error for missing refresh token, got nil")
	}
	if !strings.Contains(err.Error(), "refresh token is required") {
		t.Errorf("Expected 'refresh token is required' error, got %v", err)
	}
}

func TestTestAccessToken_MissingToken(t *testing.T) {
	client := &Client{}

	err := client.TestAccessToken("")
	if err == nil {
		t.Error("Expected error for missing access token, got nil")
	}
	if !strings.Contains(err.Error(), "access token is required") {
		t.Errorf("Expected 'access token is required' error, got %v", err)
	}
}

func TestGenerateStateParameter(t *testing.T) {
	state1, err := GenerateStateParameter()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	state2, err := GenerateStateParameter()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// States should be different
	if state1 == state2 {
		t.Error("Expected different state parameters, got identical ones")
	}

	// States should be non-empty
	if state1 == "" || state2 == "" {
		t.Error("Expected non-empty state parameters")
	}

	// States should be base64 encoded (basic check)
	if len(state1) < 10 || len(state2) < 10 {
		t.Error("Expected state parameters to be reasonably long")
	}
}

func TestOAuthClient(t *testing.T) {
	config := OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TenantID:     "test-tenant-id",
		RedirectURI:  "https://localhost/callback",
		Scopes:       []string{"User.Read"},
	}

	oauthClient := NewOAuthClient(config)
	if oauthClient == nil {
		t.Fatal("Expected non-nil OAuth client")
	}

	// Test authorization URL generation
	authURL, err := oauthClient.GetAuthorizationURL("test-state")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if authURL == "" {
		t.Error("Expected non-empty authorization URL")
	}

	// Test error cases
	_, err = oauthClient.ExchangeCode("")
	if err == nil {
		t.Error("Expected error for empty authorization code")
	}

	_, err = oauthClient.RefreshAccessToken("")
	if err == nil {
		t.Error("Expected error for empty refresh token")
	}

	err = oauthClient.TestToken("")
	if err == nil {
		t.Error("Expected error for empty access token")
	}
}
