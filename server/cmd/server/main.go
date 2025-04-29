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
	"github.com/jmoiron/sqlx" // Добавляем импорт sqlx
	_ "github.com/lib/pq"     // Драйвер PostgreSQL
	"github.com/maynagashev/gophkeeper/server/internal/handlers"
	appmiddleware "github.com/maynagashev/gophkeeper/server/internal/middleware"
	"github.com/maynagashev/gophkeeper/server/internal/repository"
	"github.com/maynagashev/gophkeeper/server/internal/services"
	"github.com/maynagashev/gophkeeper/server/internal/storage" // Добавляем импорт storage
)

const (
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 10 * time.Second
	defaultIdleTimeout  = 30 * time.Second

	// Переменные окружения для MinIO (значения по умолчанию из docker-compose).
	envMinioEndpoint     = "MINIO_ENDPOINT"
	envMinioUser         = "MINIO_USER"
	envMinioPassword     = "MINIO_PASSWORD"
	envMinioBucket       = "MINIO_BUCKET"
	defaultMinioEndpoint = "localhost:9000"
	defaultMinioUser     = "minioadmin"
	defaultMinioPassword = "minioadmin"
	defaultMinioBucket   = "gophkeeper-vaults"
	minioUseSSL          = false // Для локальной разработки
)

// Переменная для функции создания соединения с БД, для возможности мокирования в тестах.
var newPostgresDB = repository.NewPostgresDB //nolint:gochecknoglobals // Используется для мокирования в тестах

// Структура для хранения инициализированных зависимостей.
type dependencies struct {
	db           *sqlx.DB            // Используем тип *sqlx.DB
	fileStorage  storage.FileStorage // Используем интерфейс
	authHandler  *handlers.AuthHandler
	vaultHandler *handlers.VaultHandler
}

// Функция для запуска HTTP сервера (для удобства мокирования в тестах).
var startHTTPServer = //nolint:gochecknoglobals // Используется для мокирования в тестах
func(cfg *config, handler http.Handler) error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      handler,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	log.Printf("Запуск HTTPS-сервера на порту %s...", cfg.Port)
	log.Printf("Используется сертификат: %s", cfg.CertFile)
	log.Printf("Используется ключ: %s", cfg.KeyFile)

	// Запускаем сервер с TLS
	if err := server.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("ошибка запуска HTTPS-сервера: %w", err)
	}
	return nil
}

// main - точка входа. Вызывает run и обрабатывает ошибку.
func main() {
	if err := run(); err != nil {
		log.Printf("Ошибка выполнения сервера: %v", err) // Используем Printf
		os.Exit(1)                                       // Выход с кодом ошибки
	}
}

// run содержит основную логику запуска сервера и возвращает ошибку.
func run() error {
	log.Println("Запуск сервера GophKeeper...")

	// Парсинг флагов командной строки
	cfg, err := parseFlags()
	if err != nil {
		// Используем log.Fatalf, так как ошибка фатальна для запуска сервера.
		// В реальном тесте это приведет к завершению, нужна стратегия обхода.
		// Пока оставляем так, но тест должен будет мокировать parseFlags.
		log.Fatalf("Ошибка конфигурации сервера: %v", err)
	}

	// Инициализация зависимостей
	deps, err := setupDependencies(cfg)
	if err != nil {
		return fmt.Errorf("ошибка инициализации зависимостей: %w", err)
	}
	// Отложенное закрытие соединения с БД
	// Это гарантированно выполнится при выходе из run()
	defer func() {
		if deps.db != nil {
			if closeErr := deps.db.Close(); closeErr != nil {
				log.Printf("Ошибка закрытия соединения с БД: %v", closeErr)
			}
		}
	}()

	// Настройка роутера
	r := setupRouter(deps.authHandler, deps.vaultHandler)

	// --- Запуск сервера --- //
	// Используем переменную startHTTPServer вместо прямого кода запуска
	if err = startHTTPServer(cfg, r); err != nil {
		return err // Возвращаем ошибку от startHTTPServer
	}
	return nil // Успешное завершение run()
}

// setupDependencies инициализирует и возвращает все необходимые зависимости сервера.
func setupDependencies(cfg *config) (*dependencies, error) {
	deps := &dependencies{}
	var err error

	// 1. Подключение к БД
	deps.db, err = newPostgresDB(cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации БД: %w", err)
	}
	log.Println("Соединение с БД успешно установлено.")

	// 2. Инициализация клиента MinIO
	minioCfg := storage.MinioConfig{
		Endpoint:        getEnv(envMinioEndpoint, defaultMinioEndpoint),
		AccessKeyID:     getEnv(envMinioUser, defaultMinioUser),
		SecretAccessKey: getEnv(envMinioPassword, defaultMinioPassword),
		UseSSL:          minioUseSSL,
		BucketName:      getEnv(envMinioBucket, defaultMinioBucket),
	}
	// storage.NewMinioClient возвращает *storage.MinioClient, который реализует storage.FileStorage
	deps.fileStorage, err = storage.NewMinioClient(minioCfg)
	if err != nil {
		// Закрываем соединение с БД перед выходом
		if dbCloseErr := deps.db.Close(); dbCloseErr != nil {
			log.Printf("Ошибка закрытия соединения с БД при ошибке MinIO: %v", dbCloseErr)
		}
		return nil, fmt.Errorf("ошибка инициализации клиента MinIO: %w", err)
	}

	// 3. Создание репозиториев
	// Передаем *sqlx.DB в конструкторы репозиториев
	userRepo := repository.NewPostgresUserRepository(deps.db)
	vaultRepo := repository.NewPostgresVaultRepository(deps.db)
	vaultVersionRepo := repository.NewPostgresVaultVersionRepository(deps.db)

	// 4. Создание сервисов
	authService := services.NewAuthService(userRepo)
	// Передаем *sql.DB (из поля DB типа *sqlx.DB) в VaultService
	vaultService := services.NewVaultService(deps.db.DB, vaultRepo, vaultVersionRepo, deps.fileStorage)

	// 5. Создание обработчиков
	deps.authHandler = handlers.NewAuthHandler(authService)
	deps.vaultHandler = handlers.NewVaultHandler(vaultService)

	return deps, nil
}

// setupRouter настраивает и возвращает роутер chi.
func setupRouter(authHandler *handlers.AuthHandler, vaultHandler *handlers.VaultHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// --- Маршруты --- //
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong\n"))
	})

	// Определяем базовый маршрут /api
	r.Route("/api", func(r chi.Router) {
		// Публичные маршруты (регистрация, вход)
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)

		// Приватные маршруты (требуют аутентификации)
		r.Group(func(r chi.Router) {
			// Применяем middleware аутентификации ко всей группе
			r.Use(appmiddleware.Authenticator)

			// Маршруты для работы с хранилищем
			r.Route("/vault", func(r chi.Router) {
				r.Get("/", vaultHandler.GetMetadata)
				r.Post("/upload", vaultHandler.Upload)
				r.Get("/download", vaultHandler.Download)
				r.Get("/versions", vaultHandler.ListVersions)
				r.Post("/rollback", vaultHandler.Rollback)
			})
			// Маршрут для удаления аккаунта (если он есть в AuthHandler)
			// r.Delete("/account", authHandler.DeleteAccount)
		})
	})
	return r
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Printf("Переменная окружения '%s' не установлена, используется значение по умолчанию: '%s'", key, fallback)
	return fallback
}
