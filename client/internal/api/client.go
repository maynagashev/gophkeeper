package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io" // Добавляем для будущих методов
	"net/http"
	"net/url"
	"strconv" // Добавляем для ListVersions
	"time"    // Добавили time

	"github.com/maynagashev/gophkeeper/models" // Импортируем общие модели
)

// ErrAuthorization сигнализирует об ошибке авторизации (401).
var ErrAuthorization = errors.New("ошибка авторизации")

// Client определяет интерфейс для взаимодействия с API сервера GophKeeper.
type Client interface {
	// Register регистрирует нового пользователя.
	Register(ctx context.Context, username, password string) error
	// Login аутентифицирует пользователя и возвращает JWT токен.
	Login(ctx context.Context, username, password string) (string, error)
	// GetVaultMetadata получает метаданные текущей версии хранилища.
	GetVaultMetadata(ctx context.Context) (*models.VaultVersion, error)
	// UploadVault загружает файл хранилища на сервер.
	UploadVault(ctx context.Context, data io.Reader, size int64, contentModifiedAt time.Time) error
	// DownloadVault скачивает текущую версию файла хранилища.
	DownloadVault(ctx context.Context) (io.ReadCloser, *models.VaultVersion, error)
	// ListVersions получает список версий хранилища.
	ListVersions(ctx context.Context, limit, offset int) ([]models.VaultVersion, error)
	// RollbackToVersion откатывает хранилище к указанной версии.
	RollbackToVersion(ctx context.Context, versionID int64) error
	// SetAuthToken устанавливает JWT токен для аутентифицированных запросов.
	SetAuthToken(token string)
}

// httpClient реализует интерфейс Client для взаимодействия с сервером по HTTP.
type httpClient struct {
	baseURL    string       // Базовый URL сервера, например "http://localhost:8080"
	httpClient *http.Client // HTTP клиент для выполнения запросов
	authToken  string       // JWT токен для аутентифицированных запросов
}

// NewHTTPClient создает новый экземпляр API клиента.
func NewHTTPClient(baseURL string) Client {
	return &httpClient{
		baseURL:    baseURL,
		httpClient: &http.Client{}, // Используем стандартный HTTP клиент
	}
}

// Register отправляет запрос на регистрацию на сервер.
func (c *httpClient) Register(ctx context.Context, username, password string) error {
	// Формируем URL эндпоинта регистрации
	registerURL, err := url.JoinPath(c.baseURL, "/api/register")
	if err != nil {
		return fmt.Errorf("ошибка формирования URL для регистрации: %w", err)
	}

	// Создаем тело запроса
	requestBody := models.RegisterRequest{
		Username: username,
		Password: password,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("ошибка кодирования данных для регистрации: %w", err)
	}

	// Создаем HTTP запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса на регистрацию: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// TODO: Добавить обработку сетевых ошибок (таймауты, недоступность сервера)
		return fmt.Errorf("ошибка выполнения запроса на регистрацию: %w", err)
	}
	defer resp.Body.Close() // Важно закрывать тело ответа

	// Проверяем статус код ответа
	if resp.StatusCode != http.StatusCreated {
		// TODO: Читать тело ответа для получения сообщения об ошибке от сервера
		return fmt.Errorf("ошибка регистрации на сервере: статус %d", resp.StatusCode)
	}

	return nil // Успешная регистрация
}

// Login отправляет запрос на вход на сервер и сохраняет токен.
func (c *httpClient) Login(ctx context.Context, username, password string) (string, error) {
	loginURL, err := url.JoinPath(c.baseURL, "/api/login")
	if err != nil {
		return "", fmt.Errorf("ошибка формирования URL для входа: %w", err)
	}

	requestBody := models.LoginRequest{
		Username: username,
		Password: password,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("ошибка кодирования данных для входа: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса на вход: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// TODO: Добавить обработку сетевых ошибок
		return "", fmt.Errorf("ошибка выполнения запроса на вход: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO: Читать тело ответа для получения сообщения об ошибке
		if resp.StatusCode == http.StatusUnauthorized {
			return "", errors.New("неверное имя пользователя или пароль") // Можно вернуть кастомную ошибку
		}
		return "", fmt.Errorf("ошибка входа на сервере: статус %d", resp.StatusCode)
	}

	// Декодируем ответ для получения токена
	var loginResponse models.LoginResponse
	if err = json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		return "", fmt.Errorf("ошибка декодирования ответа на вход: %w", err)
	}

	if loginResponse.Token == "" {
		return "", errors.New("сервер вернул пустой токен")
	}

	// Сохраняем токен в клиенте для последующих запросов
	c.authToken = loginResponse.Token

	return loginResponse.Token, nil
}

// helper function to add auth header.
func (c *httpClient) setAuthHeader(req *http.Request) error {
	if c.authToken == "" {
		return errors.New("токен аутентификации отсутствует") // Или другая ошибка, сигнализирующая о необходимости логина
	}
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	return nil
}

// GetVaultMetadata получает метаданные текущей версии хранилища с сервера.
func (c *httpClient) GetVaultMetadata(ctx context.Context) (*models.VaultVersion, error) {
	metadataURL, err := url.JoinPath(c.baseURL, "/api/vault")
	if err != nil {
		return nil, fmt.Errorf("ошибка формирования URL для метаданных: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса на получение метаданных: %w", err)
	}

	// Добавляем заголовок авторизации
	if err = c.setAuthHeader(req); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// TODO: Обработка сетевых ошибок
		return nil, fmt.Errorf("ошибка выполнения запроса на получение метаданных: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, errors.New("хранилище не найдено на сервере") // Кастомная ошибка
		}
		if resp.StatusCode == http.StatusUnauthorized {
			// Возвращаем нашу специальную ошибку
			return nil, ErrAuthorization
		}
		// TODO: Читать тело для деталей
		return nil, fmt.Errorf("ошибка получения метаданных: статус %d", resp.StatusCode)
	}

	var metadata models.VaultVersion
	if err = json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("ошибка декодирования метаданных: %w", err)
	}

	return &metadata, nil
}

// UploadVault загружает данные хранилища на сервер.
func (c *httpClient) UploadVault(ctx context.Context, data io.Reader, size int64, contentModifiedAt time.Time) error {
	uploadURL, err := url.JoinPath(c.baseURL, "/api/vault/upload")
	if err != nil {
		return fmt.Errorf("ошибка формирования URL для загрузки: %w", err)
	}

	// Используем data напрямую как тело запроса
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, data)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса на загрузку: %w", err)
	}

	// Устанавливаем необходимые заголовки
	req.Header.Set("Content-Type", "application/octet-stream") // Тип контента для KDBX
	req.Header.Set("Content-Length", strconv.FormatInt(size, 10))
	// Добавляем заголовок с временем модификации контента
	modTimeStr := contentModifiedAt.UTC().Format(time.RFC3339)
	req.Header.Set("X-Kdbx-Content-Modified-At", modTimeStr)

	if err = c.setAuthHeader(req); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// TODO: Обработка сетевых ошибок
		return fmt.Errorf("ошибка выполнения запроса на загрузку: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close() // Закрываем тело в случае ошибки
		if resp.StatusCode == http.StatusUnauthorized {
			// Возвращаем нашу специальную ошибку
			return ErrAuthorization
		}
		if resp.StatusCode == http.StatusConflict {
			return errors.New("конфликт версий при загрузке") // Возвращаем ошибку конфликта
		}
		// TODO: Читать тело для деталей
		return fmt.Errorf("ошибка загрузки на сервер: статус %d", resp.StatusCode)
	}

	return nil // Успешная загрузка
}

// DownloadVault скачивает текущую версию файла хранилища с сервера.
func (c *httpClient) DownloadVault(ctx context.Context) (io.ReadCloser, *models.VaultVersion, error) {
	downloadURL, err := url.JoinPath(c.baseURL, "/api/vault/download")
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка формирования URL для скачивания: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка создания запроса на скачивание: %w", err)
	}
	if err = c.setAuthHeader(req); err != nil {
		return nil, nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// TODO: Обработка сетевых ошибок
		return nil, nil, fmt.Errorf("ошибка выполнения запроса на скачивание: %w", err)
	}
	// НЕ закрываем resp.Body здесь, вызывающая сторона должна это сделать

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close() // Закрываем тело в случае ошибки
		if resp.StatusCode == http.StatusNotFound {
			return nil, nil, errors.New("хранилище не найдено для скачивания")
		}
		if resp.StatusCode == http.StatusUnauthorized {
			// Возвращаем нашу специальную ошибку
			return nil, nil, ErrAuthorization
		}
		// TODO: Читать тело для деталей
		return nil, nil, fmt.Errorf("ошибка скачивания с сервера: статус %d", resp.StatusCode)
	}

	// Пытаемся извлечь метаданные из заголовков (если сервер их туда кладет - пока нет)
	// Здесь мы не можем получить метаданные версии из ответа Download так просто.
	// API сервера возвращает только файл. Клиенту нужно будет отдельно запросить метаданные,
	// если они нужны после скачивания. Либо сервер должен передавать их, например, в заголовках.
	// Пока возвращаем nil для метаданных.
	// Можно было бы распарсить Content-Length, если он есть.
	var meta *models.VaultVersion // Заглушка

	return resp.Body, meta, nil // Возвращаем тело ответа (io.ReadCloser)
}

// ListVersions получает список версий хранилища.
func (c *httpClient) ListVersions(ctx context.Context, limit, offset int) ([]models.VaultVersion, error) {
	listURL, err := url.JoinPath(c.baseURL, "/api/vault/versions")
	if err != nil {
		return nil, fmt.Errorf("ошибка формирования URL для списка версий: %w", err)
	}

	// Добавляем параметры пагинации
	query := url.Values{}
	if limit > 0 {
		query.Add("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		query.Add("offset", strconv.Itoa(offset))
	}
	if len(query) > 0 {
		listURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса на список версий: %w", err)
	}
	if err = c.setAuthHeader(req); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// TODO: Обработка сетевых ошибок
		return nil, fmt.Errorf("ошибка выполнения запроса на список версий: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			// Возвращаем нашу специальную ошибку
			return nil, ErrAuthorization
		}
		// TODO: Читать тело для деталей
		return nil, fmt.Errorf("ошибка получения списка версий: статус %d", resp.StatusCode)
	}

	var versions []models.VaultVersion
	if err = json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, fmt.Errorf("ошибка декодирования списка версий: %w", err)
	}

	return versions, nil
}

// RollbackToVersion отправляет запрос на откат к указанной версии.
func (c *httpClient) RollbackToVersion(ctx context.Context, versionID int64) error {
	rollbackURL, err := url.JoinPath(c.baseURL, "/api/vault/rollback")
	if err != nil {
		return fmt.Errorf("ошибка формирования URL для отката: %w", err)
	}

	// Создаем тело запроса
	requestBody := map[string]int64{"version_id": versionID} // Простой JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("ошибка кодирования данных для отката: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rollbackURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса на откат: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if err = c.setAuthHeader(req); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// TODO: Обработка сетевых ошибок
		return fmt.Errorf("ошибка выполнения запроса на откат: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent { // Ожидаем 204 No Content
		if resp.StatusCode == http.StatusUnauthorized {
			// Возвращаем нашу специальную ошибку
			return ErrAuthorization
		}
		if resp.StatusCode == http.StatusNotFound {
			return errors.New("указанная версия или хранилище не найдены для отката")
		}
		if resp.StatusCode == http.StatusBadRequest {
			// TODO: Читать тело ответа
			return errors.New("неверный запрос на откат (например, некорректный ID версии)")
		}
		// TODO: Читать тело для деталей
		return fmt.Errorf("ошибка отката на сервере: статус %d", resp.StatusCode)
	}

	return nil // Успешный откат
}

// SetAuthToken устанавливает токен аутентификации для клиента.
func (c *httpClient) SetAuthToken(token string) {
	c.authToken = token
	// Можно добавить логирование при необходимости
	// slog.Debug("Auth token set in API client")
}

// --- Конец методов API клиента ---
