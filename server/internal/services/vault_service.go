package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/maynagashev/gophkeeper/server/internal/repository"
	"github.com/maynagashev/gophkeeper/server/internal/storage"
)

// VaultService определяет интерфейс для сервиса работы с хранилищами.
type VaultService interface {
	GetVaultMetadata(userID int64) (*models.VaultVersion, error)
	UploadVault(userID int64, reader io.Reader, size int64, contentType string, contentModifiedAt time.Time) error
	DownloadVault(userID int64) (io.ReadCloser, *models.VaultVersion, error)
	ListVersions(userID int64, limit, offset int) ([]models.VaultVersion, error)
	RollbackToVersion(userID int64, versionID int64) error
}

// vaultService реализует логику работы с хранилищами.
var _ VaultService = (*vaultService)(nil)

type vaultService struct {
	db               *sql.DB
	vaultRepo        repository.VaultRepository
	vaultVersionRepo repository.VaultVersionRepository
	fileStorage      storage.FileStorage
}

// NewVaultService создает новый экземпляр сервиса хранилищ.
func NewVaultService(
	db *sql.DB,
	vaultRepo repository.VaultRepository,
	vaultVersionRepo repository.VaultVersionRepository,
	fileStorage storage.FileStorage,
) VaultService {
	return &vaultService{
		db:               db,
		vaultRepo:        vaultRepo,
		vaultVersionRepo: vaultVersionRepo,
		fileStorage:      fileStorage,
	}
}

// GetVaultMetadata получает метаданные ТЕКУЩЕЙ версии хранилища для пользователя.
func (s *vaultService) GetVaultMetadata(userID int64) (*models.VaultVersion, error) {
	ctx := context.Background()

	_, currentVersion, err := s.vaultRepo.GetVaultWithCurrentVersionByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrVaultNotFound) {
			log.Printf("[VaultService] Метаданные (GetVaultWithCurrentVersion) для пользователя %d не найдены", userID)
			return nil, ErrVaultNotFound
		}
		log.Printf("[VaultService] Ошибка репозитория при получении хранилища с версией для пользователя %d: %v", userID, err)
		return nil, errors.New("внутренняя ошибка сервера при получении метаданных")
	}

	if currentVersion == nil {
		// Хранилище есть, но текущей версии нет (например, после неудачного отката)
		log.Printf("[VaultService] У пользователя %d есть хранилище, но нет текущей версии", userID)
		return nil, ErrVaultNotFound // Считаем, что метаданных нет
	}

	log.Printf("[VaultService] Успешно получены метаданные текущей версии (ID: %d)"+
		" для пользователя %d", currentVersion.ID, userID)
	return currentVersion, nil
}

// Добавили contentModifiedAt в параметры.
func (s *vaultService) UploadVault(
	userID int64,
	reader io.Reader,
	size int64,
	contentType string,
	contentModifiedAt time.Time,
) error {
	ctx := context.Background()

	// Используем io.TeeReader, чтобы одновременно считать хеш и передать данные дальше
	hash := sha256.New()
	teeReader := io.TeeReader(reader, hash)

	// Генерируем уникальный ключ объекта для MinIO
	objectKey := fmt.Sprintf("user_%d/vault_%s.kdbx", userID, uuid.New().String())

	// 1. Загружаем файл в MinIO ПЕРЕД транзакцией (для упрощения, можно оптимизировать)
	err := s.fileStorage.UploadFile(ctx, objectKey, teeReader, size, contentType)
	if err != nil {
		log.Printf("[VaultService] Ошибка загрузки файла в хранилище для пользователя %d: %v", userID, err)
		return errors.New("внутренняя ошибка сервера при загрузке файла")
	}
	// Получаем вычисленную чек-сумму клиента
	checksumClient := hex.EncodeToString(hash.Sum(nil))
	log.Printf("[VaultService] Файл для пользователя %d загружен в '%s', SHA256: %s",
		userID, objectKey, checksumClient)

	// --- Транзакция БД --- //
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[VaultService] Ошибка начала транзакции для пользователя %d: %v", userID, err)
		// TODO: Попытаться удалить загруженный файл из MinIO?
		return errors.New("внутренняя ошибка сервера")
	}
	// Гарантируем откат транзакции в случае паники или ошибки
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // Передаем панику дальше
		} else if err != nil {
			log.Printf("[VaultService] Ошибка во время транзакции, откат... Error: %v", err)
			_ = tx.Rollback()
			// TODO: Попытаться удалить загруженный файл из MinIO?
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("[VaultService] Ошибка коммита транзакции: %v", err)
				// TODO: Попытаться удалить загруженный файл из MinIO?
			}
		}
	}()

	// Используем транзакционные репозитории (если они есть) или передаем tx
	// Пока будем передавать tx в существующие методы репозиториев,
	// модифицировав их для приема tx или создав tx-варианты.
	// Сейчас для простоты будем считать, что методы репо могут работать с tx (нужно будет доработать репозитории).

	// 2. Получаем текущее хранилище и его ВЕРСИЮ
	vault, currentVersion, err := s.vaultRepo.GetVaultWithCurrentVersionByUserID(ctx, userID) // TODO: Передать tx
	if err != nil && !errors.Is(err, repository.ErrVaultNotFound) {
		// Неожиданная ошибка при поиске
		log.Printf("[VaultService] Ошибка поиска хранилища/версии для пользователя %d: %v", userID, err)
		return errors.New("внутренняя ошибка сервера") // defer откатит транзакцию
	}

	// 3. Сравнение версий (если текущая версия существует)
	shouldCreateNewVersion := true
	if currentVersion != nil && currentVersion.ContentModifiedAt != nil {
		serverTime := *currentVersion.ContentModifiedAt
		clientTime := contentModifiedAt // Время из аргумента
		checksumServer := ""            // Чек-сумма текущей версии на сервере
		if currentVersion.Checksum != nil {
			checksumServer = *currentVersion.Checksum
		}

		log.Printf("[VaultService] Сравнение версий: Клиент T=%v C=%s | Сервер T=%v C=%s",
			clientTime, checksumClient, serverTime, checksumServer)

		if clientTime.Before(serverTime) {
			// Время клиента < времени сервера -> Конфликт
			log.Printf("[VaultService] Отклонено: время клиента (%v) раньше времени сервера (%v). Конфликт.",
				clientTime, serverTime)
			err = ErrConflictVersion // Устанавливаем ошибку для defer
			return err               // Возвращаем ошибку конфликта
		} else if clientTime.Equal(serverTime) {
			// Время совпадает, проверяем чек-суммы
			if checksumClient == checksumServer {
				// Идентичная версия
				log.Printf("[VaultService] Пропуск: идентичная версия (время и чек-сумма совпадают).")
				return nil
			}
			// Время совпадает, чек-суммы разные -> Конфликт
			log.Printf("[VaultService] Отклонено: время совпадает (%v), но чек-суммы разные. Конфликт.", clientTime)
			err = ErrConflictVersion // Устанавливаем ошибку для defer
			return err               // Возвращаем ошибку конфликта
		}
		// Если clientTime.After(serverTime), то shouldCreateNewVersion остается true
	}

	// Если нужно создать новую версию (т.е. не было конфликта или идентичной версии)
	if shouldCreateNewVersion {
		// 4. Найти или создать Vault (если на шаге 2 он не был найден)
		var vaultID int64
		if vault == nil {
			// Хранилище не найдено, создаем новое
			log.Printf("[VaultService] Хранилище для пользователя %d не найдено, создаем новое.", userID)
			newVault := &models.Vault{UserID: userID}
			vaultID, err = s.vaultRepo.CreateVault(ctx, newVault) // TODO: Передать tx
			if err != nil {
				log.Printf("[VaultService] Ошибка создания хранилища в транзакции для пользователя %d: %v", userID, err)
				return errors.New("внутренняя ошибка сервера") // defer откатит
			}
			log.Printf("[VaultService] Новое хранилище создано (ID: %d) для пользователя %d", vaultID, userID)
		} else {
			vaultID = vault.ID
			log.Printf("[VaultService] Используется существующее хранилище (ID: %d) для пользователя %d", vaultID, userID)
		}

		// 5. Создать запись о новой версии
		newVersion := &models.VaultVersion{
			VaultID:           vaultID,
			ObjectKey:         objectKey,
			Checksum:          &checksumClient, // Используем хеш клиента
			SizeBytes:         &size,
			ContentModifiedAt: &contentModifiedAt,
		}
		var versionID int64
		versionID, err = s.vaultVersionRepo.CreateVersion(ctx, newVersion) // TODO: Передать tx
		if err != nil {
			log.Printf("[VaultService] Ошибка создания версии в транзакции для хранилища %d: %v", vaultID, err)
			return errors.New("внутренняя ошибка сервера") // defer откатит
		}
		log.Printf("[VaultService] Новая версия создана (ID: %d) для хранилища %d", versionID, vaultID)

		// 6. Обновить current_version_id в Vault
		err = s.vaultRepo.UpdateVaultCurrentVersion(ctx, vaultID, versionID) // TODO: Передать tx
		if err != nil {
			log.Printf("[VaultService] Ошибка обновления current_version_id в транзакции для хранилища %d: %v", vaultID, err)
			return errors.New("внутренняя ошибка сервера") // defer откатит
		}
		log.Printf("[VaultService] current_version_id для хранилища %d обновлен на %d", vaultID, versionID)

		log.Printf("[VaultService] Загрузка и обновление метаданных для пользователя %d завершены успешно", userID)
	}

	// Ошибки нет (либо была идентичная версия), defer выполнит Commit
	return nil
}

// DownloadVault скачивает ТЕКУЩУЮ версию файла хранилища.
func (s *vaultService) DownloadVault(userID int64) (io.ReadCloser, *models.VaultVersion, error) {
	ctx := context.Background()

	// Получаем хранилище и текущую версию одним запросом
	_, currentVersion, err := s.vaultRepo.GetVaultWithCurrentVersionByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrVaultNotFound) {
			log.Printf("[VaultService] Запрос на скачивание: хранилище или версия для пользователя %d не найдены", userID)
			return nil, nil, ErrVaultNotFound
		}
		log.Printf("[VaultService] Ошибка получения хранилища/версии для скачивания (пользователь %d): %v", userID, err)
		return nil, nil, errors.New("внутренняя ошибка сервера при получении метаданных")
	}

	if currentVersion == nil {
		log.Printf("[VaultService] Запрос на скачивание: нет текущей активной версии для пользователя %d", userID)
		return nil, nil, ErrVaultNotFound
	}

	// Скачиваем файл из MinIO по ключу текущей версии
	fileReader, err := s.fileStorage.DownloadFile(ctx, currentVersion.ObjectKey)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			log.Printf("[VaultService] Файл '%s' не найден в хранилище"+
				" (пользователь %d, версия %d)", currentVersion.ObjectKey, userID, currentVersion.ID)
			return nil, nil, ErrVaultNotFound
		}
		log.Printf("[VaultService] Ошибка скачивания файла '%s' из хранилища"+
			" (пользователь %d, версия %d): %v", currentVersion.ObjectKey, userID, currentVersion.ID, err)
		return nil, nil, errors.New("внутренняя ошибка сервера при скачивании файла")
	}

	log.Printf("[VaultService] Файл '%s' (версия %d) для пользователя %d"+
		" готов к скачиванию", currentVersion.ObjectKey, currentVersion.ID, userID)
	return fileReader, currentVersion, nil
}

// ListVersions возвращает список версий хранилища пользователя.
func (s *vaultService) ListVersions(userID int64, limit, offset int) ([]models.VaultVersion, error) {
	ctx := context.Background()

	// Сначала находим ID хранилища пользователя
	vault, err := s.vaultRepo.GetVaultByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrVaultNotFound) {
			log.Printf("[VaultService] Запрос списка версий: хранилище для пользователя %d не найдено", userID)
			return []models.VaultVersion{}, nil // Возвращаем пустой слайс, а не ошибку
		}
		log.Printf("[VaultService] Ошибка поиска хранилища для списка версий (пользователь %d): %v", userID, err)
		return nil, errors.New("внутренняя ошибка сервера")
	}

	// Получаем список версий для найденного vaultID
	versions, err := s.vaultVersionRepo.ListVersionsByVaultID(ctx, vault.ID, limit, offset)
	if err != nil {
		log.Printf("[VaultService] Ошибка получения списка версий для хранилища %d"+
			" (пользователь %d): %v", vault.ID, userID, err)
		return nil, errors.New("внутренняя ошибка сервера")
	}

	log.Printf("[VaultService] Возвращено %d версий для пользователя %d", len(versions), userID)
	return versions, nil
}

// RollbackToVersion откатывает хранилище пользователя к указанной версии.
func (s *vaultService) RollbackToVersion(userID int64, versionID int64) error {
	ctx := context.Background()

	// 1. Найти хранилище пользователя
	vault, err := s.vaultRepo.GetVaultByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrVaultNotFound) {
			log.Printf("[VaultService] Попытка отката: хранилище для пользователя %d не найдено", userID)
			return ErrVaultNotFound
		}
		log.Printf("[VaultService] Ошибка поиска хранилища для отката (пользователь %d): %v", userID, err)
		return errors.New("внутренняя ошибка сервера")
	}

	// 2. Проверить, что указанная версия принадлежит этому хранилищу
	version, err := s.vaultVersionRepo.GetVersionByID(ctx, versionID)
	if err != nil {
		if errors.Is(err, repository.ErrVersionNotFound) {
			log.Printf("[VaultService] Попытка отката: версия %d не найдена (пользователь %d)", versionID, userID)
			return ErrVersionNotFound // А возвращаем ошибку сервиса
		}
		log.Printf("[VaultService] Ошибка поиска версии %d для отката (пользователь %d): %v", versionID, userID, err)
		return errors.New("внутренняя ошибка сервера")
	}
	if version.VaultID != vault.ID {
		log.Printf("[VaultService] Попытка отката: версия %d не принадлежит хранилищу %d"+
			" (пользователь %d)", versionID, vault.ID, userID)
		return ErrForbidden // Другая ошибка: попытка доступа к чужой версии
	}

	// 3. Обновить current_version_id в хранилище
	err = s.vaultRepo.UpdateVaultCurrentVersion(ctx, vault.ID, versionID)
	if err != nil {
		// Обрабатываем случай, если хранилище вдруг не нашлось (хотя мы его только что нашли)
		if errors.Is(err, repository.ErrVaultNotFound) {
			log.Printf("[VaultService] Ошибка отката: хранилище %d исчезло во время обновления?"+
				" (пользователь %d)", vault.ID, userID)
			return ErrVaultNotFound
		}
		log.Printf("[VaultService] Ошибка обновления current_version_id при откате"+
			" для хранилища %d (пользователь %d): %v", vault.ID, userID, err)
		return errors.New("внутренняя ошибка сервера при откате")
	}

	log.Printf("[VaultService] Пользователь %d успешно откатил хранилище %d к версии %d", userID, vault.ID, versionID)
	return nil
}

// Кастомные ошибки сервиса.
var (
	ErrVaultNotFound   = errors.New("хранилище или его версия не найдены")
	ErrVersionNotFound = errors.New("указанная версия хранилища не найдена")
	ErrForbidden       = errors.New("доступ запрещен") // Общая ошибка доступа
	ErrConflictVersion = errors.New("конфликт версий")
)
