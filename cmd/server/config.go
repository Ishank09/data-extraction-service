package server

type Config struct {
	Server struct {
		Port int64
	}
	MSGraph struct {
		ClientID     string
		ClientSecret string
		TenantID     string // Use "common" for personal accounts, specific tenant ID for work/school accounts
		UserID       string // Required for application flow when accessing user data
	}
	OAuth struct {
		RedirectURI string
		Scopes      []string
	}
}

const (
	PortEnvVar            = "PORT"
	EnvironmentNameEnvVar = "ENVIRONMENT_NAME"

	// MSGraph environment variables
	MSGraphClientIDEnvVar     = "MSGRAPH_CLIENT_ID"
	MSGraphClientSecretEnvVar = "MSGRAPH_CLIENT_SECRET"
	MSGraphTenantIDEnvVar     = "MSGRAPH_TENANT_ID" // Use "common" for personal accounts
	MSGraphUserIDEnvVar       = "MSGRAPH_USER_ID"   // New environment variable for user ID

	// OAuth environment variables
	OAuthRedirectURIEnvVar = "OAUTH_REDIRECT_URI"
	OAuthScopesEnvVar      = "OAUTH_SCOPES" // Comma-separated list of scopes
)
