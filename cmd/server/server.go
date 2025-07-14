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
	"github.com/ishank09/data-extraction-service/pkg/api/v1/health"
	"github.com/ishank09/data-extraction-service/pkg/logging"
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
			engine.GET(
				"/ping",
				health.New().Handler,
				getMetricsMiddlewareHandler("GET /ping", httpMetricsMiddlewareInstance),
			)
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
