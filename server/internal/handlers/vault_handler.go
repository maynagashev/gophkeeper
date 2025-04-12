package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/maynagashev/gophkeeper/server/internal/middleware"
	"github.com/maynagashev/gophkeeper/server/internal/services"
)

// VaultHandler обрабатывает HTTP-запросы, связанные с хранилищем.
type VaultHandler struct {
	vaultService services.VaultService
}

// NewVaultHandler создает новый экземпляр VaultHandler.
func NewVaultHandler(vs services.VaultService) *VaultHandler {
	return &VaultHandler{vaultService: vs}
}

// GetMetadata обрабатывает GET запрос на получение метаданных ТЕКУЩЕЙ версии хранилища.
func (h *VaultHandler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		log.Printf("[VaultHandler:GetMetadata] Не удалось получить userID из контекста")
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	log.Printf("[VaultHandler:GetMetadata] Запрос метаданных от пользователя %d", userID)

	// Вызываем сервис для получения метаданных ТЕКУЩЕЙ версии
	currentVersion, err := h.vaultService.GetVaultMetadata(userID)
	if err != nil {
		if errors.Is(err, services.ErrVaultNotFound) {
			log.Printf("[VaultHandler:GetMetadata] Метаданные не найдены для пользователя %d", userID)
			http.Error(w, "Хранилище не найдено", http.StatusNotFound)
		} else {
			log.Printf("[VaultHandler:GetMetadata] Внутренняя ошибка "+
				"при получении метаданных для пользователя %d: %v", userID, err)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	// Отправляем метаданные текущей версии в JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(currentVersion); err != nil {
		log.Printf("[VaultHandler:GetMetadata] Ошибка кодирования ответа с метаданными: %v", err)
	}
}

// Upload обрабатывает POST запрос на загрузку файла хранилища.
func (h *VaultHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		log.Printf("[VaultHandler:Upload] Не удалось получить userID из контекста")
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	log.Printf("[VaultHandler:Upload] Запрос на загрузку файла от пользователя %d", userID)

	// Получаем размер файла из заголовка Content-Length
	sizeStr := r.Header.Get("Content-Length")
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil || size <= 0 {
		log.Printf("[VaultHandler:Upload] Неверный или отсутствующий заголовок Content-Length: %s", sizeStr)
		http.Error(w, "Неверный или отсутствующий заголовок Content-Length", http.StatusBadRequest)
		return
	}

	// Получаем Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		// По умолчанию считаем бинарным потоком
		contentType = "application/octet-stream"
	}

	// Вызываем сервис для загрузки файла
	err = h.vaultService.UploadVault(userID, r.Body, size, contentType)
	if err != nil {
		// Обработка ошибок сервиса (пока только внутренние)
		log.Printf("[VaultHandler:Upload] Ошибка сервиса при загрузке файла для пользователя %d: %v", userID, err)
		http.Error(w, "Внутренняя ошибка сервера при загрузке файла", http.StatusInternalServerError)
		return
	}

	// Успешный ответ
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Файл успешно загружен\n"))
	log.Printf("[VaultHandler:Upload] Файл для пользователя %d успешно загружен", userID)
}

// Download обрабатывает GET запрос на скачивание ТЕКУЩЕЙ версии файла хранилища.
func (h *VaultHandler) Download(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		log.Printf("[VaultHandler:Download] Не удалось получить userID из контекста")
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	log.Printf("[VaultHandler:Download] Запрос на скачивание файла от пользователя %d", userID)

	// Вызываем сервис для скачивания ТЕКУЩЕЙ версии
	fileReader, versionMeta, err := h.vaultService.DownloadVault(userID)
	if err != nil {
		if errors.Is(err, services.ErrVaultNotFound) {
			log.Printf("[VaultHandler:Download] Хранилище/версия не найдено для пользователя %d", userID)
			http.Error(w, "Хранилище не найдено", http.StatusNotFound)
		} else {
			log.Printf("[VaultHandler:Download] Внутренняя ошибка при скачивании "+
				"файла для пользователя %d: %v", userID, err)
			http.Error(w, "Внутренняя ошибка сервера при скачивании файла", http.StatusInternalServerError)
		}
		return
	}
	defer func() {
		if closeErr := fileReader.Close(); closeErr != nil {
			log.Printf("[VaultHandler:Download] Ошибка закрытия fileReader: %v", closeErr)
		}
	}()

	// Устанавливаем заголовки для скачивания файла
	w.Header().Set("Content-Disposition", `attachment; filename="gophkeeper_vault.kdbx"`)
	contentType := "application/octet-stream"
	w.Header().Set("Content-Type", contentType)
	if versionMeta.SizeBytes != nil {
		w.Header().Set("Content-Length", strconv.FormatInt(*versionMeta.SizeBytes, 10))
	}

	// Копируем данные из fileReader в ResponseWriter
	_, err = io.Copy(w, fileReader)
	if err != nil {
		log.Printf("[VaultHandler:Download] Ошибка копирования данных файла в ответ для пользователя %d: %v", userID, err)
		return
	}

	log.Printf("[VaultHandler:Download] Файл для пользователя %d (версия %d) успешно отправлен", userID, versionMeta.ID)
}

// ListVersions обрабатывает GET запрос на получение списка версий хранилища.
func (h *VaultHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		log.Printf("[VaultHandler:ListVersions] Не удалось получить userID из контекста")
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Получаем параметры пагинации (простой вариант, без валидации)
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 || limit > 100 { // Ограничиваем максимальный лимит
		limit = 20 // Значение по умолчанию
	}
	if offset < 0 {
		offset = 0
	}

	log.Printf("[VaultHandler:ListVersions] Запрос списка версий от пользователя %d "+
		"(limit=%d, offset=%d)", userID, limit, offset)

	versions, err := h.vaultService.ListVersions(userID, limit, offset)
	if err != nil {
		log.Printf("[VaultHandler:ListVersions] Внутренняя ошибка при получении "+
			"списка версий для пользователя %d: %v", userID, err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Отправляем список версий в JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(versions); err != nil {
		log.Printf("[VaultHandler:ListVersions] Ошибка кодирования ответа со списком версий: %v", err)
	}
}

// RollbackRequest представляет тело запроса на откат к версии.
type RollbackRequest struct {
	VersionID int64 `json:"version_id"`
}

// Rollback обрабатывает POST запрос на откат к указанной версии.
func (h *VaultHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		log.Printf("[VaultHandler:Rollback] Не удалось получить userID из контекста")
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Декодируем тело запроса
	var req RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[VaultHandler:Rollback] Ошибка декодирования запроса на откат: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if req.VersionID <= 0 {
		http.Error(w, "Неверный ID версии", http.StatusBadRequest)
		return
	}

	log.Printf("[VaultHandler:Rollback] Запрос на откат к версии %d от пользователя %d", req.VersionID, userID)

	err := h.vaultService.RollbackToVersion(userID, req.VersionID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrVaultNotFound), errors.Is(err, services.ErrVersionNotFound):
			log.Printf("[VaultHandler:Rollback] Хранилище/версия %d не найдена для пользователя %d", req.VersionID, userID)
			http.Error(w, "Указанное хранилище или версия не найдены", http.StatusNotFound)
		case errors.Is(err, services.ErrForbidden):
			log.Printf("[VaultHandler:Rollback] Попытка отката к чужой версии %d пользователем %d", req.VersionID, userID)
			http.Error(w, "Доступ запрещен", http.StatusForbidden)
		default:
			log.Printf("[VaultHandler:Rollback] Внутренняя ошибка при откате "+
				"к версии %d для пользователя %d: %v", req.VersionID, userID, err)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content - успешный откат без тела ответа
	log.Printf("[VaultHandler:Rollback] Успешный откат к версии %d для пользователя %d", req.VersionID, userID)
}
