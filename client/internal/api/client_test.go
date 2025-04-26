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
				// Специальная проверка для ошибки авторизации
				if tt.name == "Ошибка авторизации (401)" {
					require.ErrorIs(err, api.ErrAuthorization)
				} else if tt.expectedErrMsg != "" {
					// Для остальных ошибок проверяем содержание
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

// TestHTTPClient_Register тестирует функцию регистрации нового пользователя.
func TestHTTPClient_Register(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	testUsername := "testuser"
	testPassword := "testpass"
	testToken := "test-jwt-token" // Для проверки ответа сервера
	testUserID := int64(123)

	// Подготавливаем ожидаемое тело запроса и ответа
	expectedRequestBody := map[string]string{
		"username": testUsername,
		"password": testPassword,
	}
	expectedResponseBody := map[string]interface{}{
		"user_id": testUserID,
		"token":   testToken,
	}

	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Успешная регистрация",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Проверяем метод и заголовки
				assert.Equal(http.MethodPost, r.Method)
				assert.Equal("application/json", r.Header.Get("Content-Type"))

				// Проверяем тело запроса
				var requestBody map[string]string
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				assert.NoError(err)
				assert.Equal(expectedRequestBody, requestBody)

				// Отправляем успешный ответ
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				err = json.NewEncoder(w).Encode(expectedResponseBody)
				assert.NoError(err)
			},
			expectedErr: false,
		},
		{
			name: "Ошибка при занятом имени пользователя (409)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusConflict)
				err := json.NewEncoder(w).Encode(map[string]string{
					"error": "Пользователь с таким именем уже существует",
				})
				assert.NoError(err)
			},
			expectedErr:    true,
			expectedErrMsg: "ошибка регистрации на сервере: статус 409",
		},
		{
			name: "Ошибка сервера (500)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedErr:    true,
			expectedErrMsg: "ошибка регистрации на сервере: статус 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := api.NewHTTPClient(server.URL)
			err := client.Register(context.Background(), testUsername, testPassword)

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
}

// TestHTTPClient_Login тестирует функцию входа пользователя.
func TestHTTPClient_Login(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	testUsername := "testuser"
	testPassword := "testpass"
	testToken := "test-jwt-token"
	testUserID := int64(123)

	// Подготавливаем ожидаемое тело запроса и ответа
	expectedRequestBody := map[string]string{
		"username": testUsername,
		"password": testPassword,
	}
	expectedResponseBody := map[string]interface{}{
		"user_id": testUserID,
		"token":   testToken,
	}

	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		expectedErr    bool
		expectedErrMsg string
		expectedToken  string
	}{
		{
			name: "Успешный вход",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Проверяем метод и заголовки
				assert.Equal(http.MethodPost, r.Method)
				assert.Equal("application/json", r.Header.Get("Content-Type"))

				// Проверяем тело запроса
				var requestBody map[string]string
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				assert.NoError(err)
				assert.Equal(expectedRequestBody, requestBody)

				// Отправляем успешный ответ
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				err = json.NewEncoder(w).Encode(expectedResponseBody)
				assert.NoError(err)
			},
			expectedErr:   false,
			expectedToken: testToken,
		},
		{
			name: "Неверные учетные данные (401)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				err := json.NewEncoder(w).Encode(map[string]string{
					"error": "Неверное имя пользователя или пароль",
				})
				assert.NoError(err)
			},
			expectedErr:    true,
			expectedErrMsg: "неверное имя пользователя или пароль",
			expectedToken:  "",
		},
		{
			name: "Пользователь не найден (404)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				err := json.NewEncoder(w).Encode(map[string]string{
					"error": "Пользователь не найден",
				})
				assert.NoError(err)
			},
			expectedErr:    true,
			expectedErrMsg: "ошибка входа на сервере: статус 404",
			expectedToken:  "",
		},
		{
			name: "Ошибка сервера (500)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedErr:    true,
			expectedErrMsg: "ошибка входа на сервере: статус 500",
			expectedToken:  "",
		},
		{
			name: "Невалидный JSON ответ",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{\"invalid_json"))
			},
			expectedErr:    true,
			expectedErrMsg: "ошибка декодирования ответа на вход",
			expectedToken:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := api.NewHTTPClient(server.URL)
			token, err := client.Login(context.Background(), testUsername, testPassword)

			if tt.expectedErr {
				require.Error(err)
				assert.Equal("", token)
				if tt.expectedErrMsg != "" {
					assert.Contains(err.Error(), tt.expectedErrMsg)
				}
			} else {
				require.NoError(err)
				assert.Equal(tt.expectedToken, token)
			}
		})
	}
}

// TestHTTPClient_DownloadVault тестирует функцию скачивания хранилища с сервера.
func TestHTTPClient_DownloadVault(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	testToken := "test-jwt-token"
	testData := "test vault data"

	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		expectedData   string
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Успешное скачивание",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Проверяем метод и заголовки
				assert.Equal(http.MethodGet, r.Method)
				assert.Equal("Bearer "+testToken, r.Header.Get("Authorization"))

				// Отправляем бинарные данные хранилища
				w.Header().Set("Content-Type", "application/octet-stream")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(testData))
			},
			expectedData: testData,
			expectedErr:  false,
		},
		{
			name: "Хранилище не найдено (404)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedData:   "",
			expectedErr:    true,
			expectedErrMsg: "хранилище не найдено для скачивания",
		},
		{
			name: "Ошибка авторизации (401)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			expectedData:   "",
			expectedErr:    true,
			expectedErrMsg: "ошибка авторизации",
		},
		{
			name: "Ошибка сервера (500)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedData:   "",
			expectedErr:    true,
			expectedErrMsg: "ошибка скачивания с сервера: статус 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := api.NewHTTPClient(server.URL)
			client.SetAuthToken(testToken)

			reader, meta, err := client.DownloadVault(context.Background())

			if tt.expectedErr {
				require.Error(err)
				assert.Nil(reader)
				assert.Nil(meta)
				if tt.expectedErrMsg != "" {
					assert.Contains(err.Error(), tt.expectedErrMsg)
				}
				// Особая проверка для ошибки авторизации
				if tt.name == "Ошибка авторизации (401)" {
					require.ErrorIs(err, api.ErrAuthorization)
				}
			} else {
				require.NoError(err)
				assert.NotNil(reader)
				// Meta в данной реализации всегда nil, проверяем это
				assert.Nil(meta)

				// Проверяем содержимое полученного reader
				readData, readErr := io.ReadAll(reader)
				require.NoError(readErr)
				assert.Equal(tt.expectedData, string(readData))

				// Закрываем reader
				closeErr := reader.Close()
				require.NoError(closeErr)
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

		reader, meta, err := client.DownloadVault(context.Background())

		require.Error(err)
		assert.Nil(reader)
		assert.Nil(meta)
		assert.Contains(err.Error(), "токен аутентификации отсутствует")
	})
}

// TestHTTPClient_ListVersions тестирует функцию получения списка версий хранилища.
func TestHTTPClient_ListVersions(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	testToken := "test-jwt-token"
	testLimit := 10
	testOffset := 5
	testCurrentVersionID := int64(105)

	// Создаем тестовые версии
	testVersions := []models.VaultVersion{
		{
			ID:        int64(105),
			VaultID:   int64(1),
			CreatedAt: time.Now().UTC().Add(-time.Hour),
		},
		{
			ID:        int64(104),
			VaultID:   int64(1),
			CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
		},
		{
			ID:        int64(103),
			VaultID:   int64(1),
			CreatedAt: time.Now().UTC().Add(-3 * time.Hour),
		},
	}

	// Готовим ожидаемый JSON-ответ
	expectedResponse := struct {
		Versions         []models.VaultVersion `json:"versions"`
		CurrentVersionID int64                 `json:"current_version_id"`
	}{
		Versions:         testVersions,
		CurrentVersionID: testCurrentVersionID,
	}
	expectedJSON, _ := json.Marshal(expectedResponse)

	tests := []struct {
		name                  string
		serverHandler         http.HandlerFunc
		expectedVersions      []models.VaultVersion
		expectedCurrentID     int64
		expectedErr           bool
		expectedErrMsg        string
		expectedQueryContains string // Для проверки параметров запроса
	}{
		{
			name: "Успешное получение списка версий",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Проверяем метод и заголовки
				assert.Equal(http.MethodGet, r.Method)
				assert.Equal("Bearer "+testToken, r.Header.Get("Authorization"))

				// Проверяем параметры запроса
				assert.Contains(r.URL.RawQuery, "limit="+strconv.Itoa(testLimit))
				assert.Contains(r.URL.RawQuery, "offset="+strconv.Itoa(testOffset))

				// Отправляем успешный ответ
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(expectedJSON)
			},
			expectedVersions:      testVersions,
			expectedCurrentID:     testCurrentVersionID,
			expectedErr:           false,
			expectedQueryContains: "limit=" + strconv.Itoa(testLimit) + "&offset=" + strconv.Itoa(testOffset),
		},
		{
			name: "Ошибка авторизации (401)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			expectedVersions:  nil,
			expectedCurrentID: 0,
			expectedErr:       true,
			expectedErrMsg:    "ошибка авторизации",
		},
		{
			name: "Ошибка сервера (500)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedVersions:  nil,
			expectedCurrentID: 0,
			expectedErr:       true,
			expectedErrMsg:    "ошибка получения списка версий: статус 500",
		},
		{
			name: "Невалидный JSON ответ",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{\"invalid_json"))
			},
			expectedVersions:  nil,
			expectedCurrentID: 0,
			expectedErr:       true,
			expectedErrMsg:    "ошибка декодирования списка версий",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := api.NewHTTPClient(server.URL)
			client.SetAuthToken(testToken)

			versions, currentID, err := client.ListVersions(context.Background(), testLimit, testOffset)

			if tt.expectedErr {
				require.Error(err)
				assert.Nil(versions)
				assert.Equal(int64(0), currentID)
				if tt.expectedErrMsg != "" {
					assert.Contains(err.Error(), tt.expectedErrMsg)
				}
				// Особая проверка для ошибки авторизации
				if tt.name == "Ошибка авторизации (401)" {
					require.ErrorIs(err, api.ErrAuthorization)
				}
			} else {
				require.NoError(err)
				assert.NotNil(versions)
				assert.Equal(tt.expectedVersions, versions)
				assert.Equal(tt.expectedCurrentID, currentID)
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

		versions, currentID, err := client.ListVersions(context.Background(), testLimit, testOffset)

		require.Error(err)
		assert.Nil(versions)
		assert.Equal(int64(0), currentID)
		assert.Contains(err.Error(), "токен аутентификации отсутствует")
	})
}

// TestHTTPClient_RollbackToVersion тестирует функцию отката хранилища к указанной версии.
func TestHTTPClient_RollbackToVersion(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	testToken := "test-jwt-token"
	testVersionID := int64(103)

	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Успешный откат",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Проверяем метод и заголовки
				assert.Equal(http.MethodPost, r.Method)
				assert.Equal("Bearer "+testToken, r.Header.Get("Authorization"))
				assert.Equal("application/json", r.Header.Get("Content-Type"))

				// Проверяем тело запроса
				var requestBody map[string]int64
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				assert.NoError(err)
				assert.Equal(testVersionID, requestBody["version_id"])

				// Отправляем успешный ответ (204 No Content для успешного отката)
				w.WriteHeader(http.StatusNoContent)
			},
			expectedErr: false,
		},
		{
			name: "Версия не найдена (404)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedErr:    true,
			expectedErrMsg: "указанная версия или хранилище не найдены для отката",
		},
		{
			name: "Неверный запрос (400)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			expectedErr:    true,
			expectedErrMsg: "неверный запрос на откат",
		},
		{
			name: "Ошибка авторизации (401)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			expectedErr:    true,
			expectedErrMsg: "ошибка авторизации",
		},
		{
			name: "Ошибка сервера (500)",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedErr:    true,
			expectedErrMsg: "ошибка отката на сервере: статус 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := api.NewHTTPClient(server.URL)
			client.SetAuthToken(testToken)

			err := client.RollbackToVersion(context.Background(), testVersionID)

			if tt.expectedErr {
				require.Error(err)
				if tt.expectedErrMsg != "" {
					assert.Contains(err.Error(), tt.expectedErrMsg)
				}
				// Особая проверка для ошибки авторизации
				if tt.name == "Ошибка авторизации (401)" {
					require.ErrorIs(err, api.ErrAuthorization)
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

		err := client.RollbackToVersion(context.Background(), testVersionID)

		require.Error(err)
		assert.Contains(err.Error(), "токен аутентификации отсутствует")
	})
}
