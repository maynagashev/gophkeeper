//nolint:testpackage // Это тесты в том же пакете для доступа к приватным компонентам
package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateServerURLInputScreen проверяет обработку сообщений на экране ввода URL сервера
func TestUpdateServerURLInputScreen(t *testing.T) {
	t.Run("ОтменаВводаURL", func(t *testing.T) {
		// Инициализируем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(serverURLInputScreen)

		// Устанавливаем исходный URL
		originalURL := "https://original.server"
		suite.Model.serverURL = originalURL

		// Имитируем нажатие Esc
		newModel, cmd := suite.SimulateKeyPress(tea.KeyEsc)
		model := toModel(t, newModel)

		// Проверяем результаты
		assert.Equal(t, syncServerScreen, model.state, "Должен произойти переход на экран синхронизации")
		assert.Equal(t, originalURL, model.serverURL, "URL не должен измениться")
		assert.Nil(t, cmd, "Не должно быть возвращено команды")
	})

	t.Run("ПодтверждениеВводаURL", func(t *testing.T) {
		// Инициализируем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(serverURLInputScreen)

		// Устанавливаем значение в поле ввода
		newURL := "https://new.server"
		suite.Model.serverURLInput.SetValue(newURL)

		// Имитируем нажатие Enter
		newModel, cmd := suite.SimulateKeyPress(tea.KeyEnter)
		model := toModel(t, newModel)

		// Проверяем результаты
		assert.Equal(t, loginRegisterChoiceScreen, model.state, "Должен произойти переход на экран выбора входа/регистрации")
		assert.Equal(t, newURL, model.serverURL, "URL должен обновиться")
		assert.NotNil(t, cmd, "Должна быть возвращена команда")
		assert.NotNil(t, model.apiClient, "API клиент должен быть инициализирован")
	})

	t.Run("ИспользованиеПлейсхолдера", func(t *testing.T) {
		// Инициализируем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(serverURLInputScreen)

		// Устанавливаем плейсхолдер, но оставляем пустое значение
		placeholder := "https://default.server"
		suite.Model.serverURLInput.Placeholder = placeholder
		suite.Model.serverURLInput.SetValue("")

		// Имитируем нажатие Enter
		newModel, cmd := suite.SimulateKeyPress(tea.KeyEnter)
		model := toModel(t, newModel)

		// Проверяем результаты
		assert.Equal(t, loginRegisterChoiceScreen, model.state, "Должен произойти переход на экран выбора входа/регистрации")
		assert.Equal(t, placeholder, model.serverURL, "URL должен быть установлен из плейсхолдера")
		assert.NotNil(t, cmd, "Должна быть возвращена команда")
	})

	t.Run("ОбновлениеПоляВвода", func(t *testing.T) {
		// Инициализируем модель
		m := &model{
			state:          serverURLInputScreen,
			serverURLInput: textinput.New(),
		}

		// Имитируем ввод символа 'a'
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		newM, cmd := m.updateServerURLInputScreen(keyMsg)
		model, ok := newM.(*model)
		require.True(t, ok, "Не удалось привести tea.Model к *model")

		// Проверяем результаты
		assert.Equal(t, "a", model.serverURLInput.Value(), "Символ должен быть добавлен в поле ввода")
		assert.NotNil(t, cmd, "Должна быть возвращена команда")
	})
}

// TestViewServerURLInputScreen проверяет отображение экрана ввода URL сервера
func TestViewServerURLInputScreen(t *testing.T) {
	// Инициализируем модель
	m := &model{
		state:          serverURLInputScreen,
		serverURLInput: textinput.New(),
	}

	// Устанавливаем значение в поле ввода
	testURL := "https://test.server"
	m.serverURLInput.SetValue(testURL)

	// Вызываем View
	view := m.viewServerURLInputScreen()

	// Проверяем результаты
	assert.Contains(t, view, "Введите URL сервера")
	assert.Contains(t, view, testURL)
}
