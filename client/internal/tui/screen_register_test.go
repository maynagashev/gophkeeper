package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateRegisterScreen проверяет обработку сообщений на экране регистрации
func TestUpdateRegisterScreen(t *testing.T) {
	tests := []struct {
		name            string
		inputMsg        tea.Msg
		initialField    int
		expectedField   int
		expectedState   screenState
		expectedCmd     bool
		usernameFocused bool
		passwordFocused bool
		initModel       func(m *model)
	}{
		{
			name:            "ПереключениеПоляВперед",
			inputMsg:        tea.KeyMsg{Type: tea.KeyTab},
			initialField:    0,
			expectedField:   1,
			expectedState:   registerScreen,
			expectedCmd:     false,
			usernameFocused: false,
			passwordFocused: true,
			initModel:       func(m *model) {},
		},
		{
			name:            "ПереключениеПоляНазад",
			inputMsg:        tea.KeyMsg{Type: tea.KeyShiftTab},
			initialField:    1,
			expectedField:   0,
			expectedState:   registerScreen,
			expectedCmd:     false,
			usernameFocused: true,
			passwordFocused: false,
			initModel:       func(m *model) {},
		},
		{
			name:            "ОтменаРегистрации",
			inputMsg:        tea.KeyMsg{Type: tea.KeyEsc},
			initialField:    0,
			expectedField:   0,
			expectedState:   loginRegisterChoiceScreen,
			expectedCmd:     false,
			usernameFocused: false,
			passwordFocused: false,
			initModel:       func(m *model) {},
		},
		{
			name:            "НажатиеEnter_ПервоеПоле",
			inputMsg:        tea.KeyMsg{Type: tea.KeyEnter},
			initialField:    0,
			expectedField:   1,
			expectedState:   registerScreen,
			expectedCmd:     false,
			usernameFocused: false,
			passwordFocused: true,
			initModel:       func(m *model) {},
		},
		{
			name:            "НажатиеEnter_ВтороеПоле_ОтправкаФормы",
			inputMsg:        tea.KeyMsg{Type: tea.KeyEnter},
			initialField:    1,
			expectedField:   1,
			expectedState:   registerScreen,
			expectedCmd:     true,
			usernameFocused: false,
			passwordFocused: true,
			initModel: func(m *model) {
				m.registerUsernameInput.SetValue("testuser")
				m.registerPasswordInput.SetValue("testpass")
				m.serverURL = "http://test.server"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &model{
				state:                     registerScreen,
				loginRegisterFocusedField: tt.initialField,
				registerUsernameInput:     textinput.New(),
				registerPasswordInput:     textinput.New(),
			}

			// Инициализируем модель с помощью функции из тест-кейса
			tt.initModel(m)

			// Устанавливаем фокус на поле, указанное в initialField
			updateInputFocus(m, tt.initialField)

			newM, cmd := m.updateRegisterScreen(tt.inputMsg)
			model, ok := newM.(*model)
			require.True(t, ok, "Не удалось привести tea.Model к *model")

			assert.Equal(t, tt.expectedState, model.state)
			assert.Equal(t, tt.expectedField, model.loginRegisterFocusedField)
			assert.Equal(t, tt.usernameFocused, model.registerUsernameInput.Focused())
			assert.Equal(t, tt.passwordFocused, model.registerPasswordInput.Focused())

			if tt.expectedCmd {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}
		})
	}
}

// TestViewRegisterScreen проверяет корректность отображения экрана регистрации
func TestViewRegisterScreen(t *testing.T) {
	m := &model{
		state:                     registerScreen,
		loginRegisterFocusedField: 0,
		registerUsernameInput:     textinput.New(),
		registerPasswordInput:     textinput.New(),
		serverURL:                 "http://test.server",
	}

	// Устанавливаем значения
	m.registerUsernameInput.SetValue("testuser")
	m.registerPasswordInput.SetValue("testpass")

	view := m.viewRegisterScreen()

	assert.Contains(t, view, "Регистрация")
	assert.Contains(t, view, "http://test.server")
}

// updateInputFocus вспомогательная функция для установки фокуса на нужное поле
func updateInputFocus(m *model, field int) {
	m.registerUsernameInput.Blur()
	m.registerPasswordInput.Blur()

	switch field {
	case 0:
		m.registerUsernameInput.Focus()
	case 1:
		m.registerPasswordInput.Focus()
	}
}
