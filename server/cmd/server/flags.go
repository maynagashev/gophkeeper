package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

const (
	// Порт по умолчанию для HTTPS (непривилегированный).
	defaultServerPort = "8443"

	// Переменные окружения.
	envServerPort  = "SERVER_PORT"
	envTLSCertFile = "TLS_CERT_FILE"
	envTLSKeyFile  = "TLS_KEY_FILE"
	envDatabaseDSN = "DATABASE_DSN"
)

// config хранит конфигурацию сервера.
type config struct {
	Port        string
	CertFile    string
	KeyFile     string
	DatabaseDSN string
}

// parseFlags разбирает флаги и переменные окружения, возвращает config или ошибку.
func parseFlags() (*config, error) {
	cfg := &config{}

	// Определяем флаги
	flag.StringVar(&cfg.Port, "port", "",
		fmt.Sprintf("Порт для запуска HTTPS-сервера (env: %s, default: %s)", envServerPort, defaultServerPort))
	flag.StringVar(&cfg.CertFile, "cert-file", "",
		fmt.Sprintf("Путь к файлу TLS-сертификата (env: %s)", envTLSCertFile))
	flag.StringVar(&cfg.KeyFile, "key-file", "",
		fmt.Sprintf("Путь к файлу TLS-ключа (env: %s)", envTLSKeyFile))
	flag.StringVar(&cfg.DatabaseDSN, "database-dsn", "",
		fmt.Sprintf("Строка подключения к базе данных (env: %s)", envDatabaseDSN))

	// Парсим флаги
	flag.Parse()

	// Применяем переменные окружения, если флаги не заданы
	if cfg.Port == "" {
		if value, ok := os.LookupEnv(envServerPort); ok {
			cfg.Port = value
		} else {
			cfg.Port = defaultServerPort
		}
	}
	if cfg.CertFile == "" {
		if value, ok := os.LookupEnv(envTLSCertFile); ok {
			cfg.CertFile = value
		}
	}
	if cfg.KeyFile == "" {
		if value, ok := os.LookupEnv(envTLSKeyFile); ok {
			cfg.KeyFile = value
		}
	}
	if cfg.DatabaseDSN == "" {
		if value, ok := os.LookupEnv(envDatabaseDSN); ok {
			cfg.DatabaseDSN = value
		}
	}

	// Проверяем обязательные параметры
	if cfg.CertFile == "" {
		return nil, errors.New("не указан путь к файлу сертификата (--cert-file или " + envTLSCertFile + ")")
	}
	if cfg.KeyFile == "" {
		return nil, errors.New("не указан путь к файлу ключа (--key-file или " + envTLSKeyFile + ")")
	}
	if cfg.DatabaseDSN == "" {
		return nil, errors.New("не указана строка подключения к БД (--database-dsn или " + envDatabaseDSN + ")")
	}

	return cfg, nil
}
