package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// viewCredentialsScreen отображает общий экран ввода данных (логин/пароль).
func (m *model) viewCredentialsScreen(title, hint string, usernameInput, passwordInput textinput.Model) string {
	var b strings.Builder

	// Определяем стили здесь, чтобы избежать дублирования в каждом вызывающем месте
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA"))
	subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))    // Серый
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F25D94")) // Красный для ошибок

	b.WriteString(titleStyle.Render(title) + "\n\n")
	b.WriteString(usernameInput.View() + "\n")
	b.WriteString(passwordInput.View() + "\n\n")
	b.WriteString(subtleStyle.Render(hint) + "\n")
	if m.err != nil {
		b.WriteString(errorStyle.Render("Ошибка: "+m.err.Error()) + "\n")
	}
	return b.String()
}
