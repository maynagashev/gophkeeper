package tui

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/client/internal/api"
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
			return m, nil
		}
	}
	// Обновляем поле ввода
	newInput, inputCmd := m.serverURLInput.Update(msg)
	m.serverURLInput = newInput
	// Возвращаем обновленную модель и команду от textinput
	return m, inputCmd
}
