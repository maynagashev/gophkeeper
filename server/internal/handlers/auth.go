package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/maynagashev/gophkeeper/models"                   // Импортируем наши модели
	"github.com/maynagashev/gophkeeper/server/internal/services" // Импортируем пакет сервисов
)

// AuthService определяет интерфейс для сервиса аутентификации.
// Это позволит нам легко подменять реализацию (например, для тестов).
// type AuthService interface { // Удаляем этот интерфейс отсюда
// 	Register(username, password string) error
// 	Login(username, password string) (string, error) // Возвращает JWT токен или ошибку
// }

// AuthHandler обрабатывает HTTP-запросы, связанные с аутентификацией.
type AuthHandler struct {
	service services.AuthService // Используем интерфейс из пакета services
}

// NewAuthHandler создает новый экземпляр AuthHandler.
func NewAuthHandler(s services.AuthService) *AuthHandler { // Принимаем интерфейс из пакета services
	return &AuthHandler{service: s}
}

// Register обрабатывает запрос на регистрацию нового пользователя.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	// Декодируем JSON из тела запроса
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[AuthHandler] Ошибка декодирования запроса регистрации: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Валидация входных данных (простая)
	if req.Username == "" || req.Password == "" {
		log.Printf("[AuthHandler] Пустое имя пользователя или пароль при регистрации")
		http.Error(w, "Имя пользователя и пароль не могут быть пустыми", http.StatusBadRequest)
		return
	}

	log.Printf("[AuthHandler] Попытка регистрации пользователя: %s", req.Username)

	// Вызываем сервис
	err := h.service.Register(req.Username, req.Password)
	if err != nil {
		// Обрабатываем ошибки от сервиса
		if errors.Is(err, services.ErrUsernameTaken) {
			log.Printf("[AuthHandler] Ошибка регистрации (имя занято): %s", req.Username)
			http.Error(w, err.Error(), http.StatusConflict) // 409 Conflict
		} else {
			// Другие ошибки считаем внутренними
			log.Printf("[AuthHandler] Внутренняя ошибка при регистрации '%s': %v", req.Username, err)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	// Возвращаем успешный статус
	w.WriteHeader(http.StatusCreated)                                // 201 Created
	_, _ = w.Write([]byte("Пользователь успешно зарегистрирован\n")) // Убираем "(заглушка)"
	log.Printf("[AuthHandler] Успешная регистрация для: %s", req.Username)
}

// Login обрабатывает запрос на вход пользователя.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	// Декодируем JSON из тела запроса
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[AuthHandler] Ошибка декодирования запроса входа: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Валидация входных данных (простая)
	if req.Username == "" || req.Password == "" {
		log.Printf("[AuthHandler] Пустое имя пользователя или пароль при входе")
		http.Error(w, "Имя пользователя и пароль не могут быть пустыми", http.StatusBadRequest)
		return
	}

	log.Printf("[AuthHandler] Попытка входа пользователя: %s", req.Username)

	// Вызываем сервис
	token, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		// Обрабатываем ошибки от сервиса
		if errors.Is(err, services.ErrInvalidCredentials) {
			log.Printf("[AuthHandler] Ошибка входа (неверные данные): %s", req.Username)
			http.Error(w, err.Error(), http.StatusUnauthorized) // 401 Unauthorized
		} else {
			// Другие ошибки считаем внутренними
			log.Printf("[AuthHandler] Внутренняя ошибка при входе '%s': %v", req.Username, err)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	// Возвращаем токен
	resp := models.LoginResponse{
		Token: token, // Используем реальный токен от сервиса
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200 OK
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[AuthHandler] Ошибка кодирования ответа входа: %v", err)
		return
	}
	log.Printf("[AuthHandler] Успешный вход для: %s", req.Username)
}
