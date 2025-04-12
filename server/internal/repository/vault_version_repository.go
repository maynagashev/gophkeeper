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

// VaultVersionRepository определяет методы для работы с версиями хранилищ.
type VaultVersionRepository interface {
	CreateVersion(ctx context.Context, version *models.VaultVersion) (int64, error)
	ListVersionsByVaultID(ctx context.Context, vaultID int64, limit, offset int) ([]models.VaultVersion, error)
	GetVersionByID(ctx context.Context, versionID int64) (*models.VaultVersion, error)
}

// postgresVaultVersionRepository реализует VaultVersionRepository для PostgreSQL.
type postgresVaultVersionRepository struct {
	db *sqlx.DB
}

// NewPostgresVaultVersionRepository создает новый экземпляр репозитория версий.
func NewPostgresVaultVersionRepository(db *sqlx.DB) VaultVersionRepository {
	return &postgresVaultVersionRepository{db: db}
}

// CreateVersion создает новую запись о версии хранилища.
func (r *postgresVaultVersionRepository) CreateVersion(
	ctx context.Context,
	version *models.VaultVersion,
) (int64, error) {
	query := `INSERT INTO vault_versions (vault_id, object_key, checksum, size_bytes)
	          VALUES ($1, $2, $3, $4) RETURNING id`
	var versionID int64

	err := r.db.QueryRowxContext(ctx, query,
		version.VaultID, version.ObjectKey, version.Checksum, version.SizeBytes,
	).Scan(&versionID)

	if err != nil {
		// Проверяем на ошибку уникальности object_key
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode {
			log.Printf("[VaultVerRepo] Ошибка создания версии: ключ объекта '%s' уже существует", version.ObjectKey)
			return 0, fmt.Errorf("версия с ключом объекта '%s' уже существует: %w", version.ObjectKey, err)
		}
		log.Printf("[VaultVerRepo] Непредвиденная ошибка при создании версии для '%s': %v", version.ObjectKey, err)
		return 0, fmt.Errorf("ошибка выполнения запроса на создание версии: %w", err)
	}

	log.Printf("[VaultVerRepo] Версия (ID: %d) успешно создана для хранилища ID %d", versionID, version.VaultID)
	return versionID, nil
}

// ListVersionsByVaultID возвращает список версий для указанного хранилища с пагинацией.
func (r *postgresVaultVersionRepository) ListVersionsByVaultID(
	ctx context.Context,
	vaultID int64,
	limit,
	offset int,
) ([]models.VaultVersion, error) {
	// Запрос с сортировкой по убыванию времени создания (сначала новые)
	query := `SELECT id, vault_id, object_key, checksum, size_bytes, created_at
	          FROM vault_versions
	          WHERE vault_id=$1
	          ORDER BY created_at DESC
	          LIMIT $2 OFFSET $3`

	versions := make([]models.VaultVersion, 0, limit)
	err := r.db.SelectContext(ctx, &versions, query, vaultID, limit, offset)
	if err != nil {
		log.Printf("[VaultVerRepo] Ошибка при получении списка версий для хранилища ID %d: %v", vaultID, err)
		return nil, fmt.Errorf("ошибка выполнения запроса на получение списка версий: %w", err)
	}

	log.Printf("[VaultVerRepo] Получено %d версий для хранилища ID %d (limit=%d, offset=%d)",
		len(versions), vaultID, limit, offset)
	return versions, nil
}

// GetVersionByID находит конкретную версию по ее ID.
func (r *postgresVaultVersionRepository) GetVersionByID(
	ctx context.Context,
	versionID int64,
) (*models.VaultVersion, error) {
	query := `SELECT id, vault_id, object_key, checksum, size_bytes, created_at` +
		` FROM vault_versions WHERE id=$1` // Разделяем длинный запрос
	var version models.VaultVersion

	err := r.db.GetContext(ctx, &version, query, versionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[VaultVerRepo] Версия с ID %d не найдена", versionID)
			return nil, ErrVersionNotFound // Кастомная ошибка
		}
		log.Printf("[VaultVerRepo] Ошибка при поиске версии ID %d: %v", versionID, err)
		return nil, fmt.Errorf("ошибка выполнения запроса на получение версии: %w", err)
	}

	log.Printf("[VaultVerRepo] Найдена версия ID %d (Хранилище ID: %d)", versionID, version.VaultID)
	return &version, nil
}

// Кастомные ошибки репозитория версий.
var (
	ErrVersionNotFound = errors.New("версия хранилища не найдена") // Возвращаем определение
)
