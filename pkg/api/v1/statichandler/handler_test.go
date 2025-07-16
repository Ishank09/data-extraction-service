package statichandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestNew(t *testing.T) {
	handler := New()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.staticClient)
}

func TestHandler_ExtractAllData(t *testing.T) {
	handler := New()
	router := setupRouter()
	router.GET("/pipeline", handler.ExtractAllData)

	req := httptest.NewRequest(http.MethodGet, "/pipeline", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "documents")
	assert.Contains(t, w.Body.String(), "source")
}

func TestHandler_ExtractDataByType(t *testing.T) {
	tests := []struct {
		name           string
		fileType       string
		expectedStatus int
	}{
		{
			name:           "valid file type - json",
			fileType:       "json",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid file type - csv",
			fileType:       "csv",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid file type",
			fileType:       "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := New()
			router := setupRouter()
			router.GET("/pipeline/type/:type", handler.ExtractDataByType)

			req := httptest.NewRequest(http.MethodGet, "/pipeline/type/"+tt.fileType, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_GetSupportedTypes(t *testing.T) {
	handler := New()
	router := setupRouter()
	router.GET("/types", handler.GetSupportedTypes)

	req := httptest.NewRequest(http.MethodGet, "/types", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "supported_types")
	assert.Contains(t, w.Body.String(), "count")
}

func TestHandler_GetHealth(t *testing.T) {
	handler := New()
	router := setupRouter()
	router.GET("/health", handler.GetHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "status")
	assert.Contains(t, w.Body.String(), "healthy")
	assert.Contains(t, w.Body.String(), "static_client")
}
