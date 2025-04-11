package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

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
