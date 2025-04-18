package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/maynagashev/gophkeeper/models"
	"github.com/maynagashev/gophkeeper/server/internal/handlers"
	"github.com/maynagashev/gophkeeper/server/internal/middleware"
	"github.com/maynagashev/gophkeeper/server/internal/services"
)

// MockVaultService is a mock implementation of VaultService interface.
type MockVaultService struct {
	mock.Mock
}

func (m *MockVaultService) GetVaultMetadata(userID int64) (*models.VaultVersion, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VaultVersion), args.Error(1) //nolint:errcheck // Acceptable for mocks
}

func (m *MockVaultService) UploadVault(userID int64, reader io.Reader, size int64, contentType string) error {
	args := m.Called(userID, reader, size, contentType)
	// Consume the reader to simulate reading the body
	_, _ = io.Copy(io.Discard, reader)
	return args.Error(0)
}

func (m *MockVaultService) DownloadVault(userID int64) (io.ReadCloser, *models.VaultVersion, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	reader, ok := args.Get(0).(io.ReadCloser)
	if !ok && args.Get(0) != nil {
		panic("mock DownloadVault reader is not io.ReadCloser")
	}
	meta, ok := args.Get(1).(*models.VaultVersion)
	if !ok && args.Get(1) != nil {
		panic("mock DownloadVault meta is not *models.VaultVersion")
	}
	return reader, meta, args.Error(2)
}

func (m *MockVaultService) ListVersions(userID int64, limit, offset int) ([]models.VaultVersion, error) {
	args := m.Called(userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.VaultVersion), args.Error(1) //nolint:errcheck // Acceptable for mocks
}

func (m *MockVaultService) RollbackToVersion(userID, versionID int64) error {
	args := m.Called(userID, versionID)
	return args.Error(0)
}

func TestVaultHandler_GetMetadata(t *testing.T) {
	testUserID := int64(1)
	testVaultID := int64(10)
	testVersionID := int64(100)
	testCreatedAt := time.Now().Truncate(time.Second)

	tests := []struct {
		name               string
		mockReturnVersion  *models.VaultVersion
		mockReturnErr      error
		expectedStatusCode int
		expectedBody       string // JSON string or error message
	}{
		{
			name: "Успех",
			mockReturnVersion: &models.VaultVersion{
				ID:        testVersionID,
				VaultID:   testVaultID,
				SizeBytes: nil, // Example, adjust as needed
				CreatedAt: testCreatedAt,
			},
			mockReturnErr:      nil,
			expectedStatusCode: http.StatusOK,
			expectedBody: func() string {
				expected := models.VaultVersion{
					ID:        testVersionID,
					VaultID:   testVaultID,
					CreatedAt: testCreatedAt,
				}
				bodyBytes, _ := json.Marshal(expected)
				return string(bodyBytes) + "\n" // Add newline from Encode
			}(),
		},
		{
			name:               "Хранилище не найдено",
			mockReturnVersion:  nil,
			mockReturnErr:      services.ErrVaultNotFound,
			expectedStatusCode: http.StatusNotFound,
			expectedBody:       "Хранилище не найдено\n",
		},
		{
			name:               "Внутренняя ошибка сервера",
			mockReturnVersion:  nil,
			mockReturnErr:      errors.New("internal error"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "Внутренняя ошибка сервера\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockVaultService)
			handler := handlers.NewVaultHandler(mockService)

			// Setup mock expectation
			mockService.On("GetVaultMetadata", testUserID).Return(tt.mockReturnVersion, tt.mockReturnErr)

			// Create request and recorder
			req := httptest.NewRequest(http.MethodGet, "/api/vault", nil)
			// Add user ID to context
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, testUserID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			// Create router and serve request
			router := chi.NewRouter()
			router.Get("/api/vault", handler.GetMetadata) // Use the actual route pattern
			router.ServeHTTP(rr, req)

			// Assertions
			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
			mockService.AssertExpectations(t)
		})
	}

	// Test case for missing user ID in context (should not happen with middleware, but good to check handler)
	t.Run("Отсутствует UserID в контексте", func(t *testing.T) {
		mockService := new(MockVaultService) // No expectations needed
		handler := handlers.NewVaultHandler(mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/vault", nil)
		rr := httptest.NewRecorder()

		router := chi.NewRouter()
		router.Get("/api/vault", handler.GetMetadata)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Equal(t, "Внутренняя ошибка сервера\n", rr.Body.String())
		// No calls expected to the service
		mockService.AssertNotCalled(t, "GetVaultMetadata", mock.Anything)
	})
}

// --- Placeholder for other tests ---

func TestVaultHandler_Upload(t *testing.T) {
	testUserID := int64(1)
	testFileSize := int64(1024)
	testContentType := "application/octet-stream"

	tests := []struct {
		name               string
		body               io.Reader
		headers            map[string]string
		mockReturnErr      error
		expectedStatusCode int
		expectedBody       string
		setupMock          func(mockSvc *MockVaultService) // Function to set up mock expectations
	}{
		{
			name: "Success",
			body: strings.NewReader(string(make([]byte, testFileSize))),
			headers: map[string]string{
				"Content-Length": strconv.FormatInt(testFileSize, 10),
				"Content-Type":   testContentType,
			},
			mockReturnErr:      nil,
			expectedStatusCode: http.StatusOK,
			expectedBody:       "Файл успешно загружен\n",
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("UploadVault", testUserID, mock.Anything, testFileSize, testContentType).Return(nil)
			},
		},
		{
			name: "Missing Content-Length",
			body: strings.NewReader("test"),
			headers: map[string]string{
				"Content-Type": testContentType,
			},
			mockReturnErr:      nil, // Service not called
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       "Неверный или отсутствующий заголовок Content-Length\n",
			// Rename unused parameter to _
			setupMock: func(_ *MockVaultService) { /* No service call expected */ },
		},
		{
			name: "Invalid Content-Length",
			body: strings.NewReader("test"),
			headers: map[string]string{
				"Content-Length": "invalid",
				"Content-Type":   testContentType,
			},
			mockReturnErr:      nil, // Service not called
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       "Неверный или отсутствующий заголовок Content-Length\n",
			// Rename unused parameter to _
			setupMock: func(_ *MockVaultService) { /* No service call expected */ },
		},
		{
			name: "Zero Content-Length",
			body: strings.NewReader(""),
			headers: map[string]string{
				"Content-Length": "0",
				"Content-Type":   testContentType,
			},
			mockReturnErr:      nil, // Service not called
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       "Неверный или отсутствующий заголовок Content-Length\n",
			// Rename unused parameter to _
			setupMock: func(_ *MockVaultService) { /* No service call expected */ },
		},
		{
			name: "Internal Service Error",
			body: strings.NewReader(string(make([]byte, testFileSize))),
			headers: map[string]string{
				"Content-Length": strconv.FormatInt(testFileSize, 10),
				"Content-Type":   testContentType,
			},
			mockReturnErr:      errors.New("service upload error"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "Внутренняя ошибка сервера при загрузке файла\n",
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("UploadVault", testUserID, mock.Anything, testFileSize, testContentType).
					Return(errors.New("service upload error")) // Shorten line
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockVaultService)
			handler := handlers.NewVaultHandler(mockService)

			// Setup mock expectation using the setup function
			tt.setupMock(mockService)

			// Create request and recorder
			req := httptest.NewRequest(http.MethodPost, "/api/vault/upload", tt.body)
			// Add user ID to context
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, testUserID)
			req = req.WithContext(ctx)

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			// If Content-Type is not set, chi router might default it.
			// Handle the case where it's expected to be defaulted.
			if _, ok := tt.headers["Content-Type"]; !ok {
				// For the test, we expect it to be defaulted if not provided
				mockService.On("UploadVault", testUserID, mock.Anything, testFileSize, "application/octet-stream").Maybe()
			}

			rr := httptest.NewRecorder()

			// Create router and serve request
			router := chi.NewRouter()
			router.Post("/api/vault/upload", handler.Upload) // Use the actual route pattern
			router.ServeHTTP(rr, req)

			// Assertions
			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
			mockService.AssertExpectations(t)
		})
	}

	// Test case for missing user ID in context
	t.Run("Отсутствует UserID в контексте", func(t *testing.T) {
		mockService := new(MockVaultService) // No expectations needed
		handler := handlers.NewVaultHandler(mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/vault/upload", strings.NewReader("test"))
		req.Header.Set("Content-Length", "4")
		rr := httptest.NewRecorder()

		router := chi.NewRouter()
		router.Post("/api/vault/upload", handler.Upload)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Equal(t, "Внутренняя ошибка сервера\n", rr.Body.String())
		// No calls expected to the service
		mockService.AssertNotCalled(t, "UploadVault", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestVaultHandler_Download(t *testing.T) {
	testUserID := int64(1)
	testVaultID := int64(10)
	testVersionID := int64(101)
	testFileSize := int64(512)
	testObjectKey := "user_1/vault_xyz.kdbx"
	testCreatedAt := time.Now()
	fileContent := "dummy file content"

	tests := []struct {
		name               string
		mockReturnReader   io.ReadCloser
		mockReturnMeta     *models.VaultVersion
		mockReturnErr      error
		expectedStatusCode int
		expectedHeaders    map[string]string
		expectedBody       string
		setupMock          func(mockSvc *MockVaultService)
	}{
		{
			name:             "Success",
			mockReturnReader: io.NopCloser(strings.NewReader(fileContent)),
			mockReturnMeta: &models.VaultVersion{
				ID:        testVersionID,
				VaultID:   testVaultID,
				ObjectKey: testObjectKey,
				SizeBytes: &testFileSize,
				CreatedAt: testCreatedAt,
			},
			mockReturnErr:      nil,
			expectedStatusCode: http.StatusOK,
			expectedHeaders: map[string]string{
				"Content-Disposition": `attachment; filename="gophkeeper_vault.kdbx"`,
				"Content-Type":        "application/octet-stream",
				"Content-Length":      strconv.FormatInt(testFileSize, 10),
			},
			expectedBody: fileContent,
			setupMock: func(mockSvc *MockVaultService) {
				mockReader := io.NopCloser(strings.NewReader(fileContent))
				mockMeta := &models.VaultVersion{
					ID:        testVersionID,
					VaultID:   testVaultID,
					ObjectKey: testObjectKey,
					SizeBytes: &testFileSize,
					CreatedAt: testCreatedAt,
				}
				mockSvc.On("DownloadVault", testUserID).Return(mockReader, mockMeta, nil)
			},
		},
		{
			name:               "Хранилище не найдено",
			mockReturnReader:   nil,
			mockReturnMeta:     nil,
			mockReturnErr:      services.ErrVaultNotFound,
			expectedStatusCode: http.StatusNotFound,
			expectedHeaders:    map[string]string{}, // No specific headers expected on error
			expectedBody:       "Хранилище не найдено\n",
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("DownloadVault", testUserID).Return(nil, nil, services.ErrVaultNotFound)
			},
		},
		{
			name:               "Внутренняя ошибка сервиса",
			mockReturnReader:   nil,
			mockReturnMeta:     nil,
			mockReturnErr:      errors.New("internal download error"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedHeaders:    map[string]string{}, // No specific headers expected on error
			expectedBody:       "Внутренняя ошибка сервера при скачивании файла\n",
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("DownloadVault", testUserID).Return(nil, nil, errors.New("internal download error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockVaultService)
			handler := handlers.NewVaultHandler(mockService)

			tt.setupMock(mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/vault/download", nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, testUserID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Get("/api/vault/download", handler.Download)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			// Check headers only on success
			if tt.expectedStatusCode == http.StatusOK {
				for key, expectedValue := range tt.expectedHeaders {
					assert.Equal(t, expectedValue, rr.Header().Get(key), "Несоответствие заголовка для ключа: %s", key)
				}
			}
			assert.Equal(t, tt.expectedBody, rr.Body.String())
			mockService.AssertExpectations(t)
		})
	}

	// Test case for missing user ID in context
	t.Run("Отсутствует UserID в контексте", func(t *testing.T) {
		mockService := new(MockVaultService)
		handler := handlers.NewVaultHandler(mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/vault/download", nil)
		rr := httptest.NewRecorder()

		router := chi.NewRouter()
		router.Get("/api/vault/download", handler.Download)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Equal(t, "Внутренняя ошибка сервера\n", rr.Body.String())
		mockService.AssertNotCalled(t, "DownloadVault", mock.Anything)
	})
}

func TestVaultHandler_ListVersions(t *testing.T) {
	testUserID := int64(1)
	testVaultID := int64(10)
	testVersion1 := models.VaultVersion{ID: 101, VaultID: testVaultID, CreatedAt: time.Now().Add(-time.Hour)}
	testVersion2 := models.VaultVersion{ID: 102, VaultID: testVaultID, CreatedAt: time.Now()}

	tests := []struct {
		name               string
		queryParams        string // e.g., "limit=10&offset=5"
		mockLimit          int
		mockOffset         int
		mockReturnVersions []models.VaultVersion
		mockReturnErr      error
		expectedStatusCode int
		expectedBody       string // JSON string or error message
		setupMock          func(mockSvc *MockVaultService)
	}{
		{
			name:               "Успех - Пагинация по умолчанию",
			queryParams:        "",
			mockLimit:          20,                                                // Default limit
			mockOffset:         0,                                                 // Default offset
			mockReturnVersions: []models.VaultVersion{testVersion2, testVersion1}, // Example order
			mockReturnErr:      nil,
			expectedStatusCode: http.StatusOK,
			expectedBody: func() string {
				bodyBytes, _ := json.Marshal([]models.VaultVersion{testVersion2, testVersion1})
				return string(bodyBytes) + "\n"
			}(),
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("ListVersions", testUserID, 20, 0).Return([]models.VaultVersion{testVersion2, testVersion1}, nil)
			},
		},
		{
			name:               "Успех - Пользовательская пагинация",
			queryParams:        "limit=1&offset=1",
			mockLimit:          1,
			mockOffset:         1,
			mockReturnVersions: []models.VaultVersion{testVersion1},
			mockReturnErr:      nil,
			expectedStatusCode: http.StatusOK,
			expectedBody: func() string {
				bodyBytes, _ := json.Marshal([]models.VaultVersion{testVersion1})
				return string(bodyBytes) + "\n"
			}(),
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("ListVersions", testUserID, 1, 1).Return([]models.VaultVersion{testVersion1}, nil)
			},
		},
		{
			name:               "Успех - Пустой список",
			queryParams:        "",
			mockLimit:          20,
			mockOffset:         0,
			mockReturnVersions: []models.VaultVersion{}, // Empty slice
			mockReturnErr:      nil,
			expectedStatusCode: http.StatusOK,
			expectedBody:       "[]\n", // Empty JSON array
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("ListVersions", testUserID, 20, 0).Return([]models.VaultVersion{}, nil)
			},
		},
		{
			name:               "Внутренняя ошибка сервиса",
			queryParams:        "",
			mockLimit:          20,
			mockOffset:         0,
			mockReturnVersions: nil,
			mockReturnErr:      errors.New("internal list error"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "Внутренняя ошибка сервера\n",
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("ListVersions", testUserID, 20, 0).Return(nil, errors.New("internal list error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockVaultService)
			handler := handlers.NewVaultHandler(mockService)

			tt.setupMock(mockService)

			url := "/api/vault/versions"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, testUserID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Get("/api/vault/versions", handler.ListVersions)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
			mockService.AssertExpectations(t)
		})
	}

	// Test case for missing user ID in context
	t.Run("Отсутствует UserID в контексте", func(t *testing.T) {
		mockService := new(MockVaultService)
		handler := handlers.NewVaultHandler(mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/vault/versions", nil)
		rr := httptest.NewRecorder()

		router := chi.NewRouter()
		router.Get("/api/vault/versions", handler.ListVersions)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Equal(t, "Внутренняя ошибка сервера\n", rr.Body.String())
		mockService.AssertNotCalled(t, "ListVersions", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestVaultHandler_Rollback(t *testing.T) {
	testUserID := int64(1)
	testValidVersionID := int64(101)

	tests := []struct {
		name               string
		requestBody        string
		mockRollbackInput  int64 // versionID passed to mock service
		mockReturnErr      error
		expectedStatusCode int
		expectedBody       string
		setupMock          func(mockSvc *MockVaultService)
	}{
		{
			name:               "Успех",
			requestBody:        `{"version_id": ` + strconv.FormatInt(testValidVersionID, 10) + `}`,
			mockRollbackInput:  testValidVersionID,
			mockReturnErr:      nil,
			expectedStatusCode: http.StatusNoContent,
			expectedBody:       "", // No body on 204
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("RollbackToVersion", testUserID, testValidVersionID).Return(nil)
			},
		},
		{
			name:               "Неверный JSON",
			requestBody:        `{"version_id": }`,
			mockRollbackInput:  0, // Not called
			mockReturnErr:      nil,
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       "Неверный формат запроса\n",
			// Rename unused parameter to _
			setupMock: func(_ *MockVaultService) { /* Not called */ },
		},
		{
			name:               "Отсутствует version_id",
			requestBody:        `{}`, // Missing field
			mockRollbackInput:  0,    // Not called
			mockReturnErr:      nil,
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       "Неверный ID версии\n", // Because VersionID will be 0
			// Rename unused parameter to _
			setupMock: func(_ *MockVaultService) { /* Not called */ },
		},
		{
			name:               "Неверный version_id (ноль)",
			requestBody:        `{"version_id": 0}`,
			mockRollbackInput:  0, // Not called
			mockReturnErr:      nil,
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       "Неверный ID версии\n",
			// Rename unused parameter to _
			setupMock: func(_ *MockVaultService) { /* Not called */ },
		},
		{
			name:               "Хранилище/Версия не найдена",
			requestBody:        `{"version_id": 999}`,
			mockRollbackInput:  999,
			mockReturnErr:      services.ErrVersionNotFound,
			expectedStatusCode: http.StatusNotFound,
			expectedBody:       "Указанное хранилище или версия не найдены\n",
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("RollbackToVersion", testUserID, int64(999)).Return(services.ErrVersionNotFound)
			},
		},
		{
			name:               "Доступ запрещен",
			requestBody:        `{"version_id": ` + strconv.FormatInt(testValidVersionID, 10) + `}`,
			mockRollbackInput:  testValidVersionID,
			mockReturnErr:      services.ErrForbidden,
			expectedStatusCode: http.StatusForbidden,
			expectedBody:       "Доступ запрещен\n",
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("RollbackToVersion", testUserID, testValidVersionID).Return(services.ErrForbidden)
			},
		},
		{
			name:               "Внутренняя ошибка сервиса",
			requestBody:        `{"version_id": ` + strconv.FormatInt(testValidVersionID, 10) + `}`,
			mockRollbackInput:  testValidVersionID,
			mockReturnErr:      errors.New("internal rollback error"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "Внутренняя ошибка сервера\n",
			setupMock: func(mockSvc *MockVaultService) {
				mockSvc.On("RollbackToVersion", testUserID, testValidVersionID).Return(errors.New("internal rollback error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockVaultService)
			handler := handlers.NewVaultHandler(mockService)

			tt.setupMock(mockService)

			req := httptest.NewRequest(http.MethodPost, "/api/vault/rollback", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, testUserID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Post("/api/vault/rollback", handler.Rollback)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
			mockService.AssertExpectations(t)
		})
	}

	// Test case for missing user ID in context
	t.Run("Отсутствует UserID в контексте", func(t *testing.T) {
		mockService := new(MockVaultService)
		handler := handlers.NewVaultHandler(mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/vault/rollback", strings.NewReader(`{"version_id": 1}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router := chi.NewRouter()
		router.Post("/api/vault/rollback", handler.Rollback)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Equal(t, "Внутренняя ошибка сервера\n", rr.Body.String())
		mockService.AssertNotCalled(t, "RollbackToVersion", mock.Anything, mock.Anything)
	})
}
