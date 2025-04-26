package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"

	"gophkeeper/models"
)

// initTestModel создает модель с базовыми настройками для тестирования
func initTestModel(t *testing.T) *model {
	t.Helper()

	m := &model{
		entries:                      make([]models.Entry, 0),
		vaultPath:                    "",
		logBuffer:                    make([]string, 0, maxLogLines),
		loginUsernameInput:           makeTextInput("Имя пользователя", ""),
		loginPasswordInput:           makePasswordInput("Пароль", ""),
		registerUsernameInput:        makeTextInput("Имя пользователя", ""),
		registerPasswordInput:        makePasswordInput("Пароль", ""),
		registerPasswordConfirmInput: makePasswordInput("Подтверждение пароля", ""),
		serverURLInput:               makeTextInput("URL Сервера", ""),
		settingsVaultPathInput:       makeTextInput("Путь к хранилищу", ""),
		settingsMasterPasswordInput:  makePasswordInput("Мастер-пароль", ""),
		newEntryTitleInput:           makeTextInput("Название", ""),
		newEntryUsernameInput:        makeTextInput("Логин", ""),
		newEntryPasswordInput:        makePasswordInput("Пароль", ""),
		newEntryUrlInput:             makeTextInput("URL", ""),
		newEntryNotesInput:           makeTextInput("Заметки", ""),
		width:                        80,
		height:                       24,
		showNewEntryForm:             false,
		showPasswordChar:             true,
		serverURL:                    "https://example.com",
		statusBadge:                  lipgloss.NewStyle().Padding(0, 1).Bold(true),
		pageBadge:                    lipgloss.NewStyle().Padding(0, 1).Bold(true),
	}

	return m
}

// pressKey имитирует нажатие клавиши и возвращает обновленную модель
func pressKey(t *testing.T, m tea.Model, key string) (tea.Model, tea.Cmd) {
	t.Helper()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return newModel, cmd
}

// pressSpecialKey имитирует нажатие специальной клавиши и возвращает обновленную модель
func pressSpecialKey(t *testing.T, m tea.Model, key tea.KeyType) (tea.Model, tea.Cmd) {
	t.Helper()

	newModel, cmd := m.Update(tea.KeyMsg{Type: key})
	return newModel, cmd
}

// asModel выполняет безопасное приведение типа tea.Model к *model с проверкой
func asModel(t *testing.T, m tea.Model) *model {
	t.Helper()

	model, ok := m.(*model)
	require.True(t, ok, "Не удалось привести tea.Model к *model")
	return model
}
