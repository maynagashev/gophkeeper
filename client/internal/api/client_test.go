package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maynagashev/gophkeeper/client/internal/api"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_UploadVault(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	testToken := "test-jwt-token"
	// testUserID := int64(1) // Не используется напрямую, удалено
	testSize := int64(len("test data"))
	testData := "test data"
	// Используем UTC для времени
	testModTime := time.Now().UTC().Truncate(time.Second)
	testModTimeStr := testModTime.Format(time.RFC3339)

	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Успех",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Проверяем заголовки
				assert.Equal(http.MethodPost, r.Method)
				assert.Equal("Bearer "+testToken, r.Header.Get("Authorization"))
				assert.Equal("application/octet-stream", r.Header.Get("Content-Type"))
				assert.Equal(strconv.FormatInt(testSize, 10), r.Header.Get("Content-Length"))
				assert.Equal(testModTimeStr, r.Header.Get("X-Kdbx-Content-Modified-At"))

				// Читаем тело, чтобы убедиться, что оно пришло
				bodyBytes, err := io.ReadAll(r.Body)
				assert.NoError(err)
				assert.Equal(testData, string(bodyBytes))

				w.WriteHeader(http.StatusOK)
			},
			expectedErr: false,
		},
		{
			name: "Ошибка авторизации (401)",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal("Bearer "+testToken, r.Header.Get("Authorization")) // Заголовок все еще должен быть
				w.WriteHeader(http.StatusUnauthorized)
			},
			expectedErr:    true,
			expectedErrMsg: "ошибка авторизации при загрузке",
		},
		{
			name: "Ошибка сервера (500)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedErr:    true,
			expectedErrMsg: "ошибка загрузки на сервер: статус 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := api.NewHTTPClient(server.URL)
			client.SetAuthToken(testToken)

			reader := strings.NewReader(testData)
			err := client.UploadVault(context.Background(), reader, testSize, testModTime)

			if tt.expectedErr {
				require.Error(err)
				if tt.expectedErrMsg != "" {
					assert.Contains(err.Error(), tt.expectedErrMsg)
				}
			} else {
				require.NoError(err)
			}
		})
	}

	// Тест без токена
	t.Run("Без токена авторизации", func(_ *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			assert.Fail("Сервер не должен был получить запрос без токена")
		}))
		defer server.Close()

		client := api.NewHTTPClient(server.URL)
		// Не вызываем SetAuthToken

		reader := strings.NewReader(testData)
		err := client.UploadVault(context.Background(), reader, testSize, testModTime)

		require.Error(err)
		assert.Contains(err.Error(), "токен аутентификации отсутствует")
	})
}

func TestHTTPClient_GetVaultMetadata(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	testToken := "test-jwt-token"
	testVersionID := int64(101)
	testVaultID := int64(11)
	testSize := int64(500)
	testHash := "abc"
	// Используем UTC и добавляем наносекунды для проверки точного парсинга
	testModTime := time.Now().UTC().Truncate(time.Microsecond)
	// testModTimeStr := testModTime.Format(time.RFC3339Nano) // Удалено, т.к. используется только в JSON

	// Готовим ожидаемый JSON ответ
	expectedMeta := models.VaultVersion{
		ID:                testVersionID,
		VaultID:           testVaultID,
		ContentModifiedAt: &testModTime,
		SizeBytes:         &testSize,
		Checksum:          &testHash,
		// CreatedAt не используется в этом тесте, но может быть в реальном ответе
	}
	expectedJSON, _ := json.Marshal(expectedMeta)

	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		expectedResult *models.VaultVersion
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Успех",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(http.MethodGet, r.Method)
				assert.Equal("Bearer "+testToken, r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(expectedJSON)
			},
			expectedResult: &expectedMeta,
			expectedErr:    false,
		},
		{
			name: "Хранилище не найдено (404)",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal("Bearer "+testToken, r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusNotFound)
			},
			expectedResult: nil,
			expectedErr:    true,
			expectedErrMsg: "хранилище не найдено на сервере",
		},
		{
			name: "Ошибка авторизации (401)",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal("Bearer "+testToken, r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusUnauthorized)
			},
			expectedResult: nil,
			expectedErr:    true,
			expectedErrMsg: "ошибка авторизации", // (невалидный или просроченный токен?)",
		},
		{
			name: "Ошибка сервера (500)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedResult: nil,
			expectedErr:    true,
			expectedErrMsg: "ошибка получения метаданных: статус 500",
		},
		{
			name: "Невалидный JSON ответ",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{\"invalid_json"))
			},
			expectedResult: nil,
			expectedErr:    true,
			expectedErrMsg: "ошибка декодирования метаданных",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := api.NewHTTPClient(server.URL)
			client.SetAuthToken(testToken)

			meta, err := client.GetVaultMetadata(context.Background())

			if tt.expectedErr {
				require.Error(err)
				assert.Nil(meta)
				if tt.expectedErrMsg != "" {
					assert.Contains(err.Error(), tt.expectedErrMsg)
				}
			} else {
				require.NoError(err)
				// Сравниваем содержимое структур, а не указатели
				assert.Equal(*tt.expectedResult, *meta)
				// Отдельно проверяем время с учетом возможной потери точности при JSON маршалинге
				assert.NotNil(meta.ContentModifiedAt)
				assert.WithinDuration(testModTime, *meta.ContentModifiedAt, time.Nanosecond)
			}
		})
	}

	// Тест без токена
	t.Run("Без токена авторизации", func(_ *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			assert.Fail("Сервер не должен был получить запрос без токена")
		}))
		defer server.Close()

		client := api.NewHTTPClient(server.URL)
		// Не вызываем SetAuthToken

		meta, err := client.GetVaultMetadata(context.Background())

		require.Error(err)
		assert.Nil(meta)
		assert.Contains(err.Error(), "токен аутентификации отсутствует")
	})
}

// TODO: Добавить TestHTTPClient_GetVaultMetadata
