package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/maynagashev/gophkeeper/server/internal/handlers"
	"github.com/maynagashev/gophkeeper/server/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock AuthService --- //

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(username, password string) error {
	args := m.Called(username, password)
	return args.Error(0)
}

func (m *MockAuthService) Login(username, password string) (string, error) {
	args := m.Called(username, password)
	return args.String(0), args.Error(1)
}

// --- Tests --- //

func TestNewAuthHandler(t *testing.T) {
	mockService := new(MockAuthService)
	h := handlers.NewAuthHandler(mockService)
	assert.NotNil(t, h)
}

// Вспомогательная функция для создания роутера с обработчиком.
func setupAuthRouter(h *handlers.AuthHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	return r
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		mockUsername    string
		mockPassword    string
		mockReturnError error
		expectedStatus  int
		expectedBody    string // Проверяем подстроку в теле ответа
	}{
		{
			name:            "Успешная регистрация",
			body:            `{"username": "testuser", "password": "password123"}`,
			mockUsername:    "testuser",
			mockPassword:    "password123",
			mockReturnError: nil,
			expectedStatus:  http.StatusCreated,
			expectedBody:    "Пользователь успешно зарегистрирован",
		},
		{
			name:           "Невалидный JSON",
			body:           `{"username": "testuser", "password": "password123"`, // Сломанный JSON
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Неверный формат запроса",
		},
		{
			name:           "Пустой username",
			body:           `{"username": "", "password": "password123"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Имя пользователя и пароль не могут быть пустыми",
		},
		{
			name:           "Пустой password",
			body:           `{"username": "testuser", "password": ""}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Имя пользователя и пароль не могут быть пустыми",
		},
		{
			name:            "Имя пользователя занято",
			body:            `{"username": "existinguser", "password": "password123"}`,
			mockUsername:    "existinguser",
			mockPassword:    "password123",
			mockReturnError: services.ErrUsernameTaken, // Ошибка от сервиса
			expectedStatus:  http.StatusConflict,
			expectedBody:    services.ErrUsernameTaken.Error(),
		},
		{
			name:            "Внутренняя ошибка сервера",
			body:            `{"username": "erroruser", "password": "password123"}`,
			mockUsername:    "erroruser",
			mockPassword:    "password123",
			mockReturnError: errors.New("some internal error"), // Другая ошибка
			expectedStatus:  http.StatusInternalServerError,
			expectedBody:    "Внутренняя ошибка сервера",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			h := handlers.NewAuthHandler(mockService)
			r := setupAuthRouter(h)

			// Настраиваем мок только если ожидается вызов сервиса
			if tt.mockUsername != "" || tt.mockPassword != "" {
				mockService.On("Register", tt.mockUsername, tt.mockPassword).Return(tt.mockReturnError).Once()
			}

			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			// Проверяем статус код
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Проверяем тело ответа (содержит ожидаемую подстроку)
			if tt.expectedBody != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			}

			// Проверяем, что мок был вызван как ожидалось
			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		mockUsername    string
		mockPassword    string
		mockReturnToken string
		mockReturnError error
		expectedStatus  int
		expectedBody    string // Проверяем подстроку
		expectedToken   string // Ожидаемый токен в JSON ответе
	}{
		{
			name:            "Успешный вход",
			body:            `{"username": "testuser", "password": "password123"}`,
			mockUsername:    "testuser",
			mockPassword:    "password123",
			mockReturnToken: "fake-jwt-token",
			mockReturnError: nil,
			expectedStatus:  http.StatusOK,
			expectedToken:   "fake-jwt-token",
		},
		{
			name:           "Невалидный JSON",
			body:           `{"username": "testuser", "password": "password123"`, // Сломанный JSON
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Неверный формат запроса",
		},
		{
			name:           "Пустой username",
			body:           `{"username": "", "password": "password123"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Имя пользователя и пароль не могут быть пустыми",
		},
		{
			name:           "Пустой password",
			body:           `{"username": "testuser", "password": ""}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Имя пользователя и пароль не могут быть пустыми",
		},
		{
			name:            "Неверные креды",
			body:            `{"username": "wronguser", "password": "wrongpassword"}`,
			mockUsername:    "wronguser",
			mockPassword:    "wrongpassword",
			mockReturnError: services.ErrInvalidCredentials, // Ошибка от сервиса
			expectedStatus:  http.StatusUnauthorized,
			expectedBody:    services.ErrInvalidCredentials.Error(),
		},
		{
			name:            "Внутренняя ошибка сервера",
			body:            `{"username": "erroruser", "password": "password123"}`,
			mockUsername:    "erroruser",
			mockPassword:    "password123",
			mockReturnError: errors.New("some internal error"), // Другая ошибка
			expectedStatus:  http.StatusInternalServerError,
			expectedBody:    "Внутренняя ошибка сервера",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			h := handlers.NewAuthHandler(mockService)
			r := setupAuthRouter(h)

			// Настраиваем мок только если ожидается вызов сервиса
			if tt.mockUsername != "" || tt.mockPassword != "" {
				mockService.On("Login", tt.mockUsername, tt.mockPassword).Return(tt.mockReturnToken, tt.mockReturnError).Once()
			}

			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			// Проверяем статус код
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Проверяем тело ответа
			if tt.expectedToken != "" {
				var resp models.LoginResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err, "Ошибка декодирования JSON ответа")
				assert.Equal(t, tt.expectedToken, resp.Token)
			} else if tt.expectedBody != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			}

			// Проверяем, что мок был вызван как ожидалось
			mockService.AssertExpectations(t)
		})
	}
}
