package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/maynagashev/gophkeeper/server/internal/models"
)

// VaultRepository определяет методы для работы с метаданными хранилищ.
type VaultRepository interface {
	GetVaultByUserID(ctx context.Context, userID int64) (*models.Vault, error)
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

// Кастомная ошибка репозитория.
var (
	ErrVaultNotFound = errors.New("метаданные хранилища не найдены")
)
