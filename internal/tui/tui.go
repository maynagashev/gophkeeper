package tui

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const statusMessageTimeout = 2 * time.Second // Время отображения статусных сообщений

// initialModel создает начальное состояние модели.
func initialModel() model {
	// Поле ввода пароля
	ti := textinput.New()
	ti.Placeholder = "Мастер-пароль"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20
	ti.EchoMode = textinput.EchoPassword

	// Компонент списка
	delegate := list.NewDefaultDelegate()
	// Настроим цвета для лучшей видимости
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.Color("252")). // Светло-серый для обычного заголовка
		Background(lipgloss.Color("235"))  // Темный фон для контраста

	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(lipgloss.Color("245")). // Темно-серый для обычного описания
		Background(lipgloss.Color("235"))  // Темный фон для контраста

	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("212")). // Яркий розовый для выделенного заголовка
		Background(lipgloss.Color("237")). // Чуть светлее фон для выделения
		BorderLeftForeground(lipgloss.Color("212"))

	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("240")). // Светло-серый для выделенного описания
		Background(lipgloss.Color("237")). // Чуть светлее фон для выделения
		BorderLeftForeground(lipgloss.Color("212"))

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Записи"
	// Убираем стандартные подсказки Quit и Help, т.к. мы их переопределим
	l.SetShowHelp(false)
	l.SetShowStatusBar(true) // Оставляем статус-бар (X items)
	l.SetFilteringEnabled(true)
	l.Styles.Title = list.DefaultStyles().Title.Bold(true)
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle

	// Список вложений для удаления
	attachmentDelList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	attachmentDelList.Title = "Выберите вложение для удаления"
	attachmentDelList.SetShowHelp(false)
	attachmentDelList.SetShowStatusBar(false)
	attachmentDelList.SetFilteringEnabled(false) // Фильтрация не нужна
	attachmentDelList.Styles.Title = list.DefaultStyles().Title.Bold(true)

	// Поле ввода пути к файлу вложения
	pathInput := textinput.New()
	pathInput.Placeholder = "/path/to/your/file"
	pathInput.CharLimit = 4096                               // Ограничение на длину пути
	pathInput.Width = defaultListWidth - passwordInputOffset // Используем ту же ширину, что и пароль

	// Поля для ввода нового пароля
	newPass1 := textinput.New()
	newPass1.Placeholder = "Новый мастер-пароль"
	newPass1.Focus() // Фокус на первом поле
	newPass1.CharLimit = 156
	newPass1.Width = 20
	newPass1.EchoMode = textinput.EchoPassword

	newPass2 := textinput.New()
	newPass2.Placeholder = "Подтвердите пароль"
	newPass2.CharLimit = 156
	newPass2.Width = 20
	newPass2.EchoMode = textinput.EchoPassword

	return model{
		state:               welcomeScreen,
		passwordInput:       ti,
		kdbxPath:            "example/test.kdbx",
		entryList:           l,
		attachmentList:      attachmentDelList,
		attachmentPathInput: pathInput,
		// Инициализируем поля для нового KDBX
		newPasswordInput1:       newPass1,
		newPasswordInput2:       newPass2,
		newPasswordFocusedField: 0, // Фокус на первом поле
	}
}

// Init - команда, выполняемая при запуске приложения.
func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

// Update обрабатывает входящие сообщения.
//
//nolint:gocognit,funlen // Снизим сложность и длину в будущем рефакторинге
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd // Собираем команды

	switch msg := msg.(type) {
	// == Глобальные сообщения (не зависят от экрана) ==
	case tea.WindowSizeMsg:
		// Обновляем размеры компонентов
		m.entryList.SetSize(msg.Width, msg.Height)
		m.passwordInput.Width = msg.Width - passwordInputOffset
		return m, nil

	case dbOpenedMsg:
		return m.handleDBOpenedMsg(msg)

	case errMsg:
		return m.handleErrorMsg(msg)

	case dbSavedMsg:
		return m.setStatusMessage("Сохранено успешно!")

	case dbSaveErrorMsg:
		return m.setStatusMessage(fmt.Sprintf("Ошибка сохранения: %v", msg.err))

	case clearStatusMsg:
		m.savingStatus = ""
		m.statusTimer = nil
		return m, nil

	// Обработка нажатия клавиш делегируется состоянию
	case tea.KeyMsg:
		// Глобальные команды (работают на всех экранах, кроме ввода пароля?)
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+s":
			// Сохраняем только из списка или деталей (не при редактировании)
			if (m.state == entryListScreen || m.state == entryDetailScreen) && m.db != nil {
				m.savingStatus = "Подготовка к сохранению..."
				slog.Info("Начало обновления m.db перед сохранением")

				// Проходим по всем элементам в списке интерфейса
				items := m.entryList.Items()
				updatedCount := 0
				for _, item := range items {
					if listItem, ok := item.(entryItem); ok {
						// Находим соответствующую запись в m.db по UUID
						dbEntryPtr := findEntryInDB(m.db, listItem.entry.UUID)
						if dbEntryPtr != nil {
							// Обновляем найденную запись данными из элемента списка
							// Создаем копию перед присваиванием, чтобы не менять listItem
							entryToSave := deepCopyEntry(listItem.entry)
							*dbEntryPtr = entryToSave
							updatedCount++
						} else {
							slog.Warn("Запись из списка не найдена в m.db", "uuid", listItem.entry.UUID)
						}
					}
				}
				slog.Info("Обновление m.db завершено", "updated_count", updatedCount)

				m.savingStatus = "Сохранение..."
				slog.Info("Запуск сохранения KDBX", "path", m.kdbxPath)
				// Используем сохраненный пароль
				return m, saveKdbxCmd(m.db, m.kdbxPath, m.password)
			}
		}
		// Если не глобальная команда, передаем дальше
	}

	// == Обновление компонентов в зависимости от состояния ==
	var updatedModel tea.Model
	var stateCmd tea.Cmd
	switch m.state {
	case welcomeScreen:
		updatedModel, stateCmd = m.updateWelcomeScreen(msg)
	case passwordInputScreen:
		updatedModel, stateCmd = m.updatePasswordInputScreen(msg)
	case newKdbxPasswordScreen:
		updatedModel, stateCmd = m.updateNewKdbxPasswordScreen(msg)
	case entryListScreen:
		updatedModel, stateCmd = m.updateEntryListScreen(msg)
	case entryDetailScreen:
		updatedModel, stateCmd = m.updateEntryDetailScreen(msg)
	case entryEditScreen:
		updatedModel, stateCmd = m.updateEntryEditScreen(msg)
	case entryAddScreen:
		updatedModel, stateCmd = m.updateEntryAddScreen(msg)
	case attachmentListDeleteScreen:
		updatedModel, stateCmd = m.updateAttachmentListDeleteScreen(msg)
	case attachmentPathInputScreen:
		updatedModel, stateCmd = m.updateAttachmentPathInputScreen(msg)
	default:
		// Неизвестное состояние - возвращаем как есть
		updatedModel = m
	}
	cmds = append(cmds, stateCmd)

	return updatedModel, tea.Batch(cmds...)
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
		help = "(↑/↓ - навигация, Enter - детали, / - поиск, a - добавить, Ctrl+S - сохр., q - выход)"
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
	default:
		mainContent = "Неизвестное состояние!"
	}

	// Добавляем статус сохранения, если он есть и мы не на определенных экранах
	statusLine := ""
	displayStatus := m.savingStatus != "" &&
		m.state != welcomeScreen &&
		m.state != passwordInputScreen &&
		m.state != attachmentPathInputScreen
	if displayStatus {
		statusLine = "\n" + m.savingStatus
	}

	// Собираем финальный вывод
	// Для list.View уже есть отступ снизу, для остальных добавляем
	if m.state == entryListScreen {
		return mainContent + help + statusLine
	}
	// Для детального, редактирования и добавления - добавляем отступ и подсказку
	if m.state == entryDetailScreen || m.state == entryEditScreen || m.state == entryAddScreen {
		return mainContent + "\n" + help + statusLine
	}

	// Для остальных (welcome, password input)
	return mainContent + "\n" + help + statusLine
}

// Start запускает TUI приложение.
func Start() {
	// Создаем начальную модель
	m := initialModel()

	// Проверяем, существует ли файл KDBX
	if _, err := os.Stat(m.kdbxPath); os.IsNotExist(err) {
		// Файл не существует, переходим на экран создания пароля
		slog.Info("Файл KDBX не найден, переходим к созданию нового.", "path", m.kdbxPath)
		m.state = newKdbxPasswordScreen
		// Устанавливаем фокус на первое поле ввода нового пароля
		m.newPasswordInput1.Focus()
		m.newPasswordInput2.Blur()
	} else if err != nil {
		// Другая ошибка при доступе к файлу
		slog.Error("Ошибка при проверке файла KDBX", "path", m.kdbxPath, "error", err)
		// Отобразим ошибку в TUI? Пока просто выйдем
		fmt.Fprintf(os.Stderr, "Ошибка доступа к файлу %s: %v\n", m.kdbxPath, err)
		os.Exit(1)
	} else {
		// Файл существует, оставляем начальное состояние (welcomeScreen -> passwordInputScreen)
		slog.Info("Файл KDBX найден, запуск стандартного TUI.", "path", m.kdbxPath)
	}

	// Используем FullAltScreen для корректной работы списка
	p := tea.NewProgram(&m, tea.WithAltScreen()) // Передаем указатель на модель '&m'
	if _, err := p.Run(); err != nil {
		slog.Error("Ошибка при запуске TUI", "error", err)
		os.Exit(1)
	}
}
