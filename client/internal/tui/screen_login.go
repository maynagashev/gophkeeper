package tui

import (
	tea "github.com/charmbracelet/bubbletea"
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
