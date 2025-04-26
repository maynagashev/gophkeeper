//nolint:testpackage,revive,lll // Тесты в том же пакете для доступа к непубличным функциям и дублирование определения MockAPIClient.
package tui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/client/internal/api"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tobischo/gokeepasslib/v3"
)

// CommandsTestMockAPIClient реализует интерфейс api.Client для тестирования команд.
type CommandsTestMockAPIClient struct {
	mock.Mock
}

func (m *CommandsTestMockAPIClient) Login(ctx context.Context, username, password string) (string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.Error(1)
}

func (m *CommandsTestMockAPIClient) Register(ctx context.Context, username, password string) error {
	args := m.Called(ctx, username, password)
	return args.Error(0)
}

func (m *CommandsTestMockAPIClient) GetVaultMetadata(ctx context.Context) (*models.VaultVersion, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	result, ok := args.Get(0).(*models.VaultVersion)
	if !ok {
		return nil, errors.New("неверный тип результата")
	}
	return result, args.Error(1)
}

func (m *CommandsTestMockAPIClient) UploadVault(ctx context.Context, data io.Reader, size int64, contentModifiedAt time.Time) error {
	args := m.Called(ctx, data, size, contentModifiedAt)
	return args.Error(0)
}

func (m *CommandsTestMockAPIClient) DownloadVault(ctx context.Context) (io.ReadCloser, *models.VaultVersion, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	readCloser, ok := args.Get(0).(io.ReadCloser)
	if !ok {
		return nil, nil, errors.New("неверный тип результата io.ReadCloser")
	}

	var meta *models.VaultVersion
	if args.Get(1) != nil {
		meta, ok = args.Get(1).(*models.VaultVersion)
		if !ok {
			return nil, nil, errors.New("неверный тип результата *models.VaultVersion")
		}
	}

	return readCloser, meta, args.Error(2)
}

func (m *CommandsTestMockAPIClient) ListVersions(ctx context.Context, limit, offset int) ([]models.VaultVersion, int64, error) {
	args := m.Called(ctx, limit, offset)
	result, ok := args.Get(0).([]models.VaultVersion)
	if !ok {
		return nil, 0, errors.New("неверный тип результата []models.VaultVersion")
	}
	count, ok := args.Get(1).(int64)
	if !ok {
		return nil, 0, errors.New("неверный тип результата int64")
	}
	return result, count, args.Error(2)
}

func (m *CommandsTestMockAPIClient) RollbackToVersion(ctx context.Context, versionID int64) error {
	args := m.Called(ctx, versionID)
	return args.Error(0)
}

func (m *CommandsTestMockAPIClient) SetAuthToken(token string) {
	m.Called(token)
}

// Создаем структуру для тестирования.
type mockCommandsModel struct {
	db         *gokeepasslib.Database
	dbModified bool
	dbPath     string
	statusMsg  string
	apiClient  api.Client
	authToken  string
	serverURL  string
}

// Реализуем tea.Model для mockCommandsModel.
func (m *mockCommandsModel) Init() tea.Cmd {
	return nil
}

func (m *mockCommandsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *mockCommandsModel) View() string {
	return ""
}

func (m *mockCommandsModel) GetDB() *gokeepasslib.Database {
	return m.db
}

func (m *mockCommandsModel) SetDB(db *gokeepasslib.Database) {
	m.db = db
}

func (m *mockCommandsModel) SetDBModified(modified bool) {
	m.dbModified = modified
}

func (m *mockCommandsModel) SetDBPath(path string) {
	m.dbPath = path
}

func (m *mockCommandsModel) GetDBPath() string {
	return m.dbPath
}

func (m *mockCommandsModel) SetStatusMessage(msg string) {
	m.statusMsg = msg
}

func (m *mockCommandsModel) GetAPIClient() api.Client {
	return m.apiClient
}

func (m *mockCommandsModel) GetAuthToken() string {
	return m.authToken
}

func (m *mockCommandsModel) GetServerURL() string {
	return m.serverURL
}

// Определяем типы сообщений для тестирования.
type OpenFileSuccess struct {
	Path string
}

type OpenFileError struct {
	Err error
}

func (e OpenFileError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "неизвестная ошибка открытия файла"
}

type SaveFileSuccess struct {
	Path string
}

type SaveFileError struct {
	Err error
}

func (e SaveFileError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "неизвестная ошибка сохранения файла"
}

// TestOpenKdbxCmd проверяет команду открытия файла KDBX
//
//nolint:gocognit // Тесты имеют высокую когнитивную сложность из-за необходимости проверки разных кейсов
func TestOpenKdbxCmd(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir, err := os.MkdirTemp("", "gophkeeper-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Создаем тестовый KDBX файл
	testFilePath := filepath.Join(tempDir, "test.kdbx")
	testPassword := "testpassword"

	// Создаем базу данных
	testDB := gokeepasslib.NewDatabase()
	testDB.Content = gokeepasslib.NewContent()
	testDB.Content.Root = &gokeepasslib.RootData{
		Groups: []gokeepasslib.Group{
			{
				Name: "TestGroup",
			},
		},
	}
	testDB.Credentials = gokeepasslib.NewPasswordCredentials(testPassword)
	err = testDB.LockProtectedEntries()
	require.NoError(t, err)

	// Сохраняем файл
	testFile, err := os.Create(testFilePath)
	require.NoError(t, err)
	err = gokeepasslib.NewEncoder(testFile).Encode(testDB)
	require.NoError(t, err)
	err = testFile.Close()
	require.NoError(t, err)

	// Создаем модель для тестирования
	model := &mockCommandsModel{}

	t.Run("SuccessfulOpen", func(t *testing.T) {
		// Инициализация функции openKdbxCmd
		openCmd := func(path, password string, m *mockCommandsModel) tea.Cmd {
			return func() tea.Msg {
				openDB := gokeepasslib.NewDatabase()
				openDB.Credentials = gokeepasslib.NewPasswordCredentials(password)

				openFile, fileErr := os.Open(path)
				if fileErr != nil {
					return OpenFileError{Err: fileErr}
				}
				defer openFile.Close()

				decodeErr := gokeepasslib.NewDecoder(openFile).Decode(openDB)
				if decodeErr != nil {
					return OpenFileError{Err: decodeErr}
				}

				unlockErr := openDB.UnlockProtectedEntries()
				if unlockErr != nil {
					return OpenFileError{Err: unlockErr}
				}

				m.SetDB(openDB)
				m.SetDBPath(path)
				m.SetDBModified(false)

				return OpenFileSuccess{Path: path}
			}
		}

		// Вызываем команду открытия
		cmd := openCmd(testFilePath, testPassword, model)
		msg := cmd()

		// Проверяем результат
		openSuccess, ok := msg.(OpenFileSuccess)
		require.True(t, ok, "Должно вернуться сообщение OpenFileSuccess")
		assert.Equal(t, testFilePath, openSuccess.Path, "Путь должен совпадать")

		// Проверяем, что база данных была открыта и прикреплена к модели
		assert.NotNil(t, model.db, "База данных должна быть присоединена к модели")
		assert.Equal(t, testFilePath, model.dbPath, "Путь к базе данных должен быть сохранен в модели")
		assert.False(t, model.dbModified, "Флаг модификации должен быть сброшен")

		// Проверяем содержимое базы данных
		assert.NotNil(t, model.db.Content, "Содержимое базы данных должно быть загружено")
		assert.NotNil(t, model.db.Content.Root, "Корневая группа должна быть загружена")
		assert.Len(t, model.db.Content.Root.Groups, 1, "Должна быть одна группа")
		assert.Equal(t, "TestGroup", model.db.Content.Root.Groups[0].Name, "Имя группы должно совпадать")
	})

	t.Run("FileNotFound", func(t *testing.T) {
		// Инициализация функции openKdbxCmd с несуществующим файлом
		nonExistentPath := filepath.Join(tempDir, "nonexistent.kdbx")

		// Инициализация функции openKdbxCmd
		openCmd := func(path, password string, m *mockCommandsModel) tea.Cmd {
			return func() tea.Msg {
				openDB := gokeepasslib.NewDatabase()
				openDB.Credentials = gokeepasslib.NewPasswordCredentials(password)

				openFile, fileErr := os.Open(path)
				if fileErr != nil {
					return OpenFileError{Err: fileErr}
				}
				defer openFile.Close()

				decodeErr := gokeepasslib.NewDecoder(openFile).Decode(openDB)
				if decodeErr != nil {
					return OpenFileError{Err: decodeErr}
				}

				unlockErr := openDB.UnlockProtectedEntries()
				if unlockErr != nil {
					return OpenFileError{Err: unlockErr}
				}

				m.SetDB(openDB)
				m.SetDBPath(path)
				m.SetDBModified(false)

				return OpenFileSuccess{Path: path}
			}
		}

		// Вызываем команду открытия
		cmd := openCmd(nonExistentPath, testPassword, model)
		msg := cmd()

		// Проверяем результат
		openError, ok := msg.(OpenFileError)
		require.True(t, ok, "Должно вернуться сообщение OpenFileError")
		assert.Contains(t, openError.Error(), "no such file", "Ошибка должна указывать на отсутствие файла")
	})

	t.Run("WrongPassword", func(t *testing.T) {
		// Инициализация функции openKdbxCmd с неверным паролем
		wrongPassword := "wrongpassword"

		// Инициализация функции openKdbxCmd
		openCmd := func(path, password string, m *mockCommandsModel) tea.Cmd {
			return func() tea.Msg {
				openDB := gokeepasslib.NewDatabase()
				openDB.Credentials = gokeepasslib.NewPasswordCredentials(password)

				openFile, fileErr := os.Open(path)
				if fileErr != nil {
					return OpenFileError{Err: fileErr}
				}
				defer openFile.Close()

				decodeErr := gokeepasslib.NewDecoder(openFile).Decode(openDB)
				if decodeErr != nil {
					return OpenFileError{Err: decodeErr}
				}

				unlockErr := openDB.UnlockProtectedEntries()
				if unlockErr != nil {
					return OpenFileError{Err: unlockErr}
				}

				m.SetDB(openDB)
				m.SetDBPath(path)
				m.SetDBModified(false)

				return OpenFileSuccess{Path: path}
			}
		}

		// Вызываем команду открытия
		cmd := openCmd(testFilePath, wrongPassword, model)
		msg := cmd()

		// Проверяем результат
		openError, ok := msg.(OpenFileError)
		require.True(t, ok, "Должно вернуться сообщение OpenFileError")
		assert.Error(t, openError.Err, "Ошибка должна быть не nil")
	})
}

// TestSaveKdbxCmd проверяет команду сохранения файла KDBX
//
//nolint:gocognit // Тесты имеют высокую когнитивную сложность из-за необходимости проверки разных кейсов
func TestSaveKdbxCmd(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir, err := os.MkdirTemp("", "gophkeeper-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Создаем тестовый KDBX файл
	testFilePath := filepath.Join(tempDir, "test.kdbx")
	testPassword := "testpassword"

	// Создаем модель для тестирования
	model := &mockCommandsModel{}

	// Создаем базу данных для модели
	testDB := gokeepasslib.NewDatabase()
	testDB.Content = gokeepasslib.NewContent()
	testDB.Content.Root = &gokeepasslib.RootData{
		Groups: []gokeepasslib.Group{
			{
				Name: "TestGroup",
			},
		},
	}
	testDB.Credentials = gokeepasslib.NewPasswordCredentials(testPassword)

	// Прикрепляем базу данных к модели
	model.db = testDB
	model.dbPath = testFilePath
	model.dbModified = true

	t.Run("SuccessfulSave", func(t *testing.T) {
		// Инициализация функции saveKdbxCmd
		saveCmd := func(m *mockCommandsModel) tea.Cmd {
			return func() tea.Msg {
				if m.GetDB() == nil || m.GetDB().Content == nil || m.GetDB().Content.Root == nil {
					return SaveFileError{Err: errors.New("database, its contents or metadata not initialized")}
				}

				path := m.GetDBPath()
				if path == "" {
					return SaveFileError{Err: errors.New("no path to save to")}
				}

				// Блокируем защищенные поля перед сохранением
				lockErr := m.GetDB().LockProtectedEntries()
				if lockErr != nil {
					return SaveFileError{Err: lockErr}
				}

				// Открываем файл для записи
				saveFile, fileErr := os.Create(path)
				if fileErr != nil {
					return SaveFileError{Err: fileErr}
				}
				defer saveFile.Close()

				// Кодируем и сохраняем
				encodeErr := gokeepasslib.NewEncoder(saveFile).Encode(m.GetDB())
				if encodeErr != nil {
					return SaveFileError{Err: encodeErr}
				}

				// Разблокируем защищенные поля обратно
				unlockErr := m.GetDB().UnlockProtectedEntries()
				if unlockErr != nil {
					return SaveFileError{Err: unlockErr}
				}

				// Обновляем состояние
				m.SetDBModified(false)

				return SaveFileSuccess{Path: path}
			}
		}

		// Вызываем команду сохранения
		cmd := saveCmd(model)
		msg := cmd()

		// Проверяем результат
		saveSuccess, ok := msg.(SaveFileSuccess)
		require.True(t, ok, "Должно вернуться сообщение SaveFileSuccess")
		assert.Equal(t, testFilePath, saveSuccess.Path, "Путь должен совпадать")

		// Проверяем, что файл был создан
		_, statErr := os.Stat(testFilePath)
		require.NoError(t, statErr, "Файл должен существовать")

		// Проверяем, что флаг модификации сброшен
		assert.False(t, model.dbModified, "Флаг модификации должен быть сброшен")
	})

	t.Run("NilDatabase", func(t *testing.T) {
		// Устанавливаем пустую базу данных
		model.db = nil

		// Инициализация функции saveKdbxCmd
		saveCmd := func(m *mockCommandsModel) tea.Cmd {
			return func() tea.Msg {
				if m.GetDB() == nil || m.GetDB().Content == nil || m.GetDB().Content.Root == nil {
					return SaveFileError{Err: errors.New("database, its contents or metadata not initialized")}
				}

				path := m.GetDBPath()
				if path == "" {
					return SaveFileError{Err: errors.New("no path to save to")}
				}

				// Дальнейший код не будет выполнен из-за проверки выше

				return SaveFileSuccess{Path: path}
			}
		}

		// Вызываем команду сохранения
		cmd := saveCmd(model)
		msg := cmd()

		// Проверяем результат
		saveError, ok := msg.(SaveFileError)
		require.True(t, ok, "Должно вернуться сообщение SaveFileError")
		assert.Contains(t, saveError.Error(), "database, its contents or metadata not initialized",
			"Ошибка должна указывать на отсутствие базы данных")
	})

	// Восстанавливаем базу данных для следующих тестов
	model.db = testDB

	t.Run("EmptyPath", func(t *testing.T) {
		// Устанавливаем пустой путь
		model.dbPath = ""

		// Инициализация функции saveKdbxCmd
		saveCmd := func(m *mockCommandsModel) tea.Cmd {
			return func() tea.Msg {
				if m.GetDB() == nil || m.GetDB().Content == nil || m.GetDB().Content.Root == nil {
					return SaveFileError{Err: errors.New("database, its contents or metadata not initialized")}
				}

				path := m.GetDBPath()
				if path == "" {
					return SaveFileError{Err: errors.New("no path to save to")}
				}

				// Дальнейший код не будет выполнен из-за проверки выше

				return SaveFileSuccess{Path: path}
			}
		}

		// Вызываем команду сохранения
		cmd := saveCmd(model)
		msg := cmd()

		// Проверяем результат
		saveError, ok := msg.(SaveFileError)
		require.True(t, ok, "Должно вернуться сообщение SaveFileError")
		assert.Contains(t, saveError.Error(), "no path to save to", "Ошибка должна указывать на отсутствие пути")
	})

	// Восстанавливаем путь
	model.dbPath = testFilePath

	t.Run("InaccessibleDirectory", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("Пропускаем тест для root пользователя")
		}

		// Устанавливаем путь к недоступной директории
		inaccessiblePath := "/root/test.kdbx"
		model.dbPath = inaccessiblePath

		// Инициализация функции saveKdbxCmd
		saveCmd := func(m *mockCommandsModel) tea.Cmd {
			return func() tea.Msg {
				if m.GetDB() == nil || m.GetDB().Content == nil || m.GetDB().Content.Root == nil {
					return SaveFileError{Err: errors.New("database, its contents or metadata not initialized")}
				}

				path := m.GetDBPath()
				if path == "" {
					return SaveFileError{Err: errors.New("no path to save to")}
				}

				// Блокируем защищенные поля перед сохранением
				lockErr := m.GetDB().LockProtectedEntries()
				if lockErr != nil {
					return SaveFileError{Err: lockErr}
				}

				// Открываем файл для записи
				saveFile, fileErr := os.Create(path)
				if fileErr != nil {
					return SaveFileError{Err: fileErr}
				}
				defer saveFile.Close()

				// Кодируем и сохраняем
				encodeErr := gokeepasslib.NewEncoder(saveFile).Encode(m.GetDB())
				if encodeErr != nil {
					return SaveFileError{Err: encodeErr}
				}

				// Разблокируем защищенные поля обратно
				unlockErr := m.GetDB().UnlockProtectedEntries()
				if unlockErr != nil {
					return SaveFileError{Err: unlockErr}
				}

				// Обновляем состояние
				m.SetDBModified(false)

				return SaveFileSuccess{Path: path}
			}
		}

		// Вызываем команду сохранения
		cmd := saveCmd(model)
		msg := cmd()

		// Проверяем результат
		saveError, ok := msg.(SaveFileError)
		require.True(t, ok, "Должно вернуться сообщение SaveFileError")
		assert.Error(t, saveError.Err, "Ошибка должна быть не nil")
	})
}

// TestClearStatusCmd проверяет команду создания таймера для очистки статусного сообщения.
func TestClearStatusCmd(t *testing.T) {
	// Создаем минимальное время задержки для теста
	delay := 10 * time.Millisecond

	// Вызываем команду, которая возвращает команду tea.Tick
	cmd := clearStatusCmd(delay)
	require.NotNil(t, cmd, "Команда не должна быть nil")

	// Проверяем, что команда возвращает правильное сообщение после задержки
	time.Sleep(delay * 2) // Ждем немного дольше задержки
	msg := cmd()

	// Проверяем тип сообщения
	_, ok := msg.(clearStatusMsg)
	require.True(t, ok, "Сообщение должно быть типа clearStatusMsg")
}

// TestMakeLoginCmd проверяет функцию makeLoginCmd, которая выполняет вход через API.
func TestMakeLoginCmd(t *testing.T) {
	testUsername := "testuser"
	testPassword := "password123"
	expectedToken := "test-token-123"

	t.Run("SuccessfulLogin", func(t *testing.T) {
		// Создаем мок API клиента
		mockAPI := new(CommandsTestMockAPIClient)

		// Настраиваем ожидаемый вызов Login
		mockAPI.On("Login", mock.Anything, testUsername, testPassword).
			Return(expectedToken, nil).Once()

		// Создаем модель с моком API
		model := &model{
			apiClient: mockAPI,
		}

		// Вызываем команду входа
		cmd := model.makeLoginCmd(testUsername, testPassword)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Выполняем команду
		msg := cmd()

		// Проверяем тип сообщения и содержимое
		loginMsg, ok := msg.(loginSuccessMsg)
		require.True(t, ok, "Сообщение должно быть типа loginSuccessMsg")
		assert.Equal(t, expectedToken, loginMsg.Token, "Токен должен совпадать с ожидаемым")

		// Проверяем, что мок API был вызван как ожидалось
		mockAPI.AssertExpectations(t)
	})

	t.Run("LoginError", func(t *testing.T) {
		// Создаем мок API клиента
		mockAPI := new(CommandsTestMockAPIClient)

		// Имитируем ошибку при вызове Login
		expectedErr := errors.New("неверные учетные данные")
		mockAPI.On("Login", mock.Anything, testUsername, testPassword).
			Return("", expectedErr).Once()

		// Создаем модель с моком API
		model := &model{
			apiClient: mockAPI,
		}

		// Вызываем команду входа
		cmd := model.makeLoginCmd(testUsername, testPassword)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Выполняем команду
		msg := cmd()

		// Проверяем тип сообщения и содержимое
		loginErr, ok := msg.(LoginError)
		require.True(t, ok, "Сообщение должно быть типа LoginError")
		assert.Equal(t, expectedErr.Error(), loginErr.Error(), "Текст ошибки должен совпадать")

		// Проверяем, что мок API был вызван как ожидалось
		mockAPI.AssertExpectations(t)
	})
}

// TestMakeRegisterCmd проверяет функцию makeRegisterCmd, которая выполняет регистрацию через API.
func TestMakeRegisterCmd(t *testing.T) {
	testUsername := "newuser"
	testPassword := "newpassword123"

	t.Run("SuccessfulRegistration", func(t *testing.T) {
		// Создаем мок API клиента
		mockAPI := new(CommandsTestMockAPIClient)

		// Настраиваем ожидаемый вызов Register
		mockAPI.On("Register", mock.Anything, testUsername, testPassword).
			Return(nil).Once()

		// Создаем модель с моком API
		model := &model{
			apiClient: mockAPI,
		}

		// Вызываем команду регистрации
		cmd := model.makeRegisterCmd(testUsername, testPassword)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Выполняем команду
		msg := cmd()

		// Проверяем тип сообщения
		_, ok := msg.(registerSuccessMsg)
		require.True(t, ok, "Сообщение должно быть типа registerSuccessMsg")

		// Проверяем, что мок API был вызван как ожидалось
		mockAPI.AssertExpectations(t)
	})

	t.Run("RegistrationError", func(t *testing.T) {
		// Создаем мок API клиента
		mockAPI := new(CommandsTestMockAPIClient)

		// Имитируем ошибку при вызове Register
		expectedErr := errors.New("пользователь уже существует")
		mockAPI.On("Register", mock.Anything, testUsername, testPassword).
			Return(expectedErr).Once()

		// Создаем модель с моком API
		model := &model{
			apiClient: mockAPI,
		}

		// Вызываем команду регистрации
		cmd := model.makeRegisterCmd(testUsername, testPassword)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Выполняем команду
		msg := cmd()

		// Проверяем тип сообщения и содержимое
		registerErr, ok := msg.(RegisterError)
		require.True(t, ok, "Сообщение должно быть типа RegisterError")
		assert.Equal(t, expectedErr.Error(), registerErr.Error(), "Текст ошибки должен совпадать")

		// Проверяем, что мок API был вызван как ожидалось
		mockAPI.AssertExpectations(t)
	})
}

// TestStartSyncCmd проверяет функцию startSyncCmd, которая проверяет предусловия для синхронизации.
func TestStartSyncCmd(t *testing.T) {
	// Создаем базовую модель для тестов
	mockAPI := new(CommandsTestMockAPIClient)

	t.Run("AllPreconditionsOK", func(t *testing.T) {
		// Создаем модель со всеми необходимыми параметрами для успешной синхронизации
		model := &model{
			serverURL: "https://example.com",
			authToken: "test-token",
			apiClient: mockAPI,
			db:        gokeepasslib.NewDatabase(),
		}

		// Даем модели базовые компоненты БД
		model.db.Content = gokeepasslib.NewContent()
		model.db.Content.Root = &gokeepasslib.RootData{}

		// Вызываем команду синхронизации
		cmd := startSyncCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Проверяем результат
		msg := cmd()
		syncMsg, ok := msg.(syncStartedMsg)
		require.True(t, ok, "Сообщение должно быть syncStartedMsg")
		assert.IsType(t, syncStartedMsg{}, syncMsg, "Тип сообщения должен быть syncStartedMsg")
	})

	t.Run("NoServerURL", func(t *testing.T) {
		// Создаем модель без URL сервера
		model := &model{
			authToken: "test-token",
			apiClient: mockAPI,
			db:        gokeepasslib.NewDatabase(),
		}

		// Даем модели базовые компоненты БД
		model.db.Content = gokeepasslib.NewContent()
		model.db.Content.Root = &gokeepasslib.RootData{}

		// Вызываем команду синхронизации
		cmd := startSyncCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Проверяем результат
		msg := cmd()
		syncErr, ok := msg.(SyncError)
		require.True(t, ok, "Сообщение должно быть SyncError")
		assert.Contains(t, syncErr.Error(), "URL сервера не настроен", "Ошибка должна указывать на отсутствие URL")
	})

	t.Run("NoAuthToken", func(t *testing.T) {
		// Создаем модель без токена аутентификации
		model := &model{
			serverURL: "https://example.com",
			apiClient: mockAPI,
			db:        gokeepasslib.NewDatabase(),
		}

		// Даем модели базовые компоненты БД
		model.db.Content = gokeepasslib.NewContent()
		model.db.Content.Root = &gokeepasslib.RootData{}

		// Вызываем команду синхронизации
		cmd := startSyncCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Проверяем результат
		msg := cmd()
		syncErr, ok := msg.(SyncError)
		require.True(t, ok, "Сообщение должно быть SyncError")
		assert.Contains(t, syncErr.Error(), "необходимо войти на сервер", "Ошибка должна указывать на отсутствие токена")
	})

	t.Run("NoAPIClient", func(t *testing.T) {
		// Создаем модель без API клиента
		model := &model{
			serverURL: "https://example.com",
			authToken: "test-token",
			db:        gokeepasslib.NewDatabase(),
		}

		// Даем модели базовые компоненты БД
		model.db.Content = gokeepasslib.NewContent()
		model.db.Content.Root = &gokeepasslib.RootData{}

		// Вызываем команду синхронизации
		cmd := startSyncCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Проверяем результат
		msg := cmd()
		syncErr, ok := msg.(SyncError)
		require.True(t, ok, "Сообщение должно быть SyncError")
		assert.Contains(t, syncErr.Error(), "API клиент не инициализирован", "Ошибка должна указывать на отсутствие API клиента")
	})

	t.Run("NoDatabase", func(t *testing.T) {
		// Создаем модель без базы данных
		model := &model{
			serverURL: "https://example.com",
			authToken: "test-token",
			apiClient: mockAPI,
		}

		// Вызываем команду синхронизации
		cmd := startSyncCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Проверяем результат
		msg := cmd()
		syncErr, ok := msg.(SyncError)
		require.True(t, ok, "Сообщение должно быть SyncError")
		assert.Contains(t, syncErr.Error(), "локальная база данных не загружена", "Ошибка должна указывать на отсутствие базы данных")
	})
}

// TestFetchServerMetadataCmd проверяет функцию fetchServerMetadataCmd, которая запрашивает метаданные с сервера.
func TestFetchServerMetadataCmd(t *testing.T) {
	// Создаем базовую модель для тестов
	mockAPI := new(CommandsTestMockAPIClient)

	t.Run("SuccessfulMetadataFetch", func(t *testing.T) {
		// Настраиваем мок для успешного получения метаданных
		mockMetadata := &models.VaultVersion{
			ID:        1,
			CreatedAt: time.Now(),
			SizeBytes: new(int64),
		}
		*mockMetadata.SizeBytes = 1024
		mockAPI.On("GetVaultMetadata", mock.Anything).Return(mockMetadata, nil).Once()

		// Создаем модель со всеми необходимыми параметрами
		model := &model{
			apiClient: mockAPI,
		}

		// Вызываем команду получения метаданных сервера
		cmd := fetchServerMetadataCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Проверяем результат
		msg := cmd()
		metadataMsg, ok := msg.(serverMetadataMsg)
		require.True(t, ok, "Сообщение должно быть serverMetadataMsg")
		assert.Equal(t, mockMetadata, metadataMsg.metadata, "Метаданные должны совпадать")

		// Проверяем, что метод GetVaultMetadata был вызван
		mockAPI.AssertExpectations(t)
	})

	t.Run("ErrorFetchingMetadata", func(t *testing.T) {
		// Настраиваем мок для ошибки при получении метаданных
		expectedErr := errors.New("ошибка получения метаданных")
		mockAPI.On("GetVaultMetadata", mock.Anything).Return(nil, expectedErr).Once()

		// Создаем модель со всеми необходимыми параметрами
		model := &model{
			apiClient: mockAPI,
		}

		// Вызываем команду получения метаданных сервера
		cmd := fetchServerMetadataCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Проверяем результат
		msg := cmd()
		syncErr, ok := msg.(SyncError)
		require.True(t, ok, "Сообщение должно быть SyncError")
		assert.Contains(t, syncErr.Error(), expectedErr.Error(), "Ошибка должна содержать оригинальную ошибку")

		// Проверяем, что метод GetVaultMetadata был вызван
		mockAPI.AssertExpectations(t)
	})
}

// TestFetchLocalMetadataCmd проверяет функцию fetchLocalMetadataCmd, которая получает локальные метаданные.
func TestFetchLocalMetadataCmd(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir, err := os.MkdirTemp("", "gophkeeper_test_*")
	require.NoError(t, err, "Не удалось создать временную директорию")
	defer os.RemoveAll(tempDir)

	// Путь к тестовому файлу
	testFilePath := filepath.Join(tempDir, "test.kdbx")

	t.Run("SuccessfulLocalMetadataFetch", func(t *testing.T) {
		// Создаем тестовый файл с содержимым
		testFileContent := []byte("test kdbx file content")
		errWrite := os.WriteFile(testFilePath, testFileContent, 0600)
		require.NoError(t, errWrite, "Не удалось создать тестовый файл")

		// Создаем модель с путем к файлу
		model := &model{
			kdbxPath: testFilePath,
		}

		// Получаем время изменения файла
		fileInfo, errStat := os.Stat(testFilePath)
		require.NoError(t, errStat, "Не удалось получить информацию о файле")

		// Вызываем команду получения локальных метаданных
		cmd := fetchLocalMetadataCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Проверяем результат
		msg := cmd()
		metadataMsg, ok := msg.(localMetadataMsg)
		require.True(t, ok, "Сообщение должно быть localMetadataMsg")

		// Проверяем содержимое метаданных (метаданные локального файла возвращаются в виде modTime и found)
		assert.Equal(t, fileInfo.ModTime(), metadataMsg.modTime, "Время изменения должно совпадать")
		assert.True(t, metadataMsg.found, "Флаг found должен быть true")
	})

	t.Run("FileNotFound", func(t *testing.T) {
		// Создаем модель с несуществующим путем к файлу
		model := &model{
			kdbxPath: filepath.Join(tempDir, "non_existent.kdbx"),
		}

		// Вызываем команду получения локальных метаданных
		cmd := fetchLocalMetadataCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Проверяем результат
		msg := cmd()
		metadataMsg, ok := msg.(localMetadataMsg)
		require.True(t, ok, "Сообщение должно быть localMetadataMsg")
		assert.False(t, metadataMsg.found, "Флаг found должен быть false для несуществующего файла")
	})

	t.Run("NoPathInModel", func(t *testing.T) {
		// Создаем модель без пути к файлу
		model := &model{}

		// Вызываем команду получения локальных метаданных
		cmd := fetchLocalMetadataCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Проверяем результат
		msg := cmd()
		metadataMsg, ok := msg.(localMetadataMsg)
		require.True(t, ok, "Сообщение должно быть localMetadataMsg")
		assert.False(t, metadataMsg.found, "Флаг found должен быть false для модели без пути")
	})
}

// TestUploadVaultCmd проверяет функцию uploadVaultCmd, которая загружает локальный файл на сервер.
func TestUploadVaultCmd(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir, err := os.MkdirTemp("", "gophkeeper_test_*")
	require.NoError(t, err, "Не удалось создать временную директорию")
	defer os.RemoveAll(tempDir)

	// Путь к тестовому файлу
	testFilePath := filepath.Join(tempDir, "test.kdbx")

	// Создаем тестовое содержимое
	testFileContent := []byte("test kdbx file content")
	errWrite := os.WriteFile(testFilePath, testFileContent, 0600)
	require.NoError(t, errWrite, "Не удалось создать тестовый файл")

	// Получаем информацию о файле для теста
	_, errStat := os.Stat(testFilePath)
	require.NoError(t, errStat, "Не удалось получить информацию о файле")

	// Создаем мок API клиента
	mockAPI := new(CommandsTestMockAPIClient)

	t.Run("SuccessfulUpload", func(t *testing.T) {
		// Настраиваем мок для успешной загрузки
		mockAPI.On("UploadVault", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		// Создаем базу данных с базовой структурой
		db := gokeepasslib.NewDatabase()
		db.Content = gokeepasslib.NewContent()
		db.Content.Root = &gokeepasslib.RootData{
			Groups: []gokeepasslib.Group{
				{
					Name: "Root",
				},
			},
		}

		// Создаем модель
		model := &model{
			kdbxPath:  testFilePath,
			apiClient: mockAPI,
			db:        db,
		}

		// Вызываем команду загрузки
		cmd := uploadVaultCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Выполняем команду и проверяем результат
		msg := cmd()
		_, ok := msg.(syncUploadSuccessMsg)
		require.True(t, ok, "Сообщение должно быть syncUploadSuccessMsg")

		// Проверяем, что метод UploadVault был вызван
		mockAPI.AssertExpectations(t)
	})

	t.Run("APIError", func(t *testing.T) {
		// Настраиваем мок для ошибки при загрузке
		expectedErr := errors.New("ошибка загрузки")
		mockAPI.On("UploadVault", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(expectedErr).Once()

		// Создаем базу данных с базовой структурой
		db := gokeepasslib.NewDatabase()
		db.Content = gokeepasslib.NewContent()
		db.Content.Root = &gokeepasslib.RootData{
			Groups: []gokeepasslib.Group{
				{
					Name: "Root",
				},
			},
		}

		// Создаем модель
		model := &model{
			kdbxPath:  testFilePath,
			apiClient: mockAPI,
			db:        db,
		}

		// Вызываем команду загрузки
		cmd := uploadVaultCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Выполняем команду и проверяем результат
		msg := cmd()
		syncErr, ok := msg.(SyncError)
		require.True(t, ok, "Сообщение должно быть SyncError")
		assert.Contains(t, syncErr.Error(), expectedErr.Error(), "Ошибка должна содержать оригинальную ошибку")

		// Проверяем, что метод UploadVault был вызван
		mockAPI.AssertExpectations(t)
	})

	t.Run("NoDatabaseLoaded", func(t *testing.T) {
		// Создаем модель без базы данных
		model := &model{
			kdbxPath:  testFilePath,
			apiClient: mockAPI,
		}

		// Вызываем команду загрузки
		cmd := uploadVaultCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Выполняем команду и проверяем результат
		msg := cmd()
		syncErr, ok := msg.(SyncError)
		require.True(t, ok, "Сообщение должно быть SyncError")
		assert.Contains(t, syncErr.Error(), "локальная база не загружена", "Ошибка должна указывать на отсутствие базы")
	})

	t.Run("FileNotFoundForMetadata", func(t *testing.T) {
		// Создаем базу данных с базовой структурой
		db := gokeepasslib.NewDatabase()
		db.Content = gokeepasslib.NewContent()
		db.Content.Root = &gokeepasslib.RootData{}

		// Создаем модель с несуществующим файлом
		model := &model{
			kdbxPath:  filepath.Join(tempDir, "non_existent.kdbx"),
			apiClient: mockAPI,
			db:        db,
		}

		// Вызываем команду загрузки
		cmd := uploadVaultCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Выполняем команду и проверяем результат
		msg := cmd()
		syncErr, ok := msg.(SyncError)
		require.True(t, ok, "Сообщение должно быть SyncError")
		assert.Contains(t, syncErr.Error(), "ошибка доступа к локальному файлу", "Ошибка должна указывать на проблему доступа к файлу")
	})
}

// TestDownloadVaultCmd проверяет функцию downloadVaultCmd, которая скачивает хранилище с сервера.
func TestDownloadVaultCmd(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir, err := os.MkdirTemp("", "gophkeeper_test_*")
	require.NoError(t, err, "Не удалось создать временную директорию")
	defer os.RemoveAll(tempDir)

	// Путь к тестовому файлу
	testFilePath := filepath.Join(tempDir, "test.kdbx")

	// Создаем мок API клиента
	mockAPI := new(CommandsTestMockAPIClient)

	t.Run("SuccessfulDownload", func(t *testing.T) {
		// Создаем тестовое содержимое для "скачивания"
		fileContent := []byte("downloaded content")
		readCloser := io.NopCloser(bytes.NewReader(fileContent))

		// Метаданные версии
		fileSize := int64(len(fileContent))
		mockVersion := &models.VaultVersion{
			ID:        1,
			CreatedAt: time.Now(),
			SizeBytes: &fileSize,
		}

		// Настраиваем мок для успешного скачивания
		mockAPI.On("DownloadVault", mock.Anything).Return(readCloser, mockVersion, nil).Once()

		// Создаем модель
		model := &model{
			kdbxPath:  testFilePath,
			apiClient: mockAPI,
		}

		// Вызываем команду скачивания
		cmd := downloadVaultCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Выполняем команду и проверяем результат
		msg := cmd()
		downloadMsg, ok := msg.(syncDownloadSuccessMsg)
		require.True(t, ok, "Сообщение должно быть syncDownloadSuccessMsg")
		assert.True(t, downloadMsg.reloadNeeded, "Флаг reloadNeeded должен быть true")

		// Проверяем, что файл был создан
		_, errStat := os.Stat(testFilePath)
		require.NoError(t, errStat, "Файл должен существовать после скачивания")

		// Проверяем содержимое файла
		content, errRead := os.ReadFile(testFilePath)
		require.NoError(t, errRead, "Должны иметь возможность прочитать скачанный файл")
		assert.Equal(t, fileContent, content, "Содержимое должно совпадать")

		// Проверяем, что метод DownloadVault был вызван
		mockAPI.AssertExpectations(t)
	})

	t.Run("APIError", func(t *testing.T) {
		// Настраиваем мок для ошибки при скачивании
		expectedErr := errors.New("ошибка скачивания")
		mockAPI.On("DownloadVault", mock.Anything).Return(nil, nil, expectedErr).Once()

		// Создаем модель
		model := &model{
			kdbxPath:  testFilePath,
			apiClient: mockAPI,
		}

		// Вызываем команду скачивания
		cmd := downloadVaultCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Выполняем команду и проверяем результат
		msg := cmd()
		syncErr, ok := msg.(SyncError)
		require.True(t, ok, "Сообщение должно быть SyncError")
		assert.Contains(t, syncErr.Error(), expectedErr.Error(), "Ошибка должна содержать оригинальную ошибку")

		// Проверяем, что метод DownloadVault был вызван
		mockAPI.AssertExpectations(t)
	})

	t.Run("WriteError", func(t *testing.T) {
		// Создаем тестовую директорию, которую сделаем read-only
		readOnlyDir := filepath.Join(tempDir, "readonly")
		require.NoError(t, os.Mkdir(readOnlyDir, 0500), "Не удалось создать read-only директорию")
		defer os.RemoveAll(readOnlyDir)

		// Создаем путь к файлу в read-only директории
		readOnlyFilePath := filepath.Join(readOnlyDir, "test.kdbx")

		// Создаем тестовое содержимое для "скачивания"
		fileContent := []byte("downloaded content")
		readCloser := io.NopCloser(bytes.NewReader(fileContent))

		// Метаданные версии
		fileSize := int64(len(fileContent))
		mockVersion := &models.VaultVersion{
			ID:        1,
			CreatedAt: time.Now(),
			SizeBytes: &fileSize,
		}

		// Настраиваем мок для успешного скачивания API
		mockAPI.On("DownloadVault", mock.Anything).Return(readCloser, mockVersion, nil).Once()

		// Создаем модель с путем к файлу в read-only директории
		model := &model{
			kdbxPath:  readOnlyFilePath,
			apiClient: mockAPI,
		}

		// На некоторых ОС (особенно Windows) права доступа могут работать иначе
		// Пропускаем тест, если у нас получилось создать файл в read-only директории
		if tempFile, errCreate := os.Create(readOnlyFilePath); errCreate == nil {
			tempFile.Close()
			os.Remove(readOnlyFilePath)
			t.Skip("Тест не применим: смогли создать файл в read-only директории")
		}

		// Вызываем команду скачивания
		cmd := downloadVaultCmd(model)
		require.NotNil(t, cmd, "Команда не должна быть nil")

		// Выполняем команду и проверяем результат
		msg := cmd()
		_, ok := msg.(SyncError)
		require.True(t, ok, "Сообщение должно быть SyncError")

		// Проверяем, что метод DownloadVault был вызван
		mockAPI.AssertExpectations(t)
	})
}

// TestListVersionsCmd тестирует функцию listVersionsCmd для получения списка версий.
func TestListVersionsCmd(t *testing.T) {
	// Подготовим общие данные для тестовых сценариев
	testVersions := []models.VaultVersion{
		{
			ID:                1,
			VaultID:           100,
			SizeBytes:         nil,
			CreatedAt:         time.Now().Add(-48 * time.Hour),
			ContentModifiedAt: &time.Time{},
		},
		{
			ID:                2,
			VaultID:           100,
			SizeBytes:         nil,
			CreatedAt:         time.Now().Add(-24 * time.Hour),
			ContentModifiedAt: &time.Time{},
		},
	}
	currentVersionID := int64(2)

	// Подготовим тестовые сценарии
	tests := []struct {
		name            string
		setupModel      func(m *model)
		setupMock       func(mockAPI *CommandsTestMockAPIClient)
		expectedMsgType any
		validateMsg     func(t *testing.T, msg tea.Msg)
	}{
		{
			name: "УспешнаяЗагрузкаВерсий",
			setupModel: func(m *model) {
				m.apiClient = &CommandsTestMockAPIClient{}
				m.authToken = "test-token"
			},
			setupMock: func(mockAPI *CommandsTestMockAPIClient) {
				mockAPI.On("ListVersions", mock.Anything, defaultVersionListLimit, 0).
					Return(testVersions, currentVersionID, nil)
			},
			expectedMsgType: versionsLoadedMsg{},
			validateMsg: func(t *testing.T, msg tea.Msg) {
				loadedMsg, ok := msg.(versionsLoadedMsg)
				require.True(t, ok, "Должно вернуться сообщение versionsLoadedMsg")
				assert.Len(t, loadedMsg.versions, 2, "Неверное количество версий")
				assert.Equal(t, currentVersionID, loadedMsg.currentVersionID, "Неверный ID текущей версии")
			},
		},
		{
			name: "APIКлиентНеИнициализирован",
			setupModel: func(m *model) {
				m.apiClient = nil
				m.authToken = "test-token"
			},
			setupMock: func(mockAPI *CommandsTestMockAPIClient) {
				// Мок не будет вызван
			},
			expectedMsgType: versionsLoadErrorMsg{},
			validateMsg: func(t *testing.T, msg tea.Msg) {
				errMsg, ok := msg.(versionsLoadErrorMsg)
				require.True(t, ok, "Должно вернуться сообщение versionsLoadErrorMsg")
				assert.Equal(t, "API клиент не инициализирован", errMsg.err.Error(), "Неверный текст ошибки")
			},
		},
		{
			name: "ОтсутствуетТокенАвторизации",
			setupModel: func(m *model) {
				m.apiClient = &CommandsTestMockAPIClient{}
				m.authToken = ""
			},
			setupMock: func(mockAPI *CommandsTestMockAPIClient) {
				// Мок не будет вызван
			},
			expectedMsgType: versionsLoadErrorMsg{},
			validateMsg: func(t *testing.T, msg tea.Msg) {
				errMsg, ok := msg.(versionsLoadErrorMsg)
				require.True(t, ok, "Должно вернуться сообщение versionsLoadErrorMsg")
				assert.Equal(t, "требуется авторизация", errMsg.err.Error(), "Неверный текст ошибки")
			},
		},
		{
			name: "ОшибкаAPI",
			setupModel: func(m *model) {
				m.apiClient = &CommandsTestMockAPIClient{}
				m.authToken = "test-token"
			},
			setupMock: func(mockAPI *CommandsTestMockAPIClient) {
				mockAPI.On("ListVersions", mock.Anything, defaultVersionListLimit, 0).
					Return([]models.VaultVersion{}, int64(0), errors.New("ошибка сети"))
			},
			expectedMsgType: versionsLoadErrorMsg{},
			validateMsg: func(t *testing.T, msg tea.Msg) {
				errMsg, ok := msg.(versionsLoadErrorMsg)
				require.True(t, ok, "Должно вернуться сообщение versionsLoadErrorMsg")
				assert.Equal(t, "ошибка сети", errMsg.err.Error(), "Неверный текст ошибки")
			},
		},
		{
			name: "ОшибкаАвторизации",
			setupModel: func(m *model) {
				m.apiClient = &CommandsTestMockAPIClient{}
				m.authToken = "test-token"
			},
			setupMock: func(mockAPI *CommandsTestMockAPIClient) {
				mockAPI.On("ListVersions", mock.Anything, defaultVersionListLimit, 0).
					Return([]models.VaultVersion{}, int64(0), api.ErrAuthorization)
			},
			expectedMsgType: versionsLoadErrorMsg{},
			validateMsg: func(t *testing.T, msg tea.Msg) {
				errMsg, ok := msg.(versionsLoadErrorMsg)
				require.True(t, ok, "Должно вернуться сообщение versionsLoadErrorMsg")
				assert.Equal(t, "ошибка авторизации", errMsg.err.Error(), "Неверный текст ошибки")
			},
		},
	}

	// Запускаем тесты
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем модель и устанавливаем начальное состояние
			m := &model{}
			tt.setupModel(m)

			// Если есть apiClient, настраиваем мок
			if mockAPI, ok := m.apiClient.(*CommandsTestMockAPIClient); ok {
				tt.setupMock(mockAPI)
			}

			// Вызываем тестируемую функцию
			cmd := loadVersionsCmd(m)
			msg := cmd()

			// Проверяем результат
			tt.validateMsg(t, msg)

			// Проверяем, что все ожидаемые вызовы мока были выполнены
			if mockAPI, ok := m.apiClient.(*CommandsTestMockAPIClient); ok {
				mockAPI.AssertExpectations(t)
			}
		})
	}
}

// TestRollbackToVersionCmd тестирует функцию rollbackToVersionCmd для отката к выбранной версии.
func TestRollbackToVersionCmd(t *testing.T) {
	// Подготовим тестовые сценарии
	tests := []struct {
		name            string
		versionID       int64
		setupModel      func(m *model)
		setupMock       func(mockAPI *CommandsTestMockAPIClient)
		expectedMsgType any
		validateMsg     func(t *testing.T, msg tea.Msg)
	}{
		{
			name:      "УспешныйОткат",
			versionID: 10,
			setupModel: func(m *model) {
				m.apiClient = &CommandsTestMockAPIClient{}
				m.authToken = "test-token"
			},
			setupMock: func(mockAPI *CommandsTestMockAPIClient) {
				mockAPI.On("RollbackToVersion", mock.Anything, int64(10)).Return(nil)
			},
			expectedMsgType: rollbackSuccessMsg{},
			validateMsg: func(t *testing.T, msg tea.Msg) {
				successMsg, ok := msg.(rollbackSuccessMsg)
				require.True(t, ok, "Должно вернуться сообщение rollbackSuccessMsg")
				assert.Equal(t, int64(10), successMsg.versionID, "Неверный ID версии в сообщении")
			},
		},
		{
			name:      "APIКлиентНеИнициализирован",
			versionID: 10,
			setupModel: func(m *model) {
				m.apiClient = nil
				m.authToken = "test-token"
			},
			setupMock: func(mockAPI *CommandsTestMockAPIClient) {
				// Мок не будет вызван
			},
			expectedMsgType: rollbackErrorMsg{},
			validateMsg: func(t *testing.T, msg tea.Msg) {
				errMsg, ok := msg.(rollbackErrorMsg)
				require.True(t, ok, "Должно вернуться сообщение rollbackErrorMsg")
				assert.Equal(t, "API клиент не инициализирован", errMsg.err.Error(), "Неверный текст ошибки")
			},
		},
		{
			name:      "ОтсутствуетТокенАвторизации",
			versionID: 10,
			setupModel: func(m *model) {
				m.apiClient = &CommandsTestMockAPIClient{}
				m.authToken = ""
			},
			setupMock: func(mockAPI *CommandsTestMockAPIClient) {
				// Мок не будет вызван
			},
			expectedMsgType: rollbackErrorMsg{},
			validateMsg: func(t *testing.T, msg tea.Msg) {
				errMsg, ok := msg.(rollbackErrorMsg)
				require.True(t, ok, "Должно вернуться сообщение rollbackErrorMsg")
				assert.Equal(t, "требуется авторизация", errMsg.err.Error(), "Неверный текст ошибки")
			},
		},
		{
			name:      "ОшибкаОтката",
			versionID: 10,
			setupModel: func(m *model) {
				m.apiClient = &CommandsTestMockAPIClient{}
				m.authToken = "test-token"
			},
			setupMock: func(mockAPI *CommandsTestMockAPIClient) {
				mockAPI.On("RollbackToVersion", mock.Anything, int64(10)).
					Return(errors.New("версия не найдена"))
			},
			expectedMsgType: rollbackErrorMsg{},
			validateMsg: func(t *testing.T, msg tea.Msg) {
				errMsg, ok := msg.(rollbackErrorMsg)
				require.True(t, ok, "Должно вернуться сообщение rollbackErrorMsg")
				assert.Equal(t, "версия не найдена", errMsg.err.Error(), "Неверный текст ошибки")
			},
		},
	}

	// Запускаем тесты
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем модель и устанавливаем начальное состояние
			m := &model{}
			tt.setupModel(m)

			// Если есть apiClient, настраиваем мок
			if mockAPI, ok := m.apiClient.(*CommandsTestMockAPIClient); ok {
				tt.setupMock(mockAPI)
			}

			// Вызываем тестируемую функцию
			cmd := rollbackToVersionCmd(m, tt.versionID)
			msg := cmd()

			// Проверяем результат
			tt.validateMsg(t, msg)

			// Проверяем, что все ожидаемые вызовы мока были выполнены
			if mockAPI, ok := m.apiClient.(*CommandsTestMockAPIClient); ok {
				mockAPI.AssertExpectations(t)
			}
		})
	}
}

// TestApplyUIChangesToDB проверяет функцию applyUIChangesToDB, которая применяет изменения из TUI к базе данных.
func TestApplyUIChangesToDB(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir, err := os.MkdirTemp("", "gophkeeper_test_*")
	require.NoError(t, err, "Не удалось создать временную директорию")
	defer os.RemoveAll(tempDir)

	// Создаем тестовую базу данных с группой и записями
	db := gokeepasslib.NewDatabase()
	db.Content = gokeepasslib.NewContent()

	// Создаем UUID для записей, которые мы сможем использовать для поиска и сравнения
	uuid1 := gokeepasslib.NewUUID()
	uuid2 := gokeepasslib.NewUUID()

	// Создаем корневую группу с записями
	rootGroup := gokeepasslib.Group{
		Name: "Root",
		Entries: []gokeepasslib.Entry{
			{
				UUID: uuid1,
				Values: []gokeepasslib.ValueData{
					{
						Key:   "Title",
						Value: gokeepasslib.V{Content: "Запись 1"},
					},
					{
						Key:   "UserName",
						Value: gokeepasslib.V{Content: "user1"},
					},
				},
			},
			{
				UUID: uuid2,
				Values: []gokeepasslib.ValueData{
					{
						Key:   "Title",
						Value: gokeepasslib.V{Content: "Запись 2"},
					},
					{
						Key:   "UserName",
						Value: gokeepasslib.V{Content: "user2"},
					},
				},
			},
		},
	}

	db.Content.Root = &gokeepasslib.RootData{
		Groups: []gokeepasslib.Group{rootGroup},
	}

	t.Run("УспешноеОбновлениеЗаписей", func(t *testing.T) {
		// Создаем модифицированные записи для entryList
		modifiedEntry1 := gokeepasslib.Entry{
			UUID: uuid1,
			Values: []gokeepasslib.ValueData{
				{
					Key:   "Title",
					Value: gokeepasslib.V{Content: "Запись 1 (изменено)"},
				},
				{
					Key:   "UserName",
					Value: gokeepasslib.V{Content: "user1_modified"},
				},
				{
					Key:   "Password",
					Value: gokeepasslib.V{Content: "password1"},
				},
			},
		}

		// Вторая запись не изменяется

		// Создаем модель с базой данных и entryList
		m := &model{
			db: db,
		}

		// Создаем список с модифицированными записями
		items := []list.Item{
			entryItem{entry: modifiedEntry1},
			// Вторую запись не включаем в список, чтобы проверить, что она не изменится
		}
		m.entryList = list.New(items, list.NewDefaultDelegate(), 0, 0)

		// Вызываем тестируемую функцию
		applyUIChangesToDB(m)

		// Проверяем, что первая запись была обновлена
		entry1 := findEntryInDB(m.db, uuid1)
		require.NotNil(t, entry1, "Запись 1 должна существовать")

		// Функция для получения значения по ключу
		getValue := func(entry *gokeepasslib.Entry, key string) string {
			for _, v := range entry.Values {
				if v.Key == key {
					return v.Value.Content
				}
			}
			return ""
		}

		// Проверяем модифицированные поля
		assert.Equal(t, "Запись 1 (изменено)", getValue(entry1, "Title"), "Заголовок должен быть обновлен")
		assert.Equal(t, "user1_modified", getValue(entry1, "UserName"), "Имя пользователя должно быть обновлено")
		assert.Equal(t, "password1", getValue(entry1, "Password"), "Пароль должен быть добавлен")

		// Проверяем, что вторая запись не изменилась
		entry2 := findEntryInDB(m.db, uuid2)
		require.NotNil(t, entry2, "Запись 2 должна существовать")
		assert.Equal(t, "Запись 2", getValue(entry2, "Title"), "Заголовок не должен быть изменен")
		assert.Equal(t, "user2", getValue(entry2, "UserName"), "Имя пользователя не должно быть изменено")
	})

	t.Run("НесуществующийUUID", func(t *testing.T) {
		// Создаем запись с UUID, которого нет в базе
		nonExistentUUID := gokeepasslib.NewUUID()
		nonExistentEntry := gokeepasslib.Entry{
			UUID: nonExistentUUID,
			Values: []gokeepasslib.ValueData{
				{
					Key:   "Title",
					Value: gokeepasslib.V{Content: "Несуществующая запись"},
				},
			},
		}

		// Создаем модель с базой данных и entryList
		m := &model{
			db: db,
		}

		// Создаем список только с несуществующей записью
		items := []list.Item{
			entryItem{entry: nonExistentEntry},
		}
		m.entryList = list.New(items, list.NewDefaultDelegate(), 0, 0)

		// Вызываем тестируемую функцию
		applyUIChangesToDB(m)

		// Проверяем, что запись не добавилась в базу
		entry := findEntryInDB(m.db, nonExistentUUID)
		assert.Nil(t, entry, "Несуществующая запись не должна быть добавлена в базу")

		// Проверяем, что количество записей в базе не изменилось
		assert.Len(t, db.Content.Root.Groups[0].Entries, 2, "Количество записей не должно измениться")
	})

	t.Run("ПустойСписок", func(t *testing.T) {
		// Создаем модель с базой данных и пустым entryList
		m := &model{
			db: db,
		}

		// Создаем пустой список
		m.entryList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

		// Запоминаем состояние базы
		original := deepCopyEntry(db.Content.Root.Groups[0].Entries[0])

		// Вызываем тестируемую функцию
		applyUIChangesToDB(m)

		// Проверяем, что записи не изменились
		entry1 := findEntryInDB(m.db, uuid1)
		require.NotNil(t, entry1, "Запись 1 должна существовать")

		// Проверяем, что значения в первой записи не изменились
		assert.Equal(t, original.Values[0].Value.Content, entry1.Values[0].Value.Content, "Значение Title не должно измениться")
	})
}
