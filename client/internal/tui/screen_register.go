package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// updateRegisterScreen обрабатывает ввод данных для регистрации.
func (m *model) updateRegisterScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	registerAction := func() (tea.Model, tea.Cmd) {
		// TODO: Call API register m.registerUsernameInput.Value(), m.registerPasswordInput.Value()
		return m.setStatusMessage("TODO: Регистрация...")
	}

	return m.handleCredentialsInput(
		msg,
		&m.registerUsernameInput,
		&m.registerPasswordInput,
		&m.loginRegisterFocusedField,
		registerAction,
		loginRegisterChoiceScreen, // Возвращаемся к выбору при Esc
	)
}
