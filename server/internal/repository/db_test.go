package repository_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/maynagashev/gophkeeper/server/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Получаем DSN из переменной окружения DATABASE_DSN (приоритет).
// Если она не установлена, строим DSN для локального docker-compose,
// используя POSTGRES_PORT (default: 5433) и стандартные креды/БД.
func getTestDSN() string {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Println("Переменная окружения DATABASE_DSN не установлена.")
		// Строим запасной DSN для локального docker-compose
		pgPort := os.Getenv("POSTGRES_PORT")
		if pgPort == "" {
			pgPort = "5433" // Порт по умолчанию из docker-compose.yml
			log.Printf("Переменная окружения POSTGRES_PORT не установлена, используем порт по умолчанию: %s", pgPort)
		} else {
			log.Printf("Используем порт из переменной окружения POSTGRES_PORT: %s", pgPort)
		}
		// Используем стандартные имя пользователя, пароль и БД из docker-compose.yml
		dsn = fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable",
			"gophkeeper", // Пользователь по умолчанию
			"secret",     // Пароль по умолчанию
			pgPort,       // Определенный выше порт
			"gophkeeper", // БД по умолчанию
		)
		log.Printf("Используется запасной DSN: %s", dsn)
	} else {
		log.Printf("Используется DSN из переменной окружения DATABASE_DSN: %s", dsn)
	}
	return dsn
}

func TestNewPostgresDB(t *testing.T) {
	t.Run("Успешное подключение", func(t *testing.T) {
		// Этот тест требует запущенной PostgreSQL базы данных
		dsn := getTestDSN()
		if dsn == "" {
			t.Skip("Пропуск теста: переменная окружения DATABASE_DSN не установлена")
		}

		db, err := repository.NewPostgresDB(dsn)

		// Проверяем, что ошибки нет
		require.NoError(t, err)
		// Проверяем, что объект БД не nil
		require.NotNil(t, db)

		// Проверяем, что соединение действительно работает (дополнительный пинг)
		err = db.Ping()
		require.NoError(t, err, "Не удалось пинговать БД после создания")

		// Важно закрыть соединение после теста
		err = db.Close()
		require.NoError(t, err, "Ошибка при закрытии соединения с БД")
	})

	t.Run("Ошибка: Невалидный DSN", func(t *testing.T) {
		invalidDSN := "это точно не dsn"

		db, err := repository.NewPostgresDB(invalidDSN)

		// Проверяем, что есть ошибка
		require.Error(t, err)
		// Проверяем, что объект БД nil
		assert.Nil(t, db)
		// Можно также проверить текст ошибки (но он может зависеть от драйвера)
		assert.Contains(t, err.Error(), "ошибка подключения к БД")
	})

	t.Run("Ошибка: Неверные креды или хост", func(t *testing.T) {
		// Этот тест также требует, чтобы *не* было доступной БД по этому адресу
		wrongDSN := "postgres://wronguser:wrongpassword@nonexistenthost:5432/wrongdb?sslmode=disable"

		db, err := repository.NewPostgresDB(wrongDSN)

		// Проверяем, что есть ошибка
		require.Error(t, err)
		// Проверяем, что объект БД nil
		assert.Nil(t, db)
		// Ошибка может быть как "ошибка подключения", так и "ошибка проверки соединения (ping)"
		// в зависимости от того, на каком этапе драйвер обнаружит проблему.
		// Поэтому проверяем общее начало сообщения
		assert.Contains(t, err.Error(), "ошибка")
	})
}
