package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/maynagashev/gophkeeper/server/internal/models" // Импортируем наши модели
)

// AuthService определяет интерфейс для сервиса аутентификации.
// Это позволит нам легко подменять реализацию (например, для тестов).
type AuthService interface {
	Register(username, password string) error
	Login(username, password string) (string, error) // Возвращает JWT токен или ошибку
}

// AuthHandler обрабатывает HTTP-запросы, связанные с аутентификацией.
type AuthHandler struct {
	service AuthService // Зависимость от интерфейса, а не конкретной реализации
}

// NewAuthHandler создает новый экземпляр AuthHandler.
func NewAuthHandler(s AuthService) *AuthHandler {
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

	// TODO: Вызвать h.service.Register(req.Username, req.Password)

	// Пока просто возвращаем успешный статус
	w.WriteHeader(http.StatusCreated) // 201 Created
	_, _ = w.Write([]byte("Пользователь успешно зарегистрирован (заглушка)\n"))
	log.Printf("[AuthHandler] Успешная регистрация (заглушка) для: %s", req.Username)
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

	// TODO: Вызвать h.service.Login(req.Username, req.Password)
	// TODO: Получить токен и отправить его в ответе

	// Пока просто возвращаем заглушку токена
	resp := models.LoginResponse{
		Token: "fake-jwt-token-placeholder",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200 OK
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[AuthHandler] Ошибка кодирования ответа входа: %v", err)
		// Клиент уже получил статус 200, сложно что-то изменить
		return
	}
	log.Printf("[AuthHandler] Успешный вход (заглушка) для: %s", req.Username)
}
