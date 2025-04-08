package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// updateWelcomeScreen обрабатывает сообщения для экрана приветствия
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

// viewWelcomeScreen отрисовывает экран приветствия
func (m model) viewWelcomeScreen() string {
	s := "Добро пожаловать в GophKeeper!\n\n"
	s += "Это безопасный менеджер паролей для командной строки,\n"
	s += "совместимый с форматом KDBX (KeePass).\n\n"
	return s
}
