package documenthandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ishank09/data-extraction-service/pkg/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDocumentService is a mock for mongodb.DocumentService
type MockDocumentService struct {
	mock.Mock
}

func (m *MockDocumentService) GetDocuments(ctx context.Context, filter mongodb.DocumentFilter) ([]mongodb.StoredDocument, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]mongodb.StoredDocument), args.Error(1)
}

func (m *MockDocumentService) GetDocumentCollections(ctx context.Context, filter mongodb.CollectionFilter) ([]mongodb.StoredDocumentCollection, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]mongodb.StoredDocumentCollection), args.Error(1)
}

func (m *MockDocumentService) GetDocumentStats(ctx context.Context) (*mongodb.DocumentStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(*mongodb.DocumentStats), args.Error(1)
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantNil bool
	}{
		{
			name:    "creates handler with nil config",
			config:  nil,
			wantNil: true,
		},
		{
			name:    "creates handler with nil document service",
			config:  &Config{DocumentService: nil},
			wantNil: true,
		},
		{
			name: "creates handler with valid document service",
			config: &Config{
				DocumentService: &mongodb.DocumentService{},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := New(tt.config)
			if tt.wantNil {
				assert.Nil(t, handler)
			} else {
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestHandler_GetHealth(t *testing.T) {
	tests := []struct {
		name           string
		hasService     bool
		expectedStatus int
	}{
		{
			name:           "returns service unavailable when no document service",
			hasService:     false,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "returns healthy when document service is available",
			hasService:     true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *Handler
			if tt.hasService {
				config := &Config{
					DocumentService: &mongodb.DocumentService{},
				}
				handler = New(config)
			} else {
				handler = &Handler{documentService: nil}
			}

			router := setupRouter()
			router.GET("/health", handler.GetHealth)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		handler  *Handler
		expected bool
	}{
		{
			name:     "returns false for nil handler",
			handler:  nil,
			expected: false,
		},
		{
			name:     "returns false for handler with nil document service",
			handler:  &Handler{documentService: nil},
			expected: false,
		},
		{
			name: "returns true for handler with document service",
			handler: &Handler{
				documentService: &mongodb.DocumentService{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.handler.IsConfigured()
			assert.Equal(t, tt.expected, result)
		})
	}
}
