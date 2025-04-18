package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// updatePasswordInputScreen обрабатывает сообщения для экрана ввода пароля.
func (m *model) updatePasswordInputScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Сначала обновляем поле ввода
	m.passwordInput, cmd = m.passwordInput.Update(msg)
	cmds = append(cmds, cmd)

	// Обработка клавиш для экрана ввода пароля
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Если была ошибка, любое нажатие ее скрывает
		if m.err != nil {
			m.err = nil
			m.passwordInput.Focus() // Возвращаем фокус
			cmds = append(cmds, textinput.Blink)
			// Не обрабатываем другие клавиши в этом цикле
		} else if keyMsg.String() == keyEnter {
			password := m.passwordInput.Value()
			m.passwordInput.Blur()
			m.passwordInput.Reset()
			// Сохраняем пароль в модели перед отправкой команды
			m.password = password
			cmds = append(cmds, openKdbxCmd(m.kdbxPath, password))
		}
	}
	return m, tea.Batch(cmds...)
}

// viewPasswordInputScreen отрисовывает экран ввода пароля.
func (m *model) viewPasswordInputScreen() string {
	var s strings.Builder
	s.WriteString("Введите мастер-пароль для " + m.kdbxPath + ":\n")
	s.WriteString(m.passwordInput.View() + "\n\n")
	if m.err != nil {
		errMsgStr := fmt.Sprintf("\nОшибка: %s\n\n(Нажмите любую клавишу для продолжения)", m.err)
		return s.String() + errMsgStr // Возвращаем основной текст + текст ошибки
	}
	return s.String()
}

// handleErrorMsg обрабатывает сообщение об ошибке.
func (m *model) handleErrorMsg(msg errMsg) tea.Model /*, tea.Cmd */ {
	// Устанавливаем статус ошибки и возвращаемся к экрану ввода пароля
	m.err = msg.err // Сохраняем ошибку в модели
	m.state = passwordInputScreen
	m.passwordInput.Focus()
	return m // , nil // Больше не возвращаем команду
}
