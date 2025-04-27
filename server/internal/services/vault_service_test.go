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
	"github.com/maynagashev/gophkeeper/server/internal/mocks"
	"github.com/maynagashev/gophkeeper/server/internal/repository"
	"github.com/maynagashev/gophkeeper/server/internal/services"
	"github.com/maynagashev/gophkeeper/server/internal/storage"
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
	services.VaultService,
	*mocks.VaultRepository,
	*mocks.VaultVersionRepository,
	*mocks.FileStorage,
	sqlmock.Sqlmock,
) {
	mockVaultRepo := new(mocks.VaultRepository)
	mockVersionRepo := new(mocks.VaultVersionRepository)
	mockFileStorage := new(mocks.FileStorage)

	mockDB, mockSQL, err := sqlmock.New()
	if err != nil {
		panic(fmt.Sprintf("Не удалось создать sqlmock: %s", err))
	}

	vaultService := services.NewVaultService(mockDB, mockVaultRepo, mockVersionRepo, mockFileStorage)

	return vaultService, mockVaultRepo, mockVersionRepo, mockFileStorage, mockSQL
}

// --- Tests ---

func TestVaultService_GetVaultMetadata(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	testUserID := int64(1)
	testVaultID := int64(101)
	testVersionID := int64(201)
	testModTime := time.Now().UTC().Truncate(time.Second)
	testS3Key := "vaults/1/version_201"

	tests := []struct {
		name             string
		mockSetup        func(*mocks.VaultRepository, *mocks.VaultVersionRepository)
		expectedMetadata *models.VaultVersion
		expectedErr      error
		checkErrorIs     bool
	}{
		{
			name: "Успех",
			mockSetup: func(mockVaultRepo *mocks.VaultRepository, _ *mocks.VaultVersionRepository) {
				modTimeCopy := testModTime
				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(
						&models.Vault{ID: testVaultID, UserID: testUserID, CurrentVersionID: &testVersionID},
						&models.VaultVersion{
							ID:                testVersionID,
							VaultID:           testVaultID,
							ContentModifiedAt: &modTimeCopy,
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
			mockSetup: func(mockVaultRepo *mocks.VaultRepository, _ *mocks.VaultVersionRepository) {
				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(nil, nil, repository.ErrVaultNotFound).Once()
			},
			expectedMetadata: nil,
			expectedErr:      services.ErrVaultNotFound,
			checkErrorIs:     true,
		},
		{
			name: "Хранилище есть, версии нет",
			mockSetup: func(mockVaultRepo *mocks.VaultRepository, _ *mocks.VaultVersionRepository) {
				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID, CurrentVersionID: nil}, nil, nil).Once()
			},
			expectedMetadata: nil,
			expectedErr:      services.ErrVaultNotFound,
			checkErrorIs:     true,
		},
		{
			name: "Ошибка репозитория",
			mockSetup: func(mockVaultRepo *mocks.VaultRepository, _ *mocks.VaultVersionRepository) {
				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(nil, nil, errors.New("db error")).Once()
			},
			expectedMetadata: nil,
			expectedErr:      errors.New("внутренняя ошибка сервера при получении метаданных"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultService, mockVaultRepo, mockVersionRepo, _, mockSQL := setupVaultServiceWithMocks()
			tt.mockSetup(mockVaultRepo, mockVersionRepo)

			metadata, err := vaultService.GetVaultMetadata(testUserID)

			// Проверяем ошибку, если она ожидается
			if tt.expectedErr != nil {
				require.Error(err)
				if tt.checkErrorIs {
					require.ErrorIs(err, tt.expectedErr)
				} else {
					require.EqualError(err, tt.expectedErr.Error())
				}
				assert.Nil(metadata)
				require.NoError(mockSQL.ExpectationsWereMet(), "Ожидания sqlmock не выполнены")
				return
			}

			// Если ошибки нет
			require.NoError(err)
			require.NotNil(metadata)

			// Сравниваем указатели на время
			if tt.expectedMetadata.ContentModifiedAt != nil && metadata.ContentModifiedAt != nil {
				assert.True(
					tt.expectedMetadata.ContentModifiedAt.Equal(*metadata.ContentModifiedAt),
					"Время модификации не совпадает",
				)
				// Обнуляем время перед сравнением остальной структуры
				tt.expectedMetadata.ContentModifiedAt = nil
				metadata.ContentModifiedAt = nil
			} else {
				assert.Equal(
					tt.expectedMetadata.ContentModifiedAt,
					metadata.ContentModifiedAt,
					"Одно из времен модификации nil",
				)
			}
			assert.Equal(tt.expectedMetadata, metadata)

			// Проверяем, что все ожидания моков были выполнены
			require.NoError(mockSQL.ExpectationsWereMet(), "Ожидания sqlmock не выполнены")
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
	testData := "test data"

	// Вычисляем чексумму для тестовых данных
	h := sha256.New()
	_, _ = io.WriteString(h, testData)
	testDataChecksum := hex.EncodeToString(h.Sum(nil))

	// Ожидаемый аргумент для CreateVersion
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
			mockVaultRepo *mocks.VaultRepository,
			mockVersionRepo *mocks.VaultVersionRepository,
			mockFileStorage *mocks.FileStorage,
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
				mockVaultRepo *mocks.VaultRepository,
				mockVersionRepo *mocks.VaultVersionRepository,
				mockFileStorage *mocks.FileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				// 1. Загрузка файла
				mockFileStorage.EXPECT().
					UploadFile(mock.Anything, mock.AnythingOfType("string"), mock.Anything, testSize, testContentType).
					Return(nil).Once()

				// 2. Начало транзакции
				mockSQL.ExpectBegin()

				// 3. Проверка существования хранилища (для проверки конфликта)
				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(nil, nil, repository.ErrVaultNotFound).Once()

				// 4. Создание нового хранилища
				mockVaultRepo.EXPECT().
					CreateVault(mock.Anything, mock.AnythingOfType("*models.Vault")).
					Return(testVaultID, nil).Once()

				// 5. Создание новой версии
				mockVersionRepo.EXPECT().
					CreateVersion(mock.Anything, expectedVersionMatcher).
					Return(testVersionID, nil).Once()

				// 6. Обновление текущей версии хранилища
				mockVaultRepo.EXPECT().
					UpdateVaultCurrentVersion(mock.Anything, testVaultID, testVersionID).
					Return(nil).Once()

				// 7. Завершение транзакции
				mockSQL.ExpectCommit()
			},
			expectedErr: nil,
		},
		{
			name:          "Успех - Существующее хранилище (новее клиента)",
			clientModTime: testModTime, // Время клиента новее, чем mockExistingVersion
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				mockVersionRepo *mocks.VaultVersionRepository,
				mockFileStorage *mocks.FileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				// 1. Загрузка файла
				mockFileStorage.EXPECT().
					UploadFile(mock.Anything, mock.AnythingOfType("string"), mock.Anything, testSize, testContentType).
					Return(nil).Once()

				// 2. Начало транзакции
				mockSQL.ExpectBegin()

				// 3. Получение существующего хранилища и текущей версии
				mockExistingVault := &models.Vault{ID: testVaultID, UserID: testUserID}
				modTimeOlderCopy := testModTimeOlder // Используем старое время для существующей версии
				serverChecksum := "server_checksum"  // Чексумма на сервере
				mockExistingVersion := &models.VaultVersion{
					ContentModifiedAt: &modTimeOlderCopy,
					Checksum:          &serverChecksum,
				}

				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(mockExistingVault, mockExistingVersion, nil).Once()

				// 4. Создание новой версии
				mockVersionRepo.EXPECT().
					CreateVersion(mock.Anything, expectedVersionMatcher).
					Return(testVersionID, nil).Once()

				// 5. Обновление текущей версии
				mockVaultRepo.EXPECT().
					UpdateVaultCurrentVersion(mock.Anything, testVaultID, testVersionID).
					Return(nil).Once()

				// 6. Завершение транзакции
				mockSQL.ExpectCommit()
			},
			expectedErr: nil,
		},
		{
			name:          "Конфликт - Существующее хранилище (старее клиента по времени)",
			clientModTime: testModTimeOlder, // Время клиента СТАРШЕ версии на сервере
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
				mockFileStorage *mocks.FileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				// 1. Загрузка файла (всегда происходит сначала)
				mockFileStorage.EXPECT().
					UploadFile(mock.Anything, mock.AnythingOfType("string"), mock.Anything, testSize, testContentType).
					Return(nil).Once()

				// 2. Начало транзакции
				mockSQL.ExpectBegin()

				// 3. Проверка метаданных для обнаружения конфликта
				mockExistingVault := &models.Vault{ID: testVaultID, UserID: testUserID}
				modTimeServer := testModTime // Версия на сервере новее клиента
				serverChecksum := "server_checksum"
				mockExistingVersion := &models.VaultVersion{
					ContentModifiedAt: &modTimeServer,
					Checksum:          &serverChecksum,
				}
				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(mockExistingVault, mockExistingVersion, nil).Once()

				// 4. Откат транзакции из-за ошибки
				mockSQL.ExpectRollback()
			},
			expectedErr:      services.ErrConflictVersion,
			checkErrorIs:     true,
			expectedConflict: true,
		},
		{
			name:          "Конфликт - Существующее хранилище (время совпадает, checksum разный)",
			clientModTime: testModTime, // Время клиента совпадает
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
				mockFileStorage *mocks.FileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				// 1. Загрузка файла
				mockFileStorage.EXPECT().
					UploadFile(mock.Anything, mock.AnythingOfType("string"), mock.Anything, testSize, testContentType).
					Return(nil).Once()

				// 2. Начало транзакции
				mockSQL.ExpectBegin()

				// 3. Проверка метаданных для обнаружения конфликта
				mockExistingVault := &models.Vault{ID: testVaultID, UserID: testUserID}
				modTimeServer := testModTime                  // Время совпадает
				serverChecksum := "different_server_checksum" // Чексумма отличается
				mockExistingVersion := &models.VaultVersion{
					ContentModifiedAt: &modTimeServer,
					Checksum:          &serverChecksum,
				}
				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(mockExistingVault, mockExistingVersion, nil).Once()

				// 4. Транзакция откатывается из-за ошибки конфликта
				mockSQL.ExpectRollback()
			},
			expectedErr:      services.ErrConflictVersion,
			checkErrorIs:     true,
			expectedConflict: true,
		},
		{
			name:          "Успех - Идентичная версия (пропуск)",
			clientModTime: testModTime, // Время клиента совпадает
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
				mockFileStorage *mocks.FileStorage,
				mockSQL sqlmock.Sqlmock,
			) {
				// Мы должны эмулировать поведение uploadFileToStorage, где сначала загружается файл,
				// а потом используется полученная чексумма.

				// 1. Эмулируем загрузку файла в хранилище
				mockFileStorage.EXPECT().
					UploadFile(mock.Anything, mock.AnythingOfType("string"), mock.Anything, testSize, testContentType).
					Return(nil).
					Run(func(_ context.Context, _ string, reader io.Reader, _ int64, _ string) {
						// Эмулируем чтение из TeeReader, который вычисляет хеш
						_, _ = io.ReadAll(reader) // Это важно! Мы должны прочитать, чтобы хеш вычислился
					}).Once()

				// 2. Начало транзакции
				mockSQL.ExpectBegin()

				// 3. Проверка метаданных - версии идентичны, поэтому мы должны
				// вернуть существующую версию с такой же контрольной суммой
				mockExistingVault := &models.Vault{ID: testVaultID, UserID: testUserID}
				modTimeServer := testModTime

				// Задаем ту же контрольную сумму, что будет вычислена при пустых данных
				// В нашем случае в тесте строка будет "test data"
				mockExistingVersion := &models.VaultVersion{
					ID:                testVersionID,
					VaultID:           testVaultID,
					ContentModifiedAt: &modTimeServer,
					Checksum:          &testDataChecksum, // Этот хеш будет сравниваться с вычисленным
				}

				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(mockExistingVault, mockExistingVersion, nil).Once()

				// 4. Транзакция должна закоммититься
				mockSQL.ExpectCommit()
			},
			expectedErr: nil, // Ошибки нет, просто пропускаем
		},
		{
			name:          "Ошибка - Загрузка в FileStorage",
			clientModTime: testModTime,
			mockSetup: func(
				_ *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
				mockFileStorage *mocks.FileStorage,
				_ sqlmock.Sqlmock,
			) {
				// 1. Ошибка при загрузке файла - файл не загружается, транзакция не начинается
				mockFileStorage.EXPECT().
					UploadFile(mock.Anything, mock.AnythingOfType("string"), mock.Anything, testSize, testContentType).
					Return(errors.New("storage error")).Once()

				// Все остальные вызовы не должны произойти
			},
			expectedErr:  errors.New("внутренняя ошибка сервера при загрузке файла"),
			checkErrorIs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Получаем моки
			service, mockVaultRepo, mockVersionRepo, mockFileStorage, mockSQL := setupVaultServiceWithMocks()

			tt.mockSetup(mockVaultRepo, mockVersionRepo, mockFileStorage, mockSQL)

			// Строка данных для тестирования
			currentReader := strings.NewReader(testData)

			// Вызываем метод сервиса
			err := service.UploadVault(testUserID, currentReader, testSize, testContentType, tt.clientModTime)

			// Проверяем результат
			if tt.expectedErr != nil {
				require.Error(err)
				if tt.checkErrorIs {
					require.ErrorIs(err, tt.expectedErr)
				} else {
					assert.Contains(err.Error(), tt.expectedErr.Error())
				}
			} else {
				require.NoError(err)
			}

			// Проверяем, что все ожидания моков были выполнены
			mockVaultRepo.AssertExpectations(t)
			mockVersionRepo.AssertExpectations(t)
			mockFileStorage.AssertExpectations(t)
			// Проверяем ожидания sqlmock только если не было ошибки загрузки файла
			// или если ошибка nil (успешный случай или пропуск)
			if !tt.expectedConflict && (tt.expectedErr == nil || !strings.Contains(tt.expectedErr.Error(), "загрузке файла")) {
				require.NoError(mockSQL.ExpectationsWereMet(), "Ожидания sqlmock не выполнены")
			}
		})
	}
}

// TestVaultService_DownloadVault проверяет функциональность скачивания текущей версии хранилища.
func TestVaultService_DownloadVault(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	testUserID := int64(1)
	testVaultID := int64(101)
	testVersionID := int64(201)
	testObjectKey := "test/file.kdbx"
	testContent := "test file content"

	tests := []struct {
		name      string
		mockSetup func(
			*mocks.VaultRepository,
			*mocks.VaultVersionRepository,
			*mocks.FileStorage,
		)
		expectedData   string
		expectedErr    error
		checkErrorIs   bool
		expectedMeta   *models.VaultVersion
		shouldReadData bool
	}{
		{
			name: "Успешное скачивание хранилища",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
				mockFileStorage *mocks.FileStorage,
			) {
				// Создаем версию
				version := &models.VaultVersion{
					ID:        testVersionID,
					VaultID:   testVaultID,
					ObjectKey: testObjectKey,
				}

				// Настраиваем мок репозитория
				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID}, version, nil).Once()

				// Настраиваем мок хранилища файлов
				mockFileStorage.EXPECT().
					DownloadFile(mock.Anything, testObjectKey).
					Return(io.NopCloser(strings.NewReader(testContent)), nil).Once()
			},
			expectedData: testContent,
			expectedErr:  nil,
			expectedMeta: &models.VaultVersion{
				ID:        testVersionID,
				VaultID:   testVaultID,
				ObjectKey: testObjectKey,
			},
			shouldReadData: true,
		},
		{
			name: "Хранилище не найдено",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
				_ *mocks.FileStorage,
			) {
				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(nil, nil, repository.ErrVaultNotFound).Once()
			},
			expectedData: "",
			expectedErr:  services.ErrVaultNotFound,
			checkErrorIs: true,
		},
		{
			name: "Ошибка репозитория",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
				_ *mocks.FileStorage,
			) {
				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(nil, nil, errors.New("db error")).Once()
			},
			expectedData: "",
			expectedErr:  errors.New("внутренняя ошибка сервера при получении метаданных"),
		},
		{
			name: "Нет активной версии",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
				_ *mocks.FileStorage,
			) {
				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID}, nil, nil).Once()
			},
			expectedData: "",
			expectedErr:  services.ErrVaultNotFound,
			checkErrorIs: true,
		},
		{
			name: "Объект не найден в хранилище",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
				mockFileStorage *mocks.FileStorage,
			) {
				version := &models.VaultVersion{
					ID:        testVersionID,
					VaultID:   testVaultID,
					ObjectKey: testObjectKey,
				}

				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID}, version, nil).Once()

				mockFileStorage.EXPECT().
					DownloadFile(mock.Anything, testObjectKey).
					Return(nil, storage.ErrObjectNotFound).Once()
			},
			expectedData: "",
			expectedErr:  services.ErrVaultNotFound,
			checkErrorIs: true,
		},
		{
			name: "Ошибка скачивания файла",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
				mockFileStorage *mocks.FileStorage,
			) {
				version := &models.VaultVersion{
					ID:        testVersionID,
					VaultID:   testVaultID,
					ObjectKey: testObjectKey,
				}

				mockVaultRepo.EXPECT().
					GetVaultWithCurrentVersionByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID}, version, nil).Once()

				mockFileStorage.EXPECT().
					DownloadFile(mock.Anything, testObjectKey).
					Return(nil, errors.New("storage error")).Once()
			},
			expectedData: "",
			expectedErr:  errors.New("внутренняя ошибка сервера при скачивании файла"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем сервис с моками
			service, mockVaultRepo, mockVersionRepo, mockFileStorage, _ := setupVaultServiceWithMocks()
			tt.mockSetup(mockVaultRepo, mockVersionRepo, mockFileStorage)

			// Вызываем тестируемый метод
			reader, metadata, err := service.DownloadVault(testUserID)

			// Проверяем результат
			if tt.expectedErr != nil {
				require.Error(err)
				if tt.checkErrorIs {
					require.ErrorIs(err, tt.expectedErr)
				} else {
					assert.Contains(err.Error(), tt.expectedErr.Error())
				}
				assert.Nil(reader)
				assert.Nil(metadata)
			} else {
				require.NoError(err)
				require.NotNil(reader)
				require.NotNil(metadata)

				// Сравниваем метаданные
				assert.Equal(tt.expectedMeta.ID, metadata.ID)
				assert.Equal(tt.expectedMeta.VaultID, metadata.VaultID)
				assert.Equal(tt.expectedMeta.ObjectKey, metadata.ObjectKey)

				// Читаем данные из reader, если нужно
				if tt.shouldReadData {
					data, readErr := io.ReadAll(reader)
					require.NoError(readErr)
					assert.Equal(tt.expectedData, string(data))
				}
			}

			// Проверяем, что все ожидания моков были выполнены
			mockVaultRepo.AssertExpectations(t)
			mockVersionRepo.AssertExpectations(t)
			mockFileStorage.AssertExpectations(t)
		})
	}
}

// TestVaultService_ListVersions проверяет функциональность получения списка версий хранилища.
func TestVaultService_ListVersions(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	testUserID := int64(1)
	testVaultID := int64(101)
	testLimit := 10
	testOffset := 0

	testVersions := []models.VaultVersion{
		{ID: 201, VaultID: testVaultID, CreatedAt: time.Now().Add(-time.Hour * 2)},
		{ID: 202, VaultID: testVaultID, CreatedAt: time.Now().Add(-time.Hour)},
		{ID: 203, VaultID: testVaultID, CreatedAt: time.Now()},
	}

	tests := []struct {
		name             string
		mockSetup        func(*mocks.VaultRepository, *mocks.VaultVersionRepository)
		expectedErr      error
		checkErrorIs     bool
		expectedVersions []models.VaultVersion
	}{
		{
			name: "Успешное получение списка версий",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				mockVersionRepo *mocks.VaultVersionRepository,
			) {
				// Настраиваем мок репозитория хранилища
				mockVaultRepo.EXPECT().
					GetVaultByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID}, nil).Once()

				// Настраиваем мок репозитория версий
				mockVersionRepo.EXPECT().
					ListVersionsByVaultID(mock.Anything, testVaultID, testLimit, testOffset).
					Return(testVersions, nil).Once()
			},
			expectedErr:      nil,
			expectedVersions: testVersions,
		},
		{
			name: "Хранилище не найдено",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
			) {
				mockVaultRepo.EXPECT().
					GetVaultByUserID(mock.Anything, testUserID).
					Return(nil, repository.ErrVaultNotFound).Once()
			},
			expectedErr:      nil, // Сервис возвращает пустой слайс, а не ошибку
			expectedVersions: []models.VaultVersion{},
		},
		{
			name: "Ошибка репозитория при поиске хранилища",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
			) {
				mockVaultRepo.EXPECT().
					GetVaultByUserID(mock.Anything, testUserID).
					Return(nil, errors.New("db error")).Once()
			},
			expectedErr: errors.New("внутренняя ошибка сервера"),
		},
		{
			name: "Ошибка репозитория при получении списка версий",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				mockVersionRepo *mocks.VaultVersionRepository,
			) {
				mockVaultRepo.EXPECT().
					GetVaultByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID}, nil).Once()

				mockVersionRepo.EXPECT().
					ListVersionsByVaultID(mock.Anything, testVaultID, testLimit, testOffset).
					Return(nil, errors.New("db error")).Once()
			},
			expectedErr: errors.New("внутренняя ошибка сервера"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем сервис с моками
			service, mockVaultRepo, mockVersionRepo, _, _ := setupVaultServiceWithMocks()
			tt.mockSetup(mockVaultRepo, mockVersionRepo)

			// Вызываем тестируемый метод
			versions, err := service.ListVersions(testUserID, testLimit, testOffset)

			// Проверяем результат
			if tt.expectedErr != nil {
				require.Error(err)
				if tt.checkErrorIs {
					require.ErrorIs(err, tt.expectedErr)
				} else {
					assert.Contains(err.Error(), tt.expectedErr.Error())
				}
				assert.Nil(versions)
			} else {
				require.NoError(err)
				require.NotNil(versions)
				assert.Equal(len(tt.expectedVersions), len(versions))

				for i, v := range tt.expectedVersions {
					assert.Equal(v.ID, versions[i].ID)
					assert.Equal(v.VaultID, versions[i].VaultID)
				}
			}

			// Проверяем, что все ожидания моков были выполнены
			mockVaultRepo.AssertExpectations(t)
			mockVersionRepo.AssertExpectations(t)
		})
	}
}

// TestVaultService_RollbackToVersion проверяет функциональность отката к указанной версии хранилища.
func TestVaultService_RollbackToVersion(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	testUserID := int64(1)
	testVaultID := int64(101)
	testVersionID := int64(201)

	tests := []struct {
		name         string
		mockSetup    func(*mocks.VaultRepository, *mocks.VaultVersionRepository)
		expectedErr  error
		checkErrorIs bool
	}{
		{
			name: "Успешный откат к версии",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				mockVersionRepo *mocks.VaultVersionRepository,
			) {
				// Настраиваем мок репозитория хранилища
				mockVaultRepo.EXPECT().
					GetVaultByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID}, nil).Once()

				// Настраиваем мок репозитория версий
				mockVersionRepo.EXPECT().
					GetVersionByID(mock.Anything, testVersionID).
					Return(&models.VaultVersion{ID: testVersionID, VaultID: testVaultID}, nil).Once()

				// Ожидаем обновление текущей версии
				mockVaultRepo.EXPECT().
					UpdateVaultCurrentVersion(mock.Anything, testVaultID, testVersionID).
					Return(nil).Once()
			},
			expectedErr: nil,
		},
		{
			name: "Хранилище не найдено",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
			) {
				mockVaultRepo.EXPECT().
					GetVaultByUserID(mock.Anything, testUserID).
					Return(nil, repository.ErrVaultNotFound).Once()
			},
			expectedErr:  services.ErrVaultNotFound,
			checkErrorIs: true,
		},
		{
			name: "Ошибка репозитория при поиске хранилища",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				_ *mocks.VaultVersionRepository,
			) {
				mockVaultRepo.EXPECT().
					GetVaultByUserID(mock.Anything, testUserID).
					Return(nil, errors.New("db error")).Once()
			},
			expectedErr: errors.New("внутренняя ошибка сервера"),
		},
		{
			name: "Версия не найдена",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				mockVersionRepo *mocks.VaultVersionRepository,
			) {
				mockVaultRepo.EXPECT().
					GetVaultByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID}, nil).Once()

				mockVersionRepo.EXPECT().
					GetVersionByID(mock.Anything, testVersionID).
					Return(nil, repository.ErrVersionNotFound).Once()
			},
			expectedErr:  services.ErrVersionNotFound,
			checkErrorIs: true,
		},
		{
			name: "Ошибка репозитория при получении версии",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				mockVersionRepo *mocks.VaultVersionRepository,
			) {
				mockVaultRepo.EXPECT().
					GetVaultByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID}, nil).Once()

				mockVersionRepo.EXPECT().
					GetVersionByID(mock.Anything, testVersionID).
					Return(nil, errors.New("db error")).Once()
			},
			expectedErr: errors.New("внутренняя ошибка сервера"),
		},
		{
			name: "Версия принадлежит другому хранилищу",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				mockVersionRepo *mocks.VaultVersionRepository,
			) {
				mockVaultRepo.EXPECT().
					GetVaultByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID}, nil).Once()

				mockVersionRepo.EXPECT().
					GetVersionByID(mock.Anything, testVersionID).
					Return(&models.VaultVersion{ID: testVersionID, VaultID: testVaultID + 1}, nil).Once()
			},
			expectedErr:  services.ErrForbidden,
			checkErrorIs: true,
		},
		{
			name: "Ошибка при обновлении текущей версии",
			mockSetup: func(
				mockVaultRepo *mocks.VaultRepository,
				mockVersionRepo *mocks.VaultVersionRepository,
			) {
				mockVaultRepo.EXPECT().
					GetVaultByUserID(mock.Anything, testUserID).
					Return(&models.Vault{ID: testVaultID, UserID: testUserID}, nil).Once()

				mockVersionRepo.EXPECT().
					GetVersionByID(mock.Anything, testVersionID).
					Return(&models.VaultVersion{ID: testVersionID, VaultID: testVaultID}, nil).Once()

				mockVaultRepo.EXPECT().
					UpdateVaultCurrentVersion(mock.Anything, testVaultID, testVersionID).
					Return(errors.New("db error")).Once()
			},
			expectedErr: errors.New("внутренняя ошибка сервера при откате"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем сервис с моками
			service, mockVaultRepo, mockVersionRepo, _, _ := setupVaultServiceWithMocks()
			tt.mockSetup(mockVaultRepo, mockVersionRepo)

			// Вызываем тестируемый метод
			err := service.RollbackToVersion(testUserID, testVersionID)

			// Проверяем результат
			if tt.expectedErr != nil {
				require.Error(err)
				if tt.checkErrorIs {
					require.ErrorIs(err, tt.expectedErr)
				} else {
					assert.Contains(err.Error(), tt.expectedErr.Error())
				}
			} else {
				require.NoError(err)
			}

			// Проверяем, что все ожидания моков были выполнены
			mockVaultRepo.AssertExpectations(t)
			mockVersionRepo.AssertExpectations(t)
		})
	}
}

// Add tests for ListVersions, DownloadVault, RollbackToVersion as needed
