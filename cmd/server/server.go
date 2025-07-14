package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/ishank09/data-extraction-service/cmd/server/env"
	"github.com/ishank09/data-extraction-service/pkg/api/v1/dataextractionhandler"
	"github.com/ishank09/data-extraction-service/pkg/api/v1/health"
	"github.com/ishank09/data-extraction-service/pkg/logging"
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

			// Create data extraction handler
			handler, err := createDataExtractionHandler(&cfg)
			if err != nil {
				log.Errorf("Failed to create data extraction handler: %v", err)
				return err
			}

			// Data extraction routes
			v1 := engine.Group("/api/v1")
			v1.GET("/documents", handler.GetAllDocuments, getMetricsMiddlewareHandler("GET /api/v1/documents", httpMetricsMiddlewareInstance))
			v1.GET("/documents/:source", handler.GetDocumentsBySource, getMetricsMiddlewareHandler("GET /api/v1/documents/:source", httpMetricsMiddlewareInstance))
			v1.GET("/documents/type/:type", handler.GetDocumentsByType, getMetricsMiddlewareHandler("GET /api/v1/documents/type/:type", httpMetricsMiddlewareInstance))
			v1.GET("/sources", handler.GetSources, getMetricsMiddlewareHandler("GET /api/v1/sources", httpMetricsMiddlewareInstance))
			v1.GET("/health", handler.GetHealth, getMetricsMiddlewareHandler("GET /api/v1/health", httpMetricsMiddlewareInstance))

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

			err = engine.Run(addr)
			if err != nil {
				log.Error("[Error] failed to start gin server due to: %s", err.Error())
				return err
			}
			return nil
		},
	}
}

// createDataExtractionHandler creates a data extraction handler with MSGraph configuration from environment variables
func createDataExtractionHandler(cfg *Config) (*dataextractionhandler.Handler, error) {
	// Check if MSGraph configuration is available
	if cfg.MSGraph.ClientID != "" && cfg.MSGraph.ClientSecret != "" && cfg.MSGraph.TenantID != "" {
		log.Infof("Creating data extraction handler with MSGraph integration")
		config := &dataextractionhandler.Config{
			MSGraphConfig: &msgraph.Config{
				ClientID:     cfg.MSGraph.ClientID,
				ClientSecret: cfg.MSGraph.ClientSecret,
				TenantID:     cfg.MSGraph.TenantID,
			},
		}
		return dataextractionhandler.New(config)
	}

	// Fallback to static files only
	log.Infof("Creating data extraction handler with static files only (MSGraph not configured)")
	return dataextractionhandler.New(nil)
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
