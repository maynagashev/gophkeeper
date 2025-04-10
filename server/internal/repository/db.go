package repository

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Драйвер PostgreSQL, импортируем для регистрации
)

const (
	maxOpenConns    = 25              // Максимальное количество открытых соединений
	maxIdleConns    = 25              // Максимальное количество простаивающих соединений
	connMaxLifetime = 5 * time.Minute // Максимальное время жизни соединения
	connMaxIdleTime = 5 * time.Minute // Максимальное время простоя соединения
)

// NewPostgresDB создает и возвращает новое подключение к PostgreSQL.
func NewPostgresDB(dsn string) (*sqlx.DB, error) {
	log.Printf("Подключение к PostgreSQL...")

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	// Проверка соединения
	if err = db.Ping(); err != nil {
		// Закрываем соединение в случае ошибки пинга
		closeErr := db.Close()
		if closeErr != nil {
			log.Printf("Ошибка закрытия соединения с БД после неудачного пинга: %v", closeErr)
		}
		return nil, fmt.Errorf("ошибка проверки соединения с БД (ping): %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	log.Println("Подключение к PostgreSQL успешно установлено.")
	return db, nil
}
