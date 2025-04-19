package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	// Количество полей, обрабатываемых handleCredentialsInput (имя/пароль).
	numCredentialFields = 2
)

// handleCredentialsKeys обрабатывает нажатия Tab, Shift+Tab и Enter в полях ввода.
// Возвращает модель, команду и флаг, указывающий, была ли клавиша обработана.
func (m *model) handleCredentialsKeys(
	keyMsg tea.KeyMsg,
	input1 *textinput.Model,
	input2 *textinput.Model,
	focusedFieldIdx *int,
	onEnterCmd func() (tea.Model, tea.Cmd),
) (tea.Model, tea.Cmd, bool) {
	switch keyMsg.String() {
	case keyTab:
		*focusedFieldIdx = (*focusedFieldIdx + 1) % numCredentialFields
		if *focusedFieldIdx == 0 {
			input2.Blur()
			input1.Focus()
		} else {
			input1.Blur()
			input2.Focus()
		}
		return m, textinput.Blink, true // Клавиша обработана
	case keyShiftTab:
		*focusedFieldIdx = (*focusedFieldIdx + numCredentialFields - 1) % numCredentialFields
		if *focusedFieldIdx == 0 {
			input2.Blur()
			input1.Focus()
		} else {
			input1.Blur()
			input2.Focus()
		}
		return m, textinput.Blink, true // Клавиша обработана
	case keyEnter:
		if *focusedFieldIdx == 0 { // Активно первое поле
			*focusedFieldIdx = 1
			input1.Blur()
			input2.Focus()
			return m, textinput.Blink, true // Клавиша обработана (переход фокуса)
		}
		// Активно второе поле - вызываем действие
		model, cmd := onEnterCmd()
		return model, cmd, true // Клавиша обработана (вызов действия)
	default:
		return m, nil, false // Клавиша не обработана этим хендлером
	}
}

// handleCredentialsInput обрабатывает ввод в двух полях (например, имя/пароль),
// переключение фокуса между ними и действия по Enter/Esc.
func (m *model) handleCredentialsInput(
	msg tea.Msg,
	input1 *textinput.Model,
	input2 *textinput.Model,
	focusedFieldIdx *int,
	onEnterCmd func() (tea.Model, tea.Cmd),
	previousState screenState,
) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Сначала обрабатываем Esc
		if keyMsg.String() == keyEsc {
			m.state = previousState
			input1.Blur()
			input2.Blur()
			return m, tea.ClearScreen
		}

		// Затем обрабатываем Tab, Shift+Tab, Enter
		newModel, keyCmd, handled := m.handleCredentialsKeys(keyMsg, input1, input2, focusedFieldIdx, onEnterCmd)
		if handled {
			return newModel, keyCmd
		}
	}

	// Если это не Esc или другая обработанная клавиша, обновляем активное поле ввода
	currentIdx := *focusedFieldIdx
	activeInput := input1
	if currentIdx == 1 {
		activeInput = input2
	}
	*activeInput, cmd = activeInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}
