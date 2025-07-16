package msgraphhandler

import (
	"context"
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
		wantNil bool
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantNil: true,
			wantErr: false,
		},
		{
			name: "empty config",
			config: &Config{
				MSGraphConfig: nil,
			},
			wantNil: true,
			wantErr: false,
		},
		{
			name: "valid config",
			config: &Config{
				MSGraphConfig: &msgraph.Config{
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					TenantID:     "test-tenant-id",
				},
			},
			wantNil: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := New(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, handler)
			} else {
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestNewWithClient(t *testing.T) {
	mockClient := &MockMSGraphClient{}
	handler := NewWithClient(mockClient)

	assert.NotNil(t, handler)
	assert.Equal(t, mockClient, handler.msgraphClient)
}

func TestHandler_GetDocuments(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*MockMSGraphClient)
		wantErr   bool
		wantNil   bool
	}{
		{
			name: "successful retrieval",
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
			wantErr: false,
			wantNil: false,
		},
		{
			name: "client error",
			setupMock: func(m *MockMSGraphClient) {
				m.On("GetOneNoteDataAsJSON", mock.Anything).Return((*types.DocumentCollection)(nil), errors.New("client error"))
			},
			wantErr: true,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMSGraphClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			handler := NewWithClient(mockClient)
			ctx := context.Background()

			result, err := handler.GetDocuments(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, result)
			} else if !tt.wantErr {
				assert.NotNil(t, result)
			}
		})
	}
}

func TestHandler_GetDocuments_NotConfigured(t *testing.T) {
	handler := &Handler{msgraphClient: nil}
	ctx := context.Background()

	result, err := handler.GetDocuments(ctx)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestHandler_ExtractAllData(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockMSGraphClient)
		useNilClient   bool
		expectedStatus int
	}{
		{
			name: "successful retrieval",
			setupMock: func(m *MockMSGraphClient) {
				collection := types.NewDocumentCollection("onenote")
				m.On("GetOneNoteDataAsJSON", mock.Anything).Return(collection, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "client error",
			setupMock: func(m *MockMSGraphClient) {
				m.On("GetOneNoteDataAsJSON", mock.Anything).Return((*types.DocumentCollection)(nil), errors.New("client error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "not configured",
			useNilClient:   true,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *Handler
			if tt.useNilClient {
				handler = &Handler{msgraphClient: nil}
			} else {
				mockClient := &MockMSGraphClient{}
				if tt.setupMock != nil {
					tt.setupMock(mockClient)
				}
				handler = NewWithClient(mockClient)
			}

			router := setupRouter()
			router.GET("/pipeline", handler.ExtractAllData)

			req := httptest.NewRequest(http.MethodGet, "/pipeline", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_GetHealth(t *testing.T) {
	tests := []struct {
		name         string
		useNilClient bool
		expectedMsg  string
	}{
		{
			name:         "configured",
			useNilClient: false,
			expectedMsg:  "healthy",
		},
		{
			name:         "not configured",
			useNilClient: true,
			expectedMsg:  "not_configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *Handler
			if tt.useNilClient {
				handler = &Handler{msgraphClient: nil}
			} else {
				mockClient := &MockMSGraphClient{}
				handler = NewWithClient(mockClient)
			}

			router := setupRouter()
			router.GET("/health", handler.GetHealth)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedMsg)
		})
	}
}

func TestHandler_IsConfigured(t *testing.T) {
	tests := []struct {
		name         string
		useNilClient bool
		expected     bool
	}{
		{
			name:         "configured",
			useNilClient: false,
			expected:     true,
		},
		{
			name:         "not configured",
			useNilClient: true,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *Handler
			if tt.useNilClient {
				handler = &Handler{msgraphClient: nil}
			} else {
				mockClient := &MockMSGraphClient{}
				handler = NewWithClient(mockClient)
			}

			result := handler.IsConfigured()
			assert.Equal(t, tt.expected, result)
		})
	}
}
