package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/maynagashev/gophkeeper/models"
	// "github.com/lib/pq" // Больше не нужен здесь, т.к. убрали CreateVault/UpdateVaultMetadata с проверкой UNIQUE.
)

// VaultRepository определяет методы для работы с основными записями хранилищ.
type VaultRepository interface {
	GetVaultByUserID(ctx context.Context, userID int64) (*models.Vault, error)
	CreateVault(ctx context.Context, vault *models.Vault) (int64, error)
	UpdateVaultCurrentVersion(ctx context.Context, vaultID int64, versionID int64) error
	GetVaultWithCurrentVersionByUserID(ctx context.Context, userID int64) (*models.Vault, *models.VaultVersion, error)
}

// postgresVaultRepository реализует VaultRepository для PostgreSQL.
type postgresVaultRepository struct {
	db *sqlx.DB
}

// NewPostgresVaultRepository создает новый экземпляр репозитория хранилищ.
func NewPostgresVaultRepository(db *sqlx.DB) VaultRepository {
	return &postgresVaultRepository{db: db}
}

// GetVaultByUserID находит основную запись хранилища по ID пользователя.
func (r *postgresVaultRepository) GetVaultByUserID(ctx context.Context, userID int64) (*models.Vault, error) {
	query := `SELECT id, user_id, current_version_id, created_at, updated_at FROM vaults WHERE user_id=$1 LIMIT 1`
	var vault models.Vault

	err := r.db.GetContext(ctx, &vault, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[VaultRepo] Хранилище для пользователя ID %d не найдено", userID)
			return nil, ErrVaultNotFound
		}
		log.Printf("[VaultRepo] Ошибка при поиске хранилища для пользователя ID %d: %v", userID, err)
		return nil, fmt.Errorf("ошибка выполнения запроса на получение хранилища: %w", err)
	}

	log.Printf("[VaultRepo] Найдено хранилище (ID: %d) для пользователя ID %d", vault.ID, userID)
	return &vault, nil
}

// CreateVault создает новую основную запись о хранилище для пользователя.
// Поле current_version_id будет NULL по умолчанию.
func (r *postgresVaultRepository) CreateVault(ctx context.Context, vault *models.Vault) (int64, error) {
	// Убедимся, что current_version_id не передается при создании
	query := `INSERT INTO vaults (user_id) VALUES ($1) RETURNING id`
	var vaultID int64

	err := r.db.QueryRowxContext(ctx, query, vault.UserID).Scan(&vaultID)
	if err != nil {
		// Здесь может быть ошибка уникальности user_id, если мы добавим такой constraint
		log.Printf("[VaultRepo] Непредвиденная ошибка при создании хранилища для пользователя ID %d: %v", vault.UserID, err)
		return 0, fmt.Errorf("ошибка выполнения запроса на создание хранилища: %w", err)
	}

	log.Printf("[VaultRepo] Хранилище (ID: %d) успешно создано для пользователя ID %d", vaultID, vault.UserID)
	return vaultID, nil
}

// UpdateVaultCurrentVersion обновляет ссылку на текущую версию в записи хранилища.
func (r *postgresVaultRepository) UpdateVaultCurrentVersion(ctx context.Context, vaultID int64, versionID int64) error {
	query := `UPDATE vaults SET current_version_id=$1, updated_at=NOW() WHERE id=$2`

	result, err := r.db.ExecContext(ctx, query, versionID, vaultID)
	if err != nil {
		log.Printf("[VaultRepo] Ошибка обновления current_version_id для хранилища ID %d: %v", vaultID, err)
		return fmt.Errorf("ошибка выполнения запроса на обновление current_version_id: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[VaultRepo] Ошибка получения rowsAffected при обновлении"+
			" current_version_id для хранилища ID %d: %v", vaultID, err)
		return fmt.Errorf("ошибка получения результата обновления current_version_id: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("[VaultRepo] Хранилище ID %d не найдено для обновления current_version_id", vaultID)
		return ErrVaultNotFound
	}

	log.Printf("[VaultRepo] current_version_id для хранилища ID %d успешно обновлен на %d", vaultID, versionID)
	return nil
}

// GetVaultWithCurrentVersionByUserID получает запись хранилища и данные его текущей версии одним запросом.
func (r *postgresVaultRepository) GetVaultWithCurrentVersionByUserID(
	ctx context.Context,
	userID int64,
) (*models.Vault, *models.VaultVersion, error) {
	// Добавили выборку vv.content_modified_at
	query := `
		SELECT
		    v.id AS vault_id, v.user_id, v.created_at AS vault_created_at, v.updated_at AS vault_updated_at,
		    vv.id AS version_id, vv.object_key, vv.checksum, vv.size_bytes,
		    vv.created_at AS version_created_at, vv.content_modified_at AS version_content_modified_at
		FROM vaults v
		LEFT JOIN vault_versions vv ON v.current_version_id = vv.id
		WHERE v.user_id = $1
		LIMIT 1`

	// Используем временную структуру для сканирования результата JOIN
	// Добавили поле для content_modified_at
	type result struct {
		VaultID                  int64      `db:"vault_id"`
		UserID                   int64      `db:"user_id"`
		VaultCreatedAt           time.Time  `db:"vault_created_at"`
		VaultUpdatedAt           time.Time  `db:"vault_updated_at"`
		VersionID                *int64     `db:"version_id"` // Указатель, т.к. LEFT JOIN может дать NULL
		ObjectKey                *string    `db:"object_key"`
		Checksum                 *string    `db:"checksum"`
		SizeBytes                *int64     `db:"size_bytes"`
		VersionCreatedAt         *time.Time `db:"version_created_at"`
		VersionContentModifiedAt *time.Time `db:"version_content_modified_at"` // Указатель, т.к. LEFT JOIN может дать NULL
	}

	var res result
	err := r.db.GetContext(ctx, &res, query, userID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[VaultRepo] Хранилище (с версией) для пользователя ID %d не найдено", userID)
			return nil, nil, ErrVaultNotFound
		}
		log.Printf("[VaultRepo] Ошибка при поиске хранилища с версией для пользователя ID %d: %v", userID, err)
		return nil, nil, fmt.Errorf("ошибка выполнения запроса на получение хранилища с версией: %w", err)
	}

	vault := &models.Vault{
		ID:               res.VaultID,
		UserID:           res.UserID,
		CurrentVersionID: res.VersionID, // Передаем указатель
		CreatedAt:        res.VaultCreatedAt,
		UpdatedAt:        res.VaultUpdatedAt,
	}

	var currentVersion *models.VaultVersion
	if res.VersionID != nil { // Если есть текущая версия
		// Добавили заполнение ContentModifiedAt
		currentVersion = &models.VaultVersion{
			ID:                *res.VersionID,
			VaultID:           res.VaultID,
			ObjectKey:         *res.ObjectKey,
			Checksum:          res.Checksum,
			SizeBytes:         res.SizeBytes,
			CreatedAt:         *res.VersionCreatedAt,
			ContentModifiedAt: res.VersionContentModifiedAt, // Указатель на время или nil
		}
		log.Printf("[VaultRepo] Найдено хранилище ID %d с текущей версией ID %d"+
			" для пользователя %d", vault.ID, currentVersion.ID, userID)
	} else {
		log.Printf("[VaultRepo] Найдено хранилище ID %d без текущей версии для пользователя %d", vault.ID, userID)
	}

	return vault, currentVersion, nil
}

// Кастомная ошибка репозитория.
var (
	ErrVaultNotFound = errors.New("метаданные хранилища не найдены")
)
