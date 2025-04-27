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

func TestNewPostgresVaultVersionRepository(t *testing.T) {
	// Можно передать nil
	repo := repository.NewPostgresVaultVersionRepository(nil)
	assert.NotNil(t, repo)

	// Или с моком
	db, _, _ := sqlmock.New()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo = repository.NewPostgresVaultVersionRepository(sqlxDB)
	assert.NotNil(t, repo)
}

// Вспомогательная функция для создания мока БД и репозитория версий хранилищ.
func setupVaultVersionRepoMock(t *testing.T) (repository.VaultVersionRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := repository.NewPostgresVaultVersionRepository(sqlxDB)
	return repo, mock
}

func TestCreateVersion(t *testing.T) {
	now := time.Now()
	checksum := "abc"
	sizeBytes := int64(1024)
	versionContentModifiedAt := now.Add(-time.Hour)

	tests := []struct {
		name        string
		version     *models.VaultVersion
		mockSetup   func(mock sqlmock.Sqlmock, version *models.VaultVersion)
		expectedID  int64
		expectedErr error
	}{
		{
			name: "Успешное создание",
			version: &models.VaultVersion{
				VaultID:           501,
				ObjectKey:         "key1",
				Checksum:          &checksum,
				SizeBytes:         &sizeBytes,
				ContentModifiedAt: &versionContentModifiedAt,
			},
			mockSetup: func(mock sqlmock.Sqlmock, version *models.VaultVersion) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(601))
				query := regexp.QuoteMeta(
					`INSERT INTO vault_versions (vault_id, object_key, checksum, size_bytes, content_modified_at)` +
						` VALUES ($1, $2, $3, $4, $5) RETURNING id`,
				)
				mock.ExpectQuery(query).
					WithArgs(version.VaultID, version.ObjectKey, version.Checksum, version.SizeBytes, version.ContentModifiedAt).
					WillReturnRows(rows)
			},
			expectedID:  601,
			expectedErr: nil,
		},
		{
			name: "Ключ объекта уже существует",
			version: &models.VaultVersion{
				VaultID:           502,
				ObjectKey:         "existing_key",
				Checksum:          &checksum,
				SizeBytes:         &sizeBytes,
				ContentModifiedAt: &versionContentModifiedAt,
			},
			mockSetup: func(mock sqlmock.Sqlmock, version *models.VaultVersion) {
				query := regexp.QuoteMeta(
					`INSERT INTO vault_versions (vault_id, object_key, checksum, size_bytes, content_modified_at)` +
						` VALUES ($1, $2, $3, $4, $5) RETURNING id`,
				)
				pqErr := &pq.Error{Code: "23505"} // unique_violation
				mock.ExpectQuery(query).
					WithArgs(version.VaultID, version.ObjectKey, version.Checksum, version.SizeBytes, version.ContentModifiedAt).
					WillReturnError(pqErr)
			},
			expectedID:  0,
			expectedErr: errors.New("версия с ключом объекта 'existing_key' уже существует"), // Ожидаем обернутую ошибку
		},
		{
			name: "Другая ошибка базы данных",
			version: &models.VaultVersion{
				VaultID:           503,
				ObjectKey:         "error_key",
				Checksum:          &checksum,
				SizeBytes:         &sizeBytes,
				ContentModifiedAt: &versionContentModifiedAt,
			},
			mockSetup: func(mock sqlmock.Sqlmock, version *models.VaultVersion) {
				query := regexp.QuoteMeta(
					`INSERT INTO vault_versions (vault_id, object_key, checksum, size_bytes, content_modified_at)` +
						` VALUES ($1, $2, $3, $4, $5) RETURNING id`,
				)
				dbErr := errors.New("connection error")
				mock.ExpectQuery(query).
					WithArgs(version.VaultID, version.ObjectKey, version.Checksum, version.SizeBytes, version.ContentModifiedAt).
					WillReturnError(dbErr)
			},
			expectedID:  0,
			expectedErr: errors.New("ошибка выполнения запроса на создание версии"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := setupVaultVersionRepoMock(t)
			tt.mockSetup(mock, tt.version)

			versionID, err := repo.CreateVersion(context.Background(), tt.version)

			assert.Equal(t, tt.expectedID, versionID)
			if tt.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			}

			assert.NoError(t, mock.ExpectationsWereMet(), "Не все ожидания мока были выполнены")
		})
	}
}

func TestListVersionsByVaultID(t *testing.T) {
	now := time.Now()
	checksum1 := "abc"
	checksum2 := "def"
	sizeBytes1 := int64(1024)
	sizeBytes2 := int64(2048)
	modTime1 := now.Add(-time.Hour)
	modTime2 := now.Add(-2 * time.Hour)

	versionsList := []models.VaultVersion{
		{
			ID: 601, VaultID: 501, ObjectKey: "key1", Checksum: &checksum1,
			SizeBytes: &sizeBytes1, CreatedAt: now, ContentModifiedAt: &modTime1,
		},
		{
			ID: 600, VaultID: 501, ObjectKey: "key0", Checksum: &checksum2,
			SizeBytes: &sizeBytes2, CreatedAt: now.Add(-time.Minute), ContentModifiedAt: &modTime2,
		},
	}

	tests := []struct {
		name             string
		vaultID          int64
		limit            int
		offset           int
		mockSetup        func(mock sqlmock.Sqlmock, vaultID int64, limit, offset int)
		expectedVersions []models.VaultVersion
		expectedErr      error
	}{
		{
			name:    "Успех: Получение списка версий",
			vaultID: 501,
			limit:   10,
			offset:  0,
			mockSetup: func(mock sqlmock.Sqlmock, vaultID int64, limit, offset int) {
				rows := sqlmock.NewRows([]string{
					"id", "vault_id", "object_key", "checksum", "size_bytes",
					"created_at", "content_modified_at",
				})
				for _, v := range versionsList {
					rows.AddRow(v.ID, v.VaultID, v.ObjectKey, v.Checksum, v.SizeBytes, v.CreatedAt, v.ContentModifiedAt)
				}
				query := regexp.QuoteMeta(
					`SELECT id, vault_id, object_key, checksum, size_bytes, created_at, content_modified_at` +
						` FROM vault_versions WHERE vault_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
				)
				mock.ExpectQuery(query).WithArgs(vaultID, limit, offset).WillReturnRows(rows)
			},
			expectedVersions: versionsList,
			expectedErr:      nil,
		},
		{
			name:    "Успех: Пустой список",
			vaultID: 502,
			limit:   10,
			offset:  0,
			mockSetup: func(mock sqlmock.Sqlmock, vaultID int64, limit, offset int) {
				rows := sqlmock.NewRows([]string{
					"id", "vault_id", "object_key", "checksum", "size_bytes",
					"created_at", "content_modified_at",
				})
				query := regexp.QuoteMeta(
					`SELECT id, vault_id, object_key, checksum, size_bytes, created_at, content_modified_at` +
						` FROM vault_versions WHERE vault_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
				)
				mock.ExpectQuery(query).WithArgs(vaultID, limit, offset).WillReturnRows(rows)
			},
			expectedVersions: []models.VaultVersion{}, // Ожидаем пустой срез
			expectedErr:      nil,
		},
		{
			name:    "Ошибка базы данных",
			vaultID: 503,
			limit:   10,
			offset:  0,
			mockSetup: func(mock sqlmock.Sqlmock, vaultID int64, limit, offset int) {
				query := regexp.QuoteMeta(
					`SELECT id, vault_id, object_key, checksum, size_bytes, created_at, content_modified_at` +
						` FROM vault_versions WHERE vault_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
				)
				dbErr := errors.New("select error")
				mock.ExpectQuery(query).WithArgs(vaultID, limit, offset).WillReturnError(dbErr)
			},
			expectedVersions: nil,
			expectedErr:      errors.New("ошибка выполнения запроса"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := setupVaultVersionRepoMock(t)
			tt.mockSetup(mock, tt.vaultID, tt.limit, tt.offset)

			versions, err := repo.ListVersionsByVaultID(context.Background(), tt.vaultID, tt.limit, tt.offset)

			if tt.expectedErr == nil {
				require.NoError(t, err)
				// Сравниваем срезы
				assert.Equal(t, tt.expectedVersions, versions)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
				assert.Nil(t, versions) // В случае ошибки ожидаем nil срез
			}

			assert.NoError(t, mock.ExpectationsWereMet(), "Не все ожидания мока были выполнены")
		})
	}
}

func TestGetVersionByID(t *testing.T) {
	now := time.Now()
	checksum := "abc"
	sizeBytes := int64(1024)
	modTime := now.Add(-time.Hour)
	testVersion := &models.VaultVersion{
		ID:                601,
		VaultID:           501,
		ObjectKey:         "key1",
		Checksum:          &checksum,
		SizeBytes:         &sizeBytes,
		CreatedAt:         now,
		ContentModifiedAt: &modTime,
	}

	tests := []struct {
		name            string
		versionID       int64
		mockSetup       func(mock sqlmock.Sqlmock, versionID int64)
		expectedVersion *models.VaultVersion
		expectedErr     error
	}{
		{
			name:      "Успешный поиск",
			versionID: 601,
			mockSetup: func(mock sqlmock.Sqlmock, versionID int64) {
				rows := sqlmock.NewRows([]string{
					"id", "vault_id", "object_key", "checksum", "size_bytes",
					"created_at", "content_modified_at",
				}).AddRow(
					testVersion.ID, testVersion.VaultID, testVersion.ObjectKey, testVersion.Checksum,
					testVersion.SizeBytes, testVersion.CreatedAt, testVersion.ContentModifiedAt,
				)
				query := regexp.QuoteMeta(
					`SELECT id, vault_id, object_key, checksum, size_bytes, created_at, content_modified_at` +
						` FROM vault_versions WHERE id=$1`,
				)
				mock.ExpectQuery(query).WithArgs(versionID).WillReturnRows(rows)
			},
			expectedVersion: testVersion,
			expectedErr:     nil,
		},
		{
			name:      "Версия не найдена",
			versionID: 602,
			mockSetup: func(mock sqlmock.Sqlmock, versionID int64) {
				query := regexp.QuoteMeta(
					`SELECT id, vault_id, object_key, checksum, size_bytes, created_at, content_modified_at` +
						` FROM vault_versions WHERE id=$1`,
				)
				mock.ExpectQuery(query).WithArgs(versionID).WillReturnError(sql.ErrNoRows)
			},
			expectedVersion: nil,
			expectedErr:     repository.ErrVersionNotFound,
		},
		{
			name:      "Ошибка базы данных",
			versionID: 603,
			mockSetup: func(mock sqlmock.Sqlmock, versionID int64) {
				query := regexp.QuoteMeta(
					`SELECT id, vault_id, object_key, checksum, size_bytes, created_at, content_modified_at` +
						` FROM vault_versions WHERE id=$1`,
				)
				dbErr := errors.New("get error")
				mock.ExpectQuery(query).WithArgs(versionID).WillReturnError(dbErr)
			},
			expectedVersion: nil,
			expectedErr:     errors.New("ошибка выполнения запроса"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := setupVaultVersionRepoMock(t)
			tt.mockSetup(mock, tt.versionID)

			version, err := repo.GetVersionByID(context.Background(), tt.versionID)

			assert.Equal(t, tt.expectedVersion, version)

			if tt.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if errors.Is(tt.expectedErr, repository.ErrVersionNotFound) {
					assert.ErrorIs(t, err, repository.ErrVersionNotFound)
				} else {
					assert.Contains(t, err.Error(), "ошибка выполнения запроса")
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet(), "Не все ожидания мока были выполнены")
		})
	}
}
