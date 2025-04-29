//nolint:testpackage // Тесты в том же пакете для доступа к приватным компонентам
package tui

import (
	"path/filepath"
	"testing"

	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Вспомогательная функция для создания модели в нужном состоянии.
func setupNewKdbxTest(t *testing.T) (*model, string) {
	t.Helper()
	// Создаем временный файл для теста
	tmpDir := t.TempDir()
	tmpFilePath := filepath.Join(tmpDir, "test_new.kdbx")

	m := &model{
		state:                   newKdbxPasswordScreen,
		kdbxPath:                tmpFilePath,
		newPasswordInput1:       textinput.New(),
		newPasswordInput2:       textinput.New(),
		newPasswordFocusedField: 0,               // Начинаем с фокуса на первом поле
		entryList:               initEntryList(), // ИНИЦИАЛИЗИРУЕМ СПИСОК
	}
	m.newPasswordInput1.Focus() // Устанавливаем фокус явно
	m.newPasswordInput1.EchoMode = textinput.EchoPassword
	m.newPasswordInput2.EchoMode = textinput.EchoPassword
	m.newPasswordInput1.Placeholder = "Введите новый пароль"
	m.newPasswordInput2.Placeholder = "Подтвердите пароль"

	return m, tmpFilePath
}

// TestViewNewKdbxPasswordScreen проверяет отображение экрана ввода пароля для нового KDBX.
func TestViewNewKdbxPasswordScreen(t *testing.T) {
	m, tmpFilePath := setupNewKdbxTest(t)

	t.Run("Базовое отображение (фокус на первом поле)", func(t *testing.T) {
		view := m.viewNewKdbxPasswordScreen()
		assert.Contains(t, view, "Создание нового файла KDBX: "+tmpFilePath)
		assert.Contains(t, view, "> "+m.newPasswordInput1.View()) // Фокус на первом
		assert.Contains(t, view, "  "+m.newPasswordInput2.View()) // Нет фокуса на втором
		assert.NotContains(t, view, "Пароли не совпадают!")       // Нет ошибки
	})

	t.Run("Отображение с фокусом на втором поле", func(t *testing.T) {
		m.newPasswordFocusedField = 1
		m.newPasswordInput1.Blur()
		m.newPasswordInput2.Focus()
		view := m.viewNewKdbxPasswordScreen()
		assert.Contains(t, view, "  "+m.newPasswordInput1.View()) // Нет фокуса на первом
		assert.Contains(t, view, "> "+m.newPasswordInput2.View()) // Фокус на втором
		assert.NotContains(t, view, "Пароли не совпадают!")       // Нет ошибки
	})

	t.Run("Отображение с ошибкой совпадения паролей", func(t *testing.T) {
		m.confirmPasswordError = "Пароли не совпадают!"
		view := m.viewNewKdbxPasswordScreen()
		assert.Contains(t, view, "Пароли не совпадают!") // Есть ошибка
		// Сбросим ошибку для других тестов
		m.confirmPasswordError = ""
	})
}

// TestUpdateNewKdbxPasswordScreen проверяет обработку сообщений на экране ввода пароля нового KDBX.
func TestUpdateNewKdbxPasswordScreen(t *testing.T) {
	t.Run("Выход по Esc", func(t *testing.T) {
		m, _ := setupNewKdbxTest(t)
		keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
		newM, cmd := m.updateNewKdbxPasswordScreen(keyMsg)
		assert.Same(t, m, newM, "Модель не должна меняться")
		assert.NotNil(t, cmd, "Должна быть возвращена команда (предположительно Quit)")
	})

	t.Run("Выход по Ctrl+C", func(t *testing.T) {
		m, _ := setupNewKdbxTest(t)
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
		newM, cmd := m.updateNewKdbxPasswordScreen(keyMsg)
		assert.Same(t, m, newM, "Модель не должна меняться")
		assert.NotNil(t, cmd, "Должна быть возвращена команда (предположительно Quit)")
	})

	t.Run("Переключение фокуса Tab/Down", func(t *testing.T) {
		m, _ := setupNewKdbxTest(t)
		assert.Equal(t, 0, m.newPasswordFocusedField, "Начальный фокус на первом поле")
		assert.True(t, m.newPasswordInput1.Focused())
		assert.False(t, m.newPasswordInput2.Focused())

		// Tab
		keyMsgTab := tea.KeyMsg{Type: tea.KeyTab}
		newM, cmd := m.updateNewKdbxPasswordScreen(keyMsgTab)
		model := asModel(t, newM)
		assert.NotNil(t, cmd)
		assert.Equal(t, 1, model.newPasswordFocusedField, "Фокус должен перейти на второе поле")
		assert.False(t, model.newPasswordInput1.Focused())
		assert.True(t, model.newPasswordInput2.Focused())

		// Down (аналогично Tab)
		keyMsgDown := tea.KeyMsg{Type: tea.KeyDown}
		newM, cmd = model.updateNewKdbxPasswordScreen(keyMsgDown)
		model = asModel(t, newM)
		assert.NotNil(t, cmd)
		assert.Equal(t, 0, model.newPasswordFocusedField, "Фокус должен вернуться на первое поле")
		assert.True(t, model.newPasswordInput1.Focused())
		assert.False(t, model.newPasswordInput2.Focused())
	})

	t.Run("Переключение фокуса Shift+Tab/Up", func(t *testing.T) {
		m, _ := setupNewKdbxTest(t)
		// Ставим фокус на второе поле для начала
		m.newPasswordFocusedField = 1
		m.newPasswordInput1.Blur()
		m.newPasswordInput2.Focus()
		assert.Equal(t, 1, m.newPasswordFocusedField)

		// Shift+Tab
		keyMsgShiftTab := tea.KeyMsg{Type: tea.KeyShiftTab}
		newM, cmd := m.updateNewKdbxPasswordScreen(keyMsgShiftTab)
		model := asModel(t, newM)
		assert.NotNil(t, cmd)
		assert.Equal(t, 0, model.newPasswordFocusedField, "Фокус должен перейти на первое поле")
		assert.True(t, model.newPasswordInput1.Focused())
		assert.False(t, model.newPasswordInput2.Focused())

		// Up (аналогично Shift+Tab)
		keyMsgUp := tea.KeyMsg{Type: tea.KeyUp}
		newM, cmd = model.updateNewKdbxPasswordScreen(keyMsgUp)
		model = asModel(t, newM)
		assert.NotNil(t, cmd)
		assert.Equal(t, 1, model.newPasswordFocusedField, "Фокус должен вернуться на второе поле")
		assert.False(t, model.newPasswordInput1.Focused())
		assert.True(t, model.newPasswordInput2.Focused())
	})

	t.Run("Enter на первом поле (пустой пароль)", func(t *testing.T) {
		m, _ := setupNewKdbxTest(t)
		keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
		newM, cmd := m.updateNewKdbxPasswordScreen(keyMsg)
		model := asModel(t, newM)
		assert.Nil(t, cmd)
		assert.Equal(t, "Пароль не может быть пустым!", model.confirmPasswordError)
		assert.Equal(t, 0, model.newPasswordFocusedField) // Остаемся на первом поле
	})

	t.Run("Enter на первом поле (непустой пароль)", func(t *testing.T) {
		m, _ := setupNewKdbxTest(t)
		m.newPasswordInput1.SetValue("password123")
		keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
		newM, cmd := m.updateNewKdbxPasswordScreen(keyMsg)
		model := asModel(t, newM)
		assert.NotNil(t, cmd)
		assert.Equal(t, "", model.confirmPasswordError)
		assert.Equal(t, 1, model.newPasswordFocusedField, "Фокус должен перейти на второе поле")
		assert.False(t, model.newPasswordInput1.Focused())
		assert.True(t, model.newPasswordInput2.Focused())
	})

	t.Run("Enter на втором поле (пароли не совпадают)", func(t *testing.T) {
		m, _ := setupNewKdbxTest(t)
		m.newPasswordInput1.SetValue("pass1")
		m.newPasswordInput2.SetValue("pass2")
		m.newPasswordFocusedField = 1 // Фокус на втором поле
		m.newPasswordInput1.Blur()
		m.newPasswordInput2.Focus()

		keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
		newM, cmd := m.updateNewKdbxPasswordScreen(keyMsg)
		model := asModel(t, newM)
		assert.NotNil(t, cmd)
		assert.Equal(t, "Пароли не совпадают!", model.confirmPasswordError)
		assert.Equal(t, "", model.newPasswordInput1.Value(), "Поле 1 должно очиститься")
		assert.Equal(t, "", model.newPasswordInput2.Value(), "Поле 2 должно очиститься")
		assert.Equal(t, 0, model.newPasswordFocusedField, "Фокус должен вернуться на первое поле")
		assert.True(t, model.newPasswordInput1.Focused())
		assert.False(t, model.newPasswordInput2.Focused())
	})

	t.Run("Enter на втором поле (успешное создание)", func(t *testing.T) {
		m, tmpFilePath := setupNewKdbxTest(t)
		password := "correctPassword"
		m.newPasswordInput1.SetValue(password)
		m.newPasswordInput2.SetValue(password)
		m.newPasswordFocusedField = 1 // Фокус на втором поле
		m.newPasswordInput1.Blur()
		m.newPasswordInput2.Focus()

		keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
		newM, cmd := m.updateNewKdbxPasswordScreen(keyMsg)
		model := asModel(t, newM)

		// Проверяем переход состояния и команду
		assert.Equal(t, entryListScreen, model.state, "Должен произойти переход на экран списка записей")
		assert.NotNil(t, cmd, "Должна быть возвращена команда (предположительно ClearScreen)")

		// Проверяем, что база данных создана и сохранена
		assert.NotNil(t, model.db, "База данных должна быть создана")
		assert.Equal(t, password, model.password, "Пароль должен быть сохранен в модели")

		// Проверяем, что файл физически создан
		_, err := os.Stat(tmpFilePath)
		require.NoError(t, err, "Файл KDBX должен быть создан по пути %s", tmpFilePath)

		// Проверяем, что список записей инициализирован (хотя бы не nil)
		assert.NotNil(t, model.entryList, "Список записей должен быть инициализирован")
		// Можно добавить проверку, что он пуст или содержит 'General' группу
	})

	t.Run("Ввод символов в активное поле", func(t *testing.T) {
		m, _ := setupNewKdbxTest(t)
		assert.Equal(t, 0, m.newPasswordFocusedField)
		initialValue := m.newPasswordInput1.Value()

		// Вводим символ 'a'
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		newM, _ := m.updateNewKdbxPasswordScreen(keyMsg)
		model := asModel(t, newM)

		assert.Equal(t, initialValue+"a", model.newPasswordInput1.Value(), "Символ 'a' должен добавиться к первому полю")
		assert.Equal(t, "", model.newPasswordInput2.Value(), "Второе поле не должно измениться")

		// Переключаем фокус на второе поле
		model.newPasswordFocusedField = 1
		model.newPasswordInput1.Blur()
		model.newPasswordInput2.Focus()

		// Вводим символ 'b'
		keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
		newM, _ = model.updateNewKdbxPasswordScreen(keyMsg)
		model = asModel(t, newM)

		assert.Equal(t, initialValue+"a", model.newPasswordInput1.Value(), "Первое поле не должно измениться")
		assert.Equal(t, "b", model.newPasswordInput2.Value(), "Символ 'b' должен добавиться ко второму полю")
	})
}

// -//- TestUpdateNewKdbxPasswordScreen
// Убираем старый TODO
// -//- // TODO: Добавить тест TestUpdateNewKdbxPasswordScreen
