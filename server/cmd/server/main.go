package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/maynagashev/gophkeeper/server/internal/handlers"
)

const (
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 10 * time.Second
	defaultIdleTimeout  = 30 * time.Second
	defaultServerPort   = "8080"
)

// Временная заглушка для AuthService.
type dummyAuthService struct{}

func (s *dummyAuthService) Register(username string, _ string) error {
	log.Printf("[DummyService] Попытка регистрации: %s", username)
	// Ничего не делаем, всегда успешно
	return nil
}

func (s *dummyAuthService) Login(username string, _ string) (string, error) {
	log.Printf("[DummyService] Попытка входа: %s", username)
	// Возвращаем фейковый токен
	return "dummy-jwt-from-service", nil
}

// --- Конец заглушки ---

// main - точка входа для сервера GophKeeper.
func main() {
	log.Println("Запуск сервера GophKeeper...")

	// Создаем новый роутер
	r := chi.NewRouter()

	// Используем стандартные middleware для логирования, восстановления после паник и т.д.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)    // Логирование запросов
	r.Use(middleware.Recoverer) // Восстановление после паник

	// Базовый маршрут для проверки работы сервера
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("pong\n"))
		if err != nil {
			log.Printf("Ошибка записи ответа: %v", err)
		}
	})

	// Создаем экземпляры зависимостей (пока с заглушками)
	authService := &dummyAuthService{}
	authHandler := handlers.NewAuthHandler(authService)

	// Маршруты API
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", authHandler.Register) // POST /api/user/register
		r.Post("/login", authHandler.Login)       // POST /api/user/login
	})

	// Задаем порт сервера (можно вынести в конфигурацию)
	port := defaultServerPort
	log.Printf("Сервер слушает на порту %s", port)

	// Создаем и настраиваем HTTP-сервер с таймаутами
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  defaultReadTimeout,  // Таймаут чтения запроса
		WriteTimeout: defaultWriteTimeout, // Таймаут записи ответа
		IdleTimeout:  defaultIdleTimeout,  // Таймаут простоя соединения
	}

	// Запускаем HTTP-сервер
	log.Printf("Запуск HTTP-сервера на %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
