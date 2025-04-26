//nolint:testpackage // Это тесты в том же пакете для доступа к приватным компонентам
package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordInputScreen(t *testing.T) {
	t.Run("ОтрисовкаЭкрана", func(t *testing.T) {
		// Настраиваем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(passwordInputScreen)

		// Устанавливаем путь к файлу
		testPath := "/path/to/test.kdbx"
		suite.Model.kdbxPath = testPath

		// Инициализируем поле ввода пароля
		suite.Model.passwordInput = textinput.New()

		// Вызываем View() напрямую
		view := suite.Model.viewPasswordInputScreen()

		// Проверяем результат
		assert.Contains(t, view, "мастер-пароль", "View должен содержать текст с просьбой о вводе пароля")
		assert.Contains(t, view, testPath, "View должен содержать путь к файлу")
	})

	t.Run("ОтображениеОшибки", func(t *testing.T) {
		// Настраиваем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(passwordInputScreen)

		// Инициализируем поле ввода пароля
		suite.Model.passwordInput = textinput.New()

		// Устанавливаем ошибку
		testError := errors.New("тестовая ошибка")
		suite.Model.err = testError

		// Вызываем View() напрямую
		view := suite.Model.viewPasswordInputScreen()

		// Проверяем результат
		assert.Contains(t, view, "Ошибка:", "View должен содержать заголовок ошибки")
		assert.Contains(t, view, testError.Error(), "View должен содержать текст ошибки")
		assert.Contains(t, view, "Нажмите любую клавишу", "View должен содержать инструкции для продолжения")
	})

	t.Run("ВводПароляИОтправкаКоманды", func(t *testing.T) {
		// Настраиваем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(passwordInputScreen)

		// Устанавливаем путь к файлу
		testPath := "/path/to/test.kdbx"
		suite.Model.kdbxPath = testPath

		// Инициализируем поле ввода пароля и устанавливаем фокус
		suite.Model.passwordInput = textinput.New()
		suite.Model.passwordInput.Focus()

		// Имитируем ввод пароля
		testPassword := "secretpassword"
		suite.Model.passwordInput.SetValue(testPassword)

		// Имитируем нажатие Enter для отправки
		newModel, cmd := suite.SimulateKeyPress(tea.KeyEnter)
		m := toModel(t, newModel)

		// Проверяем, что пароль сохранен в модели и поле очищено
		assert.Equal(t, testPassword, m.password, "Пароль должен быть сохранен в модели")
		assert.Empty(t, m.passwordInput.Value(), "Поле ввода пароля должно быть очищено")
		assert.False(t, m.passwordInput.Focused(), "Фокус должен быть снят с поля ввода")

		// Проверяем, что возвращена команда openKdbxCmd
		require.NotNil(t, cmd, "Должна быть возвращена команда")

		// Выполняем команду и проверяем, что она пытается открыть файл
		// Примечание: в реальном приложении это бы открыло файл KDBX,
		// но в тесте это вернет ошибку, так как файл не существует
		msg := suite.ExecuteCmd(cmd)
		errorMsg, ok := msg.(errMsg)
		require.True(t, ok, "Сообщение должно быть типа errMsg, т.к. файл не существует")
		assert.Contains(t, errorMsg.err.Error(), "open "+testPath, "Ошибка должна указывать на попытку открыть файл")
	})

	t.Run("ОбработкаErrMsg", func(t *testing.T) {
		// Настраиваем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(entryListScreen) // Изначально другой экран

		// Инициализируем поле ввода пароля
		suite.Model.passwordInput = textinput.New()

		// Создаем ошибку и сообщение
		testError := errors.New("тестовая ошибка открытия")
		errorMessage := errMsg{err: testError} // Переименовываем переменную msg в errorMessage

		// Обрабатываем сообщение
		newModel := suite.Model.handleErrorMsg(errorMessage)
		m := toModel(t, newModel)

		// Проверяем результаты
		assert.Equal(t, passwordInputScreen, m.state, "Должен быть переход к экрану ввода пароля")
		assert.True(t, m.passwordInput.Focused(), "Поле ввода пароля должно получить фокус")
		assert.Equal(t, testError, m.err, "Ошибка должна быть сохранена в модели")
	})

	t.Run("ОчисткаОшибкиПриНажатииКлавиши", func(t *testing.T) {
		// Настраиваем тестовую среду
		suite := NewScreenTestSuite()
		suite.WithState(passwordInputScreen)

		// Инициализируем поле ввода пароля
		suite.Model.passwordInput = textinput.New()

		// Устанавливаем ошибку
		suite.Model.err = errors.New("тестовая ошибка")

		// Имитируем нажатие любой клавиши
		newModel, _ := suite.SimulateKeyPress(tea.KeySpace)
		m := toModel(t, newModel)

		// Проверяем, что ошибка очищена и фокус возвращен
		require.NoError(t, m.err, "Ошибка должна быть очищена после нажатия клавиши")
		assert.True(t, m.passwordInput.Focused(), "Поле ввода должно получить фокус")
	})
}
