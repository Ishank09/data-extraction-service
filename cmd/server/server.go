package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/ishank09/data-extraction-service/cmd/server/env"
	"github.com/ishank09/data-extraction-service/pkg/api/v1/documenthandler"
	"github.com/ishank09/data-extraction-service/pkg/api/v1/health"
	"github.com/ishank09/data-extraction-service/pkg/api/v1/msgraphhandler"
	"github.com/ishank09/data-extraction-service/pkg/api/v1/pipelinehandler"
	"github.com/ishank09/data-extraction-service/pkg/logging"
	"github.com/ishank09/data-extraction-service/pkg/mongodb"
	"github.com/ishank09/data-extraction-service/pkg/msgraph"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/slok/go-http-metrics/metrics/prometheus"

	httpMetricsMiddleware "github.com/slok/go-http-metrics/middleware"
	ginMetricsMiddleware "github.com/slok/go-http-metrics/middleware/gin"

	"github.com/spf13/cobra"
)

const (
	localEnvironmentName = "local"
	defaultPort          = 8080
)

var log = logging.GetLogger()

func GetServerCmd() *cobra.Command {
	var cfg Config

	return &cobra.Command{
		Use:     "serve",
		Aliases: []string{"s"},
		Short:   "Run the server",
		Long:    "Run the server for creating and managing https://github.com/Ishank09/data-extraction-service#",
		RunE: func(cmd *cobra.Command, _ []string) error {
			setCmdFlagsFromEnv(cmd, &cfg)
			log.Infof("ENVIRONMENT_NAME is %s", os.Getenv(EnvironmentNameEnvVar))
			if os.Getenv(EnvironmentNameEnvVar) != "" &&
				os.Getenv(EnvironmentNameEnvVar) != localEnvironmentName {
				gin.SetMode(gin.ReleaseMode)
			}

			engine := gin.New()
			engine.Use(gin.Recovery())
			engine.Use(logging.GetGinLoggerMiddleware())
			engine.Use(
				gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/metrics"})),
			)
			err := engine.SetTrustedProxies(nil)
			if err != nil {
				return err
			}

			httpMetricsMiddlewareInstance := httpMetricsMiddleware.New(
				httpMetricsMiddleware.Config{
					Recorder: prometheus.NewRecorder(prometheus.Config{}),
				},
			)

			// Custom CORS configuration to allow all origins
			// Apply the custom CORS configuration to the engine
			engine.Use(cors.New(cors.Config{
				AllowOrigins: []string{"*"},
				AllowHeaders: []string{"*"},
				AllowMethods: []string{"*"},
			}))
			engine.Use(requestid.New())
			engine.Use(logging.GetGinRequestLogDecoratorMiddleware())

			// Health endpoint
			engine.GET(
				"/ping",
				health.New().Handler,
				getMetricsMiddlewareHandler("GET /ping", httpMetricsMiddlewareInstance),
			)

			// Create ETL pipeline handler
			handler, err := createPipelineHandler(&cfg)
			if err != nil {
				log.Errorf("Failed to create pipeline handler: %v", err)
				return err
			}

			// Create OAuth-enabled msgraph handler
			msgraphHandler, err := createMSGraphHandler(&cfg)
			if err != nil {
				log.Errorf("Failed to create msgraph handler: %v", err)
			}

			// Create MongoDB client and document service
			mongoClient, documentService, err := createMongoDBClient(&cfg)
			if err != nil {
				log.Errorf("Failed to create MongoDB client: %v", err)
				// Continue without MongoDB - not a critical failure
			}

			// Update pipeline handler with document service if MongoDB is available
			if documentService != nil {
				log.Infof("MongoDB integration enabled")
				// Recreate pipeline handler with document service
				handler, err = createPipelineHandlerWithMongoDB(&cfg, documentService)
				if err != nil {
					log.Errorf("Failed to recreate pipeline handler with MongoDB: %v", err)
					return err
				}
			} else {
				log.Infof("MongoDB integration disabled - documents will not be stored")
			}

			// Create document handler for MongoDB operations
			var documentHandler *documenthandler.Handler
			if documentService != nil {
				documentConfig := &documenthandler.Config{
					DocumentService: documentService,
				}
				documentHandler = documenthandler.New(documentConfig)
			}

			// ETL Pipeline routes
			v1 := engine.Group("/api/v1")
			v1.GET("/pipeline", handler.ExtractAllData, getMetricsMiddlewareHandler("GET /api/v1/pipeline", httpMetricsMiddlewareInstance))
			v1.GET("/pipeline/:source", handler.ExtractDataBySource, getMetricsMiddlewareHandler("GET /api/v1/pipeline/:source", httpMetricsMiddlewareInstance))
			v1.GET("/pipeline/type/:type", handler.ExtractDataByType, getMetricsMiddlewareHandler("GET /api/v1/pipeline/type/:type", httpMetricsMiddlewareInstance))
			v1.GET("/sources", handler.GetSources, getMetricsMiddlewareHandler("GET /api/v1/sources", httpMetricsMiddlewareInstance))
			v1.GET("/health", handler.GetHealth, getMetricsMiddlewareHandler("GET /api/v1/health", httpMetricsMiddlewareInstance))

			// Document storage routes (if MongoDB is configured)
			if documentHandler != nil {
				documents := v1.Group("/documents")
				documents.GET("", documentHandler.GetDocuments, getMetricsMiddlewareHandler("GET /api/v1/documents", httpMetricsMiddlewareInstance))
				documents.GET("/collections", documentHandler.GetDocumentCollections, getMetricsMiddlewareHandler("GET /api/v1/documents/collections", httpMetricsMiddlewareInstance))
				documents.GET("/stats", documentHandler.GetDocumentStats, getMetricsMiddlewareHandler("GET /api/v1/documents/stats", httpMetricsMiddlewareInstance))
				documents.DELETE("/cleanup", documentHandler.DeleteOldDocuments, getMetricsMiddlewareHandler("DELETE /api/v1/documents/cleanup", httpMetricsMiddlewareInstance))
				documents.GET("/health", documentHandler.GetHealth, getMetricsMiddlewareHandler("GET /api/v1/documents/health", httpMetricsMiddlewareInstance))
			}

			// OAuth routes for Microsoft Graph
			if msgraphHandler != nil {
				oauth := v1.Group("/oauth")
				oauth.POST("/authorize", msgraphHandler.Authorize, getMetricsMiddlewareHandler("POST /api/v1/oauth/authorize", httpMetricsMiddlewareInstance))
				oauth.GET("/callback", msgraphHandler.Callback, getMetricsMiddlewareHandler("GET /api/v1/oauth/callback", httpMetricsMiddlewareInstance))
				oauth.POST("/refresh", msgraphHandler.RefreshToken, getMetricsMiddlewareHandler("POST /api/v1/oauth/refresh", httpMetricsMiddlewareInstance))
				oauth.POST("/test", msgraphHandler.TestToken, getMetricsMiddlewareHandler("POST /api/v1/oauth/test", httpMetricsMiddlewareInstance))

				// MSGraph routes
				msgraph := v1.Group("/msgraph")
				msgraph.GET("/pipeline", msgraphHandler.ExtractAllData, getMetricsMiddlewareHandler("GET /api/v1/msgraph/pipeline", httpMetricsMiddlewareInstance))
				msgraph.GET("/health", msgraphHandler.GetHealth, getMetricsMiddlewareHandler("GET /api/v1/msgraph/health", httpMetricsMiddlewareInstance))
			}

			// Register "/metrics" endpoint with Gin to expose Prometheus metrics.
			engine.GET(
				"/metrics",
				gin.WrapH(promhttp.Handler()),
				getMetricsMiddlewareHandler("GET /metrics", httpMetricsMiddlewareInstance),
			)
			// Test endpoint that returns provided http status code to setup error rate, Apdex alerts
			engine.GET(
				"/testalerts",
				testStatusCodeAlertHandler,
				getMetricsMiddlewareHandler("GET /testalerts", httpMetricsMiddlewareInstance),
			)

			log.Infof("Running on port %d", cfg.Server.Port)
			addr := fmt.Sprintf(":%d", cfg.Server.Port)
			if os.Getenv("ENVIRONMENT_NAME") == localEnvironmentName {
				addr = "localhost" + addr
			}

			// Cleanup MongoDB connection on shutdown
			defer func() {
				if mongoClient != nil {
					if err := mongoClient.Disconnect(context.Background()); err != nil {
						log.Errorf("Failed to disconnect from MongoDB: %v", err)
					}
				}
			}()

			err = engine.Run(addr)
			if err != nil {
				log.Error("[Error] failed to start gin server due to: %s", err.Error())
				return err
			}
			return nil
		},
	}
}

// createPipelineHandler creates a pipeline handler with MSGraph configuration from environment variables
func createPipelineHandler(cfg *Config) (*pipelinehandler.Handler, error) {
	// Check if MSGraph configuration is available
	if cfg.MSGraph.ClientID != "" && cfg.MSGraph.ClientSecret != "" && cfg.MSGraph.TenantID != "" {
		log.Infof("Creating pipeline handler with MSGraph integration")
		log.Infof("OneNote concurrency: %d section workers, %d content workers", cfg.OneNote.MaxSectionWorkers, cfg.OneNote.MaxContentWorkers)

		config := &pipelinehandler.Config{
			MSGraphConfig: &msgraph.Config{
				ClientID:     cfg.MSGraph.ClientID,
				ClientSecret: cfg.MSGraph.ClientSecret,
				TenantID:     cfg.MSGraph.TenantID,
				OneNoteConcurrency: &msgraph.ConcurrencyConfig{
					MaxSectionWorkers: cfg.OneNote.MaxSectionWorkers,
					MaxContentWorkers: cfg.OneNote.MaxContentWorkers,
				},
			},
			UserID: cfg.MSGraph.UserID, // Pass user ID for application flow
		}
		return pipelinehandler.New(config)
	}

	// Fallback to static files only
	log.Infof("Creating pipeline handler with static files only (MSGraph not configured)")
	return pipelinehandler.New(nil)
}

// createMSGraphHandler creates a msgraph handler with OAuth configuration
func createMSGraphHandler(cfg *Config) (*msgraphhandler.Handler, error) {
	// Check if OAuth configuration is available
	if cfg.MSGraph.ClientID != "" && cfg.MSGraph.ClientSecret != "" && cfg.MSGraph.TenantID != "" && cfg.OAuth.RedirectURI != "" {
		log.Infof("Creating msgraph handler with OAuth integration")

		config := &msgraphhandler.Config{
			MSGraphConfig: &msgraph.Config{
				ClientID:     cfg.MSGraph.ClientID,
				ClientSecret: cfg.MSGraph.ClientSecret,
				TenantID:     cfg.MSGraph.TenantID,
				OneNoteConcurrency: &msgraph.ConcurrencyConfig{
					MaxSectionWorkers: cfg.OneNote.MaxSectionWorkers,
					MaxContentWorkers: cfg.OneNote.MaxContentWorkers,
				},
			},
			UserID: cfg.MSGraph.UserID,
			OAuthConfig: &msgraph.OAuthConfig{
				ClientID:     cfg.MSGraph.ClientID,
				ClientSecret: cfg.MSGraph.ClientSecret,
				TenantID:     cfg.MSGraph.TenantID,
				RedirectURI:  cfg.OAuth.RedirectURI,
				Scopes:       cfg.OAuth.Scopes,
			},
		}
		return msgraphhandler.New(config)
	}

	// Check if basic MSGraph configuration is available (without OAuth)
	if cfg.MSGraph.ClientID != "" && cfg.MSGraph.ClientSecret != "" && cfg.MSGraph.TenantID != "" {
		log.Infof("Creating msgraph handler with basic MSGraph integration (no OAuth)")
		config := &msgraphhandler.Config{
			MSGraphConfig: &msgraph.Config{
				ClientID:     cfg.MSGraph.ClientID,
				ClientSecret: cfg.MSGraph.ClientSecret,
				TenantID:     cfg.MSGraph.TenantID,
				OneNoteConcurrency: &msgraph.ConcurrencyConfig{
					MaxSectionWorkers: cfg.OneNote.MaxSectionWorkers,
					MaxContentWorkers: cfg.OneNote.MaxContentWorkers,
				},
			},
			UserID: cfg.MSGraph.UserID,
		}
		return msgraphhandler.New(config)
	}

	log.Infof("MSGraph handler not configured (missing required configuration)")
	return nil, nil
}

// createMongoDBClient creates a MongoDB client and document service
func createMongoDBClient(cfg *Config) (mongodb.Interface, *mongodb.DocumentService, error) {
	// Check if MongoDB configuration is available
	if cfg.MongoDB.URI == "" {
		log.Infof("MongoDB not configured - skipping MongoDB initialization")
		return nil, nil, nil
	}

	// Create MongoDB configuration
	mongoConfig := mongodb.NewConfig()
	mongoConfig.MongoDB.URI = cfg.MongoDB.URI

	// All MongoDB configuration must come from environment variables
	if cfg.MongoDB.Database != "" {
		mongoConfig.MongoDB.Database = cfg.MongoDB.Database
	}
	if cfg.MongoDB.Username != "" {
		mongoConfig.MongoDB.Username = cfg.MongoDB.Username
	}
	if cfg.MongoDB.Password != "" {
		mongoConfig.MongoDB.Password = cfg.MongoDB.Password
	}
	if cfg.MongoDB.AuthSource != "" {
		mongoConfig.Security.AuthSource = cfg.MongoDB.AuthSource
	}

	// Create MongoDB client
	mongoClient := mongodb.NewClient(mongoConfig)

	// Connect to MongoDB
	ctx := context.Background()
	err := mongoClient.Connect(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	err = mongoClient.Ping(ctx)
	if err != nil {
		mongoClient.Disconnect(ctx)
		return nil, nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	log.Infof("Successfully connected to MongoDB at %s", cfg.MongoDB.URI)

	// Create document service
	documentService := mongodb.NewDocumentService(mongoClient)

	return mongoClient, documentService, nil
}

// createPipelineHandlerWithMongoDB creates a pipeline handler with MongoDB integration
func createPipelineHandlerWithMongoDB(cfg *Config, documentService *mongodb.DocumentService) (*pipelinehandler.Handler, error) {
	// Check if MSGraph configuration is available
	if cfg.MSGraph.ClientID != "" && cfg.MSGraph.ClientSecret != "" && cfg.MSGraph.TenantID != "" {
		log.Infof("Creating pipeline handler with MSGraph and MongoDB integration")
		log.Infof("OneNote concurrency: %d section workers, %d content workers", cfg.OneNote.MaxSectionWorkers, cfg.OneNote.MaxContentWorkers)

		config := &pipelinehandler.Config{
			MSGraphConfig: &msgraph.Config{
				ClientID:     cfg.MSGraph.ClientID,
				ClientSecret: cfg.MSGraph.ClientSecret,
				TenantID:     cfg.MSGraph.TenantID,
				OneNoteConcurrency: &msgraph.ConcurrencyConfig{
					MaxSectionWorkers: cfg.OneNote.MaxSectionWorkers,
					MaxContentWorkers: cfg.OneNote.MaxContentWorkers,
				},
			},
			UserID:          cfg.MSGraph.UserID, // Pass user ID for application flow
			DocumentService: documentService,    // Add MongoDB document service
		}
		return pipelinehandler.New(config)
	}

	// Fallback to static files with MongoDB
	log.Infof("Creating pipeline handler with static files and MongoDB integration")
	config := &pipelinehandler.Config{
		DocumentService: documentService,
	}
	return pipelinehandler.New(config)
}

func getMetricsMiddlewareHandler(
	handlerID string,
	httpMetricsMiddlewareInstance httpMetricsMiddleware.Middleware,
) gin.HandlerFunc {
	return ginMetricsMiddleware.Handler(handlerID, httpMetricsMiddlewareInstance)
}

func setCmdFlagsFromEnv(command *cobra.Command, cfg *Config) {
	command.Flags().Int64VarP(
		&cfg.Server.Port,
		"port",
		"p",
		env.ParseInt(PortEnvVar, defaultPort),
		"port to run server",
	)

	// Set MSGraph configuration from environment variables
	cfg.MSGraph.ClientID = os.Getenv(MSGraphClientIDEnvVar)
	cfg.MSGraph.ClientSecret = os.Getenv(MSGraphClientSecretEnvVar)
	cfg.MSGraph.TenantID = os.Getenv(MSGraphTenantIDEnvVar)
	cfg.MSGraph.UserID = os.Getenv(MSGraphUserIDEnvVar)

	// Set OAuth configuration from environment variables
	cfg.OAuth.RedirectURI = os.Getenv(OAuthRedirectURIEnvVar)

	// Parse scopes from comma-separated string
	scopesStr := os.Getenv(OAuthScopesEnvVar)
	if scopesStr != "" {
		cfg.OAuth.Scopes = strings.Split(scopesStr, ",")
		// Trim spaces from scopes
		for i, scope := range cfg.OAuth.Scopes {
			cfg.OAuth.Scopes[i] = strings.TrimSpace(scope)
		}
	}

	// Set OneNote concurrency configuration
	cfg.OneNote.MaxSectionWorkers = int(env.ParseInt(OneNoteSectionWorkersEnvVar, 5))  // Default: 5 workers
	cfg.OneNote.MaxContentWorkers = int(env.ParseInt(OneNoteContentWorkersEnvVar, 10)) // Default: 10 workers

	// Set MongoDB configuration from environment variables
	// No default values - all MongoDB configuration must be explicitly provided
	cfg.MongoDB.URI = os.Getenv(MongoDBURIEnvVar)
	cfg.MongoDB.Database = os.Getenv(MongoDBDatabaseEnvVar)
	cfg.MongoDB.Username = os.Getenv(MongoDBUsernameEnvVar)
	cfg.MongoDB.Password = os.Getenv(MongoDBPasswordEnvVar)
	cfg.MongoDB.AuthSource = os.Getenv(MongoDBAuthSourceEnvVar)
}

func testStatusCodeAlertHandler(c *gin.Context) {
	statusCode := c.Query("code")
	code, err := strconv.Atoi(statusCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is not an integer"})
		return
	} else if code < 100 || code > 599 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is not a valid HTTP status code"})
		return
	}
	c.JSON(code, gin.H{"message": code})
}
