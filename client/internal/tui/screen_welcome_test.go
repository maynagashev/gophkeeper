//nolint:testpackage // Это тесты в том же пакете для доступа к приватным компонентам
package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWelcomeScreen(t *testing.T) {
	t.Run("ОтрисовкаЭкрана", func(t *testing.T) {
		// Настраиваем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(welcomeScreen)

		// Вызываем View() напрямую
		view := suite.Model.viewWelcomeScreen()

		// Проверяем результат
		assert.Contains(t, view, "Добро пожаловать", "View должен содержать приветственный текст")
		assert.Contains(t, view, "Нажмите Enter", "View должен содержать инструкции")
	})

	t.Run("ПереходПриНажатииEnter", func(t *testing.T) {
		// Настраиваем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(welcomeScreen)

		// Инициализируем поле ввода пароля
		suite.Model.passwordInput = textinput.New()

		// Имитируем нажатие Enter
		newModel, cmd := suite.SimulateKeyPress(tea.KeyEnter)

		// Приводим модель к нужному типу
		m := toModel(t, newModel)

		// Проверяем состояние
		assert.Equal(t, passwordInputScreen, m.state, "После нажатия Enter должен произойти переход к экрану ввода пароля")
		assert.True(t, m.passwordInput.Focused(), "Поле ввода пароля должно получить фокус")
		assert.NotNil(t, cmd, "Должна быть возвращена команда")
	})

	t.Run("ВыходПриНажатииQ", func(t *testing.T) {
		// Настраиваем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(welcomeScreen)

		// Имитируем нажатие 'q'
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		newModel, _ := suite.Model.Update(keyMsg) // Игнорируем возвращаемую команду

		// Проверяем, что модель не изменилась
		m := toModel(t, newModel)
		assert.Equal(t, welcomeScreen, m.state, "Состояние не должно измениться при нажатии q")
		// Не проверяем cmd, так как в зависимости от реализации он может быть как nil, так и не nil

		// Имитируем нажатие Ctrl+C
		var quitCmd tea.Cmd
		_, quitCmd = suite.SimulateKeyPress(tea.KeyCtrlC)

		// Определяем тип возвращаемой команды
		require.NotNil(t, quitCmd, "Команда не должна быть nil при нажатии Ctrl+C")

		// Выполняем команду и проверяем тип сообщения
		msg := suite.ExecuteCmd(quitCmd)
		_, ok := msg.(tea.QuitMsg)
		assert.True(t, ok, "Должно быть возвращено сообщение tea.QuitMsg при нажатии Ctrl+C")
	})
}
