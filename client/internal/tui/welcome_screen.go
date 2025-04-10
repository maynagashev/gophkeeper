package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// updateWelcomeScreen обрабатывает сообщения для экрана приветствия.
func (m *model) updateWelcomeScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyQuit:
			return m, tea.Quit
		case keyEnter:
			m.state = passwordInputScreen
			m.passwordInput.Focus()
			cmds = append(cmds, textinput.Blink, tea.ClearScreen)
		}
	}
	return m, tea.Batch(cmds...)
}

// viewWelcomeScreen отрисовывает приветственный экран.
func (m *model) viewWelcomeScreen() string {
	return "Добро пожаловать в GophKeeper!\n\nНажмите Enter для продолжения..."
}
