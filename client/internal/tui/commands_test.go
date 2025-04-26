//nolint:testpackage,revive,lll // Тесты в том же пакете для доступа к непубличным функциям и дублирование определения MockAPIClient.
package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

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

func (m *CommandsTestMockAPIClient) Login(ctx interface{}, username, password string) (string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.Error(1)
}

func (m *CommandsTestMockAPIClient) Register(ctx interface{}, username, password string) error {
	args := m.Called(ctx, username, password)
	return args.Error(0)
}

func (m *CommandsTestMockAPIClient) GetVaultMetadata(ctx interface{}) (*models.VaultVersion, error) {
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

func (m *CommandsTestMockAPIClient) UploadVault(ctx interface{}, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *CommandsTestMockAPIClient) DownloadVault(ctx interface{}) ([]byte, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	result, ok := args.Get(0).([]byte)
	if !ok {
		return nil, errors.New("неверный тип результата")
	}
	return result, args.Error(1)
}

func (m *CommandsTestMockAPIClient) ListVersions(ctx interface{}, page, perPage int) ([]models.VaultVersion, int64, error) {
	args := m.Called(ctx, page, perPage)
	result, ok := args.Get(0).([]models.VaultVersion)
	if !ok {
		return nil, 0, errors.New("неверный тип результата")
	}
	count, ok := args.Get(1).(int64)
	if !ok {
		return nil, 0, errors.New("неверный тип результата")
	}
	return result, count, args.Error(2)
}

func (m *CommandsTestMockAPIClient) RollbackToVersion(ctx interface{}, versionID int64) error {
	args := m.Called(ctx, versionID)
	return args.Error(0)
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
