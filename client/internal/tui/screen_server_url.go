package tui

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/client/internal/api"
	"github.com/stretchr/testify/assert"
)

// updateServerURLInputScreen обрабатывает ввод URL сервера.
func (m *model) updateServerURLInputScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Обрабатываем Esc и Enter, остальное передаем в textinput
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEsc:
			m.state = syncServerScreen // Возвращаемся к меню синхронизации
			return m, nil
		case keyEnter:
			newURL := m.serverURLInput.Value()
			if newURL == "" {
				newURL = m.serverURLInput.Placeholder // Используем плейсхолдер если пусто
			}
			// TODO: Добавить валидацию URL?
			m.serverURL = newURL
			// Сбрасываем статус, т.к. URL изменился
			m.loginStatus = "Не выполнен"
			m.authToken = ""
			m.apiClient = api.NewHTTPClient(newURL) // Пересоздаем клиент с новым URL
			slog.Info("URL сервера обновлен", "url", newURL)
			// Переходим к выбору логина/регистрации
			m.state = loginRegisterChoiceScreen
			// Добавляем ClearScreen для очистки артефактов
			return m, tea.ClearScreen
		}
	}
	// Обновляем поле ввода
	newInput, inputCmd := m.serverURLInput.Update(msg)
	m.serverURLInput = newInput
	// Возвращаем обновленную модель и команду от textinput
	return m, inputCmd
}

// viewServerURLInputScreen отображает экран ввода URL сервера.
func (m *model) viewServerURLInputScreen() string {
	return fmt.Sprintf("Введите URL сервера:\n%s", m.serverURLInput.View())
}

//nolint:testpackage // Тесты в том же пакете для доступа к приватным компонентам
func TestServerURLInputScreen_CancelInput(t *testing.T) {
	// Создаем модель
	m := &model{
		state:          serverURLInputScreen,
		serverURL:      "https://original.server",
		serverURLInput: textinput.New(),
	}

	// Создаем сообщение о нажатии Esc
	msg := tea.KeyMsg{Type: tea.KeyEsc}

	// Вызываем функцию обновления
	newModel, cmd := m.updateServerURLInputScreen(msg)
	model, ok := newModel.(*model)

	// Проверяем результаты
	assert.True(t, ok, "Должен быть возвращен указатель на model")
	assert.Equal(t, syncServerScreen, model.state, "Должен произойти переход на экран синхронизации")
	assert.Equal(t, "https://original.server", model.serverURL, "URL не должен измениться")
	assert.Nil(t, cmd, "Не должно быть возвращено команды")
}

func TestServerURLInputScreen_ConfirmInput(t *testing.T) {
	// Создаем модель
	m := &model{
		state:          serverURLInputScreen,
		serverURLInput: textinput.New(),
	}

	// Устанавливаем значение URL
	newURL := "https://new.server"
	m.serverURLInput.SetValue(newURL)

	// Создаем сообщение о нажатии Enter
	msg := tea.KeyMsg{Type: tea.KeyEnter}

	// Вызываем функцию обновления
	newModel, cmd := m.updateServerURLInputScreen(msg)
	model, ok := newModel.(*model)

	// Проверяем результаты
	assert.True(t, ok, "Должен быть возвращен указатель на model")
	assert.Equal(t, loginRegisterChoiceScreen, model.state, "Должен произойти переход на экран выбора входа/регистрации")
	assert.Equal(t, newURL, model.serverURL, "URL должен обновиться")
	assert.NotNil(t, cmd, "Должна быть возвращена команда")
	assert.NotNil(t, model.apiClient, "API клиент должен быть инициализирован")
	assert.Equal(t, "Не выполнен", model.loginStatus, "Статус логина должен быть сброшен")
	assert.Equal(t, "", model.authToken, "Токен авторизации должен быть сброшен")
}

func TestServerURLInputScreen_UseDefaultURL(t *testing.T) {
	// Создаем модель
	m := &model{
		state:          serverURLInputScreen,
		serverURLInput: textinput.New(),
	}

	// Устанавливаем плейсхолдер и пустое значение
	placeholder := "https://default.server"
	m.serverURLInput.Placeholder = placeholder
	m.serverURLInput.SetValue("")

	// Создаем сообщение о нажатии Enter
	msg := tea.KeyMsg{Type: tea.KeyEnter}

	// Вызываем функцию обновления
	newModel, _ := m.updateServerURLInputScreen(msg)
	model, ok := newModel.(*model)

	// Проверяем результаты
	assert.True(t, ok, "Должен быть возвращен указатель на model")
	assert.Equal(t, placeholder, model.serverURL, "URL должен быть установлен из плейсхолдера")
}

func TestServerURLInputScreen_UpdateInput(t *testing.T) {
	// Создаем модель
	m := &model{
		state:          serverURLInputScreen,
		serverURLInput: textinput.New(),
	}

	// Создаем сообщение о вводе символа 'a'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}

	// Вызываем функцию обновления
	newModel, cmd := m.updateServerURLInputScreen(msg)
	model, ok := newModel.(*model)

	// Проверяем результаты
	assert.True(t, ok, "Должен быть возвращен указатель на model")
	assert.Equal(t, "a", model.serverURLInput.Value(), "Символ должен быть добавлен в поле ввода")
	assert.NotNil(t, cmd, "Должна быть возвращена команда")
}

func TestServerURLInputScreen_View(t *testing.T) {
	// Создаем модель
	m := &model{
		state:          serverURLInputScreen,
		serverURLInput: textinput.New(),
	}

	// Устанавливаем значение URL
	testURL := "https://test.server"
	m.serverURLInput.SetValue(testURL)

	// Вызываем функцию отображения
	view := m.viewServerURLInputScreen()

	// Проверяем результаты
	assert.Contains(t, view, "Введите URL сервера", "View должен содержать заголовок")
	assert.Contains(t, view, testURL, "View должен содержать введенный URL")
}
