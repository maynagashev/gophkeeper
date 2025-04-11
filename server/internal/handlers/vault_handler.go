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

// GetMetadata обрабатывает GET запрос на получение метаданных хранилища.
func (h *VaultHandler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста, установленного middleware
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		log.Printf("[VaultHandler] Не удалось получить userID из контекста")
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	log.Printf("[VaultHandler] Запрос метаданных от пользователя %d", userID)

	// Вызываем сервис для получения метаданных
	vault, err := h.vaultService.GetVaultMetadata(userID)
	if err != nil {
		if errors.Is(err, services.ErrVaultNotFound) {
			log.Printf("[VaultHandler] Метаданные не найдены для пользователя %d", userID)
			http.Error(w, "Хранилище не найдено", http.StatusNotFound) // 404 Not Found
		} else {
			log.Printf("[VaultHandler] Внутренняя ошибка при получении метаданных для пользователя %d: %v", userID, err)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	// Отправляем метаданные в JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(vault); err != nil {
		log.Printf("[VaultHandler] Ошибка кодирования ответа с метаданными: %v", err)
		// Статус уже отправлен, просто логируем
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

// Download обрабатывает GET запрос на скачивание файла хранилища.
func (h *VaultHandler) Download(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		log.Printf("[VaultHandler:Download] Не удалось получить userID из контекста")
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	log.Printf("[VaultHandler:Download] Запрос на скачивание файла от пользователя %d", userID)

	// Вызываем сервис для скачивания
	fileReader, vaultMeta, err := h.vaultService.DownloadVault(userID)
	if err != nil {
		if errors.Is(err, services.ErrVaultNotFound) {
			log.Printf("[VaultHandler:Download] Хранилище не найдено для пользователя %d", userID)
			http.Error(w, "Хранилище не найдено", http.StatusNotFound)
		} else {
			log.Printf("[VaultHandler:Download] Внутренняя ошибка при скачивании файла для пользователя %d: %v", userID, err)
			http.Error(w, "Внутренняя ошибка сервера при скачивании файла", http.StatusInternalServerError)
		}
		return
	}
	// Важно закрыть fileReader после завершения запроса
	defer func() {
		// Используем отдельную переменную или игнорируем ошибку, если она не критична
		if closeErr := fileReader.Close(); closeErr != nil {
			log.Printf("[VaultHandler:Download] Ошибка закрытия fileReader: %v", closeErr)
		}
	}()

	// Устанавливаем заголовки для скачивания файла
	w.Header().Set("Content-Disposition", `attachment; filename="gophkeeper_vault.kdbx"`) // Имя файла для скачивания
	contentType := "application/octet-stream"                                             // По умолчанию
	// Можно попытаться установить Content-Type из метаданных, если они есть и надежны
	w.Header().Set("Content-Type", contentType)
	if vaultMeta.SizeBytes != nil {
		w.Header().Set("Content-Length", strconv.FormatInt(*vaultMeta.SizeBytes, 10))
	}

	// Копируем данные из fileReader в ResponseWriter
	_, err = io.Copy(w, fileReader)
	if err != nil {
		log.Printf("[VaultHandler:Download] Ошибка копирования данных файла в ответ для пользователя %d: %v", userID, err)
		// Статус уже, скорее всего, отправлен (200 OK по умолчанию), сложно что-то сделать
		// Можно попробовать отправить http.StatusInternalServerError, но клиент может его не увидеть
		return
	}

	log.Printf("[VaultHandler:Download] Файл для пользователя %d успешно отправлен", userID)
}
