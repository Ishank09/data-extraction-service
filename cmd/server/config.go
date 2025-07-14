package server

type Config struct {
	Server struct {
		Port int64
	}
	MSGraph struct {
		ClientID     string
		ClientSecret string
		TenantID     string
	}
}

const (
	PortEnvVar            = "PORT"
	EnvironmentNameEnvVar = "ENVIRONMENT_NAME"

	// MSGraph environment variables
	MSGraphClientIDEnvVar     = "MSGRAPH_CLIENT_ID"
	MSGraphClientSecretEnvVar = "MSGRAPH_CLIENT_SECRET"
	MSGraphTenantIDEnvVar     = "MSGRAPH_TENANT_ID"
)
