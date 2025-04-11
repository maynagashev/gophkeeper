package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq" // Драйвер PostgreSQL
	"github.com/maynagashev/gophkeeper/server/internal/handlers"
	appmiddleware "github.com/maynagashev/gophkeeper/server/internal/middleware"
	"github.com/maynagashev/gophkeeper/server/internal/repository"
	"github.com/maynagashev/gophkeeper/server/internal/services"
)

const (
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 10 * time.Second
	defaultIdleTimeout  = 30 * time.Second
	defaultServerPort   = "8080"

	// Переменные окружения для БД (значения по умолчанию из Makefile/docker-compose).
	envDBUser     = "POSTGRES_USER"
	envDBPass     = "POSTGRES_PASSWORD" //nolint:gosec // Ложное срабатывание, это имя переменной окружения
	envDBName     = "POSTGRES_DB"
	envDBHost     = "POSTGRES_HOST"
	envDBPort     = "POSTGRES_PORT"
	defaultDBUser = "gophkeeper"
	defaultDBPass = "secret"
	defaultDBName = "gophkeeper"
	defaultDBHost = "localhost"
	defaultDBPort = "5433"
)

// main - точка входа для сервера GophKeeper.
func main() {
	log.Println("Запуск сервера GophKeeper...")

	// --- Инициализация зависимостей --- //

	// 1. Подключение к БД
	dsn := getDSNFromEnv()
	db, err := repository.NewPostgresDB(dsn)
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer func() {
		if err = db.Close(); err != nil {
			log.Printf("Ошибка закрытия соединения с БД: %v", err)
		}
	}()
	log.Println("Соединение с БД успешно установлено.")

	// 2. Создание репозиториев
	userRepo := repository.NewPostgresUserRepository(db)
	vaultRepo := repository.NewPostgresVaultRepository(db) // Создаем репозиторий хранилищ

	// 3. Создание сервисов
	authService := services.NewAuthService(userRepo)
	vaultService := services.NewVaultService(vaultRepo) // Создаем сервис хранилищ

	// 4. Создание обработчиков
	authHandler := handlers.NewAuthHandler(authService)
	vaultHandler := handlers.NewVaultHandler(vaultService) // Создаем обработчик хранилищ

	// --- Настройка роутера --- //
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// --- Маршруты --- //
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong\n"))
	})

	// Публичные маршруты (регистрация, вход)
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	// Приватные маршруты (требуют аутентификации)
	r.Group(func(r chi.Router) {
		// Применяем middleware аутентификации ко всей группе
		r.Use(appmiddleware.Authenticator)

		// Маршруты для работы с хранилищем
		r.Route("/api/vault", func(r chi.Router) {
			// Используем реальный обработчик
			r.Get("/", vaultHandler.GetMetadata)
			// TODO: Добавить POST /upload, GET /download, GET /versions, POST /rollback
		})
	})

	// --- Запуск сервера --- //
	port := getEnv("SERVER_PORT", defaultServerPort)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	log.Printf("Запуск HTTP-сервера на порту %s", port)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		//nolint:gocritic // Завершение через Fatalf приемлемо здесь
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

// getDSNFromEnv формирует строку подключения к БД из переменных окружения.
func getDSNFromEnv() string {
	user := getEnv(envDBUser, defaultDBUser)
	password := getEnv(envDBPass, defaultDBPass)
	host := getEnv(envDBHost, defaultDBHost)
	port := getEnv(envDBPort, defaultDBPort)
	dbname := getEnv(envDBName, defaultDBName)

	// sslmode=disable - небезопасно для продакшена, но удобно для локальной разработки с Docker
	//nolint:nosprintfhostport // DSN - это URL, а не просто host:port
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
