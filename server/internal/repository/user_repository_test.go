package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/maynagashev/gophkeeper/server/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgresUserRepository(t *testing.T) {
	// Можно передать nil, так как конструктор его просто сохраняет
	repo := repository.NewPostgresUserRepository(nil)
	assert.NotNil(t, repo)

	// Или с моком
	db, _, _ := sqlmock.New()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo = repository.NewPostgresUserRepository(sqlxDB)
	assert.NotNil(t, repo)
}

// Вспомогательная функция для создания мока БД и репозитория.
func setupUserRepoMock(t *testing.T) (repository.UserRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := repository.NewPostgresUserRepository(sqlxDB)
	return repo, mock
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name        string
		user        *models.User
		mockSetup   func(mock sqlmock.Sqlmock, user *models.User)
		expectedID  int64
		expectedErr error
	}{
		{
			name: "Успешное создание",
			user: &models.User{Username: "newuser", PasswordHash: "hash123"},
			mockSetup: func(mock sqlmock.Sqlmock, user *models.User) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(1))
				// Используем regexp.QuoteMeta для экранирования SQL запроса
				query := regexp.QuoteMeta(`INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id`)
				mock.ExpectQuery(query).WithArgs(user.Username, user.PasswordHash).WillReturnRows(rows)
			},
			expectedID:  1,
			expectedErr: nil,
		},
		{
			name: "Имя пользователя занято",
			user: &models.User{Username: "existinguser", PasswordHash: "hash456"},
			mockSetup: func(mock sqlmock.Sqlmock, user *models.User) {
				query := regexp.QuoteMeta(`INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id`)
				// Создаем ошибку PostgreSQL unique_violation, используя строковый код
				pqErr := &pq.Error{Code: "23505"} // Используем строковое значение
				mock.ExpectQuery(query).WithArgs(user.Username, user.PasswordHash).WillReturnError(pqErr)
			},
			expectedID:  0,
			expectedErr: repository.ErrUsernameTaken,
		},
		{
			name: "Ошибка базы данных",
			user: &models.User{Username: "erroruser", PasswordHash: "hash789"},
			mockSetup: func(mock sqlmock.Sqlmock, user *models.User) {
				query := regexp.QuoteMeta(`INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id`)
				dbErr := errors.New("database error")
				mock.ExpectQuery(query).WithArgs(user.Username, user.PasswordHash).WillReturnError(dbErr)
			},
			expectedID:  0,
			expectedErr: errors.New("ошибка выполнения запроса"), // Ожидаем обернутую ошибку
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := setupUserRepoMock(t)
			tt.mockSetup(mock, tt.user)

			userID, err := repo.CreateUser(context.Background(), tt.user)

			assert.Equal(t, tt.expectedID, userID)
			if tt.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if errors.Is(tt.expectedErr, repository.ErrUsernameTaken) {
					assert.ErrorIs(t, err, repository.ErrUsernameTaken)
				} else {
					assert.Contains(t, err.Error(), "ошибка выполнения запроса")
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet(), "Не все ожидания мока были выполнены")
		})
	}
}

func TestGetUserByUsername(t *testing.T) {
	// Определяем тестового пользователя заранее
	now := time.Now()
	testUser := &models.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: "hash123",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	tests := []struct {
		name         string
		username     string
		mockSetup    func(mock sqlmock.Sqlmock, username string)
		expectedUser *models.User
		expectedErr  error
	}{
		{
			name:     "Успешный поиск",
			username: "testuser",
			mockSetup: func(mock sqlmock.Sqlmock, username string) {
				rows := sqlmock.NewRows([]string{"id", "username", "password_hash", "created_at", "updated_at"}).
					AddRow(testUser.ID, testUser.Username, testUser.PasswordHash, testUser.CreatedAt, testUser.UpdatedAt)
				query := regexp.QuoteMeta(`SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username=$1`)
				mock.ExpectQuery(query).WithArgs(username).WillReturnRows(rows)
			},
			expectedUser: testUser,
			expectedErr:  nil,
		},
		{
			name:     "Пользователь не найден",
			username: "notfounduser",
			mockSetup: func(mock sqlmock.Sqlmock, username string) {
				query := regexp.QuoteMeta(`SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username=$1`)
				mock.ExpectQuery(query).WithArgs(username).WillReturnError(sql.ErrNoRows)
			},
			expectedUser: nil,
			expectedErr:  repository.ErrUserNotFound,
		},
		{
			name:     "Ошибка базы данных",
			username: "erroruser",
			mockSetup: func(mock sqlmock.Sqlmock, username string) {
				query := regexp.QuoteMeta(`SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username=$1`)
				dbErr := errors.New("database error")
				mock.ExpectQuery(query).WithArgs(username).WillReturnError(dbErr)
			},
			expectedUser: nil,
			expectedErr:  errors.New("ошибка выполнения запроса"), // Ожидаем обернутую ошибку
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := setupUserRepoMock(t)
			tt.mockSetup(mock, tt.username)

			user, err := repo.GetUserByUsername(context.Background(), tt.username)

			assert.Equal(t, tt.expectedUser, user)

			if tt.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if errors.Is(tt.expectedErr, repository.ErrUserNotFound) {
					assert.ErrorIs(t, err, repository.ErrUserNotFound)
				} else {
					assert.Contains(t, err.Error(), "ошибка выполнения запроса")
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet(), "Не все ожидания мока были выполнены")
		})
	}
}
