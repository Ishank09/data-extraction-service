//go:build unit

package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/test", New().Handler)
	return r
}

func TestHealth_Handler(t *testing.T) {
	t.Run("basic test", func(t *testing.T) {
		router := setupRouter()

		w := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "{\"message\":\"healthy\"}", w.Body.String())
	})
}
