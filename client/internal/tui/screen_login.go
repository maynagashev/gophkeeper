package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// updateLoginScreen обрабатывает ввод данных для входа.
func (m *model) updateLoginScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	loginAction := func() (tea.Model, tea.Cmd) {
		// TODO: Call API login m.loginUsernameInput.Value(), m.loginPasswordInput.Value()
		return m.setStatusMessage("TODO: Логин...")
	}

	return m.handleCredentialsInput(
		msg,
		&m.loginUsernameInput,
		&m.loginPasswordInput,
		&m.loginRegisterFocusedField,
		loginAction,
		loginRegisterChoiceScreen, // Возвращаемся к выбору при Esc
	)
}
