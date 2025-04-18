package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// updateLoginRegisterChoiceScreen обрабатывает выбор между входом и регистрацией.
func (m *model) updateLoginRegisterChoiceScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "r", "R":
			m.state = registerScreen
			m.registerUsernameInput.Focus()
			m.loginRegisterFocusedField = 0
			return m, textinput.Blink
		case "l", "L":
			m.state = loginScreen
			m.loginUsernameInput.Focus()
			m.loginRegisterFocusedField = 0
			return m, textinput.Blink
		case keyEsc, keyBack:
			m.state = entryListScreen // Возвращаемся к списку записей
			return m, nil
		}
	}
	// Если сообщение не было обработано (не keyMsg или не нужная клавиша)
	return m, nil
}
