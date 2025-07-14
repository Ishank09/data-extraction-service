package logging

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log" //nolint:depguard
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

const (
	ctxLoggerKey           = "flexAPILogger"
	logFormatEnvVar        = "LOG_FORMAT"
	gitHubDeliveryIDHeader = "X-GitHub-Delivery"
)

var logger = log.NewWithOptions(os.Stdout, log.Options{
	ReportCaller:    true,
	ReportTimestamp: true,
	Formatter:       getLogFmt(),
	TimeFormat:      time.Kitchen,
})

func getLogFields(c *gin.Context) map[string]any {
	logFields, fieldsValid := c.Value("logFields").(map[string]any)
	if !fieldsValid {
		logFields = make(map[string]any, 0)
		c.Set("logFields", logFields)
	}
	return logFields
}

func SetLogField(c *gin.Context, key string, value any) {
	logFields := getLogFields(c)
	logFields[key] = value
	c.Set("logFields", logFields)
}

func NewContextLoggerWithFields(c *gin.Context, key, val string) *log.Logger {
	SetLogField(c, key, val)

	return GetOrCreateContextLogger(c)
}

func getLogFmt() log.Formatter {
	switch os.Getenv(logFormatEnvVar) {
	case "json":
		return log.JSONFormatter
	case "logfmt":
		return log.LogfmtFormatter
	case "", "text":
		return log.TextFormatter
	default:
		panic(fmt.Sprintf("unrecognized log format: %s", os.Getenv(logFormatEnvVar)))
	}
}

func GetLogger() *log.Logger {
	return logger
}

func SetContextLogger(c *gin.Context, ctxLogger *log.Logger) {
	logMeta := c.Value(ctxLoggerKey)
	if _, valid := logMeta.(*log.Logger); valid {
		return
	}

	c.Set(ctxLoggerKey, ctxLogger)
}

func getContextLogger(c *gin.Context) *log.Logger {
	logMeta := c.Value(ctxLoggerKey)
	ctxLogger, valid := logMeta.(*log.Logger)
	if !valid {
		return nil
	}

	return ctxLogger
}

func GetLoggerWithRequestID(c *gin.Context) *log.Logger {
	// can happen during testing
	if c == nil || c.Request == nil {
		return logger
	}

	l := logger.With("x-request-id", requestid.Get(c))

	if c.GetHeader(gitHubDeliveryIDHeader) != "" {
		l = l.With("x-github-delivery", c.GetHeader(gitHubDeliveryIDHeader))
	}

	return l
}

func GetOrCreateContextLogger(c *gin.Context) *log.Logger {
	ctxLogger := getContextLogger(c)

	if ctxLogger == nil {
		ctxLogger = GetLoggerWithRequestID(c)
	}

	// gin.Context drops all fields in the logger when it goes between middlewares
	for k, v := range getLogFields(c) {
		ctxLogger = ctxLogger.With(k, v)
	}

	SetContextLogger(c, ctxLogger)

	return ctxLogger
}

func SetLogLevel(level log.Level) {
	logger.SetLevel(level)
}

func RedactIfNonEmpty(i string) string {
	if i == "" {
		return i
	}
	return "<REDACTED>"
}

func RedactJWTSignature(token string) string {
	logSafeToken := token
	// If token has a jwt format, remove the signature section for safe logging:
	if i := strings.LastIndex(token, "."); strings.Count(token, ".") == 2 && i > 0 &&
		len(token[i:]) > 1 {
		logSafeToken = token[:i] + ".<REDACTED>"
	}
	return logSafeToken
}

func SHA256IfNonEmpty(i string) string {
	if i == "" {
		return i
	}
	h := sha256.New()
	h.Write([]byte(i))
	return hex.EncodeToString(h.Sum(nil))
}

func GetGinLoggerMiddleware() gin.HandlerFunc {
	switch getLogFmt() {
	case log.JSONFormatter:
		return gin.LoggerWithFormatter(
			func(params gin.LogFormatterParams) string {
				reqLog := map[string]any{
					"status_code":   params.StatusCode,
					"path":          params.Path,
					"method":        params.Method,
					"start_time":    params.TimeStamp.Format("2006/01/02 - 15:04:05"),
					"remote_addr":   params.ClientIP,
					"response_time": params.Latency.String(),
					"x-request-id":  params.Request.Header.Get("X-Request-Id"),
				}
				githubDeliveryID := params.Request.Header.Get("X-GitHub-Delivery")
				if githubDeliveryID != "" {
					reqLog["x-github-delivery"] = githubDeliveryID
				}
				s, err := json.Marshal(reqLog)
				if err != nil {
					panic(err)
				}
				return string(s) + "\n"
			})
	case log.LogfmtFormatter:
		return gin.Logger()
	case log.TextFormatter:
		return gin.Logger()
	default:
		return gin.Logger()
	}
}

func GetGinRequestLogDecoratorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request != nil && c.Request.URL != nil {
			SetLogField(c, "path", c.Request.URL.Path)
			SetLogField(c, "query", c.Request.URL.RawQuery)
		}
		c.Next()
	}
}
