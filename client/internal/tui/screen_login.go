package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// updateLoginScreen обрабатывает ввод данных для входа.
func (m *model) updateLoginScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	loginAction := func() (tea.Model, tea.Cmd) {
		username := m.loginUsernameInput.Value()
		password := m.loginPasswordInput.Value()
		// Вызываем команду для выполнения входа
		cmd := m.makeLoginCmd(username, password)
		// Можно добавить сообщение о начале процесса входа
		m, statusCmd := m.setStatusMessage("Выполняется вход...")
		return m, tea.Batch(cmd, statusCmd) // Возвращаем команду входа и команду статуса
	}

	return m.handleCredentialsInput(
		msg,
		&m.loginUsernameInput,
		&m.loginPasswordInput,
		&m.loginRegisterFocusedField,
		loginAction,               // Передаем нашу функцию loginAction
		loginRegisterChoiceScreen, // Возвращаемся к выбору при Esc
	)
}

// viewLoginScreen отображает экран ввода данных для входа.
func (m *model) viewLoginScreen() string {
	// Используем общую функцию
	return m.viewCredentialsScreen(
		"Вход в учетную запись",
		"Нажмите Enter для входа, Esc для возврата",
		m.loginUsernameInput,
		m.loginPasswordInput,
	)
}

//nolint:testpackage // Тесты в том же пакете для доступа к приватным компонентам
func TestLoginScreen_TabNavigation(t *testing.T) {
	t.Run("ПереключениеФокусаПоTab", func(t *testing.T) {
		// Создаем модель
		m := &model{
			state:                     loginScreen,
			loginRegisterFocusedField: 0,
			loginUsernameInput:        textinput.New(),
			loginPasswordInput:        textinput.New(),
		}

		// Устанавливаем фокус на первое поле
		m.loginUsernameInput.Focus()
		m.loginPasswordInput.Blur()

		// Создаем сообщение о нажатии Tab
		msg := tea.KeyMsg{Type: tea.KeyTab}

		// Вызываем функцию обновления
		newModel, _ := m.updateLoginScreen(msg)
		model, ok := newModel.(*model)

		// Проверяем результаты
		assert.True(t, ok, "Должен быть возвращен указатель на model")
		assert.Equal(t, 1, model.loginRegisterFocusedField, "Фокус должен переключиться на поле пароля")
		assert.False(t, model.loginUsernameInput.Focused(), "Поле логина должно потерять фокус")
		assert.True(t, model.loginPasswordInput.Focused(), "Поле пароля должно получить фокус")
	})

	t.Run("ПереключениеФокусаПоShiftTab", func(t *testing.T) {
		// Создаем модель
		m := &model{
			state:                     loginScreen,
			loginRegisterFocusedField: 1,
			loginUsernameInput:        textinput.New(),
			loginPasswordInput:        textinput.New(),
		}

		// Устанавливаем фокус на второе поле
		m.loginUsernameInput.Blur()
		m.loginPasswordInput.Focus()

		// Создаем сообщение о нажатии Shift+Tab
		msg := tea.KeyMsg{Type: tea.KeyShiftTab}

		// Вызываем функцию обновления
		newModel, _ := m.updateLoginScreen(msg)
		model, ok := newModel.(*model)

		// Проверяем результаты
		assert.True(t, ok, "Должен быть возвращен указатель на model")
		assert.Equal(t, 0, model.loginRegisterFocusedField, "Фокус должен переключиться на поле логина")
		assert.True(t, model.loginUsernameInput.Focused(), "Поле логина должно получить фокус")
		assert.False(t, model.loginPasswordInput.Focused(), "Поле пароля должно потерять фокус")
	})
}

func TestLoginScreen_EscapeKey(t *testing.T) {
	// Создаем модель
	m := &model{
		state:                     loginScreen,
		loginRegisterFocusedField: 0,
		loginUsernameInput:        textinput.New(),
		loginPasswordInput:        textinput.New(),
	}

	// Создаем сообщение о нажатии Esc
	msg := tea.KeyMsg{Type: tea.KeyEsc}

	// Вызываем функцию обновления
	newModel, _ := m.updateLoginScreen(msg)
	model, ok := newModel.(*model)

	// Проверяем результаты
	assert.True(t, ok, "Должен быть возвращен указатель на model")
	assert.Equal(t, loginRegisterChoiceScreen, model.state, "Должен произойти переход на экран выбора")
}

func TestLoginScreen_EnterSubmit(t *testing.T) {
	// Создаем модель
	m := &model{
		state:                     loginScreen,
		loginRegisterFocusedField: 1, // Фокус на поле пароля
		loginUsernameInput:        textinput.New(),
		loginPasswordInput:        textinput.New(),
		serverURL:                 "http://test.server",
	}

	// Устанавливаем значения полей
	m.loginUsernameInput.SetValue("testuser")
	m.loginPasswordInput.SetValue("testpass")

	// Создаем сообщение о нажатии Enter
	msg := tea.KeyMsg{Type: tea.KeyEnter}

	// Вызываем функцию обновления
	newModel, cmd := m.updateLoginScreen(msg)

	// Проверяем результаты
	assert.NotNil(t, newModel, "Должна быть возвращена модель")
	assert.NotNil(t, cmd, "Должна быть возвращена команда")
}

func TestLoginScreen_View(t *testing.T) {
	// Создаем модель
	m := &model{
		state:              loginScreen,
		loginUsernameInput: textinput.New(),
		loginPasswordInput: textinput.New(),
		serverURL:          "http://test.server",
	}

	// Устанавливаем значения полей
	m.loginUsernameInput.SetValue("testuser")
	m.loginPasswordInput.SetValue("testpass")

	// Вызываем функцию отображения
	view := m.viewLoginScreen()

	// Проверяем результаты
	assert.Contains(t, view, "Вход в учетную запись", "View должен содержать заголовок")
	assert.Contains(t, view, "http://test.server", "View должен содержать URL сервера")
}
