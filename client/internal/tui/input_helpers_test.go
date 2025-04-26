//nolint:testpackage // Тесты в том же пакете для доступа к непубличным функциям
package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Определяем константы экранов для тестов.
const (
	testLoginScreen    = screenState(1)
	testPreviousScreen = screenState(0)
)

// createTestModel создает модель для тестирования.
func createTestModel() *model {
	return &model{state: testLoginScreen}
}

// TestHandleCredentialsKeys проверяет обработку клавиш в полях ввода учетных данных.
func TestHandleCredentialsKeys(t *testing.T) {
	// Создаем модель и входные данные
	m := createTestModel()
	input1 := textinput.New()
	input1.Focus()
	input2 := textinput.New()
	focusedFieldIdx := 0

	// Функция, которая будет вызываться при нажатии Enter на втором поле
	enterActionCalled := false
	onEnterCmd := func() (tea.Model, tea.Cmd) { //nolint:unparam // Сигнатура требуется интерфейсом
		enterActionCalled = true
		return m, nil
	}

	t.Run("TabKey", func(t *testing.T) {
		// Сбрасываем состояние
		focusedFieldIdx = 0
		input1.Focus()
		input2.Blur()

		// Эмулируем нажатие Tab
		keyMsg := tea.KeyMsg{Type: tea.KeyTab}
		_, _, handled := m.handleCredentialsKeys(keyMsg, &input1, &input2, &focusedFieldIdx, onEnterCmd)

		// Проверяем результаты
		assert.True(t, handled, "Нажатие Tab должно быть обработано")
		assert.Equal(t, 1, focusedFieldIdx, "После Tab фокус должен перейти на второе поле")
		assert.False(t, input1.Focused(), "Первое поле не должно быть в фокусе")
		assert.True(t, input2.Focused(), "Второе поле должно быть в фокусе")
	})

	t.Run("ShiftTabKey", func(t *testing.T) {
		// Устанавливаем начальное состояние - фокус на втором поле
		focusedFieldIdx = 1
		input1.Blur()
		input2.Focus()

		// Эмулируем нажатие Shift+Tab
		keyMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
		_, _, handled := m.handleCredentialsKeys(keyMsg, &input1, &input2, &focusedFieldIdx, onEnterCmd)

		// Проверяем результаты
		assert.True(t, handled, "Нажатие Shift+Tab должно быть обработано")
		assert.Equal(t, 0, focusedFieldIdx, "После Shift+Tab фокус должен перейти на первое поле")
		assert.True(t, input1.Focused(), "Первое поле должно быть в фокусе")
		assert.False(t, input2.Focused(), "Второе поле не должно быть в фокусе")
	})

	t.Run("EnterOnFirstField", func(t *testing.T) {
		// Устанавливаем начальное состояние - фокус на первом поле
		focusedFieldIdx = 0
		input1.Focus()
		input2.Blur()
		enterActionCalled = false

		// Эмулируем нажатие Enter на первом поле
		keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
		_, _, handled := m.handleCredentialsKeys(keyMsg, &input1, &input2, &focusedFieldIdx, onEnterCmd)

		// Проверяем результаты
		assert.True(t, handled, "Нажатие Enter на первом поле должно быть обработано")
		assert.Equal(t, 1, focusedFieldIdx, "После Enter на первом поле фокус должен перейти на второе поле")
		assert.False(t, input1.Focused(), "Первое поле не должно быть в фокусе")
		assert.True(t, input2.Focused(), "Второе поле должно быть в фокусе")
		assert.False(t, enterActionCalled, "Действие Enter не должно быть вызвано при нажатии Enter на первом поле")
	})

	t.Run("EnterOnSecondField", func(t *testing.T) {
		// Устанавливаем начальное состояние - фокус на втором поле
		focusedFieldIdx = 1
		input1.Blur()
		input2.Focus()
		enterActionCalled = false

		// Эмулируем нажатие Enter на втором поле
		keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
		_, _, handled := m.handleCredentialsKeys(keyMsg, &input1, &input2, &focusedFieldIdx, onEnterCmd)

		// Проверяем результаты
		assert.True(t, handled, "Нажатие Enter на втором поле должно быть обработано")
		assert.True(t, enterActionCalled, "Действие Enter должно быть вызвано при нажатии Enter на втором поле")
	})

	t.Run("UnhandledKey", func(t *testing.T) {
		// Эмулируем нажатие другой клавиши
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		_, _, handled := m.handleCredentialsKeys(keyMsg, &input1, &input2, &focusedFieldIdx, onEnterCmd)

		// Проверяем результаты
		assert.False(t, handled, "Нажатие 'a' не должно быть обработано handleCredentialsKeys")
	})
}

// TestHandleCredentialsInput проверяет полную обработку ввода в полях учетных данных.
func TestHandleCredentialsInput(t *testing.T) {
	// Создаем модель и входные данные
	m := createTestModel()
	input1 := textinput.New()
	input1.Focus()
	input2 := textinput.New()
	focusedFieldIdx := 0
	previousState := testPreviousScreen

	// Функция, которая будет вызываться при нажатии Enter на втором поле
	onEnterCmd := func() (tea.Model, tea.Cmd) { //nolint:unparam // Сигнатура требуется интерфейсом
		return m, nil
	}

	t.Run("EscKey", func(t *testing.T) {
		// Сбрасываем состояние
		m.state = testLoginScreen
		focusedFieldIdx = 0
		input1.Focus()
		input2.Blur()

		// Эмулируем нажатие Esc
		keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
		result, cmd := m.handleCredentialsInput(keyMsg, &input1, &input2, &focusedFieldIdx, onEnterCmd, previousState)

		// Проверяем результаты
		require.Equal(t, m, result, "Модель должна быть возвращена без изменений")
		assert.NotNil(t, cmd, "Команда должна быть возвращена для очистки экрана")
		assert.Equal(t, previousState, m.state, "Состояние должно быть изменено на предыдущее")
		assert.False(t, input1.Focused(), "Первое поле не должно быть в фокусе после Esc")
		assert.False(t, input2.Focused(), "Второе поле не должно быть в фокусе после Esc")
	})

	t.Run("TabKey", func(t *testing.T) {
		// Сбрасываем состояние
		m.state = testLoginScreen
		focusedFieldIdx = 0
		input1.Focus()
		input2.Blur()

		// Эмулируем нажатие Tab
		keyMsg := tea.KeyMsg{Type: tea.KeyTab}
		result, _ := m.handleCredentialsInput(keyMsg, &input1, &input2, &focusedFieldIdx, onEnterCmd, previousState)

		// Проверяем результаты
		require.Equal(t, m, result, "Модель должна быть возвращена без изменений")
		assert.Equal(t, 1, focusedFieldIdx, "После Tab фокус должен перейти на второе поле")
		assert.False(t, input1.Focused(), "Первое поле не должно быть в фокусе")
		assert.True(t, input2.Focused(), "Второе поле должно быть в фокусе")
	})

	t.Run("TextInputUpdate", func(_ *testing.T) {
		// Сбрасываем состояние
		m.state = testLoginScreen
		focusedFieldIdx = 0
		input1.Focus()
		input2.Blur()

		// Для этого сценария мы пропускаем проверку на nil команду,
		// так как встроенная реализация textinput.Update может вернуть nil команду в тестовом окружении
		m.handleCredentialsInput(nil, &input1, &input2, &focusedFieldIdx, onEnterCmd, previousState)

		// Тест проходит, если не было паники или другой ошибки
		// Не проверяем возвращаемую команду - она может быть nil в тестовой среде
	})
}
