package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/maynagashev/gophkeeper/models"
)

// Коды ошибок PostgreSQL.
const (
	pgUniqueViolationCode = "23505"
)

// UserRepository определяет методы для работы с данными пользователей в хранилище.
type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) (int64, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
}

// postgresUserRepository реализует UserRepository для PostgreSQL.
type postgresUserRepository struct {
	db *sqlx.DB
}

// NewPostgresUserRepository создает новый экземпляр репозитория пользователей для PostgreSQL.
func NewPostgresUserRepository(db *sqlx.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

// CreateUser создает нового пользователя в базе данных.
// Возвращает ID созданного пользователя или ошибку.
func (r *postgresUserRepository) CreateUser(ctx context.Context, user *models.User) (int64, error) {
	query := `INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id`
	var userID int64

	err := r.db.QueryRowxContext(ctx, query, user.Username, user.PasswordHash).Scan(&userID)
	if err != nil {
		// Проверяем на ошибку нарушения уникальности (duplicate key)
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode {
			log.Printf("[Repo] Ошибка создания пользователя: имя пользователя '%s' уже занято", user.Username)
			return 0, ErrUsernameTaken // Возвращаем кастомную ошибку
		}
		log.Printf("[Repo] Непредвиденная ошибка при создании пользователя '%s': %v", user.Username, err)
		return 0, fmt.Errorf("ошибка выполнения запроса на создание пользователя: %w", err)
	}

	log.Printf("[Repo] Пользователь '%s' успешно создан с ID %d", user.Username, userID)
	return userID, nil
}

// GetUserByUsername находит пользователя по его имени.
// Возвращает пользователя или ошибку, если пользователь не найден или произошла другая ошибка.
func (r *postgresUserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username=$1`
	var user models.User

	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[Repo] Пользователь с именем '%s' не найден", username)
			return nil, ErrUserNotFound // Пользователь не найден
		}
		log.Printf("[Repo] Ошибка при поиске пользователя '%s': %v", username, err)
		return nil, fmt.Errorf("ошибка выполнения запроса на получение пользователя: %w", err)
	}

	log.Printf("[Repo] Найден пользователь '%s' (ID: %d)", username, user.ID)
	return &user, nil
}

// Кастомные ошибки репозитория.
var (
	ErrUserNotFound  = errors.New("пользователь не найден")
	ErrUsernameTaken = errors.New("имя пользователя уже занято")
)
