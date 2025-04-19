package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Тип для элементов меню синхронизации --- //

// syncMenuItem представляет элемент в меню синхронизации.
type syncMenuItem struct {
	title string
	id    string // Идентификатор для обработки выбора
}

func (i syncMenuItem) Title() string       { return i.title }
func (i syncMenuItem) Description() string { return "" } // Описание не нужно
func (i syncMenuItem) FilterValue() string { return i.title }

// --- Функции экрана --- //

// viewSyncServerScreen отображает экран "Синхронизация и Сервер".
func (m *model) viewSyncServerScreen() string {
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

	// Объединяем информацию о статусе и РЕНДЕР МЕНЮ
	// Добавляем перенос строки между ними для четкого разделения.
	return statusInfo + m.syncServerMenu.View()
}

// updateSyncServerScreen обрабатывает сообщения для экрана "Синхронизация и Сервер".
//
//nolint:gocognit,nestif // TODO: Рефакторить для уменьшения вложенности и когнитивной сложности.
func (m *model) updateSyncServerScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEnter:
			selectedItem := m.syncServerMenu.SelectedItem()
			if item, itemOk := selectedItem.(syncMenuItem); itemOk {
				switch item.id {
				case "configure_url":
					m.state = serverURLInputScreen
					if m.serverURL != "" {
						m.serverURLInput.SetValue(m.serverURL)
					} else {
						m.serverURLInput.Placeholder = defaultServerURL
						m.serverURLInput.SetValue("")
					}
					m.serverURLInput.Focus()
					return m, tea.Batch(textinput.Blink, tea.ClearScreen)
				case "login_register":
					if m.serverURL == "" {
						m.state = serverURLInputScreen
						m.serverURLInput.Placeholder = defaultServerURL
						m.serverURLInput.SetValue("")
						m.serverURLInput.Focus()
						return m, tea.Batch(textinput.Blink, tea.ClearScreen)
					}
					m.state = loginRegisterChoiceScreen
					return m, tea.ClearScreen
				case "sync_now":
					return m.setStatusMessage("TODO: Запуск синхронизации...")
				case "logout":
					m.authToken = ""
					m.loginStatus = "Не выполнен"
					// При выходе из приложения токен не удаляется из KDBX, время жизни токена ограничено параметрами JWT
					// err := kdbx.SaveAuthData(m.db, m.serverURL, "")
					return m.setStatusMessage("Выход выполнен.")
				}
			}
		case keyEsc, keyBack:
			m.state = entryListScreen
			return m, tea.ClearScreen // Очистка экрана добавлена
		}
	}

	// Обновляем список меню
	var listCmd tea.Cmd
	m.syncServerMenu, listCmd = m.syncServerMenu.Update(msg)
	cmds = append(cmds, listCmd)

	return m, tea.Batch(cmds...)
}
