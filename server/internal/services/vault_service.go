package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/maynagashev/gophkeeper/server/internal/models"
	"github.com/maynagashev/gophkeeper/server/internal/repository"
	"github.com/maynagashev/gophkeeper/server/internal/storage"
)

// VaultService определяет интерфейс для сервиса работы с хранилищами.
type VaultService interface {
	GetVaultMetadata(userID int64) (*models.Vault, error)
	UploadVault(userID int64, reader io.Reader, size int64, contentType string) error
	DownloadVault(userID int64) (io.ReadCloser, *models.Vault, error)
}

// vaultService реализует логику работы с хранилищами.
var _ VaultService = (*vaultService)(nil)

type vaultService struct {
	vaultRepo   repository.VaultRepository // Зависимость от репозитория хранилищ
	fileStorage storage.FileStorage        // Зависимость от файлового хранилища (MinIO)
}

// NewVaultService создает новый экземпляр сервиса хранилищ.
func NewVaultService(vaultRepo repository.VaultRepository, fileStorage storage.FileStorage) VaultService {
	return &vaultService{vaultRepo: vaultRepo, fileStorage: fileStorage}
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

// UploadVault обрабатывает загрузку файла хранилища.
func (s *vaultService) UploadVault(userID int64, reader io.Reader, size int64, contentType string) error {
	ctx := context.Background()

	// Генерируем ключ объекта для MinIO (например, user_123/vault.kdbx)
	objectKey := fmt.Sprintf("user_%d/vault.kdbx", userID)

	// Создаем TeeReader для одновременной загрузки в MinIO и расчета хеша
	hash := sha256.New()
	teeReader := io.TeeReader(reader, hash)

	// Загружаем файл в MinIO
	err := s.fileStorage.UploadFile(ctx, objectKey, teeReader, size, contentType)
	if err != nil {
		log.Printf("[VaultService] Ошибка загрузки файла в хранилище для пользователя %d: %v", userID, err)
		return errors.New("внутренняя ошибка сервера при загрузке файла")
	}

	// Получаем хеш загруженного файла
	checksum := hex.EncodeToString(hash.Sum(nil))
	log.Printf("[VaultService] Файл для пользователя %d загружен, SHA256: %s", userID, checksum)

	// Проверяем, существует ли уже запись в БД
	existingVault, err := s.vaultRepo.GetVaultByUserID(ctx, userID)
	if err != nil && !errors.Is(err, repository.ErrVaultNotFound) {
		// Ошибка, не связанная с отсутствием записи
		log.Printf("[VaultService] Ошибка проверки существующих метаданных для пользователя %d: %v", userID, err)
		return errors.New("внутренняя ошибка сервера при проверке метаданных")
	}

	vaultData := &models.Vault{
		UserID:    userID,
		ObjectKey: objectKey,
		Checksum:  &checksum,
		SizeBytes: &size,
	}

	if existingVault == nil {
		// Создаем новую запись в БД
		log.Printf("[VaultService] Создание новой записи метаданных для пользователя %d", userID)
		_, err = s.vaultRepo.CreateVault(ctx, vaultData)
		if err != nil {
			log.Printf("[VaultService] Ошибка создания метаданных для пользователя %d: %v", userID, err)
			return errors.New("внутренняя ошибка сервера при создании метаданных")
		}
	} else {
		// Обновляем существующую запись в БД
		log.Printf("[VaultService] Обновление записи метаданных для пользователя %d", userID)
		err = s.vaultRepo.UpdateVaultMetadata(ctx, vaultData)
		if err != nil {
			log.Printf("[VaultService] Ошибка обновления метаданных для пользователя %d: %v", userID, err)
			return errors.New("внутренняя ошибка сервера при обновлении метаданных")
		}
	}

	log.Printf("[VaultService] Загрузка и обновление метаданных для пользователя %d завершены успешно", userID)
	return nil
}

// DownloadVault скачивает файл хранилища.
// Возвращает io.ReadCloser (который нужно закрыть!), метаданные и ошибку.
func (s *vaultService) DownloadVault(userID int64) (io.ReadCloser, *models.Vault, error) {
	ctx := context.Background()

	// Получаем метаданные из БД
	vault, err := s.vaultRepo.GetVaultByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrVaultNotFound) {
			log.Printf("[VaultService] Запрос на скачивание: метаданные для пользователя %d не найдены", userID)
			return nil, nil, ErrVaultNotFound
		}
		log.Printf("[VaultService] Ошибка получения метаданных для скачивания (пользователь %d): %v", userID, err)
		return nil, nil, errors.New("внутренняя ошибка сервера при получении метаданных")
	}

	// Скачиваем файл из MinIO по ключу из метаданных
	fileReader, err := s.fileStorage.DownloadFile(ctx, vault.ObjectKey)
	if err != nil {
		// Обрабатываем ошибку, если объект не найден в хранилище
		if errors.Is(err, storage.ErrObjectNotFound) {
			log.Printf("[VaultService] Файл '%s' не найден в хранилище (пользователь %d)", vault.ObjectKey, userID)
			// Возможно, стоит удалить запись из БД или пометить как невалидную?
			return nil, nil, ErrVaultNotFound // Возвращаем ту же ошибку для клиента
		}
		log.Printf("[VaultService] Ошибка скачивания файла '%s'"+
			" из хранилища (пользователь %d): %v", vault.ObjectKey, userID, err)
		return nil, nil, errors.New("внутренняя ошибка сервера при скачивании файла")
	}

	log.Printf("[VaultService] Файл '%s' для пользователя %d готов к скачиванию", vault.ObjectKey, userID)
	return fileReader, vault, nil
}

// Кастомные ошибки сервиса.
var (
	ErrVaultNotFound = errors.New("хранилище не найдено")
)
