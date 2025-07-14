package dataextractionhandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishank09/data-extraction-service/internal/types"
	"github.com/ishank09/data-extraction-service/pkg/msgraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMSGraphClient is a mock for msgraph.Interface
type MockMSGraphClient struct {
	mock.Mock
}

func (m *MockMSGraphClient) GetOneNoteDataAsJSON(ctx context.Context) (*types.DocumentCollection, error) {
	args := m.Called(ctx)
	return args.Get(0).(*types.DocumentCollection), args.Error(1)
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "creates handler with nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "creates handler with empty config",
			config:  &Config{},
			wantErr: false,
		},
		{
			name: "creates handler with valid msgraph config",
			config: &Config{
				MSGraphConfig: &msgraph.Config{
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					TenantID:     "test-tenant-id",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := New(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, handler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
				assert.NotNil(t, handler.staticHandler)
			}
		})
	}
}

func TestNewWithMSGraphClient(t *testing.T) {
	mockClient := &MockMSGraphClient{}
	handler := NewWithMSGraphClient(mockClient)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.staticHandler)
	assert.NotNil(t, handler.msgraphHandler)
}

func TestHandler_GetAllDocuments(t *testing.T) {
	tests := []struct {
		name             string
		setupMock        func(*MockMSGraphClient)
		useMSGraphClient bool
		expectedStatus   int
		expectedDocCount int
		expectError      bool
	}{
		{
			name:             "returns documents from static source only",
			useMSGraphClient: false,
			expectedStatus:   http.StatusOK,
			expectedDocCount: 0, // Empty directories
			expectError:      false,
		},
		{
			name:             "returns documents from both sources",
			useMSGraphClient: true,
			setupMock: func(m *MockMSGraphClient) {
				collection := types.NewDocumentCollection("onenote")
				collection.AddDocument(types.Document{
					ID:        "test-doc-1",
					Source:    "onenote",
					Type:      "note",
					Title:     "Test Note",
					Content:   "Test content",
					CreatedAt: time.Now(),
					FetchedAt: time.Now(),
				})
				m.On("GetOneNoteDataAsJSON", mock.Anything).Return(collection, nil)
			},
			expectedStatus:   http.StatusOK,
			expectedDocCount: 1, // 1 from msgraph + 0 from static
			expectError:      false,
		},
		{
			name:             "handles msgraph client error",
			useMSGraphClient: true,
			setupMock: func(m *MockMSGraphClient) {
				m.On("GetOneNoteDataAsJSON", mock.Anything).Return((*types.DocumentCollection)(nil), errors.New("msgraph error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *Handler
			if tt.useMSGraphClient {
				mockClient := &MockMSGraphClient{}
				if tt.setupMock != nil {
					tt.setupMock(mockClient)
				}
				handler = NewWithMSGraphClient(mockClient)
			} else {
				handler, _ = New(nil)
			}

			router := setupRouter()
			router.GET("/documents", handler.GetAllDocuments)

			req := httptest.NewRequest(http.MethodGet, "/documents", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				var response types.DocumentCollection
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedDocCount, len(response.Documents))
				assert.Equal(t, "data_extraction_service", response.Source)
			}
		})
	}
}

func TestHandler_GetDocumentsBySource(t *testing.T) {
	tests := []struct {
		name             string
		source           string
		setupMock        func(*MockMSGraphClient)
		useMSGraphClient bool
		expectedStatus   int
		expectError      bool
	}{
		{
			name:           "returns static documents",
			source:         "static",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:             "returns msgraph documents",
			source:           "msgraph",
			useMSGraphClient: true,
			setupMock: func(m *MockMSGraphClient) {
				collection := types.NewDocumentCollection("onenote")
				m.On("GetOneNoteDataAsJSON", mock.Anything).Return(collection, nil)
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:             "returns onenote documents",
			source:           "onenote",
			useMSGraphClient: true,
			setupMock: func(m *MockMSGraphClient) {
				collection := types.NewDocumentCollection("onenote")
				m.On("GetOneNoteDataAsJSON", mock.Anything).Return(collection, nil)
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:             "handles msgraph client not configured",
			source:           "msgraph",
			useMSGraphClient: false,
			expectedStatus:   http.StatusServiceUnavailable,
			expectError:      true,
		},
		{
			name:           "handles invalid source",
			source:         "invalid",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:             "handles msgraph client error",
			source:           "msgraph",
			useMSGraphClient: true,
			setupMock: func(m *MockMSGraphClient) {
				m.On("GetOneNoteDataAsJSON", mock.Anything).Return((*types.DocumentCollection)(nil), errors.New("msgraph error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *Handler
			if tt.useMSGraphClient {
				mockClient := &MockMSGraphClient{}
				if tt.setupMock != nil {
					tt.setupMock(mockClient)
				}
				handler = NewWithMSGraphClient(mockClient)
			} else {
				handler, _ = New(nil)
			}

			router := setupRouter()
			router.GET("/documents/:source", handler.GetDocumentsBySource)

			req := httptest.NewRequest(http.MethodGet, "/documents/"+tt.source, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				var response types.DocumentCollection
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandler_GetDocumentsByType(t *testing.T) {
	tests := []struct {
		name           string
		fileType       string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "returns documents by valid type",
			fileType:       "json",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "returns documents by another valid type",
			fileType:       "csv",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "handles invalid file type",
			fileType:       "invalid",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _ := New(nil)

			router := setupRouter()
			router.GET("/documents/type/:type", handler.GetDocumentsByType)

			req := httptest.NewRequest(http.MethodGet, "/documents/type/"+tt.fileType, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				var response types.DocumentCollection
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "static_"+tt.fileType, response.Source)
			}
		})
	}
}

func TestHandler_GetSources(t *testing.T) {
	tests := []struct {
		name             string
		useMSGraphClient bool
		expectedSources  int
		expectMSGraph    bool
	}{
		{
			name:             "returns sources without msgraph",
			useMSGraphClient: false,
			expectedSources:  2,
			expectMSGraph:    false,
		},
		{
			name:             "returns sources with msgraph",
			useMSGraphClient: true,
			expectedSources:  2,
			expectMSGraph:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *Handler
			if tt.useMSGraphClient {
				mockClient := &MockMSGraphClient{}
				handler = NewWithMSGraphClient(mockClient)
			} else {
				handler, _ = New(nil)
			}

			router := setupRouter()
			router.GET("/sources", handler.GetSources)

			req := httptest.NewRequest(http.MethodGet, "/sources", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			sources := response["sources"].([]interface{})
			assert.Equal(t, tt.expectedSources, len(sources))

			// Check if msgraph source availability is correct
			var msgraphSource map[string]interface{}
			for _, source := range sources {
				s := source.(map[string]interface{})
				if s["name"] == "msgraph" {
					msgraphSource = s
					break
				}
			}

			assert.NotNil(t, msgraphSource)
			assert.Equal(t, tt.expectMSGraph, msgraphSource["available"])
		})
	}
}

func TestHandler_GetHealth(t *testing.T) {
	tests := []struct {
		name             string
		useMSGraphClient bool
		expectedStatus   string
	}{
		{
			name:             "returns health without msgraph",
			useMSGraphClient: false,
			expectedStatus:   "healthy",
		},
		{
			name:             "returns health with msgraph",
			useMSGraphClient: true,
			expectedStatus:   "healthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *Handler
			if tt.useMSGraphClient {
				mockClient := &MockMSGraphClient{}
				handler = NewWithMSGraphClient(mockClient)
			} else {
				handler, _ = New(nil)
			}

			router := setupRouter()
			router.GET("/health", handler.GetHealth)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, response["status"])

			components := response["components"].(map[string]interface{})
			assert.Equal(t, "healthy", components["static_handler"])

			if tt.useMSGraphClient {
				assert.Equal(t, "healthy", components["msgraph_handler"])
			} else {
				assert.Equal(t, "not_configured", components["msgraph_handler"])
			}
		})
	}
}
