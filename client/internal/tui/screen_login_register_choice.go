package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
