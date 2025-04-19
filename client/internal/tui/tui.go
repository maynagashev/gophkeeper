package tui

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofrs/flock"

	"github.com/maynagashev/gophkeeper/client/internal/api" // Импортируем пакет API клиента
)

const (
	statusMessageTimeout     = 2 * time.Second         // Время отображения статусных сообщений
	defaultServerURL         = "http://localhost:8080" // Временный URL сервера по умолчанию
	helpStatusHeightOffset   = 2                       // Высота строки помощи и статуса
	docStyleMarginVertical   = 1
	docStyleMarginHorizontal = 2
)

// Init - команда, выполняемая при запуске приложения.
func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

// setStatusMessage устанавливает статусное сообщение и запускает таймер для его очистки.
func (m *model) setStatusMessage(status string) (tea.Model, tea.Cmd) {
	m.savingStatus = status
	// Если таймер уже есть, останавливаем его
	if m.statusTimer != nil {
		m.statusTimer.Stop()
		// Мы не можем повторно использовать старый таймер, поэтому обнуляем
		m.statusTimer = nil
	}
	// Запускаем команду для очистки статуса через заданное время
	cmd := clearStatusCmd(statusMessageTimeout)
	// Примечание: Мы не сохраняем сам таймер, так как tea.Tick управляет им.
	// Если нужно будет отменять таймер до его срабатывания, понадобится другой подход.
	return m, cmd
}

// View отрисовывает пользовательский интерфейс.
//
//nolint:funlen
func (m *model) View() string {
	var mainContent string
	var help string

	switch m.state {
	case welcomeScreen:
		mainContent = m.viewWelcomeScreen()
		help = "(Enter - продолжить, Ctrl+C/q - выход)"
	case passwordInputScreen:
		mainContent = m.viewPasswordInputScreen()
		help = "(Enter - подтвердить, Ctrl+C - выход)"
	case newKdbxPasswordScreen:
		mainContent = m.viewNewKdbxPasswordScreen()
		help = "(Tab - сменить поле, Enter - создать, Esc/Ctrl+C - выход)"
	case entryListScreen:
		mainContent = m.entryList.View()
		help = "(↑/↓, Enter - детали, / - поиск, a - доб, s - синхр, l - логин, Ctrl+S - сохр, q - вых)"
	case entryDetailScreen:
		mainContent = m.viewEntryDetailScreen()
		help = "(e - ред., Ctrl+S - сохр., Esc/b - назад)"
	case entryEditScreen:
		mainContent = m.viewEntryEditScreen()
		help = "(Tab/↑/↓, Enter - сохр., Esc - отмена, ^O - влож+, ^D - влож-)"
	case entryAddScreen:
		mainContent = m.viewEntryAddScreen()
		help = "(Tab/↑/↓, Enter - доб., ^O - влож+, Esc - отмена)"
	case attachmentListDeleteScreen:
		mainContent = m.viewAttachmentListDeleteScreen()
		help = "(↑/↓ - навигация, Enter/d - удалить, Esc/b - отмена)"
	case attachmentPathInputScreen:
		mainContent = m.viewAttachmentPathInputScreen()
		help = "(Enter - подтвердить, Esc - отмена)"
	case syncServerScreen:
		mainContent = m.viewSyncServerScreen()
		help = "(↑/↓ - навигация, Enter - выбрать, Esc/b - назад)"
	case serverURLInputScreen:
		mainContent = m.viewServerURLInputScreen()
		help = "(Enter - подтвердить, Esc - назад)"
	case loginRegisterChoiceScreen:
		mainContent = m.viewLoginRegisterChoiceScreen()
		help = "(R - регистрация, L - вход, Esc/b - назад)"
	case loginScreen:
		mainContent = m.viewLoginScreen()
		help = "(Tab - след. поле, Enter - войти, Esc - назад)"
	case registerScreen:
		mainContent = m.viewRegisterScreen()
		help = "(Tab - след. поле, Enter - зарегистрироваться, Esc - назад)"
	default:
		mainContent = "Неизвестное состояние!"
		// Для неизвестного состояния тоже добавим имя, если оно есть
		if m.state.String() != "" {
			help = "State: " + m.state.String()
		} else {
			help = "Unknown state"
		}
	}

	// Добавляем имя состояния к help для отладки
	help = help + "\n---\n" + m.state.String()

	// Добавляем статус сохранения или Read-Only, если он есть и мы не на определенных экранах
	statusLine := ""
	readOnlyIndicator := ""
	if m.readOnlyMode {
		readOnlyIndicator = " [Read-Only]"
	}
	displayStatus := (m.savingStatus != "" || m.readOnlyMode) &&
		m.state != welcomeScreen &&
		m.state != passwordInputScreen &&
		m.state != newKdbxPasswordScreen &&
		m.state != attachmentPathInputScreen
	if displayStatus {
		statusLine = "\n" + m.savingStatus + readOnlyIndicator
	}

	// Собираем финальный вывод
	// Применяем общий стиль к основному контенту
	styledContent := m.docStyle.Render(mainContent)
	// Собираем все вместе, всегда добавляя перенос строки перед help
	return fmt.Sprintf("%s\n%s\n%s", styledContent, help, statusLine)
}

// Start запускает TUI приложение.
func Start(kdbxPath string) {
	// Создаем начальную модель
	m := initModel(kdbxPath) // Используем initModel из initialization.go

	// --- Инициализация API клиента ---
	// TODO: Сделать URL конфигурируемым (флаг, env, KDBX)
	m.apiClient = api.NewHTTPClient(defaultServerURL)
	m.serverURL = defaultServerURL // Сохраняем URL в модели
	slog.Info("API клиент инициализирован", "baseURL", defaultServerURL)

	// --- Реализация flock ---
	lockPath := kdbxPath + ".lock"
	m.fileLock = flock.New(lockPath)
	var flockErr error
	m.lockAcquired, flockErr = m.fileLock.TryLock()

	if flockErr != nil {
		// Критическая ошибка при попытке блокировки
		slog.Error("Критическая ошибка при попытке блокировки файла", "lockPath", lockPath, "error", flockErr)
		fmt.Fprintf(os.Stderr, "Ошибка блокировки файла %s: %v\n", lockPath, flockErr)
		// Попробуем разблокировать перед выходом
		_ = m.fileLock.Unlock()
		os.Exit(1)
	}

	if m.lockAcquired {
		slog.Info("Эксклюзивная блокировка файла получена.", "lockPath", lockPath)
		// Регистрируем разблокировку при выходе ИЗ ФУНКЦИИ START
		defer func() {
			if errUnlock := m.fileLock.Unlock(); errUnlock != nil {
				slog.Error("Ошибка при снятии блокировки файла", "lockPath", lockPath, "error", errUnlock)
			} else {
				slog.Info("Блокировка файла снята.", "lockPath", lockPath)
			}
		}()
	} else {
		m.readOnlyMode = true
		slog.Warn("Блокировка не получена (файл используется?). Read-Only.", "lockPath", lockPath)
	}
	// --- Конец реализации flock ---

	// Проверяем, существует ли файл KDBX
	if _, errStat := os.Stat(m.kdbxPath); os.IsNotExist(errStat) {
		// Файл не существует, переходим на экран создания пароля
		slog.Info("Файл KDBX не найден, переходим к созданию нового.", "path", m.kdbxPath)
		m.state = newKdbxPasswordScreen // Используем константу в нижнем регистре
		m.newPasswordInput1.Focus()
		m.newPasswordInput2.Blur()
	} else if errStat != nil {
		// Другая ошибка при доступе к файлу
		slog.Error("Ошибка при проверке файла KDBX", "path", m.kdbxPath, "error", errStat)
		fmt.Fprintf(os.Stderr, "Ошибка доступа к файлу %s: %v\n", m.kdbxPath, errStat)
		// Разблокируем файл перед выходом
		if m.lockAcquired {
			_ = m.fileLock.Unlock()
		}
		//nolint:gocritic // Unlock вызывается вручную перед выходом
		os.Exit(1)
	} else {
		// Файл существует, оставляем начальное состояние (welcomeScreen -> passwordInputScreen)
		slog.Info("Файл KDBX найден, запуск стандартного TUI.", "path", m.kdbxPath)
		// Состояние по умолчанию welcomeScreen в initModel
	}

	// Используем FullAltScreen для корректной работы списка
	p := tea.NewProgram(&m, tea.WithAltScreen()) // Передаем указатель на модель
	if _, errRun := p.Run(); errRun != nil {
		slog.Error("Ошибка при запуске TUI", "error", errRun)
		// Разблокируем файл перед выходом
		if m.lockAcquired {
			_ = m.fileLock.Unlock()
		}
		os.Exit(1)
	}
	// Успешный выход ПОСЛЕ defer Unlock
}

// --- Вспомогательные типы и функции ---

// syncMenuItem представляет элемент в меню синхронизации.
type syncMenuItem struct {
	title string
	id    string // Идентификатор для обработки выбора
}

func (i syncMenuItem) Title() string       { return i.title }
func (i syncMenuItem) Description() string { return "" } // Описание не нужно
func (i syncMenuItem) FilterValue() string { return i.title }

// --- Функции-заглушки для отображения других экранов были удалены ---

// viewServerURLInputScreen отображает экран ввода URL сервера.
func (m *model) viewServerURLInputScreen() string {
	return fmt.Sprintf("Введите URL сервера:\n%s", m.serverURLInput.View())
}

// viewSyncServerScreen отображает экран "Синхронизация и Сервер".
func (m *model) viewSyncServerScreen() string {
	serverURLText := m.serverURL // Используем правильное имя переменной
	if serverURLText == "" {
		serverURLText = "Не настроен"
	}

	statusInfo := fmt.Sprintf(
		"URL Сервера: %s\nСтатус входа: %s\nПоследняя синх.: %s\n",
		serverURLText,
		m.loginStatus,
		m.lastSyncStatus,
	)

	// Объединяем информацию о статусе и меню действий
	return statusInfo + "\n" + m.syncServerMenu.View()
}

// updateSyncServerScreen обрабатывает сообщения для экрана "Синхронизация и Сервер".
//
//nolint:gocognit,nestif // TODO: Упростить вложенность и когнитивную сложность
func (m *model) updateSyncServerScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Используем if вместо switch т.к. обрабатываем только один тип
	if keyMsg, ok := msg.(tea.KeyMsg); ok { // Внешний 'ok'
		switch keyMsg.String() {
		case keyEnter:
			selectedItem := m.syncServerMenu.SelectedItem()
			if item, itemOk := selectedItem.(syncMenuItem); itemOk { // Используем 'itemOk' для внутреннего блока
				switch item.id {
				case "configure_url":
					m.state = serverURLInputScreen
					// Устанавливаем текущий URL в поле ввода или плейсхолдер
					if m.serverURL != "" {
						m.serverURLInput.SetValue(m.serverURL)
					} else {
						m.serverURLInput.Placeholder = defaultServerURL
						m.serverURLInput.SetValue("")
					}
					m.serverURLInput.Focus()
					return m, textinput.Blink
				case "login_register":
					if m.serverURL == "" {
						// Сначала нужно настроить URL
						m.state = serverURLInputScreen
						m.serverURLInput.Placeholder = defaultServerURL
						m.serverURLInput.SetValue("")
						m.serverURLInput.Focus()
						return m, textinput.Blink
					}
					// URL есть, переходим к выбору Вход/Регистрация (убираем else)
					m.state = loginRegisterChoiceScreen
					return m, nil
				case "sync_now":
					// TODO: Реализовать логику синхронизации
					return m.setStatusMessage("TODO: Запуск синхронизации...")
				case "logout":
					// TODO: Реализовать логику выхода (очистка токена и т.д.)
					m.authToken = ""
					m.loginStatus = "Не выполнен"
					return m.setStatusMessage("Выход выполнен.")
					// case "view_versions": // TODO
				}
			}
		case keyEsc, keyBack:
			// Возврат к списку записей
			m.state = entryListScreen
			return m, nil
		}
	}

	// Обновляем список меню, если это не было KeyMsg или не обработанное
	var listCmd tea.Cmd
	m.syncServerMenu, listCmd = m.syncServerMenu.Update(msg)
	cmds = append(cmds, listCmd)

	return m, tea.Batch(cmds...)
}
