//nolint:testpackage // Тесты в том же пакете для доступа к приватным компонентам
package tui

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: Добавить тесты для syncMenuItem и viewSyncServerScreen

// TestSyncMenuItem_Title проверяет метод Title.
func TestSyncMenuItem_Title(t *testing.T) {
	tests := []struct {
		name     string
		item     syncMenuItem
		expected string
	}{
		{"Настройка URL", syncMenuItem{title: "Настроить URL сервера"}, "Настроить URL сервера"},
		{"Вход/Регистрация", syncMenuItem{title: "Войти / Зарегистрироваться"}, "Войти / Зарегистрироваться"},
		{"Синхронизировать", syncMenuItem{title: "Синхронизировать сейчас"}, "Синхронизировать сейчас"},
		{"Просмотр версий", syncMenuItem{title: "Просмотреть версии"}, "Просмотреть версии"},
		{"Выход", syncMenuItem{title: "Выйти на сервере"}, "Выйти на сервере"},
		{"Пустой заголовок", syncMenuItem{title: ""}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.item.Title())
		})
	}
}

// TestSyncMenuItem_Description проверяет метод Description.
func TestSyncMenuItem_Description(t *testing.T) {
	// У syncMenuItem нет Description, он всегда пустой
	item1 := syncMenuItem{title: "Настроить URL сервера"}
	assert.Equal(t, "", item1.Description())

	item2 := syncMenuItem{title: ""}
	assert.Equal(t, "", item2.Description())
}

// TestSyncMenuItem_FilterValue проверяет метод FilterValue.
func TestSyncMenuItem_FilterValue(t *testing.T) {
	tests := []struct {
		name     string
		item     syncMenuItem
		expected string
	}{
		{"Настройка URL", syncMenuItem{title: "Настроить URL сервера"}, "Настроить URL сервера"},
		{"Пустой заголовок", syncMenuItem{title: ""}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// FilterValue должен возвращать Title
			assert.Equal(t, tt.expected, tt.item.FilterValue())
			assert.Equal(t, tt.item.Title(), tt.item.FilterValue())
		})
	}
}

// TestViewSyncServerScreen проверяет функцию viewSyncServerScreen.
func TestViewSyncServerScreen(t *testing.T) {
	m := model{
		state:          syncServerScreen,
		width:          80,
		height:         24,
		syncServerMenu: initSyncMenu(),
		serverURL:      "",
		loginStatus:    "",
		lastSyncStatus: "",
	}

	serverURLText := m.serverURL
	if serverURLText == "" {
		serverURLText = "Не настроен"
	}
	statusInfo := fmt.Sprintf(
		"URL Сервера: %s\nСтатус входа: %s\nПоследняя синх.: %s\n",
		serverURLText,
		m.loginStatus,
		m.lastSyncStatus,
	)
	expected := fmt.Sprintf("%s\n\n%s", statusInfo, m.syncServerMenu.View())

	actual := m.viewSyncServerScreen()

	assert.Equal(t, expected, actual)
}

// Mock API Client for testing - УДАЛЕНО, т.к. уже есть в api_messages_test.go

// TestHandleSyncMenuConfigureURL проверяет функцию handleSyncMenuConfigureURL.
func TestHandleSyncMenuConfigureURL(t *testing.T) {
	tests := []struct {
		name             string
		initialServerURL string
		expectedValue    string
	}{
		{
			name:             "URL не задан",
			initialServerURL: "",
			expectedValue:    "",
		},
		{
			name:             "URL задан",
			initialServerURL: "http://example.com",
			expectedValue:    "http://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Используем существующий mock API клиента из screen_test_helpers.go или api_messages_test.go
			mockAPI := new(MockAPIClient) // Предполагаем, что MockAPIClient доступен
			m := initModel("", false, "", mockAPI)
			m.serverURL = tt.initialServerURL
			m.state = syncServerScreen // Устанавливаем начальное состояние

			cmd := m.handleSyncMenuConfigureURL()

			assert.Equal(t, serverURLInputScreen, m.state, "Состояние должно измениться на serverURLInputScreen")
			assert.True(t, m.serverURLInput.Focused(), "Поле ввода URL должно быть в фокусе")
			assert.Equal(t, "https://...", m.serverURLInput.Placeholder, "Placeholder должен быть 'https://...'")
			assert.Equal(t, tt.expectedValue, m.serverURLInput.Value(), "Значение поля ввода должно соответствовать ожидаемому")

			// Проверяем, что возвращается команда Blink
			assert.NotNil(t, cmd, "Команда не должна быть nil")
			// Прямая проверка типа команды Blink затруднительна,
			// поэтому просто убеждаемся, что команда возвращена.
		})
	}
}

// TODO: Добавить тест для viewSyncServerScreen
