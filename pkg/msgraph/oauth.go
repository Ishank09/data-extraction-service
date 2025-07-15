package msgraph

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuthConfig represents OAuth configuration for Microsoft Graph
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	TenantID     string // Use "common" for personal accounts, specific tenant ID for work/school accounts
	RedirectURI  string
	Scopes       []string
}

// NewPersonalAccountOAuthConfig creates OAuth config for personal Microsoft accounts
func NewPersonalAccountOAuthConfig(clientID, clientSecret, redirectURI string, scopes []string) OAuthConfig {
	if len(scopes) == 0 {
		// Default scopes for personal accounts - note: some scopes may not be available for personal accounts
		scopes = []string{"offline_access", "User.Read", "Files.Read"}
	}

	return OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     "common", // "common" allows both personal and work accounts
		RedirectURI:  redirectURI,
		Scopes:       scopes,
	}
}

// NewWorkSchoolAccountOAuthConfig creates OAuth config for work/school Microsoft accounts
func NewWorkSchoolAccountOAuthConfig(clientID, clientSecret, tenantID, redirectURI string, scopes []string) OAuthConfig {
	if len(scopes) == 0 {
		// Default scopes for work/school accounts
		scopes = []string{"offline_access", "User.Read", "Mail.Read", "Notes.Read"}
	}

	return OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     tenantID, // Specific tenant ID for work/school accounts
		RedirectURI:  redirectURI,
		Scopes:       scopes,
	}
}

// IsPersonalAccountConfig returns true if this config is set up for personal accounts
func (c OAuthConfig) IsPersonalAccountConfig() bool {
	return c.TenantID == "common" || c.TenantID == ""
}

// TokenResponse represents the response from the token endpoint
type TokenResponse struct {
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	ExtExpiresIn int    `json:"ext_expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthorizationURL generates the authorization URL for OAuth 2.0 flow
func (c *Client) GenerateAuthorizationURL(oauthConfig OAuthConfig, state string) (string, error) {
	if oauthConfig.ClientID == "" || oauthConfig.RedirectURI == "" {
		return "", errors.New("client_id and redirect_uri are required")
	}

	// Set default scopes if not provided
	scopes := oauthConfig.Scopes
	if len(scopes) == 0 {
		scopes = []string{"offline_access", "User.Read", "Mail.Read"}
	}

	// Use "common" as tenant if not specified or if explicitly set to "common"
	// This allows both personal and work/school accounts
	tenant := oauthConfig.TenantID
	if tenant == "" || tenant == "common" {
		tenant = "common"
	}

	// Build authorization URL
	baseURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenant)

	params := url.Values{}
	params.Set("client_id", oauthConfig.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", oauthConfig.RedirectURI)
	params.Set("response_mode", "query")
	params.Set("scope", strings.Join(scopes, " "))
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", baseURL, params.Encode()), nil
}

// ExchangeCodeForToken exchanges authorization code for access token
func (c *Client) ExchangeCodeForToken(oauthConfig OAuthConfig, code string) (*TokenResponse, error) {
	if code == "" {
		return nil, errors.New("authorization code is required")
	}

	// Use "common" as tenant if not specified or if explicitly set to "common"
	// This allows both personal and work/school accounts
	tenant := oauthConfig.TenantID
	if tenant == "" || tenant == "common" {
		tenant = "common"
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant)

	// Set default scopes if not provided
	scopes := oauthConfig.Scopes
	if len(scopes) == 0 {
		scopes = []string{"User.Read", "Mail.Read"}
	}

	// Prepare form data
	data := url.Values{}
	data.Set("client_id", oauthConfig.ClientID)
	data.Set("scope", strings.Join(scopes, " "))
	data.Set("code", code)
	data.Set("redirect_uri", oauthConfig.RedirectURI)
	data.Set("grant_type", "authorization_code")
	data.Set("client_secret", oauthConfig.ClientSecret)

	// Make POST request
	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to make token request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResponse, nil
}

// RefreshToken refreshes an expired access token using refresh token
func (c *Client) RefreshToken(oauthConfig OAuthConfig, refreshToken string) (*TokenResponse, error) {
	if refreshToken == "" {
		return nil, errors.New("refresh token is required")
	}

	// Use "common" as tenant if not specified or if explicitly set to "common"
	// This allows both personal and work/school accounts
	tenant := oauthConfig.TenantID
	if tenant == "" || tenant == "common" {
		tenant = "common"
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant)

	// Set default scopes if not provided
	scopes := oauthConfig.Scopes
	if len(scopes) == 0 {
		scopes = []string{"User.Read", "Mail.Read"}
	}

	// Prepare form data
	data := url.Values{}
	data.Set("client_id", oauthConfig.ClientID)
	data.Set("scope", strings.Join(scopes, " "))
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")
	data.Set("client_secret", oauthConfig.ClientSecret)

	// Make POST request
	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to make refresh token request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	return &tokenResponse, nil
}

// TestAccessToken tests if an access token is valid by making a request to Microsoft Graph
func (c *Client) TestAccessToken(accessToken string) error {
	if accessToken == "" {
		return errors.New("access token is required")
	}

	// Create a simple GET request to test the token
	req, err := http.NewRequest("GET", "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to test access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("access token test failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GenerateStateParameter generates a cryptographically secure random state parameter for OAuth
func GenerateStateParameter() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// OAuthClient represents a client specifically for OAuth operations
type OAuthClient struct {
	config OAuthConfig
}

// NewOAuthClient creates a new OAuth client
func NewOAuthClient(config OAuthConfig) *OAuthClient {
	return &OAuthClient{
		config: config,
	}
}

// GetAuthorizationURL generates authorization URL
func (oc *OAuthClient) GetAuthorizationURL(state string) (string, error) {
	client := &Client{} // Create temporary client for method access
	return client.GenerateAuthorizationURL(oc.config, state)
}

// ExchangeCode exchanges authorization code for tokens
func (oc *OAuthClient) ExchangeCode(code string) (*TokenResponse, error) {
	client := &Client{} // Create temporary client for method access
	return client.ExchangeCodeForToken(oc.config, code)
}

// RefreshAccessToken refreshes access token using refresh token
func (oc *OAuthClient) RefreshAccessToken(refreshToken string) (*TokenResponse, error) {
	client := &Client{} // Create temporary client for method access
	return client.RefreshToken(oc.config, refreshToken)
}

// TestToken tests if access token is valid
func (oc *OAuthClient) TestToken(accessToken string) error {
	client := &Client{} // Create temporary client for method access
	return client.TestAccessToken(accessToken)
}
