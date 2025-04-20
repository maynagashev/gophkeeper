package services_test

import (
	"context"
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
	testModTime := time.Now().Truncate(time.Second)
	testS3Key := "vaults/1/version_201"

	tests := []struct {
		name             string
		mockSetup        func(*MockVaultRepository, *MockVaultVersionRepository)
		expectedMetadata *models.VaultVersion
		expectedErr      error
	}{
		{
			name: "Успех",
			mockSetup: func(mockVaultRepo *MockVaultRepository, _ *MockVaultVersionRepository) {
				mockVaultRepo.On(
					"GetVaultWithCurrentVersionByUserID",
					mock.Anything,
					testUserID,
				).Return(
					&models.Vault{ID: testVaultID, UserID: testUserID, CurrentVersionID: &testVersionID},
					&models.VaultVersion{
						ID:                testVersionID,
						VaultID:           testVaultID,
						ContentModifiedAt: &testModTime,
						ObjectKey:         testS3Key,
					},
					nil,
				).Once()
			},
			expectedMetadata: &models.VaultVersion{
				ID:                testVersionID,
				VaultID:           testVaultID,
				ContentModifiedAt: &testModTime,
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
				).Return(
					nil,
					nil,
					repository.ErrVaultNotFound,
				).Once()
			},
			expectedMetadata: nil,
			expectedErr:      services.ErrVaultNotFound,
		},
		{
			name: "Хранилище есть, версии нет",
			mockSetup: func(mockVaultRepo *MockVaultRepository, _ *MockVaultVersionRepository) {
				mockVaultRepo.On(
					"GetVaultWithCurrentVersionByUserID",
					mock.Anything,
					testUserID,
				).Return(
					&models.Vault{ID: testVaultID, UserID: testUserID, CurrentVersionID: nil},
					nil,
					nil,
				).Once()
			},
			expectedMetadata: nil,
			expectedErr:      services.ErrVaultNotFound,
		},
		{
			name: "Ошибка репозитория",
			mockSetup: func(mockVaultRepo *MockVaultRepository, _ *MockVaultVersionRepository) {
				mockVaultRepo.On(
					"GetVaultWithCurrentVersionByUserID",
					mock.Anything,
					testUserID,
				).Return(
					nil,
					nil,
					errors.New("db error"),
				).Once()
			},
			expectedMetadata: nil,
			expectedErr:      errors.New("внутренняя ошибка сервера"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Получаем моки, игнорируем mockSQL и mockFileStorage
			service, mockVaultRepo, mockVersionRepo, _, _ := setupVaultServiceWithMocks()

			tt.mockSetup(mockVaultRepo, mockVersionRepo)

			metadata, err := service.GetVaultMetadata(testUserID)

			if tt.expectedErr != nil {
				require.Error(err)
				require.ErrorIs(err, tt.expectedErr)
			} else {
				require.NoError(err)
				assert.Equal(tt.expectedMetadata, metadata)
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
	// Используем UTC для времени, чтобы избежать проблем с часовыми поясами
	testModTime := time.Now().UTC().Truncate(time.Second)
	testSize := int64(1234)
	testContentType := "application/octet-stream"

	// Ожидаемый аргумент для CreateVersion
	expectedVersionMatcher := mock.MatchedBy(func(v *models.VaultVersion) bool {
		return v.VaultID == testVaultID &&
			v.Checksum != nil && // Checksum генерируется в сервисе, проверяем наличие
			v.SizeBytes != nil && *v.SizeBytes == testSize &&
			v.ContentModifiedAt != nil && v.ContentModifiedAt.Equal(testModTime) &&
			strings.HasPrefix(v.ObjectKey, fmt.Sprintf("user_%d/vault_", testUserID)) // Проверяем префикс ключа
	})

	tests := []struct {
		name      string
		mockSetup func(
			mockVaultRepo *MockVaultRepository,
			mockVersionRepo *MockVaultVersionRepository,
			mockFileStorage *MockFileStorage,
			mockSQL sqlmock.Sqlmock,
		)
		expectedErr error
	}{
		{
			name: "Успех - Новое хранилище",
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				mockVersionRepo *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				// Ожидания для транзакции
				mockSQL.ExpectBegin()
				// Настраиваем моки репозиториев и хранилища
				mockVaultRepo.On("GetVaultByUserID", mock.Anything, testUserID).Return(nil, repository.ErrVaultNotFound).Once()
				mockVaultRepo.On("CreateVault", mock.Anything, mock.AnythingOfType("*models.Vault")).Return(testVaultID, nil).Once()
				mockFileStorage.On(
					"UploadFile",
					mock.Anything,
					mock.AnythingOfType("string"),
					mock.Anything,
					testSize,
					testContentType,
				).Return(nil).Once()
				mockVersionRepo.On("CreateVersion", mock.Anything, expectedVersionMatcher).Return(testVersionID, nil).Once()
				mockVaultRepo.On("UpdateVaultCurrentVersion", mock.Anything, testVaultID, testVersionID).Return(nil).Once()
				// Ожидаем коммит
				mockSQL.ExpectCommit()
			},
			expectedErr: nil,
		},
		{
			name: "Успех - Существующее хранилище",
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				mockVersionRepo *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				mockSQL.ExpectBegin()
				mockExistingVault := &models.Vault{ID: testVaultID, UserID: testUserID}
				mockVaultRepo.On("GetVaultByUserID", mock.Anything, testUserID).Return(mockExistingVault, nil).Once()
				mockFileStorage.On(
					"UploadFile",
					mock.Anything,
					mock.AnythingOfType("string"),
					mock.Anything,
					testSize,
					testContentType,
				).Return(nil).Once()
				mockVersionRepo.On("CreateVersion", mock.Anything, expectedVersionMatcher).Return(testVersionID, nil).Once()
				mockVaultRepo.On("UpdateVaultCurrentVersion", mock.Anything, testVaultID, testVersionID).Return(nil).Once()
				mockSQL.ExpectCommit()
			},
			expectedErr: nil,
		},
		{
			name: "Ошибка - Загрузка в FileStorage",
			mockSetup: func(
				_ *MockVaultRepository,
				_ *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage,
				_ sqlmock.Sqlmock, // mockSQL не используется здесь, т.к. до транзакции ошибка
			) {
				mockFileStorage.On(
					"UploadFile",
					mock.Anything,
					mock.AnythingOfType("string"),
					mock.Anything,
					testSize,
					testContentType,
				).Return(errors.New("storage error")).Once()
				// Транзакция не должна начаться
			},
			expectedErr: errors.New("внутренняя ошибка сервера при загрузке файла"),
		},
		{
			name: "Ошибка - Начало транзакции", // Новый кейс для проверки отката
			mockSetup: func(
				_ *MockVaultRepository,
				_ *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				// Загрузка успешна
				mockFileStorage.On(
					"UploadFile",
					mock.Anything,
					mock.AnythingOfType("string"),
					mock.Anything,
					testSize,
					testContentType,
				).Return(nil).Once()
				// Ошибка при начале транзакции
				mockSQL.ExpectBegin().WillReturnError(errors.New("begin error"))
				// Commit/Rollback не ожидаются
			},
			expectedErr: errors.New("внутренняя ошибка сервера"),
		},
		{
			name: "Ошибка - Поиск хранилища (не NotFound)",
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				_ *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				mockSQL.ExpectBegin()
				mockFileStorage.On(
					"UploadFile",
					mock.Anything,
					mock.AnythingOfType("string"),
					mock.Anything,
					testSize,
					testContentType,
				).Return(nil).Once()
				mockVaultRepo.On("GetVaultByUserID", mock.Anything, testUserID).Return(nil, errors.New("db error")).Once()
				mockSQL.ExpectRollback() // Ожидаем откат
			},
			expectedErr: errors.New("внутренняя ошибка сервера"),
		},
		{
			name: "Ошибка - Создание хранилища",
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				_ *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				mockSQL.ExpectBegin()
				mockFileStorage.On(
					"UploadFile",
					mock.Anything,
					mock.AnythingOfType("string"),
					mock.Anything,
					testSize,
					testContentType,
				).Return(nil).Once()
				mockVaultRepo.On("GetVaultByUserID", mock.Anything, testUserID).Return(nil, repository.ErrVaultNotFound).Once()
				mockVaultRepo.On(
					"CreateVault",
					mock.Anything,
					mock.AnythingOfType("*models.Vault"),
				).Return(int64(0), errors.New("create vault error")).Once()
				mockSQL.ExpectRollback() // Ожидаем откат
			},
			expectedErr: errors.New("внутренняя ошибка сервера"),
		},
		{
			name: "Ошибка - Создание версии",
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				mockVersionRepo *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				mockSQL.ExpectBegin()
				mockFileStorage.On(
					"UploadFile",
					mock.Anything,
					mock.AnythingOfType("string"),
					mock.Anything,
					testSize,
					testContentType,
				).Return(nil).Once()
				mockVaultRepo.On("GetVaultByUserID", mock.Anything, testUserID).Return(&models.Vault{ID: testVaultID}, nil).Once()
				mockVersionRepo.On(
					"CreateVersion",
					mock.Anything,
					expectedVersionMatcher,
				).Return(int64(0), errors.New("create version error")).Once()
				mockSQL.ExpectRollback() // Ожидаем откат
			},
			expectedErr: errors.New("внутренняя ошибка сервера"),
		},
		{
			name: "Ошибка - Обновление текущей версии",
			mockSetup: func(
				mockVaultRepo *MockVaultRepository,
				mockVersionRepo *MockVaultVersionRepository,
				mockFileStorage *MockFileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				mockSQL.ExpectBegin()
				mockFileStorage.On(
					"UploadFile",
					mock.Anything,
					mock.AnythingOfType("string"),
					mock.Anything,
					testSize,
					testContentType,
				).Return(nil).Once()
				mockVaultRepo.On("GetVaultByUserID", mock.Anything, testUserID).Return(&models.Vault{ID: testVaultID}, nil).Once()
				mockVersionRepo.On("CreateVersion", mock.Anything, expectedVersionMatcher).Return(testVersionID, nil).Once()
				mockVaultRepo.On(
					"UpdateVaultCurrentVersion",
					mock.Anything,
					testVaultID,
					testVersionID,
				).Return(errors.New("update error")).Once()
				mockSQL.ExpectRollback() // Ожидаем откат
			},
			expectedErr: errors.New("внутренняя ошибка сервера"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentReader := strings.NewReader("test data")
			// Получаем mockSQL из setup
			service, mockVaultRepo, mockVersionRepo, mockFileStorage, mockSQL := setupVaultServiceWithMocks()

			// Настраиваем моки, передавая mockSQL
			tt.mockSetup(mockVaultRepo, mockVersionRepo, mockFileStorage, mockSQL)

			// Вызываем метод сервиса
			err := service.UploadVault(testUserID, currentReader, testSize, testContentType, testModTime)

			// Проверяем результат
			if tt.expectedErr != nil {
				require.Error(err)
				assert.Contains(err.Error(), tt.expectedErr.Error())
			} else {
				require.NoError(err)
			}

			// Проверяем вызовы репозиториев и хранилища
			mockVaultRepo.AssertExpectations(t)
			mockVersionRepo.AssertExpectations(t)
			mockFileStorage.AssertExpectations(t)
			// Проверяем ожидания mockSQL
			require.NoError(mockSQL.ExpectationsWereMet(), "Ожидания sqlmock не выполнены")
		})
	}
}

// Add tests for ListVersions, DownloadVault, RollbackToVersion as needed
