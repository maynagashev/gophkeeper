package tui //nolint:testpackage // Используем тот же пакет для доступа к неэкспортируемым типам

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScreenTestMockAPIClient_Login проверяет мок метода Login.
func TestScreenTestMockAPIClient_Login(t *testing.T) {
	mockClient := new(ScreenTestMockAPIClient)
	ctx := context.Background()
	username := "testuser"
	password := "testpass"
	expectedToken := "mock-token"
	expectedErr := errors.New("mock login error")

	// Тест успешного входа
	t.Run("Success", func(t *testing.T) {
		mockClient.On("Login", ctx, username, password).Return(expectedToken, nil).Once()
		token, err := mockClient.Login(ctx, username, password)
		require.NoError(t, err) // Используем require
		assert.Equal(t, expectedToken, token)
		mockClient.AssertExpectations(t)
	})

	// Тест ошибки входа
	t.Run("Error", func(t *testing.T) {
		mockClient.On("Login", ctx, username, password).Return("", expectedErr).Once()
		token, err := mockClient.Login(ctx, username, password)
		require.Error(t, err) // Используем require
		assert.Equal(t, expectedErr, err)
		assert.Empty(t, token)
		mockClient.AssertExpectations(t)
	})
}

// TestScreenTestMockAPIClient_Register проверяет мок метода Register.
func TestScreenTestMockAPIClient_Register(t *testing.T) {
	mockClient := new(ScreenTestMockAPIClient)
	ctx := context.Background()
	username := "newuser"
	password := "newpass"
	expectedErr := errors.New("mock register error")

	// Тест успешной регистрации
	t.Run("Success", func(t *testing.T) {
		mockClient.On("Register", ctx, username, password).Return(nil).Once()
		err := mockClient.Register(ctx, username, password)
		require.NoError(t, err) // Используем require
		mockClient.AssertExpectations(t)
	})

	// Тест ошибки регистрации
	t.Run("Error", func(t *testing.T) {
		mockClient.On("Register", ctx, username, password).Return(expectedErr).Once()
		err := mockClient.Register(ctx, username, password)
		require.Error(t, err) // Используем require
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})
}

// TestScreenTestMockAPIClient_GetVaultMetadata проверяет мок метода GetVaultMetadata.
func TestScreenTestMockAPIClient_GetVaultMetadata(t *testing.T) {
	mockClient := new(ScreenTestMockAPIClient)
	ctx := context.Background()
	// Используем models.VaultVersion и соответствующие поля
	now := time.Now()
	expectedChecksum := "testhash"
	expectedSize := int64(1024)
	expectedMetadata := &models.VaultVersion{
		ID:                1,
		VaultID:           1,
		ObjectKey:         "testkey",
		Checksum:          &expectedChecksum,
		SizeBytes:         &expectedSize,
		CreatedAt:         now,
		ContentModifiedAt: &now,
	}
	expectedErr := errors.New("mock get metadata error")

	// Тест успешного получения метаданных
	t.Run("Success", func(t *testing.T) {
		mockClient.On("GetVaultMetadata", ctx).Return(expectedMetadata, nil).Once()
		metadata, err := mockClient.GetVaultMetadata(ctx)
		require.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)
		mockClient.AssertExpectations(t)
	})

	// Тест ошибки получения метаданных
	t.Run("Error", func(t *testing.T) {
		mockClient.On("GetVaultMetadata", ctx).Return(nil, expectedErr).Once()
		metadata, err := mockClient.GetVaultMetadata(ctx)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, metadata)
		mockClient.AssertExpectations(t)
	})
}

// TestScreenTestMockAPIClient_UploadVault проверяет мок метода UploadVault.
func TestScreenTestMockAPIClient_UploadVault(t *testing.T) {
	mockClient := new(ScreenTestMockAPIClient)
	ctx := context.Background()
	fileContent := []byte("test content")
	reader := bytes.NewReader(fileContent)
	size := int64(len(fileContent))
	modTime := time.Now()
	expectedErr := errors.New("mock upload error")

	// Тест успешной загрузки
	t.Run("Success", func(t *testing.T) {
		mockClient.On("UploadVault", ctx, reader, size, modTime).Return(nil).Once()
		err := mockClient.UploadVault(ctx, reader, size, modTime)
		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	// Тест ошибки загрузки
	t.Run("Error", func(t *testing.T) {
		// Пересоздаем reader, так как он мог быть прочитан в предыдущем тесте
		reader = bytes.NewReader(fileContent)
		mockClient.On("UploadVault", ctx, reader, size, modTime).Return(expectedErr).Once()
		err := mockClient.UploadVault(ctx, reader, size, modTime)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})
}

// TestScreenTestMockAPIClient_DownloadVault проверяет мок метода DownloadVault.
func TestScreenTestMockAPIClient_DownloadVault(t *testing.T) {
	mockClient := new(ScreenTestMockAPIClient)
	ctx := context.Background()
	fileContent := "downloaded content"
	reader := io.NopCloser(strings.NewReader(fileContent)) // Используем io.ReadCloser
	expectedVersion := &models.VaultVersion{
		ID:        3,
		VaultID:   1,
		ObjectKey: "downloadedkey",
	}
	expectedErr := errors.New("mock download error")

	// Тест успешной загрузки
	t.Run("Success", func(t *testing.T) {
		mockClient.On("DownloadVault", ctx).Return(reader, expectedVersion, nil).Once()
		downloadedReader, version, err := mockClient.DownloadVault(ctx)
		require.NoError(t, err)
		assert.Equal(t, expectedVersion, version)
		// Читаем содержимое ридера для проверки
		downloadedBytes, readErr := io.ReadAll(downloadedReader)
		require.NoError(t, readErr)
		assert.Equal(t, fileContent, string(downloadedBytes))
		// Закрываем ридер
		closeErr := downloadedReader.Close()
		require.NoError(t, closeErr)
		mockClient.AssertExpectations(t)
	})

	// Тест ошибки загрузки
	t.Run("Error", func(t *testing.T) {
		mockClient.On("DownloadVault", ctx).Return(nil, nil, expectedErr).Once()
		downloadedReader, version, err := mockClient.DownloadVault(ctx)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, downloadedReader)
		assert.Nil(t, version)
		mockClient.AssertExpectations(t)
	})

	// Тест успешной загрузки только ридера (без версии)
	t.Run("SuccessOnlyReader", func(t *testing.T) {
		// Пересоздаем reader
		reader = io.NopCloser(strings.NewReader(fileContent))
		mockClient.On("DownloadVault", ctx).Return(reader, nil, nil).Once()
		downloadedReader, version, err := mockClient.DownloadVault(ctx)
		require.NoError(t, err)
		assert.Nil(t, version) // Версия должна быть nil
		// Читаем содержимое ридера для проверки
		downloadedBytes, readErr := io.ReadAll(downloadedReader)
		require.NoError(t, readErr)
		assert.Equal(t, fileContent, string(downloadedBytes))
		// Закрываем ридер
		closeErr := downloadedReader.Close()
		require.NoError(t, closeErr)
		mockClient.AssertExpectations(t)
	})
}

// TestScreenTestMockAPIClient_ListVersions проверяет мок метода ListVersions.
func TestScreenTestMockAPIClient_ListVersions(t *testing.T) {
	mockClient := new(ScreenTestMockAPIClient)
	ctx := context.Background()
	limit := 10
	offset := 0
	now := time.Now()
	expectedVersions := []models.VaultVersion{
		{ID: 1, VaultID: 1, ObjectKey: "key1", CreatedAt: now, ContentModifiedAt: &now},
		{ID: 2, VaultID: 1, ObjectKey: "key2", CreatedAt: now.Add(-time.Hour), ContentModifiedAt: &now},
	}
	expectedCount := int64(2)
	expectedErr := errors.New("mock list versions error")

	// Тест успешного получения списка версий
	t.Run("Success", func(t *testing.T) {
		mockClient.On("ListVersions", ctx, limit, offset).Return(expectedVersions, expectedCount, nil).Once()
		versions, count, err := mockClient.ListVersions(ctx, limit, offset)
		require.NoError(t, err)
		assert.Equal(t, expectedVersions, versions)
		assert.Equal(t, expectedCount, count)
		mockClient.AssertExpectations(t)
	})

	// Тест ошибки получения списка версий
	t.Run("Error", func(t *testing.T) {
		mockClient.On("ListVersions", ctx, limit, offset).Return(nil, int64(0), expectedErr).Once()
		versions, count, err := mockClient.ListVersions(ctx, limit, offset)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, versions)
		assert.Zero(t, count)
		mockClient.AssertExpectations(t)
	})
}

// TestScreenTestMockAPIClient_RollbackToVersion проверяет мок метода RollbackToVersion.
func TestScreenTestMockAPIClient_RollbackToVersion(t *testing.T) {
	mockClient := new(ScreenTestMockAPIClient)
	ctx := context.Background()
	versionID := int64(5)
	expectedErr := errors.New("mock rollback error")

	// Тест успешного отката
	t.Run("Success", func(t *testing.T) {
		mockClient.On("RollbackToVersion", ctx, versionID).Return(nil).Once()
		err := mockClient.RollbackToVersion(ctx, versionID)
		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	// Тест ошибки отката
	t.Run("Error", func(t *testing.T) {
		mockClient.On("RollbackToVersion", ctx, versionID).Return(expectedErr).Once()
		err := mockClient.RollbackToVersion(ctx, versionID)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})
}

// TestScreenTestMockAPIClient_SetAuthToken проверяет мок метода SetAuthToken.
func TestScreenTestMockAPIClient_SetAuthToken(t *testing.T) {
	mockClient := new(ScreenTestMockAPIClient)
	token := "new-auth-token"

	mockClient.On("SetAuthToken", token).Return().Once()
	mockClient.SetAuthToken(token)
	mockClient.AssertExpectations(t)
}

// TestScreenTestSuite_BuilderMethods проверяет методы-конструкторы ScreenTestSuite.
func TestScreenTestSuite_BuilderMethods(t *testing.T) {
	s := NewScreenTestSuite() // Создаем тестовый набор

	// Проверяем WithServerURL
	t.Run("WithServerURL", func(t *testing.T) {
		url := "http://localhost:8080"
		s.WithServerURL(url)
		assert.Equal(t, url, s.Model.serverURL, "Model.serverURL должен быть обновлен")
	})

	// Проверяем WithAuthToken
	t.Run("WithAuthToken", func(t *testing.T) {
		token := "builder-token"
		s.WithAuthToken(token)
		assert.Equal(t, token, s.Model.authToken, "Model.authToken должен быть обновлен")
	})

	// Проверяем WithDatabase
	t.Run("WithDatabase", func(t *testing.T) {
		db := CreateBasicTestDB() // Используем хелпер для создания тестовой БД
		s.WithDatabase(db)
		assert.Equal(t, db, s.Model.db, "Model.db должен быть обновлен")
	})
}

// TestScreenTestSuite_AssertViewContains проверяет хелпер AssertViewContains.
func TestScreenTestSuite_AssertViewContains(t *testing.T) {
	s := NewScreenTestSuite() // Создаем тестовый набор
	// Устанавливаем состояние, для которого View() известен
	s.WithState(welcomeScreen)

	// Ожидаемые подстроки в welcomeScreen View
	expectedSubstring1 := "GophKeeper"
	expectedSubstring2 := "Добро пожаловать"

	// Проверяем позитивный случай: AssertViewContains не должен паниковать или фейлить тест,
	// если подстрока присутствует.
	s.AssertViewContains(t, expectedSubstring1)
	s.AssertViewContains(t, expectedSubstring2)

	// Негативный случай (когда подстрока отсутствует) не проверяем явно,
	// так как он полагается на корректную работу assert.Contains из testify,
	// которая вызовет t.FailNow(), прервав тест, что сложно перехватить.
}

// TestScreenTestSuite_CaptureView проверяет хелпер CaptureView.
func TestScreenTestSuite_CaptureView(t *testing.T) {
	t.Run("With Command", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.WithState(welcomeScreen)

		// Команда, которая меняет состояние с welcome на passwordInput
		changeStateCmd := func() tea.Msg {
			return tea.KeyMsg{Type: tea.KeyEnter}
		}

		// Ожидаем View от passwordInputScreen
		view := s.CaptureView(t, changeStateCmd)

		// Проверяем, что View соответствует новому состоянию
		assert.Contains(t, view, "Введите мастер-пароль", "View должен соответствовать passwordInputScreen")
		// Проверяем, что состояние модели действительно изменилось
		s.AssertState(t, passwordInputScreen)
	})

	t.Run("With Nil Command", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.WithState(welcomeScreen)

		// Ожидаем View от welcomeScreen
		view := s.CaptureView(t, nil)

		// Проверяем, что View соответствует исходному состоянию
		assert.Contains(t, view, "Добро пожаловать", "View должен соответствовать welcomeScreen")
		// Проверяем, что состояние модели не изменилось
		s.AssertState(t, welcomeScreen)
	})
}

// TestScreenTestSuite_CaptureOutput проверяет хелпер CaptureOutput.
func TestScreenTestSuite_CaptureOutput(t *testing.T) {
	t.Run("Sequence of Commands", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.WithState(welcomeScreen)

		// Команды: Enter (welcome -> passwordInput), Escape (passwordInput -> welcome)
		cmd1 := func() tea.Msg { return tea.KeyMsg{Type: tea.KeyEnter} }
		cmd2 := func() tea.Msg { return tea.KeyMsg{Type: tea.KeyEscape} }

		// Ожидаем View от welcomeScreen (финальное состояние)
		view := s.CaptureOutput(t, cmd1, cmd2)

		assert.Contains(t, view, "Добро пожаловать", "View должен соответствовать финальному welcomeScreen")
		s.AssertState(t, welcomeScreen)
	})

	t.Run("With Nil Commands", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.WithState(passwordInputScreen)

		cmd1 := func() tea.Msg { return tea.KeyMsg{Type: tea.KeyEnter} } // Должен сработать

		// Ожидаем View от passwordInputScreen, так как nil команды игнорируются,
		// а Enter на этом экране не меняет состояние (если не введен пароль).
		// Точнее, Enter вызовет команду openKdbxCmd, которая может изменить состояние асинхронно,
		// но CaptureOutput выполняет команды синхронно. Поэтому состояние останется passwordInputScreen.
		view := s.CaptureOutput(t, nil, cmd1, nil)

		assert.Contains(t, view, "Введите мастер-пароль", "View должен остаться passwordInputScreen")
		s.AssertState(t, passwordInputScreen)
	})

	t.Run("No Commands", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.WithState(entryListScreen)

		view := s.CaptureOutput(t)

		assert.Contains(t, view, "No items", "View должен остаться entryListScreen (и быть пустым)")
		s.AssertState(t, entryListScreen)
	})
}

// TestScreenTestSuite_SetupMockAPILogin проверяет хелпер SetupMockAPILogin.
func TestScreenTestSuite_SetupMockAPILogin(t *testing.T) {
	s := NewScreenTestSuite()
	username := "test-user"
	password := "test-pass"
	ctx := context.Background()

	t.Run("Success Case", func(t *testing.T) {
		expectedToken := "success-token"
		response := MockResponse{Success: true, Token: expectedToken}
		s.SetupMockAPILogin(username, password, response)

		// Проверяем, что мок настроен правильно, вызвав метод
		token, err := s.Mocks.APIClient.Login(ctx, username, password)
		require.NoError(t, err)
		assert.Equal(t, expectedToken, token)
		s.Mocks.APIClient.AssertExpectations(t) // Убедимся, что вызов был
	})

	t.Run("Error Case", func(t *testing.T) {
		expectedErr := errors.New("login setup error")
		response := MockResponse{Success: false, Error: expectedErr}
		s.SetupMockAPILogin(username, password, response)

		// Проверяем, что мок настроен правильно
		token, err := s.Mocks.APIClient.Login(ctx, username, password)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Empty(t, token)
		s.Mocks.APIClient.AssertExpectations(t)
	})
}

// TestScreenTestSuite_SetupMockAPIRegister проверяет хелпер SetupMockAPIRegister.
func TestScreenTestSuite_SetupMockAPIRegister(t *testing.T) {
	s := NewScreenTestSuite()
	username := "reg-user"
	password := "reg-pass"
	ctx := context.Background()

	t.Run("Success Case", func(t *testing.T) {
		response := MockResponse{Success: true}
		s.SetupMockAPIRegister(username, password, response)

		// Проверяем, что мок настроен правильно
		err := s.Mocks.APIClient.Register(ctx, username, password)
		require.NoError(t, err)
		s.Mocks.APIClient.AssertExpectations(t)
	})

	t.Run("Error Case", func(t *testing.T) {
		expectedErr := errors.New("register setup error")
		response := MockResponse{Success: false, Error: expectedErr}
		s.SetupMockAPIRegister(username, password, response)

		// Проверяем, что мок настроен правильно
		err := s.Mocks.APIClient.Register(ctx, username, password)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		s.Mocks.APIClient.AssertExpectations(t)
	})
}

// TestScreenTestSuite_RenderScreen проверяет хелпер RenderScreen.
func TestScreenTestSuite_RenderScreen(t *testing.T) {
	s := NewScreenTestSuite()
	// Используем состояние welcomeScreen для предсказуемого View
	s.WithState(welcomeScreen)

	// Ожидаемый результат ПОСЛЕ очистки ANSI и \r функцией RenderScreen
	part1 := "Добро пожаловать в GophKeeper!\n\nНажмите Enter для продолжения..."
	part2 := "\n(Enter - продолжить, Ctrl+C/q - выход)"
	expectedCleanView := part1 + part2

	actualCleanView := s.RenderScreen()

	// Убираем лишние пробелы в конце строк, которые могут появиться из-за рендеринга
	lines := strings.Split(actualCleanView, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	actualCleanViewTrimmed := strings.Join(lines, "\n")

	// Сравниваем очищенные строки
	assert.Equal(t, expectedCleanView, actualCleanViewTrimmed, "RenderScreen должен возвращать очищенный View")
}
