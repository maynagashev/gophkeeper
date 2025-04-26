package tui

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofrs/flock"

	"github.com/maynagashev/gophkeeper/client/internal/api" // Импортируем пакет API клиента
	// Импортируем пакет models для использования в debug info.
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

// getMainContentView возвращает основное содержимое для текущего состояния.
func (m *model) getMainContentView() string {
	switch m.state {
	case welcomeScreen:
		return m.viewWelcomeScreen()
	case passwordInputScreen:
		return m.viewPasswordInputScreen()
	case newKdbxPasswordScreen:
		return m.viewNewKdbxPasswordScreen()
	case entryListScreen:
		return m.entryList.View()
	case entryDetailScreen:
		return m.viewEntryDetailScreen()
	case entryEditScreen:
		return m.viewEntryEditScreen()
	case entryAddScreen:
		return m.viewEntryAddScreen()
	case attachmentListDeleteScreen:
		return m.viewAttachmentListDeleteScreen()
	case attachmentPathInputScreen:
		return m.viewAttachmentPathInputScreen()
	case syncServerScreen:
		return m.viewSyncServerScreen()
	case serverURLInputScreen:
		return m.viewServerURLInputScreen()
	case loginRegisterChoiceScreen:
		return m.viewLoginRegisterChoiceScreen()
	case loginScreen:
		return m.viewLoginScreen()
	case registerScreen:
		return m.viewRegisterScreen()
	case versionListScreen:
		return m.viewVersionListScreen()
	default:
		return "Неизвестное состояние!"
	}
}

// Helper function to get the main content and help string based on the state.
func (m *model) getContentAndHelp() (string, string) {
	mainContent := m.getMainContentView()
	// Используем карту из модели
	help, ok := m.helpTextMap[m.state]
	if !ok {
		help = "Unknown state" // Default help for unknown state
	}
	return mainContent, help
}

// getDBModTimeString возвращает отформатированную строку времени модификации для БД.
func (m *model) getDBModTimeString() string {
	// Предполагаем, что db, Content и Root проверены перед вызовом
	if len(m.db.Content.Root.Groups) == 0 {
		return "<not set>" // No groups to get time from
	}
	rootGroup := &m.db.Content.Root.Groups[0]
	if rootGroup.Times.LastModificationTime != nil {
		return rootGroup.Times.LastModificationTime.Time.Format(time.RFC3339) + " (group0 mod)"
	}
	if rootGroup.Times.CreationTime != nil {
		return rootGroup.Times.CreationTime.Time.Format(time.RFC3339) + " (group0 creation)"
	}
	return "<not set>"
}

// getDBDebugInfo генерирует отладочную информацию, связанную с базой данных.
func (m *model) getDBDebugInfo() string {
	if m.db == nil {
		return " [DB: not loaded]\n"
	}
	if m.db.Content == nil || m.db.Content.Root == nil || m.db.Content.Meta == nil {
		return " [DB: Content, Root or Meta missing]\n"
	}

	var dbDebugInfo strings.Builder
	dbDebugInfo.WriteString(fmt.Sprintf(" [DB Name: %s]\n", m.db.Content.Meta.DatabaseName))
	modTimeStr := m.getDBModTimeString()
	dbDebugInfo.WriteString(fmt.Sprintf(" [DB ModTime: %s]\n", modTimeStr))
	return dbDebugInfo.String()
}

// Helper function to generate the debug info string.
func (m *model) getDebugInfoString() string {
	var debugInfo strings.Builder
	debugInfo.WriteString(fmt.Sprintf(" [State: %s]\n", m.state.String()))
	debugInfo.WriteString(fmt.Sprintf(" [URL: %s]\n", m.serverURL))
	debugInfo.WriteString(fmt.Sprintf(" [Token: %s]\n", m.authToken)) // Keep showing token in debug
	debugInfo.WriteString(fmt.Sprintf(" [Lock Acquired: %t]\n", m.lockAcquired))

	// Добавляем информацию о БД через helper
	dbDebugInfo := m.getDBDebugInfo()
	debugInfo.WriteString(dbDebugInfo)

	return debugInfo.String()
}

// View отрисовывает пользовательский интерфейс.
func (m *model) View() string {
	mainContent, help := m.getContentAndHelp()

	// --- Формируем подвал (статус + отладка) --- //
	var footer strings.Builder

	// Добавляем статус, если он есть
	readOnlyIndicator := ""
	if m.readOnlyMode {
		readOnlyIndicator = " [Read-Only]"
	}
	displayStatus := m.savingStatus != "" || m.readOnlyMode
	if displayStatus {
		footer.WriteString("\n") // Перенос перед статусом
		footer.WriteString(m.savingStatus)
		footer.WriteString(readOnlyIndicator)
	}

	// Добавляем отладку, если включен режим
	if m.debugMode {
		// Убедимся, что help для неизвестного состояния установлен, если нужно
		if help == "Unknown state" {
			help = fmt.Sprintf("State: %s", m.state.String())
		}
		// Добавляем разделитель и отладку
		footer.WriteString("\n\n---\nОтладка:\n") // Двойной перенос перед отладкой
		footer.WriteString(m.getDebugInfoString())
	}

	// Собираем финальный вывод
	styledContent := m.docStyle.Render(mainContent)
	// Сначала основной контент, потом помощь, потом весь подвал
	return fmt.Sprintf("%s\n%s%s", styledContent, help, footer.String())
}

// Start запускает TUI приложение.
func Start(kdbxPath string, debugMode bool, serverURL string) {
	// --- Инициализация API клиента ---
	var apiClient api.Client // Объявляем переменную
	if serverURL != "" {     // Создаем клиент, только если URL не пустой
		apiClient = api.NewHTTPClient(serverURL)
		slog.Info("API клиент инициализирован", "baseURL", serverURL)
	} else {
		slog.Warn("URL сервера не указан (--server-url), функции API будут недоступны.")
		// apiClient остается nil
	}

	// Создаем начальную модель, передавая флаг
	m := initModel(kdbxPath, debugMode, serverURL, apiClient)

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
