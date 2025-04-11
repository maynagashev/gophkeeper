package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/maynagashev/gophkeeper/server/internal/models"
)

// VaultRepository определяет методы для работы с метаданными хранилищ.
type VaultRepository interface {
	GetVaultByUserID(ctx context.Context, userID int64) (*models.Vault, error)
	CreateVault(ctx context.Context, vault *models.Vault) (int64, error)
	UpdateVaultMetadata(ctx context.Context, vault *models.Vault) error
	// TODO: Добавить методы CreateVault, UpdateVault и т.д., когда они понадобятся
}

// postgresVaultRepository реализует VaultRepository для PostgreSQL.
type postgresVaultRepository struct {
	db *sqlx.DB
}

// NewPostgresVaultRepository создает новый экземпляр репозитория хранилищ.
func NewPostgresVaultRepository(db *sqlx.DB) VaultRepository {
	return &postgresVaultRepository{db: db}
}

// GetVaultByUserID находит метаданные хранилища по ID пользователя.
// Предполагается, что у одного пользователя пока только одно хранилище.
// Возвращает метаданные или ошибку (включая ErrVaultNotFound).
func (r *postgresVaultRepository) GetVaultByUserID(ctx context.Context, userID int64) (*models.Vault, error) {
	query := `SELECT id, user_id, object_key, checksum, size_bytes, last_modified_server, created_at, updated_at
	          FROM vaults WHERE user_id=$1 LIMIT 1` // LIMIT 1, т.к. пока одно хранилище на пользователя
	var vault models.Vault

	err := r.db.GetContext(ctx, &vault, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[VaultRepo] Метаданные хранилища для пользователя ID %d не найдены", userID)
			return nil, ErrVaultNotFound // Метаданные не найдены
		}
		log.Printf("[VaultRepo] Ошибка при поиске метаданных для пользователя ID %d: %v", userID, err)
		return nil, fmt.Errorf("ошибка выполнения запроса на получение метаданных: %w", err)
	}

	log.Printf("[VaultRepo] Найдены метаданные хранилища (ID: %d) для пользователя ID %d", vault.ID, userID)
	return &vault, nil
}

// CreateVault создает новую запись о метаданных хранилища.
func (r *postgresVaultRepository) CreateVault(ctx context.Context, vault *models.Vault) (int64, error) {
	query := `INSERT INTO vaults (user_id, object_key, checksum, size_bytes)
	          VALUES ($1, $2, $3, $4) RETURNING id`
	var vaultID int64

	err := r.db.QueryRowxContext(ctx, query,
		vault.UserID, vault.ObjectKey, vault.Checksum, vault.SizeBytes,
	).Scan(&vaultID)

	if err != nil {
		// Проверяем на ошибку нарушения уникальности object_key или user_id (если решим добавить такой constraint)
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode {
			log.Printf("[VaultRepo] Ошибка создания метаданных: ключ объекта '%s'"+
				" или user_id %d уже существует", vault.ObjectKey, vault.UserID)
			return 0, fmt.Errorf("метаданные для этого пользователя или ключа объекта уже существуют: %w", err)
		}
		log.Printf("[VaultRepo] Непредвиденная ошибка при создании метаданных для '%s': %v", vault.ObjectKey, err)
		return 0, fmt.Errorf("ошибка выполнения запроса на создание метаданных: %w", err)
	}

	log.Printf("[VaultRepo] Метаданные хранилища (ID: %d) успешно созданы для пользователя ID %d", vaultID, vault.UserID)
	return vaultID, nil
}

// UpdateVaultMetadata обновляет метаданные существующего хранилища (checksum, size_bytes, last_modified_server).
// Обновление происходит по user_id, предполагая одно хранилище на пользователя.
func (r *postgresVaultRepository) UpdateVaultMetadata(ctx context.Context, vault *models.Vault) error {
	// Мы обновляем по user_id, но можно и по vault.ID, если он известен.
	// last_modified_server обновится автоматически триггером, если передать другие поля.
	query := `UPDATE vaults SET checksum=$1, size_bytes=$2, updated_at=NOW() WHERE user_id=$3`

	result, err := r.db.ExecContext(ctx, query, vault.Checksum, vault.SizeBytes, vault.UserID)
	if err != nil {
		log.Printf("[VaultRepo] Ошибка обновления метаданных для пользователя ID %d: %v", vault.UserID, err)
		return fmt.Errorf("ошибка выполнения запроса на обновление метаданных: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// Ошибка получения количества затронутых строк (редко)
		log.Printf("[VaultRepo] Ошибка получения rowsAffected при обновлении"+
			" метаданных для пользователя ID %d: %v", vault.UserID, err)
		return fmt.Errorf("ошибка получения результата обновления метаданных: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("[VaultRepo] Метаданные для обновления не найдены для пользователя ID %d", vault.UserID)
		return ErrVaultNotFound // Используем ту же ошибку, что и при Get
	}

	log.Printf("[VaultRepo] Метаданные хранилища для пользователя ID %d успешно обновлены", vault.UserID)
	return nil
}

// Кастомная ошибка репозитория.
var (
	ErrVaultNotFound = errors.New("метаданные хранилища не найдены")
)
