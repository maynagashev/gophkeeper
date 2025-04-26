package services_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/maynagashev/gophkeeper/server/internal/repository"
	"github.com/maynagashev/gophkeeper/server/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

// MockVaultRepository is a mock for VaultRepository.
type MockVaultRepository struct {
	mock.Mock
}

func (m *MockVaultRepository) GetVaultByUserID(ctx context.Context, userID int64) (*models.Vault, error) {
	args := m.Called(ctx, userID)
	ret := args.Get(0)
	if ret == nil {
		return nil, args.Error(1)
	}
	//nolint:errcheck // Ошибки кастования в моках приемлемы
	return ret.(*models.Vault), args.Error(1)
}

func (m *MockVaultRepository) CreateVault(ctx context.Context, vault *models.Vault) (int64, error) {
	args := m.Called(ctx, vault)
	//nolint:errcheck // Ошибки кастования в моках приемлемы
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockVaultRepository) UpdateVaultCurrentVersion(ctx context.Context, vaultID int64, versionID int64) error {
	args := m.Called(ctx, vaultID, versionID)
	return args.Error(0)
}

func (m *MockVaultRepository) GetVaultWithCurrentVersionByUserID(
	ctx context.Context,
	userID int64,
) (*models.Vault, *models.VaultVersion, error) {
	args := m.Called(ctx, userID)
	retVault := args.Get(0)
	retVersion := args.Get(1)

	var vault *models.Vault
	if retVault != nil {
		//nolint:errcheck // Ошибки кастования в моках приемлемы
		vault = retVault.(*models.Vault)
	}
	var version *models.VaultVersion
	if retVersion != nil {
		//nolint:errcheck // Ошибки кастования в моках приемлемы
		version = retVersion.(*models.VaultVersion)
	}

	return vault, version, args.Error(2)
}

// MockVaultVersionRepository is a mock for VaultVersionRepository.
type MockVaultVersionRepository struct {
	mock.Mock
}

func (m *MockVaultVersionRepository) CreateVersion(ctx context.Context, version *models.VaultVersion) (int64, error) {
	args := m.Called(ctx, version)
	//nolint:errcheck // Ошибки кастования в моках приемлемы
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockVaultVersionRepository) ListVersionsByVaultID(
	ctx context.Context,
	vaultID int64,
	limit,
	offset int,
) ([]models.VaultVersion, error) {
	args := m.Called(ctx, vaultID, limit, offset)
	ret := args.Get(0)
	if ret == nil {
		return nil, args.Error(1)
	}
	//nolint:errcheck // Ошибки кастования в моках приемлемы
	return ret.([]models.VaultVersion), args.Error(1)
}

func (m *MockVaultVersionRepository) GetVersionByID(
	ctx context.Context,
	versionID int64,
) (*models.VaultVersion, error) {
	args := m.Called(ctx, versionID)
	ret := args.Get(0)
	if ret == nil {
		return nil, args.Error(1)
	}
	//nolint:errcheck // Ошибки кастования в моках приемлемы
	return ret.(*models.VaultVersion), args.Error(1)
}

// MockFileStorage is a mock for FileStorage.
type MockFileStorage struct {
	mock.Mock
}

func (m *MockFileStorage) UploadFile(
	ctx context.Context,
	objectKey string,
	reader io.Reader,
	size int64,
	contentType string,
) error {
	// Consume the reader to simulate reading
	_, _ = io.Copy(io.Discard, reader)
	args := m.Called(ctx, objectKey, reader, size, contentType)
	return args.Error(0)
}

func (m *MockFileStorage) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	args := m.Called(ctx, objectKey)
	ret := args.Get(0)
	if ret == nil {
		return nil, args.Error(1)
	}
	//nolint:errcheck // Ошибки кастования в моках приемлемы
	return ret.(io.ReadCloser), args.Error(1)
}

// --- Helper to setup service with mocks ---.
func setupVaultServiceWithMocks() (
	services.VaultService, // Возвращаем интерфейс напрямую
	*MockVaultRepository,
	*MockVaultVersionRepository,
	*MockFileStorage,
	sqlmock.Sqlmock,
) {
	mockVaultRepo := new(MockVaultRepository)
	mockVersionRepo := new(MockVaultVersionRepository)
	mockFileStorage := new(MockFileStorage)

	// Создаем мок DB с помощью sqlmock
	mockDB, mockSQL, err := sqlmock.New()
	if err != nil {
		// В тестах обычно паникуем при ошибке настройки мока
		panic(fmt.Sprintf("Не удалось создать sqlmock: %s", err))
	}
	// Не закрываем mockDB здесь, тесты должны это делать через mockSQL.ExpectationsWereMet()

	vaultService := services.NewVaultService(mockDB, mockVaultRepo, mockVersionRepo, mockFileStorage)
	// Убрано лишнее утверждение типа

	return vaultService, mockVaultRepo, mockVersionRepo, mockFileStorage, mockSQL // Возвращаем интерфейс
}

// --- Tests ---

func TestVaultService_GetVaultMetadata(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	testUserID := int64(1)
	testVaultID := int64(101)
	testVersionID := int64(201)
	// Используем UTC и Truncate для консистентности времени
	testModTime := time.Now().UTC().Truncate(time.Second)
	testS3Key := "vaults/1/version_201"

	tests := []struct {
		name             string
		mockSetup        func(*MockVaultRepository, *MockVaultVersionRepository)
		expectedMetadata *models.VaultVersion
		expectedErr      error
		checkErrorIs     bool // Флаг для использования ErrorIs
	}{
		{
			name: "Успех",
			mockSetup: func(mockVaultRepo *MockVaultRepository, _ *MockVaultVersionRepository) {
				// Создаем копию testModTime для мока, чтобы избежать случайного изменения
				modTimeCopy := testModTime
				mockVaultRepo.On(
					"GetVaultWithCurrentVersionByUserID",
					mock.Anything,
					testUserID,
				).Return(
					&models.Vault{ID: testVaultID, UserID: testUserID, CurrentVersionID: &testVersionID},
					&models.VaultVersion{
						ID:                testVersionID,
						VaultID:           testVaultID,
						ContentModifiedAt: &modTimeCopy, // Используем копию UTC Truncated времени
						ObjectKey:         testS3Key,
					},
					nil,
				).Once()
			},
			expectedMetadata: &models.VaultVersion{
				ID:                testVersionID,
				VaultID:           testVaultID,
				ContentModifiedAt: &testModTime, // Сравниваем с оригиналом UTC Truncated времени
				ObjectKey:         testS3Key,
			},
			expectedErr: nil,
		},
		{
			name: "Хранилище не найдено",
			mockSetup: func(mockVaultRepo *MockVaultRepository, _ *MockVaultVersionRepository) {
				mockVaultRepo.On(
					"GetVaultWithCurrentVersionByUserID",
					mock.Anything,
					testUserID,
				).Return(nil, nil, repository.ErrVaultNotFound).Once()
			},
			expectedMetadata: nil,
			expectedErr:      services.ErrVaultNotFound,
			checkErrorIs:     true,
		},
		{
			name: "Хранилище есть, версии нет",
			mockSetup: func(mockVaultRepo *MockVaultRepository, _ *MockVaultVersionRepository) {
				mockVaultRepo.On(
					"GetVaultWithCurrentVersionByUserID",
					mock.Anything,
					testUserID,
				).Return(&models.Vault{ID: testVaultID, UserID: testUserID, CurrentVersionID: nil}, nil, nil).Once()
			},
			expectedMetadata: nil,
			expectedErr:      services.ErrVaultNotFound, // Сервис должен вернуть эту ошибку
			checkErrorIs:     true,
		},
		{
			name: "Ошибка репозитория",
			mockSetup: func(mockVaultRepo *MockVaultRepository, _ *MockVaultVersionRepository) {
				mockVaultRepo.On(
					"GetVaultWithCurrentVersionByUserID",
					mock.Anything,
					testUserID,
				).Return(nil, nil, errors.New("db error")).Once()
			},
			expectedMetadata: nil,
			expectedErr:      errors.New("внутренняя ошибка сервера при получении метаданных"), // Ошибка из сервиса
			checkErrorIs:     false,                                                            // Используем Contains для этой общей ошибки
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Получаем моки, игнорируем mockSQL и mockFileStorage
			service, mockVaultRepo, mockVersionRepo, _, _ := setupVaultServiceWithMocks()

			tt.mockSetup(mockVaultRepo, mockVersionRepo)

			metadata, err := service.GetVaultMetadata(testUserID)

			if tt.expectedErr != nil {
				require.Error(err, t) // ПОРЯДОК АРГУМЕНТОВ: err, t
				if tt.checkErrorIs {
					// Теперь используем ErrorIs, так как ожидаем конкретную ошибку
					require.ErrorIs(err, tt.expectedErr, t) // ПОРЯДОК АРГУМЕНТОВ: err, tt.expectedErr, t
				} else {
					// Проверяем текст ошибки для общих случаев
					assert.Contains(err.Error(), tt.expectedErr.Error(), t)
				}
			} else {
				require.NoError(err, t) // ПОРЯДОК АРГУМЕНТОВ: err, t
				require.NotNil(metadata, t, "Метаданные не должны быть nil при успехе")
				// Сравниваем указатели на время, предварительно убедившись, что они не nil
				require.NotNil(metadata.ContentModifiedAt, t, "metadata.ContentModifiedAt is nil")
				require.NotNil(tt.expectedMetadata.ContentModifiedAt, t, "expectedMetadata.ContentModifiedAt is nil")
				// assert.True уже был исправлен ранее
				assert.True(tt.expectedMetadata.ContentModifiedAt.Equal(*metadata.ContentModifiedAt), t, "Times are not equal")
				// Сравниваем остальные поля структуры, игнорируя время
				expectedCopy := *tt.expectedMetadata
				metadataCopy := *metadata
				expectedCopy.ContentModifiedAt = nil
				metadataCopy.ContentModifiedAt = nil
				assert.Equal(expectedCopy, metadataCopy, t, "Metadata structs (excluding time) are not equal")
			}

			mockVaultRepo.AssertExpectations(t)
			mockVersionRepo.AssertExpectations(t)
		})
	}
}

func TestVaultService_UploadVault(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	testUserID := int64(1)
	testVaultID := int64(10)
	testVersionID := int64(101)
	// Используем UTC и Truncate для консистентности времени
	testModTime := time.Now().UTC().Truncate(time.Second)
	testModTimeOlder := testModTime.Add(-time.Hour) // Время старой версии на сервере
	testSize := int64(1234)
	testContentType := "application/octet-stream"
	// fakeChecksum больше не используется напрямую в вызове UploadVault, но может быть полезен для сравнения в моках
	// _ = fakeChecksum // Используем _ чтобы подавить ошибку "declared and not used"

	// Ожидаемый аргумент для CreateVersion (теперь с проверкой Checksum, вычисленным сервисом)
	// Мы не знаем точный checksum в тесте, поэтому используем mock.AnythingOfType или более сложный Matcher
	expectedVersionMatcher := mock.MatchedBy(func(v *models.VaultVersion) bool {
		return v.VaultID == testVaultID &&
			v.Checksum != nil && // Проверяем, что чексумма не nil
			v.SizeBytes != nil && *v.SizeBytes == testSize &&
			v.ContentModifiedAt != nil && v.ContentModifiedAt.Equal(testModTime) &&
			strings.HasPrefix(v.ObjectKey, fmt.Sprintf("user_%d/vault_", testUserID))
	})

	tests := []struct {
		name          string
		clientModTime time.Time // Время модификации, передаваемое клиентом
		mockSetup     func(
			mockVaultRepo *MockVaultRepository,
			mockVersionRepo *MockVaultVersionRepository,
			mockFileStorage *MockFileStorage,
			mockSQL sqlmock.Sqlmock,
		)
		expectedErr      error
		checkErrorIs     bool
		expectedConflict bool // Ожидается ли ошибка конфликта
	}{
		{
			name:          "Успех - Новое хранилище",
			clientModTime: testModTime,
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				mockVersionRepo *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				// Мок GetVaultWithCurrentVersionByUserID должен быть ПЕРВЫМ
				mockVaultRepo.On("GetVaultWithCurrentVersionByUserID", mock.Anything, testUserID).Return(nil, nil, repository.ErrVaultNotFound).Once()

				// Загрузка файла происходит ПОСЛЕ проверки конфликта
				mockFileStorage.On(
					"UploadFile",
					mock.Anything,                 // ctx
					mock.AnythingOfType("string"), // Object key
					mock.Anything,                 // Reader (TeeReader)
					testSize,
					testContentType,
				).Return(nil).Once()

				// Транзакция начинается ПОСЛЕ загрузки файла
				mockSQL.ExpectBegin()
				// Внутри транзакции: GetVaultByUserID (снова), CreateVault, CreateVersion, UpdateVaultCurrentVersion
				mockVaultRepo.On("GetVaultByUserID", mock.Anything, testUserID).Return(nil, repository.ErrVaultNotFound).Once() // Все еще не найден внутри транзакции
				mockVaultRepo.On("CreateVault", mock.Anything, mock.AnythingOfType("*models.Vault")).Return(testVaultID, nil).Once()
				mockVersionRepo.On("CreateVersion", mock.Anything, expectedVersionMatcher).Return(testVersionID, nil).Once()
				mockVaultRepo.On("UpdateVaultCurrentVersion", mock.Anything, testVaultID, testVersionID).Return(nil).Once()
				mockSQL.ExpectCommit()
			},
			expectedErr: nil,
		},
		{
			name:          "Успех - Существующее хранилище (новее клиента)",
			clientModTime: testModTime, // Время клиента новее, чем mockExistingVersion
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				mockVersionRepo *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				// Мок для проверки конфликта ПЕРЕД транзакцией
				mockExistingVault := &models.Vault{ID: testVaultID, UserID: testUserID}
				modTimeOlderCopy := testModTimeOlder // Используем старое время для существующей версии
				serverChecksum := "server_checksum"  // Чексумма на сервере (важно, чтобы она отличалась от вычисленной)
				mockExistingVersion := &models.VaultVersion{ContentModifiedAt: &modTimeOlderCopy, Checksum: &serverChecksum}
				mockVaultRepo.On("GetVaultWithCurrentVersionByUserID", mock.Anything, testUserID).Return(mockExistingVault, mockExistingVersion, nil).Once()

				// Загрузка файла
				mockFileStorage.On("UploadFile", mock.Anything, mock.AnythingOfType("string"), mock.Anything, testSize, testContentType).Return(nil).Once()

				// Транзакция
				mockSQL.ExpectBegin()
				mockVaultRepo.On("GetVaultByUserID", mock.Anything, testUserID).Return(mockExistingVault, nil).Once() // Находим существующее хранилище
				mockVersionRepo.On("CreateVersion", mock.Anything, expectedVersionMatcher).Return(testVersionID, nil).Once()
				mockVaultRepo.On("UpdateVaultCurrentVersion", mock.Anything, testVaultID, testVersionID).Return(nil).Once()
				mockSQL.ExpectCommit()
			},
			expectedErr: nil,
		},
		{
			name:          "Конфликт - Существующее хранилище (старее клиента по времени)",
			clientModTime: testModTimeOlder, // Время клиента СТАРШЕ версии на сервере
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				_ *MockVaultVersionRepository,
				_ *MockFileStorage,
				_ sqlmock.Sqlmock,
			) {
				// Мок для проверки конфликта ПЕРЕД транзакцией
				mockExistingVault := &models.Vault{ID: testVaultID, UserID: testUserID}
				modTimeServer := testModTime // Версия на сервере новее клиента
				serverChecksum := "server_checksum"
				mockExistingVersion := &models.VaultVersion{ContentModifiedAt: &modTimeServer, Checksum: &serverChecksum}
				mockVaultRepo.On("GetVaultWithCurrentVersionByUserID", mock.Anything, testUserID).Return(mockExistingVault, mockExistingVersion, nil).Once()
				// UploadFile и транзакция НЕ должны вызываться
			},
			expectedErr:      services.ErrConflictVersion, // Исправлено: ожидаем конфликт
			checkErrorIs:     true,
			expectedConflict: true,
		},
		{
			name:          "Конфликт - Существующее хранилище (время совпадает, checksum разный)",
			clientModTime: testModTime, // Время клиента совпадает
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				_ *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage, // Нужен для Upload, чтобы получить checksum
				_ sqlmock.Sqlmock,
			) {
				// Мок для GetVaultWithCurrentVersionByUserID
				mockExistingVault := &models.Vault{ID: testVaultID, UserID: testUserID}
				modTimeServer := testModTime // Время совпадает
				// _ = modTimeServer // Используем _ чтобы подавить ошибку
				serverChecksum := "different_server_checksum" // Чексумма отличается
				mockExistingVersion := &models.VaultVersion{ContentModifiedAt: &modTimeServer, Checksum: &serverChecksum}
				mockVaultRepo.On("GetVaultWithCurrentVersionByUserID", mock.Anything, testUserID).Return(mockExistingVault, mockExistingVersion, nil).Once()

				// Mock UploadFile, так как проверка checksum происходит после загрузки
				// Этот мок нужен, чтобы вычислить "клиентский" checksum внутри сервиса
				mockFileStorage.On("UploadFile", mock.Anything, mock.AnythingOfType("string"), mock.Anything, testSize, testContentType).Return(nil).Once()
				// Транзакция не должна начаться, так как будет конфликт по чексуммам
			},
			expectedErr:      services.ErrConflictVersion,
			checkErrorIs:     true,
			expectedConflict: true,
		},
		{
			name:          "Успех - Идентичная версия (пропуск)",
			clientModTime: testModTime, // Время клиента совпадает
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				_ *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage, // Нужен для Upload
				_ sqlmock.Sqlmock,
			) {
				mockExistingVault := &models.Vault{ID: testVaultID, UserID: testUserID}
				modTimeServer := testModTime // Время совпадает
				// Вычисляем ожидаемый checksum, который вернет TeeReader в сервисе
				h := sha256.New()
				_, _ = io.WriteString(h, "test data") // Данные из currentReader ниже
				expectedChecksum := hex.EncodeToString(h.Sum(nil))
				serverChecksum := expectedChecksum // Чексумма совпадает

				mockExistingVersion := &models.VaultVersion{ContentModifiedAt: &modTimeServer, Checksum: &serverChecksum}
				mockVaultRepo.On("GetVaultWithCurrentVersionByUserID", mock.Anything, testUserID).Return(mockExistingVault, mockExistingVersion, nil).Once()

				// Mock UploadFile - он вызовется, но транзакции не будет
				mockFileStorage.On("UploadFile", mock.Anything, mock.AnythingOfType("string"), mock.Anything, testSize, testContentType).Return(nil).Once()
				// Транзакция не должна начаться
			},
			expectedErr: nil, // Ошибки нет, просто пропускаем
		},
		{
			name:          "Ошибка - Загрузка в FileStorage",
			clientModTime: testModTime,
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				_ *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage,
				_ sqlmock.Sqlmock,
			) {
				// Мок для проверки конфликта (возвращает не найдено, чтобы пройти проверку времени)
				mockVaultRepo.On("GetVaultWithCurrentVersionByUserID", mock.Anything, testUserID).Return(nil, nil, repository.ErrVaultNotFound).Once()
				// Ошибка при загрузке файла
				mockFileStorage.On("UploadFile", mock.Anything, mock.AnythingOfType("string"), mock.Anything, testSize, testContentType).Return(errors.New("storage error")).Once()
				// Транзакция не должна начаться
			},
			expectedErr:  errors.New("внутренняя ошибка сервера при загрузке файла"),
			checkErrorIs: false,
		},
		// TODO: Добавить тесты на ошибки внутри транзакции (CreateVault, CreateVersion, UpdateVaultCurrentVersion)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Используем ReadSeeker для возможности повторного чтения, если потребуется
			// strings.Reader реализует ReadSeeker
			currentReader := strings.NewReader("test data")
			service, mockVaultRepo, mockVersionRepo, mockFileStorage, mockSQL := setupVaultServiceWithMocks()

			tt.mockSetup(mockVaultRepo, mockVersionRepo, mockFileStorage, mockSQL)

			// Вызываем метод сервиса БЕЗ checksum (он вычисляется внутри)
			err := service.UploadVault(testUserID, currentReader, testSize, testContentType, tt.clientModTime)

			// Проверяем результат
			if tt.expectedErr != nil {
				require.Error(err, t) // ПОРЯДОК АРГУМЕНТОВ: err, t
				if tt.checkErrorIs {
					require.ErrorIs(err, tt.expectedErr, t) // ПОРЯДОК АРГУМЕНТОВ: err, tt.expectedErr, t
				} else {
					assert.Contains(err.Error(), tt.expectedErr.Error(), t)
				}
			} else {
				require.NoError(err, t) // ПОРЯДОК АРГУМЕНТОВ: err, t
			}

			// Проверяем вызовы репозиториев и хранилища
			mockVaultRepo.AssertExpectations(t)
			mockVersionRepo.AssertExpectations(t)
			mockFileStorage.AssertExpectations(t)
			// Проверяем ожидания SQL только если не ожидалось ошибки конфликта или ошибки загрузки файла
			// или если ошибка nil (успешный случай или пропуск)
			if !tt.expectedConflict && (tt.expectedErr == nil || !strings.Contains(tt.expectedErr.Error(), "загрузке файла")) {
				require.NoError(mockSQL.ExpectationsWereMet(), t, "Ожидания sqlmock не выполнены") // ПОРЯДОК АРГУМЕНТОВ: err, t
			}
		})
	}
}

// Add tests for ListVersions, DownloadVault, RollbackToVersion as needed
