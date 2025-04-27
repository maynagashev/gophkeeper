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

// TODO: Добавить тест для viewSyncServerScreen
