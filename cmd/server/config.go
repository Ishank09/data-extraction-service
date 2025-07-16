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
	OneNote struct {
		MaxSectionWorkers int // Maximum concurrent section workers for OneNote processing
		MaxContentWorkers int // Maximum concurrent content workers for OneNote processing
	}
	MongoDB struct {
		URI        string
		Database   string
		Username   string
		Password   string
		AuthSource string
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

	// OneNote performance tuning environment variables
	OneNoteSectionWorkersEnvVar = "ONENOTE_SECTION_WORKERS" // Max concurrent section workers (default: 5)
	OneNoteContentWorkersEnvVar = "ONENOTE_CONTENT_WORKERS" // Max concurrent content workers (default: 10)

	// MongoDB environment variables
	MongoDBURIEnvVar        = "MONGODB_URI"
	MongoDBDatabaseEnvVar   = "MONGODB_DATABASE"
	MongoDBUsernameEnvVar   = "MONGODB_USERNAME"
	MongoDBPasswordEnvVar   = "MONGODB_PASSWORD"
	MongoDBAuthSourceEnvVar = "MONGODB_AUTH_SOURCE"
)
