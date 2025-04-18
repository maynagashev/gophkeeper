package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	// Количество полей, обрабатываемых handleCredentialsInput (имя/пароль).
	numCredentialFields = 2
)

// handleCredentialsInput обрабатывает ввод в двух полях (например, имя/пароль),
// переключение фокуса между ними и действия по Enter/Esc.
func (m *model) handleCredentialsInput(
	msg tea.Msg,
	input1 *textinput.Model, // Первое поле (например, username)
	input2 *textinput.Model, // Второе поле (например, password)
	focusedFieldIdx *int, // Указатель на индекс активного поля (0 или 1)
	onEnterCmd func() (tea.Model, tea.Cmd), // Функция, вызываемая по Enter
	previousState screenState, // Состояние для возврата по Esc
) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	// Обработка клавиши Esc для возврата
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == keyEsc {
		m.state = previousState
		input1.Blur()
		input2.Blur()
		return m, nil
	}

	currentIdx := *focusedFieldIdx
	activeInput := input1
	if currentIdx == 1 {
		activeInput = input2
	}

	// Обновляем активное поле ввода
	*activeInput, cmd = activeInput.Update(msg)
	cmds = append(cmds, cmd)

	// Обработка переключения фокуса и Enter
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyTab:
			*focusedFieldIdx = (*focusedFieldIdx + 1) % numCredentialFields // Используем константу
			if *focusedFieldIdx == 0 {
				input2.Blur()
				input1.Focus()
			} else {
				input1.Blur()
				input2.Focus()
			}
			cmds = append(cmds, textinput.Blink)
		case keyShiftTab:
			*focusedFieldIdx = (*focusedFieldIdx + 1) % numCredentialFields // Используем константу
			if *focusedFieldIdx == 0 {
				input2.Blur()
				input1.Focus()
			} else {
				input1.Blur()
				input2.Focus()
			}
			cmds = append(cmds, textinput.Blink)
		case keyEnter:
			// Выполняем специфичное действие (логин или регистрация)
			return onEnterCmd()
		}
	}

	return m, tea.Batch(cmds...)
}
