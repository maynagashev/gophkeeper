package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

// updateLoginRegisterChoiceScreen обрабатывает выбор между входом и регистрацией.
func (m *model) updateLoginRegisterChoiceScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "r", "R":
			m.state = registerScreen
			m.registerUsernameInput.Focus()
			m.loginRegisterFocusedField = 0
			return m, tea.Batch(textinput.Blink, tea.ClearScreen)
		case "l", "L":
			m.state = loginScreen
			m.loginUsernameInput.Focus()
			m.loginRegisterFocusedField = 0
			return m, tea.Batch(textinput.Blink, tea.ClearScreen)
		case keyEsc, keyBack:
			m.state = entryListScreen // Возвращаемся к списку записей
			return m, nil
		}
	}
	// Если сообщение не было обработано (не keyMsg или не нужная клавиша)
	return m, nil
}

// viewLoginRegisterChoiceScreen отображает экран выбора входа или регистрации.
func (m *model) viewLoginRegisterChoiceScreen() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA"))
	focusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")) // Пурпурный
	subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))  // Серый

	b.WriteString(titleStyle.Render("Настройка сервера") + "\n\n")
	b.WriteString("Сервер настроен: " + m.serverURL + "\n\n") // Показываем настроенный URL
	b.WriteString("Выберите действие:\n")
	b.WriteString("- Регистрация нового пользователя " + focusedStyle.Render("(R)") + "\n")
	b.WriteString("- Вход с существующими данными " + focusedStyle.Render("(L)") + "\n\n")
	b.WriteString(subtleStyle.Render("Нажмите Esc для возврата"))

	return b.String()
}

//nolint:testpackage // Тесты в том же пакете для доступа к приватным компонентам
func TestLoginRegisterChoiceScreen_SimulateKeyPress(t *testing.T) {
	t.Run("ПереходНаЭкранРегистрации", func(t *testing.T) {
		// Создаем модель
		m := &model{
			state:                 loginRegisterChoiceScreen,
			registerUsernameInput: textinput.New(),
		}

		// Создаем сообщение о нажатии клавиши 'r'
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}

		// Вызываем функцию обновления
		newModel, cmd := m.updateLoginRegisterChoiceScreen(msg)

		// Проверяем результаты
		model, ok := newModel.(*model)
		assert.True(t, ok, "Должен быть возвращен указатель на model")
		assert.Equal(t, registerScreen, model.state, "Должно произойти переключение на экран регистрации")
		assert.Equal(t, 0, model.loginRegisterFocusedField, "Должно быть выбрано первое поле")
		assert.True(t, model.registerUsernameInput.Focused(), "Поле имени пользователя должно получить фокус")
		assert.NotNil(t, cmd, "Должна быть возвращена команда")
	})

	t.Run("ПереходНаЭкранВхода", func(t *testing.T) {
		// Создаем модель
		m := &model{
			state:              loginRegisterChoiceScreen,
			loginUsernameInput: textinput.New(),
		}

		// Создаем сообщение о нажатии клавиши 'l'
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}

		// Вызываем функцию обновления
		newModel, cmd := m.updateLoginRegisterChoiceScreen(msg)

		// Проверяем результаты
		model, ok := newModel.(*model)
		assert.True(t, ok, "Должен быть возвращен указатель на model")
		assert.Equal(t, loginScreen, model.state, "Должно произойти переключение на экран входа")
		assert.Equal(t, 0, model.loginRegisterFocusedField, "Должно быть выбрано первое поле")
		assert.True(t, model.loginUsernameInput.Focused(), "Поле имени пользователя должно получить фокус")
		assert.NotNil(t, cmd, "Должна быть возвращена команда")
	})

	t.Run("ВозвратНаСписокЗаписей", func(t *testing.T) {
		// Создаем модель
		m := &model{
			state: loginRegisterChoiceScreen,
		}

		// Создаем сообщение о нажатии Esc
		msg := tea.KeyMsg{Type: tea.KeyEsc}

		// Вызываем функцию обновления
		newModel, cmd := m.updateLoginRegisterChoiceScreen(msg)

		// Проверяем результаты
		model, ok := newModel.(*model)
		assert.True(t, ok, "Должен быть возвращен указатель на model")
		assert.Equal(t, entryListScreen, model.state, "Должно произойти переключение на экран списка записей")
		assert.Nil(t, cmd, "Не должно быть возвращено команды")
	})

	t.Run("ИгнорированиеДругихКлавиш", func(t *testing.T) {
		// Создаем модель
		m := &model{
			state: loginRegisterChoiceScreen,
		}

		// Сохраняем начальное состояние
		initialState := m.state

		// Создаем сообщение о нажатии другой клавиши
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}

		// Вызываем функцию обновления
		newModel, cmd := m.updateLoginRegisterChoiceScreen(msg)

		// Проверяем результаты
		model, ok := newModel.(*model)
		assert.True(t, ok, "Должен быть возвращен указатель на model")
		assert.Equal(t, initialState, model.state, "Состояние не должно измениться")
		assert.Nil(t, cmd, "Не должно быть возвращено команды")
	})
}

func TestLoginRegisterChoiceScreen_View(t *testing.T) {
	// Создаем модель
	m := &model{
		state:     loginRegisterChoiceScreen,
		serverURL: "https://test.server",
	}

	// Вызываем функцию отображения
	view := m.viewLoginRegisterChoiceScreen()

	// Проверяем результаты
	assert.Contains(t, view, "Настройка сервера", "View должен содержать заголовок")
	assert.Contains(t, view, "https://test.server", "View должен содержать URL сервера")
	assert.Contains(t, view, "Регистрация нового пользователя", "View должен содержать опцию регистрации")
	assert.Contains(t, view, "Вход с существующими данными", "View должен содержать опцию входа")
	assert.Contains(t, view, "(R)", "View должен содержать горячую клавишу для регистрации")
	assert.Contains(t, view, "(L)", "View должен содержать горячую клавишу для входа")
}
