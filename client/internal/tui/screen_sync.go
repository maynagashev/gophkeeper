package tui

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/client/internal/kdbx"
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
	return fmt.Sprintf("%s\n\n%s", statusInfo, m.syncServerMenu.View())
}

// handleSyncMenuConfigureURL обрабатывает выбор пункта "Настроить URL сервера".
func (m *model) handleSyncMenuConfigureURL() tea.Cmd {
	m.serverURLInput.Reset()
	m.serverURLInput.Placeholder = "https://..."
	m.serverURLInput.Focus()
	if m.serverURL != "" {
		m.serverURLInput.SetValue(m.serverURL)
	}
	m.state = serverURLInputScreen
	return textinput.Blink
}

// handleSyncMenuLoginRegister обрабатывает выбор пункта "Войти / Зарегистрироваться".
func (m *model) handleSyncMenuLoginRegister() tea.Cmd {
	if m.serverURL == "" {
		_, cmd := m.setStatusMessage("Сначала настройте URL сервера")
		return cmd
	}
	m.loginUsernameInput.Focus()
	m.loginPasswordInput.Blur()
	m.loginRegisterFocusedField = 0
	m.state = loginRegisterChoiceScreen
	return textinput.Blink
}

// handleSyncMenuSyncNow обрабатывает выбор пункта "Синхронизировать сейчас".
func (m *model) handleSyncMenuSyncNow() tea.Cmd {
	if m.authToken == "" {
		_, cmd := m.setStatusMessage("Необходимо войти перед синхронизацией")
		return cmd
	}
	newM, statusCmd := m.setStatusMessage("Запуск синхронизации...")
	mModel, ok := newM.(*model)
	if !ok {
		slog.Error("Неожиданный тип модели после setStatusMessage")
		return tea.Batch(statusCmd, func() tea.Msg {
			return errMsg{err: errors.New("внутренняя ошибка типа модели")}
		})
	}
	syncCmd := startSyncCmd(mModel)
	return tea.Batch(statusCmd, syncCmd)
}

// handleSyncMenuViewVersions обрабатывает выбор пункта "Просмотреть версии".
func (m *model) handleSyncMenuViewVersions() tea.Cmd {
	if m.authToken == "" {
		_, cmd := m.setStatusMessage("Необходимо войти для просмотра версий")
		return cmd
	}
	m.state = versionListScreen
	m.loadingVersions = true
	return tea.Batch(tea.ClearScreen, loadVersionsCmd(m))
}

// handleSyncMenuLogout обрабатывает выбор пункта "Выйти на сервере".
func (m *model) handleSyncMenuLogout() tea.Cmd {
	if m.authToken == "" {
		_, cmd := m.setStatusMessage("Вы не авторизованы")
		return cmd
	}
	oldToken := m.authToken
	m.authToken = ""
	m.loginStatus = statusNotLoggedIn
	if m.apiClient != nil {
		m.apiClient.SetAuthToken("")
	}
	var saveCmd tea.Cmd
	if m.db != nil {
		errSave := kdbx.SaveAuthData(m.db, m.serverURL, "")
		if errSave != nil {
			slog.Error("Ошибка сохранения пустого токена в KDBX при выходе", "error", errSave)
			_, saveCmd = m.setStatusMessage("Ошибка сохранения данных при выходе")
		} else {
			slog.Info("Локальный токен очищен и сохранен в KDBX.")
			_, saveCmd = m.setStatusMessage("Успешно вышли")
		}
	} else {
		slog.Warn("Не удалось сохранить пустой токен в KDBX при выходе: база не загружена.")
		_, saveCmd = m.setStatusMessage("Успешно вышли (локально)")
	}
	slog.Info("Выполнен выход", "token_existed", oldToken != "")
	return saveCmd
}

// Меняем возвращаемый тип на *model.
func (m *model) updateSyncServerScreen(msg tea.Msg) (*model, tea.Cmd) {
	var cmds []tea.Cmd
	var listCmd tea.Cmd

	// Обновляем список меню сначала
	m.syncServerMenu, listCmd = m.syncServerMenu.Update(msg)
	cmds = append(cmds, listCmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEnter:
			// Обработка выбора пункта меню
			cmd := m.handleSyncMenuAction()
			cmds = append(cmds, cmd)
			// Может потребоваться ClearScreen в зависимости от действия
		case keyEsc, keyBack:
			m.state = entryListScreen
			// Возвращаем указатель на модель
			return m, tea.ClearScreen // Очистка экрана добавлена
		}
	}

	// Возвращаем указатель на модель
	return m, tea.Batch(cmds...)
}

// handleSyncMenuAction обрабатывает действие выбора в меню синхронизации.
func (m *model) handleSyncMenuAction() tea.Cmd {
	selectedItem, ok := m.syncServerMenu.SelectedItem().(syncMenuItem)
	if !ok {
		return nil
	}

	switch selectedItem.id {
	case "configure_url":
		return m.handleSyncMenuConfigureURL()
	case "login_register":
		return m.handleSyncMenuLoginRegister()
	case "sync_now":
		return m.handleSyncMenuSyncNow()
	case "view_versions":
		return m.handleSyncMenuViewVersions()
	case "logout":
		return m.handleSyncMenuLogout()
	}

	return nil
}
