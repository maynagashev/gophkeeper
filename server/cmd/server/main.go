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
	// Меняем порт по умолчанию на 443 для HTTPS.
	defaultServerPort = "443"
	envServerPort     = "SERVER_PORT"
	// Переменные окружения для TLS.
	envTLSCertFile = "TLS_CERT_FILE"
	envTLSKeyFile  = "TLS_KEY_FILE"

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

// Структура для хранения инициализированных зависимостей.
type dependencies struct {
	db           *sqlx.DB            // Используем тип *sqlx.DB
	fileStorage  storage.FileStorage // Используем интерфейс
	authHandler  *handlers.AuthHandler
	vaultHandler *handlers.VaultHandler
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

	// Инициализация зависимостей
	deps, err := setupDependencies()
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
	port := getEnv(envServerPort, defaultServerPort)
	certFile := getEnv(envTLSCertFile, "")
	keyFile := getEnv(envTLSKeyFile, "")

	// Проверяем наличие путей к сертификату и ключу
	if certFile == "" || keyFile == "" {
		// Возвращаем ошибку вместо Fatalf
		return errors.New("не указаны пути к файлам сертификата (TLS_CERT_FILE) и/или ключа (TLS_KEY_FILE)")
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	log.Printf("Запуск HTTPS-сервера на порту %s...", port)
	log.Printf("Используется сертификат: %s", certFile)
	log.Printf("Используется ключ: %s", keyFile)

	// Запускаем сервер с TLS
	if err = server.ListenAndServeTLS(certFile, keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		// Возвращаем ошибку вместо Fatalf
		return fmt.Errorf("ошибка запуска HTTPS-сервера: %w", err)
	}
	return nil // Успешное завершение run()
}

// setupDependencies инициализирует и возвращает все необходимые зависимости сервера.
func setupDependencies() (*dependencies, error) {
	deps := &dependencies{}
	var err error

	// 1. Подключение к БД
	dsn := getDSNFromEnv()
	// repository.NewPostgresDB возвращает *sqlx.DB
	deps.db, err = repository.NewPostgresDB(dsn)
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

// getDSNFromEnv формирует строку подключения к БД из переменных окружения.
func getDSNFromEnv() string {
	user := getEnv(envDBUser, defaultDBUser)
	password := getEnv(envDBPass, defaultDBPass)
	host := getEnv(envDBHost, defaultDBHost)
	port := getEnv(envDBPort, defaultDBPort)
	dbname := getEnv(envDBName, defaultDBName)

	// sslmode=disable - небезопасно для продакшена, но удобно для локальной разработки с Docker
	// TODO: Сделать sslmode конфигурируемым для продакшена (sslmode=require или verify-full)
	//nolint:nosprintfhostport // DSN - это URL, а не просто host:port
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Printf("Переменная окружения '%s' не установлена, используется значение по умолчанию: '%s'", key, fallback)
	return fallback
}
