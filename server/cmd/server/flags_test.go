package main

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Вспомогательная функция для сброса флагов между тестами.
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestParseFlags(t *testing.T) {
	// Сохраняем оригинальные аргументы командной строки
	originalArgs := os.Args

	// Сохраняем и очищаем переменные окружения
	originalEnv := map[string]string{
		envServerPort:  os.Getenv(envServerPort),
		envTLSCertFile: os.Getenv(envTLSCertFile),
		envTLSKeyFile:  os.Getenv(envTLSKeyFile),
		envDatabaseDSN: os.Getenv(envDatabaseDSN),
	}
	defer func() {
		for k, v := range originalEnv {
			os.Setenv(k, v)
		}
	}()
	os.Unsetenv(envServerPort)
	os.Unsetenv(envTLSCertFile)
	os.Unsetenv(envTLSKeyFile)
	os.Unsetenv(envDatabaseDSN)

	t.Run("Все параметры из флагов", func(t *testing.T) {
		resetFlags()
		// Восстанавливаем os.Args после теста
		defer func() { os.Args = originalArgs }()

		os.Args = []string{"cmd", "-port=8080", "-cert-file=cert.pem", "-key-file=key.pem", "-database-dsn=postgres://..."}
		cfg, err := parseFlags()
		require.NoError(t, err)
		assert.Equal(t, "8080", cfg.Port)
		assert.Equal(t, "cert.pem", cfg.CertFile)
		assert.Equal(t, "key.pem", cfg.KeyFile)
		assert.Equal(t, "postgres://...", cfg.DatabaseDSN)
	})

	t.Run("Все параметры из переменных окружения", func(t *testing.T) {
		resetFlags()
		defer func() { os.Args = originalArgs }() // Восстанавливаем os.Args
		os.Args = []string{"cmd"}                 // Сбрасываем аргументы командной строки

		os.Setenv(envServerPort, "9090")
		os.Setenv(envTLSCertFile, "env_cert.pem")
		os.Setenv(envTLSKeyFile, "env_key.pem")
		os.Setenv(envDatabaseDSN, "env_postgres://...")
		defer func() { // Очищаем переменные после теста
			os.Unsetenv(envServerPort)
			os.Unsetenv(envTLSCertFile)
			os.Unsetenv(envTLSKeyFile)
			os.Unsetenv(envDatabaseDSN)
		}()

		cfg, err := parseFlags()
		require.NoError(t, err)
		assert.Equal(t, "9090", cfg.Port)
		assert.Equal(t, "env_cert.pem", cfg.CertFile)
		assert.Equal(t, "env_key.pem", cfg.KeyFile)
		assert.Equal(t, "env_postgres://...", cfg.DatabaseDSN)
	})

	t.Run("Порт по умолчанию", func(t *testing.T) {
		resetFlags()
		defer func() { os.Args = originalArgs }()
		os.Args = []string{"cmd", "-cert-file=cert.pem", "-key-file=key.pem", "-database-dsn=postgres://..."}

		cfg, err := parseFlags()
		require.NoError(t, err)
		assert.Equal(t, defaultServerPort, cfg.Port) // Проверяем порт по умолчанию
	})

	t.Run("Отсутствует обязательный параметр cert-file", func(t *testing.T) {
		resetFlags()
		defer func() { os.Args = originalArgs }()
		os.Args = []string{"cmd", "-key-file=key.pem", "-database-dsn=postgres://..."}

		_, err := parseFlags()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "не указан путь к файлу сертификата")
	})

	t.Run("Отсутствует обязательный параметр key-file", func(t *testing.T) {
		resetFlags()
		defer func() { os.Args = originalArgs }()
		os.Args = []string{"cmd", "-cert-file=cert.pem", "-database-dsn=postgres://..."}

		_, err := parseFlags()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "не указан путь к файлу ключа")
	})

	t.Run("Отсутствует обязательный параметр database-dsn", func(t *testing.T) {
		resetFlags()
		defer func() { os.Args = originalArgs }()
		os.Args = []string{"cmd", "-cert-file=cert.pem", "-key-file=key.pem"}

		_, err := parseFlags()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "не указана строка подключения к БД")
	})

	t.Run("Флаги переопределяют переменные окружения", func(t *testing.T) {
		resetFlags()
		defer func() { os.Args = originalArgs }() // Восстанавливаем os.Args

		os.Setenv(envServerPort, "9090")
		os.Setenv(envTLSCertFile, "env_cert.pem")
		os.Setenv(envTLSKeyFile, "env_key.pem")
		os.Setenv(envDatabaseDSN, "env_postgres://...")
		defer func() { // Очищаем переменные после теста
			os.Unsetenv(envServerPort)
			os.Unsetenv(envTLSCertFile)
			os.Unsetenv(envTLSKeyFile)
			os.Unsetenv(envDatabaseDSN)
		}()

		os.Args = []string{
			"cmd",
			"-port=8080",
			"-cert-file=flag_cert.pem",
			"-key-file=flag_key.pem",
			"-database-dsn=flag_postgres://...",
		}
		cfg, err := parseFlags()
		require.NoError(t, err)
		assert.Equal(t, "8080", cfg.Port)
		assert.Equal(t, "flag_cert.pem", cfg.CertFile)
		assert.Equal(t, "flag_key.pem", cfg.KeyFile)
		assert.Equal(t, "flag_postgres://...", cfg.DatabaseDSN)
	})
}
