package services

import (
	"context"
	"errors"
	"log"

	"github.com/maynagashev/gophkeeper/server/internal/models"
	"github.com/maynagashev/gophkeeper/server/internal/repository"
)

// VaultService определяет интерфейс для сервиса работы с хранилищами.
type VaultService interface {
	GetVaultMetadata(userID int64) (*models.Vault, error)
	// TODO: Добавить другие методы сервиса (Upload, Download и т.д.)
}

// vaultService реализует логику работы с хранилищами.
var _ VaultService = (*vaultService)(nil) // Проверка соответствия интерфейсу

type vaultService struct {
	vaultRepo repository.VaultRepository // Зависимость от репозитория хранилищ
}

// NewVaultService создает новый экземпляр сервиса хранилищ.
func NewVaultService(vaultRepo repository.VaultRepository) VaultService {
	return &vaultService{vaultRepo: vaultRepo}
}

// GetVaultMetadata получает метаданные хранилища для указанного пользователя.
func (s *vaultService) GetVaultMetadata(userID int64) (*models.Vault, error) {
	ctx := context.Background()

	vault, err := s.vaultRepo.GetVaultByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrVaultNotFound) {
			log.Printf("[VaultService] Метаданные для пользователя %d не найдены", userID)
			return nil, ErrVaultNotFound // Возвращаем ошибку сервисного слоя
		}
		log.Printf("[VaultService] Ошибка репозитория при получении метаданных для пользователя %d: %v", userID, err)
		return nil, errors.New("внутренняя ошибка сервера при получении метаданных")
	}

	log.Printf("[VaultService] Успешно получены метаданные (ID: %d) для пользователя %d", vault.ID, userID)
	return vault, nil
}

// Кастомные ошибки сервиса.
var (
	ErrVaultNotFound = errors.New("хранилище не найдено")
)
