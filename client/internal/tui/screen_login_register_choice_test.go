//nolint:testpackage // Это тесты в том же пакете для доступа к приватным компонентам
package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestUpdateLoginRegisterChoiceScreen проверяет обработку сообщений на экране выбора входа/регистрации.
func TestUpdateLoginRegisterChoiceScreen(t *testing.T) {
	// Выводим значения констант для отладки
	t.Logf("Значения констант - syncServerScreen: %d, loginRegisterChoiceScreen: %d",
		syncServerScreen, loginRegisterChoiceScreen)

	t.Run("ПереходНаЭкранРегистрации", func(t *testing.T) {
		// Инициализируем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(loginRegisterChoiceScreen)

		// Инициализируем поле ввода для регистрации
		suite.Model.registerUsernameInput = textinput.New()

		// Имитируем нажатие клавиши 'r'
		newModel, cmd := suite.SimulateKeyRune('r')
		model := asModel(t, newModel)

		// Проверяем результаты
		assert.Equal(t, registerScreen, model.state, "Должен произойти переход на экран регистрации")
		assert.Equal(t, 0, model.loginRegisterFocusedField, "Должно быть выбрано первое поле")
		assert.True(t, model.registerUsernameInput.Focused(), "Поле имени пользователя должно получить фокус")
		assert.NotNil(t, cmd, "Должна быть возвращена команда")
	})

	t.Run("ПереходНаЭкранРегистрации_ВерхнийРегистр", func(t *testing.T) {
		// Инициализируем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(loginRegisterChoiceScreen)

		// Инициализируем поле ввода для регистрации
		suite.Model.registerUsernameInput = textinput.New()

		// Имитируем нажатие клавиши 'R' (верхний регистр)
		newModel, cmd := suite.SimulateKeyRune('R')
		model := asModel(t, newModel)

		// Проверяем результаты
		assert.Equal(t, registerScreen, model.state, "Должен произойти переход на экран регистрации")
		assert.Equal(t, 0, model.loginRegisterFocusedField, "Должно быть выбрано первое поле")
		assert.True(t, model.registerUsernameInput.Focused(), "Поле имени пользователя должно получить фокус")
		assert.NotNil(t, cmd, "Должна быть возвращена команда")
	})

	t.Run("ПереходНаЭкранВхода", func(t *testing.T) {
		// Инициализируем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(loginRegisterChoiceScreen)

		// Инициализируем поле ввода для входа
		suite.Model.loginUsernameInput = textinput.New()

		// Имитируем нажатие клавиши 'l'
		newModel, cmd := suite.SimulateKeyRune('l')
		model := asModel(t, newModel)

		// Проверяем результаты
		assert.Equal(t, loginScreen, model.state, "Должен произойти переход на экран входа")
		assert.Equal(t, 0, model.loginRegisterFocusedField, "Должно быть выбрано первое поле")
		assert.True(t, model.loginUsernameInput.Focused(), "Поле имени пользователя должно получить фокус")
		assert.NotNil(t, cmd, "Должна быть возвращена команда")
	})

	t.Run("ПереходНаЭкранВхода_ВерхнийРегистр", func(t *testing.T) {
		// Инициализируем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(loginRegisterChoiceScreen)

		// Инициализируем поле ввода для входа
		suite.Model.loginUsernameInput = textinput.New()

		// Имитируем нажатие клавиши 'L' (верхний регистр)
		newModel, cmd := suite.SimulateKeyRune('L')
		model := asModel(t, newModel)

		// Проверяем результаты
		assert.Equal(t, loginScreen, model.state, "Должен произойти переход на экран входа")
		assert.Equal(t, 0, model.loginRegisterFocusedField, "Должно быть выбрано первое поле")
		assert.True(t, model.loginUsernameInput.Focused(), "Поле имени пользователя должно получить фокус")
		assert.NotNil(t, cmd, "Должна быть возвращена команда")
	})

	t.Run("ВозвратНаСписокЗаписей", func(t *testing.T) {
		// Инициализируем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(loginRegisterChoiceScreen)

		// Имитируем нажатие Esc
		newModel, cmd := suite.SimulateKeyPress(tea.KeyEsc)
		model := asModel(t, newModel)

		// Проверяем результаты - Фактически возвращается на syncServerScreen (ID 9)
		assert.Equal(t, syncServerScreen, model.state, "Должен произойти переход на экран синхронизации и сервера")
		assert.Nil(t, cmd, "Не должно быть возвращено команды")
	})

	t.Run("ВозвратНаСписокЗаписей_Backspace", func(t *testing.T) {
		m := &model{
			state: loginRegisterChoiceScreen,
		}

		// Используем keyBack ("b") вместо KeyBackspace
		updatedModel, cmd := m.updateLoginRegisterChoiceScreen(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'b'},
		})

		model, ok := updatedModel.(*model)
		assert.True(t, ok, "Должен быть возвращен указатель на model")
		assert.Equal(t, syncServerScreen, model.state, "Должен произойти переход на экран синхронизации и сервера")
		assert.Nil(t, cmd, "Не должно быть возвращено команды")
	})

	t.Run("ИгнорированиеДругихКлавиш", func(t *testing.T) {
		// Инициализируем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(loginRegisterChoiceScreen)

		// Сохраняем начальное состояние
		initialState := suite.Model.state

		// Имитируем нажатие другой клавиши
		newModel, cmd := suite.SimulateKeyRune('x')
		model := asModel(t, newModel)

		// Проверяем результаты
		assert.Equal(t, initialState, model.state, "Состояние не должно измениться")
		assert.Nil(t, cmd, "Не должно быть возвращено команды")
	})
}

// TestViewLoginRegisterChoiceScreen проверяет отображение экрана выбора входа/регистрации.
func TestViewLoginRegisterChoiceScreen(t *testing.T) {
	// Инициализируем модель
	m := &model{
		state:     loginRegisterChoiceScreen,
		serverURL: "https://test.server",
	}

	// Вызываем View
	view := m.viewLoginRegisterChoiceScreen()

	// Проверяем результаты
	assert.Contains(t, view, "Настройка сервера", "View должен содержать заголовок")
	assert.Contains(t, view, "https://test.server", "View должен содержать URL сервера")
	assert.Contains(t, view, "Регистрация нового пользователя", "View должен содержать опцию регистрации")
	assert.Contains(t, view, "Вход с существующими данными", "View должен содержать опцию входа")
	assert.Contains(t, view, "(R)", "View должен содержать горячую клавишу для регистрации")
	assert.Contains(t, view, "(L)", "View должен содержать горячую клавишу для входа")
}
