package msgraphhandler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ishank09/data-extraction-service/internal/types"
	"github.com/ishank09/data-extraction-service/pkg/msgraph"
)

// Handler handles Microsoft Graph operations
type Handler struct {
	msgraphClient msgraph.Interface
	oauthClient   *msgraph.OAuthClient
}

// Config represents the configuration for the msgraph handler
type Config struct {
	MSGraphConfig *msgraph.Config      `json:"msgraph_config,omitempty"`
	UserID        string               `json:"user_id,omitempty"`      // Required for application flow when accessing user data
	OAuthConfig   *msgraph.OAuthConfig `json:"oauth_config,omitempty"` // OAuth configuration
}

// AuthorizeRequest represents the request for authorization URL generation
type AuthorizeRequest struct {
	Scopes      []string `json:"scopes,omitempty"`
	RedirectURI string   `json:"redirect_uri,omitempty"`
}

// AuthorizeResponse represents the authorization URL response
type AuthorizeResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
}

// CallbackRequest represents the callback request with authorization code
type CallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// RefreshTokenRequest represents the refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// TestTokenRequest represents the test token request
type TestTokenRequest struct {
	AccessToken string `json:"access_token"`
}

// New creates a new msgraph handler with application authentication (client credentials flow)
func New(config *Config) (*Handler, error) {
	if config == nil || config.MSGraphConfig == nil {
		return nil, nil // Return nil if no configuration provided
	}

	var graphClient msgraph.Interface
	var err error

	if config.UserID != "" {
		// Create client with user ID for application flow
		graphClient, err = msgraph.NewClientWithUserID(*config.MSGraphConfig, config.UserID)
	} else {
		// Create standard client for application flow (may fail for user-specific operations)
		graphClient, err = msgraph.NewClient(*config.MSGraphConfig)
	}

	if err != nil {
		return nil, err
	}

	// Create OAuth client if OAuth config is provided
	var oauthClient *msgraph.OAuthClient
	if config.OAuthConfig != nil {
		oauthClient = msgraph.NewOAuthClient(*config.OAuthConfig)
	}

	return &Handler{
		msgraphClient: graphClient,
		oauthClient:   oauthClient,
	}, nil
}

// NewWithToken creates a new msgraph handler with delegated authentication (access token flow)
func NewWithToken(accessToken string) (*Handler, error) {
	if accessToken == "" {
		return nil, nil
	}

	graphClient, err := msgraph.NewClientWithToken(accessToken)
	if err != nil {
		return nil, err
	}

	return &Handler{
		msgraphClient: graphClient,
	}, nil
}

// NewWithClient creates a new msgraph handler with an existing client
func NewWithClient(client msgraph.Interface) *Handler {
	return &Handler{
		msgraphClient: client,
	}
}

// NewWithOAuth creates a new msgraph handler with OAuth configuration
func NewWithOAuth(oauthConfig msgraph.OAuthConfig) *Handler {
	return &Handler{
		oauthClient: msgraph.NewOAuthClient(oauthConfig),
	}
}

// GetDocuments retrieves documents from Microsoft Graph
func (h *Handler) GetDocuments(ctx context.Context) (*types.DocumentCollection, error) {
	if h.msgraphClient == nil {
		return nil, nil
	}

	return h.msgraphClient.GetOneNoteDataAsJSON(ctx)
}

// ExtractAllData returns all OneNote documents
func (h *Handler) ExtractAllData(c *gin.Context) {
	// Check for Authorization header with Bearer token
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		// Extract token from Authorization header
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Create a temporary client with the provided token
		tempHandler, err := NewWithToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid access token",
				"details": err.Error(),
			})
			return
		}

		// Use the temporary handler to get documents
		ctx := c.Request.Context()
		collection, err := tempHandler.GetDocuments(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to retrieve msgraph documents with provided token",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, collection)
		return
	}

	// Fall back to the existing handler if no Authorization header
	if h.msgraphClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Microsoft Graph client not configured and no access token provided",
			"message": "Either configure the service with client credentials or provide an Authorization header with Bearer token",
		})
		return
	}

	ctx := c.Request.Context()

	collection, err := h.msgraphClient.GetOneNoteDataAsJSON(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve msgraph documents",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, collection)
}

// GetHealth returns health status of msgraph client
func (h *Handler) GetHealth(c *gin.Context) {
	if h.msgraphClient == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":    "not_configured",
			"component": "msgraph_client",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"component": "msgraph_client",
	})
}

// IsConfigured returns whether the msgraph client is configured
func (h *Handler) IsConfigured() bool {
	return h.msgraphClient != nil
}

// OAuth Endpoints

// Authorize generates authorization URL for OAuth 2.0 flow
func (h *Handler) Authorize(c *gin.Context) {
	if h.oauthClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OAuth client not configured",
		})
		return
	}

	var req AuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Generate state parameter
	state, err := msgraph.GenerateStateParameter()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate state parameter",
			"details": err.Error(),
		})
		return
	}

	// Generate authorization URL
	authURL, err := h.oauthClient.GetAuthorizationURL(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate authorization URL",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuthorizeResponse{
		AuthorizationURL: authURL,
		State:            state,
	})
}

// Callback handles the OAuth callback and exchanges code for tokens
func (h *Handler) Callback(c *gin.Context) {
	if h.oauthClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OAuth client not configured",
		})
		return
	}

	// Check for error from Microsoft
	if errorParam := c.Query("error"); errorParam != "" {
		errorDesc := c.Query("error_description")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "OAuth authorization failed",
			"error_code":        errorParam,
			"error_description": errorDesc,
		})
		return
	}

	// Get authorization code from query parameters
	code := c.Query("code")
	_ = c.Query("state") // TODO: Validate state parameter for CSRF protection

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Authorization code is required",
		})
		return
	}

	// Exchange code for tokens
	tokenResponse, err := h.oauthClient.ExchangeCode(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to exchange authorization code for tokens",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tokenResponse)
}

// RefreshToken refreshes an expired access token
func (h *Handler) RefreshToken(c *gin.Context) {
	if h.oauthClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OAuth client not configured",
		})
		return
	}

	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Refresh token is required",
		})
		return
	}

	// Refresh access token
	tokenResponse, err := h.oauthClient.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to refresh access token",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tokenResponse)
}

// TestToken tests if an access token is valid
func (h *Handler) TestToken(c *gin.Context) {
	if h.oauthClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OAuth client not configured",
		})
		return
	}

	var req TestTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if req.AccessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Access token is required",
		})
		return
	}

	// Test access token
	err := h.oauthClient.TestToken(req.AccessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Access token is invalid or expired",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "valid",
		"message": "Access token is valid",
	})
}
