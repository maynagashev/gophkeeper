package tui

import (
	"fmt"
	"log/slog"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	// Убедимся, что импорт есть.
	"github.com/maynagashev/gophkeeper/client/internal/api"
	"github.com/maynagashev/gophkeeper/client/internal/kdbx"
)

// Добавляем константу для статуса.
const (
	statusNotLoggedIn = "Не выполнен"
)

// updateEntryListScreen обрабатывает сообщения для экрана списка записей.
func (m *model) updateEntryListScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Сначала обновляем список
	m.entryList, cmd = m.entryList.Update(msg)
	cmds = append(cmds, cmd)

	// Обработка клавиш для экрана списка
	//nolint:nestif // Вложенность из-за разных клавиш
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyQuit:
			// Выход по 'q', если не активен режим фильтрации
			if m.entryList.FilterState() == list.Unfiltered {
				return m, tea.Quit
			}
		case keyEnter:
			selectedItem := m.entryList.SelectedItem()
			if selectedItem != nil {
				// Убеждаемся, что это наш тип entryItem
				if item, isEntryItem := selectedItem.(entryItem); isEntryItem {
					m.selectedEntry = &item
					m.state = entryDetailScreen
					slog.Info("Переход к деталям записи", "title", item.Title())
					cmds = append(cmds, tea.ClearScreen)
				}
			}
		case keyAdd:
			// Переход к добавлению новой записи (только если не Read-Only)
			if !m.readOnlyMode {
				m.prepareAddScreen()
				m.state = entryAddScreen
				slog.Info("Переход к добавлению новой записи")
				return m, tea.ClearScreen
			}
		case "s":
			m.state = syncServerScreen
			// Добавляем ClearScreen при переходе
			return m, tea.ClearScreen
		case "l":
			// TODO: Проверить, настроен ли URL и валиден ли токен
			// Если URL не настроен -> serverUrlInputScreen
			// Если токен есть, но невалиден -> loginScreen
			// Если токен валиден -> может быть, просто показать статус?
			// Пока просто переходим к выбору
			m.state = loginRegisterChoiceScreen
			return m, nil
		}
	}
	return m, tea.Batch(cmds...)
}

// _handleAuthLoadError обрабатывает ошибку загрузки Auth данных.
func (m *model) _handleAuthLoadError(errLoad error, urlFromFlag bool) {
	slog.Error("Ошибка загрузки Auth данных из KDBX", "error", errLoad)
	if !urlFromFlag {
		m.serverURL = ""
		m.apiClient = nil
	}
	m.authToken = ""
	m.loginStatus = statusNotLoggedIn + " (ошибка загрузки)"
}

// _handleAuthLoadSuccess обрабатывает успешную загрузку Auth данных.
func (m *model) _handleAuthLoadSuccess(loadedURL, loadedToken string, urlFromFlag bool) {
	if urlFromFlag {
		// URL задан флагом, загружаем только токен
		m.authToken = loadedToken
		slog.Info("URL сервера задан флагом, загружен только токен из KDBX", "token_found", m.authToken != "")
	} else {
		// URL не задан флагом, используем загруженный URL
		m.serverURL = loadedURL
		m.authToken = loadedToken
		if m.serverURL != "" {
			m.apiClient = api.NewHTTPClient(m.serverURL)
			slog.Info("URL сервера загружен из KDBX, создан API клиент", "url", m.serverURL, "token_found", m.authToken != "")
		} else {
			m.apiClient = nil
			slog.Info("URL сервера не задан и не найден в KDBX.")
		}
	}

	// Обновляем статус входа
	if m.authToken != "" {
		m.loginStatus = "Вход выполнен (сессия загружена)"
	} else {
		m.loginStatus = statusNotLoggedIn
	}
}

// handleDBOpenedMsg обрабатывает сообщение об успешном открытии базы.
func (m *model) handleDBOpenedMsg(msg dbOpenedMsg) (tea.Model, tea.Cmd) {
	slog.Debug("handleDBOpenedMsg: Начало")
	m.db = msg.db
	m.err = nil
	prevState := m.state
	m.state = entryListScreen
	slog.Info("База KDBX успешно открыта", "path", m.kdbxPath)

	// --- Обновленная логика загрузки Auth данных ---
	// Определяем, был ли URL задан через флаг (т.е. apiClient уже инициализирован)
	urlFromFlag := m.serverURL != "" && m.apiClient != nil
	slog.Debug("Проверка URL из флага", "urlFromFlag", urlFromFlag, "initialURL", m.serverURL)

	loadedURL, loadedToken, errLoad := kdbx.LoadAuthData(m.db)
	// Вызываем соответствующие хелперы
	if errLoad != nil {
		m._handleAuthLoadError(errLoad, urlFromFlag)
	} else {
		m._handleAuthLoadSuccess(loadedURL, loadedToken, urlFromFlag)
	}

	// Устанавливаем токен в API клиенте, если клиент существует
	if m.apiClient != nil {
		m.apiClient.SetAuthToken(m.authToken) // m.authToken будет либо загруженным, либо пустым
		slog.Debug("Установлен токен в API клиенте после загрузки/проверки KDBX", "token_set", m.authToken != "")
	} else {
		// Эта ситуация возможна, если URL не задан ни флагом, ни в KDBX
		slog.Warn("API клиент не инициализирован (URL не задан), токен не установлен.")
	}

	// --- Существующий код для заполнения списка ---
	entries := kdbx.GetAllEntries(m.db)
	slog.Debug("Записи, полученные из KDBX", "count", len(entries))

	items := make([]list.Item, len(entries))
	for i, entry := range entries {
		items[i] = entryItem{entry: entry}
	}

	slog.Debug("Элементы, подготовленные для списка", "count", len(items))
	// Используем m.entryList.SetItems, а не listCmd, так как команда теперь не используется
	_ = m.entryList.SetItems(items) // Команду от SetItems пока игнорируем

	slog.Debug("Элементы в списке после SetItems", "count", len(m.entryList.Items()))

	m.entryList.SetWidth(defaultListWidth)
	m.entryList.SetHeight(defaultListHeight)
	m.entryList.Title = fmt.Sprintf("Записи в '%s' (%d)", m.kdbxPath, len(items))

	// --- Команды для возврата ---
	dbOpenedCmds := []tea.Cmd{}
	if prevState != entryListScreen {
		dbOpenedCmds = append(dbOpenedCmds, tea.ClearScreen)
	}

	slog.Debug("handleDBOpenedMsg: Конец, m.db обновлен")
	return m, tea.Batch(dbOpenedCmds...)
}
