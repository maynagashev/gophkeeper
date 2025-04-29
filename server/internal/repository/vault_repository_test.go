package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/maynagashev/gophkeeper/server/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgresVaultRepository(t *testing.T) {
	// Можно передать nil
	repo := repository.NewPostgresVaultRepository(nil)
	assert.NotNil(t, repo)

	// Или с моком
	db, _, _ := sqlmock.New()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo = repository.NewPostgresVaultRepository(sqlxDB)
	assert.NotNil(t, repo)
}

// Вспомогательная функция для создания мока БД и репозитория хранилищ.
func setupVaultRepoMock(t *testing.T) (repository.VaultRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := repository.NewPostgresVaultRepository(sqlxDB)
	return repo, mock
}

func TestCreateVault(t *testing.T) {
	tests := []struct {
		name        string
		vault       *models.Vault
		mockSetup   func(mock sqlmock.Sqlmock, vault *models.Vault)
		expectedID  int64
		expectedErr error
	}{
		{
			name:  "Успешное создание",
			vault: &models.Vault{UserID: 101},
			mockSetup: func(mock sqlmock.Sqlmock, vault *models.Vault) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(501))
				query := regexp.QuoteMeta(`INSERT INTO vaults (user_id) VALUES ($1) RETURNING id`)
				mock.ExpectQuery(query).WithArgs(vault.UserID).WillReturnRows(rows)
			},
			expectedID:  501,
			expectedErr: nil,
		},
		{
			name:  "Ошибка базы данных",
			vault: &models.Vault{UserID: 102},
			mockSetup: func(mock sqlmock.Sqlmock, vault *models.Vault) {
				query := regexp.QuoteMeta(`INSERT INTO vaults (user_id) VALUES ($1) RETURNING id`)
				dbErr := errors.New("db connection error")
				mock.ExpectQuery(query).WithArgs(vault.UserID).WillReturnError(dbErr)
			},
			expectedID:  0,
			expectedErr: errors.New("ошибка выполнения запроса"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := setupVaultRepoMock(t)
			tt.mockSetup(mock, tt.vault)

			vaultID, err := repo.CreateVault(context.Background(), tt.vault)

			assert.Equal(t, tt.expectedID, vaultID)
			if tt.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "ошибка выполнения запроса")
			}

			assert.NoError(t, mock.ExpectationsWereMet(), "Не все ожидания мока были выполнены")
		})
	}
}

func TestGetVaultByUserID(t *testing.T) {
	now := time.Now()
	versionID := int64(601)
	testVault := &models.Vault{
		ID:               501,
		UserID:           101,
		CurrentVersionID: &versionID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	tests := []struct {
		name          string
		userID        int64
		mockSetup     func(mock sqlmock.Sqlmock, userID int64)
		expectedVault *models.Vault
		expectedErr   error
	}{
		{
			name:   "Успешный поиск",
			userID: 101,
			mockSetup: func(mock sqlmock.Sqlmock, userID int64) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "current_version_id", "created_at", "updated_at"}).
					AddRow(testVault.ID, testVault.UserID, testVault.CurrentVersionID, testVault.CreatedAt, testVault.UpdatedAt)
				query := regexp.QuoteMeta(
					`SELECT id, user_id, current_version_id, created_at, updated_at FROM vaults WHERE user_id=$1 LIMIT 1`,
				)
				mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)
			},
			expectedVault: testVault,
			expectedErr:   nil,
		},
		{
			name:   "Хранилище не найдено",
			userID: 102,
			mockSetup: func(mock sqlmock.Sqlmock, userID int64) {
				query := regexp.QuoteMeta(
					`SELECT id, user_id, current_version_id, created_at, updated_at FROM vaults WHERE user_id=$1 LIMIT 1`,
				)
				mock.ExpectQuery(query).WithArgs(userID).WillReturnError(sql.ErrNoRows)
			},
			expectedVault: nil,
			expectedErr:   repository.ErrVaultNotFound,
		},
		{
			name:   "Ошибка базы данных",
			userID: 103,
			mockSetup: func(mock sqlmock.Sqlmock, userID int64) {
				query := regexp.QuoteMeta(
					`SELECT id, user_id, current_version_id, created_at, updated_at FROM vaults WHERE user_id=$1 LIMIT 1`,
				)
				dbErr := errors.New("connection failed")
				mock.ExpectQuery(query).WithArgs(userID).WillReturnError(dbErr)
			},
			expectedVault: nil,
			expectedErr:   errors.New("ошибка выполнения запроса"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := setupVaultRepoMock(t)
			tt.mockSetup(mock, tt.userID)

			vault, err := repo.GetVaultByUserID(context.Background(), tt.userID)

			assert.Equal(t, tt.expectedVault, vault)

			if tt.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if errors.Is(tt.expectedErr, repository.ErrVaultNotFound) {
					assert.ErrorIs(t, err, repository.ErrVaultNotFound)
				} else {
					assert.Contains(t, err.Error(), "ошибка выполнения запроса")
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet(), "Не все ожидания мока были выполнены")
		})
	}
}

func TestUpdateVaultCurrentVersion(t *testing.T) {
	tests := []struct {
		name        string
		vaultID     int64
		versionID   int64
		mockSetup   func(mock sqlmock.Sqlmock, vaultID, versionID int64)
		expectedErr error
	}{
		{
			name:      "Успешное обновление",
			vaultID:   501,
			versionID: 601,
			mockSetup: func(mock sqlmock.Sqlmock, vaultID, versionID int64) {
				query := regexp.QuoteMeta(`UPDATE vaults SET current_version_id=$1, updated_at=NOW() WHERE id=$2`)
				mock.ExpectExec(query).WithArgs(versionID, vaultID).
					WillReturnResult(sqlmock.NewResult(0, 1)) // lastInsertId=0, rowsAffected=1
			},
			expectedErr: nil,
		},
		{
			name:      "Хранилище не найдено",
			vaultID:   502, // Несуществующий ID
			versionID: 602,
			mockSetup: func(mock sqlmock.Sqlmock, vaultID, versionID int64) {
				query := regexp.QuoteMeta(`UPDATE vaults SET current_version_id=$1, updated_at=NOW() WHERE id=$2`)
				mock.ExpectExec(query).WithArgs(versionID, vaultID).
					WillReturnResult(sqlmock.NewResult(0, 0)) // rowsAffected=0
			},
			expectedErr: repository.ErrVaultNotFound,
		},
		{
			name:      "Ошибка базы данных при Exec",
			vaultID:   503,
			versionID: 603,
			mockSetup: func(mock sqlmock.Sqlmock, vaultID, versionID int64) {
				query := regexp.QuoteMeta(`UPDATE vaults SET current_version_id=$1, updated_at=NOW() WHERE id=$2`)
				mock.ExpectExec(query).WithArgs(versionID, vaultID).
					WillReturnError(errors.New("exec error"))
			},
			expectedErr: errors.New("ошибка выполнения запроса"),
		},
		{
			name:      "Ошибка при получении RowsAffected",
			vaultID:   504,
			versionID: 604,
			mockSetup: func(mock sqlmock.Sqlmock, vaultID, versionID int64) {
				query := regexp.QuoteMeta(`UPDATE vaults SET current_version_id=$1, updated_at=NOW() WHERE id=$2`)
				// Мокируем результат, который вызовет ошибку при RowsAffected()
				mock.ExpectExec(query).WithArgs(versionID, vaultID).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedErr: errors.New("ошибка получения результата"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := setupVaultRepoMock(t)
			tt.mockSetup(mock, tt.vaultID, tt.versionID)

			err := repo.UpdateVaultCurrentVersion(context.Background(), tt.vaultID, tt.versionID)

			if tt.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if errors.Is(tt.expectedErr, repository.ErrVaultNotFound) {
					assert.ErrorIs(t, err, repository.ErrVaultNotFound)
				} else {
					// Проверяем частичное совпадение для обернутых ошибок
					assert.Contains(t, err.Error(), tt.expectedErr.Error())
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet(), "Не все ожидания мока были выполнены")
		})
	}
}

func TestGetVaultWithCurrentVersionByUserID(t *testing.T) {
	now := time.Now()
	versionID := int64(601)
	checksum := "abc"
	sizeBytes := int64(1024)
	versionContentModifiedAt := now.Add(-time.Hour)

	testVault := &models.Vault{
		ID:               501,
		UserID:           101,
		CurrentVersionID: &versionID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	testVersion := &models.VaultVersion{
		ID:                versionID,
		VaultID:           testVault.ID,
		ObjectKey:         fmt.Sprintf("%d_%d.vault", testVault.UserID, versionID),
		Checksum:          &checksum,
		SizeBytes:         &sizeBytes,
		CreatedAt:         now,
		ContentModifiedAt: &versionContentModifiedAt,
	}
	// Хранилище без текущей версии
	testVaultNoVersion := &models.Vault{
		ID:               502,
		UserID:           102,
		CurrentVersionID: nil,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	tests := []struct {
		name            string
		userID          int64
		mockSetup       func(mock sqlmock.Sqlmock, userID int64)
		expectedVault   *models.Vault
		expectedVersion *models.VaultVersion
		expectedErr     error
	}{
		{
			name:   "Успех: Хранилище с текущей версией",
			userID: 101,
			mockSetup: func(mock sqlmock.Sqlmock, userID int64) {
				rows := sqlmock.NewRows([]string{
					"vault_id", "user_id", "vault_created_at", "vault_updated_at",
					"version_id", "object_key", "checksum", "size_bytes",
					"version_created_at", "version_content_modified_at",
				}).AddRow(
					testVault.ID, testVault.UserID, testVault.CreatedAt, testVault.UpdatedAt,
					testVersion.ID, testVersion.ObjectKey, testVersion.Checksum, testVersion.SizeBytes,
					testVersion.CreatedAt, testVersion.ContentModifiedAt,
				)
				// Используем частичный матчинг запроса, т.к. он многострочный
				mock.ExpectQuery(`SELECT v.id AS vault_id`).WithArgs(userID).WillReturnRows(rows)
			},
			expectedVault:   testVault,
			expectedVersion: testVersion,
			expectedErr:     nil,
		},
		{
			name:   "Успех: Хранилище без текущей версии",
			userID: 102,
			mockSetup: func(mock sqlmock.Sqlmock, userID int64) {
				rows := sqlmock.NewRows([]string{
					"vault_id", "user_id", "vault_created_at", "vault_updated_at",
					"version_id", "object_key", "checksum", "size_bytes",
					"version_created_at", "version_content_modified_at",
				}).AddRow(
					testVaultNoVersion.ID, testVaultNoVersion.UserID, testVaultNoVersion.CreatedAt, testVaultNoVersion.UpdatedAt,
					nil, nil, nil, nil, nil, nil, // Все поля версии NULL
				)
				mock.ExpectQuery(`SELECT v.id AS vault_id`).WithArgs(userID).WillReturnRows(rows)
			},
			expectedVault:   testVaultNoVersion,
			expectedVersion: nil,
			expectedErr:     nil,
		},
		{
			name:   "Хранилище не найдено",
			userID: 103,
			mockSetup: func(mock sqlmock.Sqlmock, userID int64) {
				mock.ExpectQuery(`SELECT v.id AS vault_id`).WithArgs(userID).WillReturnError(sql.ErrNoRows)
			},
			expectedVault:   nil,
			expectedVersion: nil,
			expectedErr:     repository.ErrVaultNotFound,
		},
		{
			name:   "Ошибка базы данных",
			userID: 104,
			mockSetup: func(mock sqlmock.Sqlmock, userID int64) {
				dbErr := errors.New("query failed")
				mock.ExpectQuery(`SELECT v.id AS vault_id`).WithArgs(userID).WillReturnError(dbErr)
			},
			expectedVault:   nil,
			expectedVersion: nil,
			expectedErr:     errors.New("ошибка выполнения запроса"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := setupVaultRepoMock(t)
			tt.mockSetup(mock, tt.userID)

			vault, version, err := repo.GetVaultWithCurrentVersionByUserID(context.Background(), tt.userID)

			assert.Equal(t, tt.expectedVault, vault)
			assert.Equal(t, tt.expectedVersion, version)

			if tt.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if errors.Is(tt.expectedErr, repository.ErrVaultNotFound) {
					assert.ErrorIs(t, err, repository.ErrVaultNotFound)
				} else {
					assert.Contains(t, err.Error(), "ошибка выполнения запроса")
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet(), "Не все ожидания мока были выполнены")
		})
	}
}
