package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// updateRegisterScreen обрабатывает ввод данных для регистрации.
func (m *model) updateRegisterScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	registerAction := func() (tea.Model, tea.Cmd) {
		username := m.registerUsernameInput.Value()
		password := m.registerPasswordInput.Value()
		// Вызываем команду для выполнения регистрации (создадим её позже)
		cmd := m.makeRegisterCmd(username, password)
		m, statusCmd := m.setStatusMessage("Выполняется регистрация...")
		return m, tea.Batch(cmd, statusCmd)
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

// viewRegisterScreen отображает экран ввода данных для регистрации.
func (m *model) viewRegisterScreen() string {
	// Используем общую функцию
	return m.viewCredentialsScreen(
		"Регистрация новой учетной записи",
		"Нажмите Enter для регистрации, Esc для возврата",
		m.registerUsernameInput,
		m.registerPasswordInput,
	)
}
