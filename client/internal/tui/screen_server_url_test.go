//nolint:testpackage // Это тесты в том же пакете для доступа к приватным компонентам
package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateServerURLInputScreen проверяет обработку сообщений на экране ввода URL сервера.
func TestUpdateServerURLInputScreen(t *testing.T) {
	// Создаем модель напрямую (более простой подход)
	t.Run("ОтменаВводаURL", func(t *testing.T) {
		// Инициализируем модель напрямую
		m := &model{
			state:          serverURLInputScreen,
			serverURL:      "https://original.server",
			serverURLInput: textinput.New(),
		}

		// Имитируем нажатие Esc
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		newModel, cmd := m.updateServerURLInputScreen(msg)
		model, ok := newModel.(*model)
		require.True(t, ok, "Должен быть возвращен указатель на model")

		// Проверяем результаты
		assert.Equal(t, syncServerScreen, model.state, "Должен произойти переход на экран синхронизации")
		assert.Equal(t, "https://original.server", model.serverURL, "URL не должен измениться")
		assert.Nil(t, cmd, "Не должно быть возвращено команды")
	})

	t.Run("ПодтверждениеВводаURL", func(t *testing.T) {
		// Инициализируем модель напрямую
		m := &model{
			state:          serverURLInputScreen,
			serverURLInput: textinput.New(),
		}

		// Устанавливаем значение в поле ввода
		newURL := "https://new.server"
		m.serverURLInput.SetValue(newURL)

		// Имитируем нажатие Enter
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := m.updateServerURLInputScreen(msg)
		model, ok := newModel.(*model)
		require.True(t, ok, "Должен быть возвращен указатель на model")

		// Проверяем результаты
		assert.Equal(t, loginRegisterChoiceScreen, model.state, "Должен произойти переход на экран выбора входа/регистрации")
		assert.Equal(t, newURL, model.serverURL, "URL должен обновиться")
		assert.NotNil(t, cmd, "Должна быть возвращена команда")
		assert.NotNil(t, model.apiClient, "API клиент должен быть инициализирован")
		assert.Equal(t, "Не выполнен", model.loginStatus, "Статус логина должен быть сброшен")
		assert.Equal(t, "", model.authToken, "Токен авторизации должен быть сброшен")
	})

	t.Run("ИспользованиеПлейсхолдера", func(t *testing.T) {
		// Инициализируем модель напрямую
		m := &model{
			state:          serverURLInputScreen,
			serverURLInput: textinput.New(),
		}

		// Устанавливаем плейсхолдер, но оставляем пустое значение
		placeholder := "https://default.server"
		m.serverURLInput.Placeholder = placeholder
		m.serverURLInput.SetValue("")

		// Имитируем нажатие Enter
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.updateServerURLInputScreen(msg)
		model, ok := newModel.(*model)
		require.True(t, ok, "Должен быть возвращен указатель на model")

		// Проверяем результаты
		assert.Equal(t, loginRegisterChoiceScreen, model.state, "Должен произойти переход на экран выбора входа/регистрации")
		assert.Equal(t, placeholder, model.serverURL, "URL должен быть установлен из плейсхолдера")
	})

	t.Run("ОбновлениеПоляВвода", func(t *testing.T) {
		// Создаем модель с полем ввода
		input := textinput.New()
		input.Focus() // Важно: фокусируем поле, чтобы оно могло принимать ввод

		m := &model{
			state:          serverURLInputScreen,
			serverURLInput: input,
		}

		// Имитируем ввод символа 'a'
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		newM, _ := m.updateServerURLInputScreen(keyMsg)
		model, ok := newM.(*model)
		require.True(t, ok, "Не удалось привести tea.Model к *model")

		// Команда может быть nil, но само поле должно обновиться
		assert.NotNil(t, model.serverURLInput, "Поле ввода должно существовать")
		assert.Contains(t, model.serverURLInput.Value(), "a", "Символ 'a' должен быть добавлен в поле ввода")
	})
}

// TestViewServerURLInputScreen проверяет отображение экрана ввода URL сервера.
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

// TestDirectServerURLInputScreen проверяет работу функций экрана напрямую без TestSuite.
func TestDirectServerURLInputScreen(t *testing.T) {
	t.Run("ОтменаВвода", func(t *testing.T) {
		// Создаем модель напрямую
		m := &model{
			state:          serverURLInputScreen,
			serverURL:      "https://original.server",
			serverURLInput: textinput.New(),
		}

		// Создаем сообщение о нажатии Esc
		msg := tea.KeyMsg{Type: tea.KeyEsc}

		// Вызываем функцию обновления
		newModel, cmd := m.updateServerURLInputScreen(msg)
		model, ok := newModel.(*model)

		// Проверяем результаты
		assert.True(t, ok, "Должен быть возвращен указатель на model")
		assert.Equal(t, syncServerScreen, model.state, "Должен произойти переход на экран синхронизации")
		assert.Equal(t, "https://original.server", model.serverURL, "URL не должен измениться")
		assert.Nil(t, cmd, "Не должно быть возвращено команды")
	})

	t.Run("ПодтверждениеПустогоВвода", func(t *testing.T) {
		// Создаем модель
		m := &model{
			state:          serverURLInputScreen,
			serverURLInput: textinput.New(),
		}

		// Устанавливаем плейсхолдер и пустое значение
		placeholder := "https://default.server"
		m.serverURLInput.Placeholder = placeholder
		m.serverURLInput.SetValue("")

		// Создаем сообщение о нажатии Enter
		msg := tea.KeyMsg{Type: tea.KeyEnter}

		// Вызываем функцию обновления
		newModel, _ := m.updateServerURLInputScreen(msg)
		model, ok := newModel.(*model)

		// Проверяем результаты
		assert.True(t, ok, "Должен быть возвращен указатель на model")
		assert.Equal(t, placeholder, model.serverURL, "URL должен быть установлен из плейсхолдера")
	})
}
