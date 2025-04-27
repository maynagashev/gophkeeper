package tui //nolint:testpackage // Тесты в том же пакете для доступа к непубличным функциям

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/client/internal/api"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tobischo/gokeepasslib/v3"
)

// Вспомогательная функция для безопасного приведения типа.
func asModel(t *testing.T, m tea.Model) *model {
	t.Helper()
	result, ok := m.(*model)
	require.True(t, ok, "Модель должна быть типа *model")
	return result
}

// MockAPIClient - мок для API клиента.
type MockAPIClient struct {
	mock.Mock
	api.Client
}

func (m *MockAPIClient) SetAuthToken(token string) {
	m.Called(token)
}

// ListVersions mocks base method.
func (m *MockAPIClient) ListVersions(ctx context.Context, limit, offset int) ([]models.VaultVersion, int64, error) {
	args := m.Called(ctx, limit, offset)
	var versions []models.VaultVersion
	var currentID int64
	var err error

	if v := args.Get(0); v != nil {
		var ok bool
		versions, ok = v.([]models.VaultVersion)
		if !ok {
			panic("mock: ListVersions: argument 0 is not []models.VaultVersion")
		}
	}

	if id := args.Get(1); id != nil {
		var ok bool
		currentID, ok = id.(int64)
		if !ok {
			panic("mock: ListVersions: argument 1 is not int64")
		}
	}

	err = args.Error(2)

	return versions, currentID, err
}

// TestHandleAPIMessages проверяет обработку различных сообщений API.
func TestHandleAPIMessages(t *testing.T) {
	t.Run("УспешныйВход", func(t *testing.T) {
		// Создаем базовую модель для тестирования
		m := createTestModelForUpdate()

		// Создаем мок API клиента
		mockClient := new(MockAPIClient)
		mockClient.On("SetAuthToken", "test-token-12345").Return()
		m.apiClient = mockClient

		// Создаем полноценную структуру БД
		db := gokeepasslib.NewDatabase()
		db.Content = &gokeepasslib.DBContent{
			Meta: &gokeepasslib.MetaData{},
			Root: &gokeepasslib.RootData{
				Groups: []gokeepasslib.Group{
					{
						Name: "Root",
					},
				},
			},
		}
		db.Credentials = gokeepasslib.NewPasswordCredentials("test_password")
		m.db = db

		m.loginUsernameInput.SetValue("testuser")
		m.loginPasswordInput.SetValue("password123")
		m.state = loginScreen

		// Создаем сообщение успешного входа
		msg := loginSuccessMsg{Token: "test-token-12345"}

		// Проверяем обработку
		newM, cmd, handled := handleAPIMsg(m, msg)
		updatedModel := asModel(t, newM)

		// Проверяем результаты
		require.True(t, handled, "Сообщение должно быть обработано")
		require.NotNil(t, cmd, "Должна быть возвращена команда")
		require.Equal(t, "test-token-12345", updatedModel.authToken, "Токен должен быть сохранен")
		require.Contains(t, updatedModel.loginStatus, "Вход выполнен", "Статус должен содержать сообщение об успешном входе")
		require.Equal(t, entryListScreen, updatedModel.state, "Должен быть выполнен переход на экран списка")
		require.Empty(t, updatedModel.loginUsernameInput.Value(), "Поле имени пользователя должно быть очищено")
		require.Empty(t, updatedModel.loginPasswordInput.Value(), "Поле пароля должно быть очищено")

		// Проверяем, что метод SetAuthToken был вызван с правильным параметром
		mockClient.AssertExpectations(t)
	})

	t.Run("ОшибкаВхода", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.state = loginScreen

		// Создаем сообщение ошибки входа
		testErr := errors.New("неверный логин или пароль")
		msg := LoginError{err: testErr}

		// Проверяем обработку
		newM, cmd, handled := handleAPIMsg(m, msg)
		updatedModel := asModel(t, newM)

		// Проверяем результаты
		require.True(t, handled, "Сообщение должно быть обработано")
		require.NotNil(t, cmd, "Должна быть возвращена команда")
		require.Equal(t, testErr, updatedModel.err, "Ошибка должна быть сохранена в модели")
		require.Equal(t, loginScreen, updatedModel.state, "Должны остаться на экране входа")
	})

	t.Run("УспешнаяРегистрация", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.state = registerScreen
		m.registerUsernameInput.SetValue("newuser")
		m.registerPasswordInput.SetValue("password123")

		// Создаем сообщение успешной регистрации
		msg := registerSuccessMsg{}

		// Проверяем обработку
		newM, cmd, handled := handleAPIMsg(m, msg)
		updatedModel := asModel(t, newM)

		// Проверяем результаты
		require.True(t, handled, "Сообщение должно быть обработано")
		require.NotNil(t, cmd, "Должна быть возвращена команда")
		require.Empty(t, updatedModel.registerUsernameInput.Value(), "Поле имени пользователя должно быть очищено")
		require.Empty(t, updatedModel.registerPasswordInput.Value(), "Поле пароля должно быть очищено")
		require.Equal(t, loginScreen, updatedModel.state, "Должен быть выполнен переход на экран входа")
		require.NoError(t, updatedModel.err, "Ошибки не должно быть")
	})

	t.Run("ОшибкаРегистрации", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.state = registerScreen

		// Создаем сообщение ошибки регистрации
		testErr := errors.New("пользователь уже существует")
		msg := RegisterError{err: testErr}

		// Проверяем обработку
		newM, cmd, handled := handleAPIMsg(m, msg)
		updatedModel := asModel(t, newM)

		// Проверяем результаты
		require.True(t, handled, "Сообщение должно быть обработано")
		require.NotNil(t, cmd, "Должна быть возвращена команда")
		require.Equal(t, testErr, updatedModel.err, "Ошибка должна быть сохранена в модели")
		require.Equal(t, registerScreen, updatedModel.state, "Должны остаться на экране регистрации")
	})

	t.Run("НеобрабатываемоеСообщение", func(t *testing.T) {
		m := createTestModelForUpdate()

		// Создаем необрабатываемое сообщение
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}

		// Проверяем обработку
		_, _, handled := handleAPIMsg(m, msg)

		// Проверяем результаты
		assert.False(t, handled, "Сообщение не должно быть обработано")
	})
}
