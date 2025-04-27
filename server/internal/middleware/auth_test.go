package middleware_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/maynagashev/gophkeeper/server/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Вынести секретный ключ в общее место или передавать в Authenticator.
const jwtSecretKey = "your-very-secret-key"

type jwtClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func TestGetUserIDFromContext(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		expectedID int64
		expectedOK bool
	}{
		{
			name:       "Контекст с UserID",
			ctx:        context.WithValue(context.Background(), middleware.UserIDKey, int64(123)),
			expectedID: 123,
			expectedOK: true,
		},
		{
			name:       "Пустой контекст",
			ctx:        context.Background(),
			expectedID: 0,
			expectedOK: false,
		},
		{
			name:       "Контекст с UserID неверного типа",
			ctx:        context.WithValue(context.Background(), middleware.UserIDKey, "not-an-int64"),
			expectedID: 0,
			expectedOK: false,
		},
		{
			name:       "Nil контекст",
			ctx:        nil,
			expectedID: 0,
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Обработка nil контекста
			if tt.ctx == nil {
				userID, ok := middleware.GetUserIDFromContext(context.Background()) // Передаем пустой контекст
				assert.Equal(t, tt.expectedID, userID)
				assert.Equal(t, tt.expectedOK, ok)
			} else {
				userID, ok := middleware.GetUserIDFromContext(tt.ctx)
				assert.Equal(t, tt.expectedID, userID)
				assert.Equal(t, tt.expectedOK, ok)
			}
		})
	}
}

// Вспомогательная функция для генерации JWT токена.
func generateTestToken(userID int64, secretKey string, expiresAt time.Time) (string, error) {
	claims := jwtClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "test-issuer", // Пример
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

func TestAuthenticator(t *testing.T) {
	// Обработчик, который будет вызван после middleware
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		assert.True(t, ok, "UserID должен быть в контексте")
		assert.NotEqual(t, int64(0), userID, "UserID не должен быть 0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf("OK for user %d", userID)))
	})

	// Оборачиваем обработчик в middleware
	authMiddleware := middleware.Authenticator(nextHandler)

	// Создаем тестовый сервер
	server := httptest.NewServer(authMiddleware)
	defer server.Close()

	tests := []struct {
		name           string
		header         string // Содержимое заголовка Authorization
		expectedStatus int
		expectedBody   string // Подстрока в теле ответа
	}{
		{
			name:           "Успешная аутентификация",
			header:         generateAuthHeader(t, 123, jwtSecretKey, time.Now().Add(time.Hour)),
			expectedStatus: http.StatusOK,
			expectedBody:   "OK for user 123",
		},
		{
			name:           "Нет заголовка Authorization",
			header:         "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Требуется аутентификация",
		},
		{
			name:           "Неверный формат заголовка (нет Bearer)",
			header:         generateTestTokenOnly(t, 456, jwtSecretKey, time.Now().Add(time.Hour)),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Неверный формат токена",
		},
		{
			name:           "Неверный формат заголовка (лишнее слово)",
			header:         "Bearer extra " + generateTestTokenOnly(t, 789, jwtSecretKey, time.Now().Add(time.Hour)),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Неверный формат токена",
		},
		{
			name:           "Невалидный токен (неверный секрет)",
			header:         generateAuthHeader(t, 111, "wrong-secret-key", time.Now().Add(time.Hour)),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Невалидный токен",
		},
		{
			name:           "Истекший токен",
			header:         generateAuthHeader(t, 222, jwtSecretKey, time.Now().Add(-time.Hour)), // Токен истек час назад
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Невалидный токен",
		},
		{
			name:           "Невалидный токен (пустой)",
			header:         "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Неверный формат токена",
		},
		{
			name:           "Невалидный токен (мусор)",
			header:         "Bearer garbage",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Невалидный токен",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Проверяем статус код
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Проверяем тело ответа
			bodyBytes, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(bodyBytes), tt.expectedBody)
		})
	}
}

// Вспомогательная функция для генерации токена и заголовка.
func generateAuthHeader(t *testing.T, userID int64, secretKey string, expiresAt time.Time) string {
	t.Helper()
	token, err := generateTestToken(userID, secretKey, expiresAt)
	require.NoError(t, err, "Ошибка генерации тестового токена")
	return "Bearer " + token
}

// Вспомогательная функция для генерации только токена (для тестов с неверным форматом заголовка).
func generateTestTokenOnly(t *testing.T, userID int64, secretKey string, expiresAt time.Time) string {
	t.Helper()
	token, err := generateTestToken(userID, secretKey, expiresAt)
	require.NoError(t, err, "Ошибка генерации тестового токена")
	return token
}
