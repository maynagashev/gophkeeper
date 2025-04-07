package tui

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tobischo/gokeepasslib/v3"

	"github.com/maynagashev/gophkeeper/internal/kdbx"
)

// Состояния (экраны) приложения.
type screenState int

const (
	welcomeScreen       screenState = iota // Приветственный экран
	passwordInputScreen                    // Экран ввода пароля
	entryListScreen                        // Экран списка записей
	// TODO: Добавить другие экраны (детали записи и т.д.)
)

// Константы для TUI.
const (
	defaultListWidth    = 80 // Стандартная ширина терминала для списка
	defaultListHeight   = 24 // Стандартная высота терминала для списка
	passwordInputOffset = 4  // Отступ для поля ввода пароля
)

// entryItem представляет элемент списка записей.
// Реализует интерфейс list.Item.
type entryItem struct {
	entry gokeepasslib.Entry
}

func (i entryItem) Title() string {
	// Пытаемся получить значение поля "Title"
	title := i.entry.GetTitle()
	if title == "" {
		// Если Title пустой, используем Username
		title = i.entry.GetContent("UserName")
	}
	if title == "" {
		// Если и Username пустой, используем UUID
		title = hex.EncodeToString(i.entry.UUID[:])
	}
	return title
}

func (i entryItem) Description() string {
	// В описании можно показать Username или URL
	username := i.entry.GetContent("UserName")
	url := i.entry.GetContent("URL")
	switch {
	case username != "" && url != "":
		return fmt.Sprintf("User: %s | URL: %s", username, url)
	case username != "":
		return fmt.Sprintf("User: %s", username)
	case url != "":
		return fmt.Sprintf("URL: %s", url)
	default:
		return ""
	}
}

func (i entryItem) FilterValue() string { return i.Title() }

// Модель представляет состояние TUI приложения.
type model struct {
	state         screenState            // Текущее состояние (экран)
	passwordInput textinput.Model        // Поле ввода для пароля
	db            *gokeepasslib.Database // Объект открытой базы KDBX
	kdbxPath      string                 // Путь к KDBX файлу (пока захардкожен)
	err           error                  // Последняя ошибка для отображения
	entryList     list.Model             // Компонент списка записей
}

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

	return model{
		state:         welcomeScreen,
		passwordInput: ti,
		kdbxPath:      "example/test.kdbx",
		entryList:     l,
	}
}

// Init - команда, выполняемая при запуске приложения.
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Структура для сообщения об успешном открытии файла.
type dbOpenedMsg struct {
	db *gokeepasslib.Database
}

// Структура для сообщения об ошибке.
type errMsg struct {
	err error
}

// Команда для асинхронного открытия файла.
func openKdbxCmd(path, password string) tea.Cmd {
	return func() tea.Msg {
		db, err := kdbx.OpenFile(path, password)
		if err != nil {
			return errMsg{err: err}
		}
		return dbOpenedMsg{db: db}
	}
}

// Update обрабатывает входящие сообщения.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// var cmd tea.Cmd
	// var cmds []tea.Cmd // Собираем команды

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
		m.err = msg.err
		slog.Error("Ошибка при работе с KDBX", "error", m.err)
		m.passwordInput.Blur() // Снимаем фокус, чтобы показать ошибку
		return m, nil

	// Обработка нажатия клавиш делегируется состоянию
	case tea.KeyMsg:
		// Сочетание Ctrl+C всегда приводит к выходу
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}

	// == Обновление компонентов в зависимости от состояния ==
	switch m.state {
	case welcomeScreen:
		return m.updateWelcomeScreen(msg)
	case passwordInputScreen:
		return m.updatePasswordInputScreen(msg)
	case entryListScreen:
		return m.updateEntryListScreen(msg)
	default:
		// Для неизвестных состояний возвращаем модель без изменений и команд
		return m, nil
	}

	// Возвращаем модель и собранные команды
	// return m, tea.Batch(cmds...)
}

// updateWelcomeScreen обрабатывает сообщения для экрана приветствия.
func (m *model) updateWelcomeScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "q" {
			return m, tea.Quit
		} else if keyMsg.String() == "enter" {
			m.state = passwordInputScreen
			m.passwordInput.Focus()
			// Добавляем явную очистку экрана при переходе
			cmds = append(cmds, textinput.Blink, tea.ClearScreen)
		}
	}
	return m, tea.Batch(cmds...)
}

// updatePasswordInputScreen обрабатывает сообщения для экрана ввода пароля.
func (m *model) updatePasswordInputScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Сначала обновляем поле ввода
	m.passwordInput, cmd = m.passwordInput.Update(msg)
	cmds = append(cmds, cmd)

	// Обработка клавиш для экрана ввода пароля
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Если была ошибка, любое нажатие ее скрывает
		if m.err != nil {
			m.err = nil
			m.passwordInput.Focus() // Возвращаем фокус
			cmds = append(cmds, textinput.Blink)
			// Не обрабатываем другие клавиши в этом цикле
		} else if keyMsg.String() == "enter" {
			password := m.passwordInput.Value()
			m.passwordInput.Blur()
			m.passwordInput.Reset()
			cmds = append(cmds, openKdbxCmd(m.kdbxPath, password))
		}
	}
	return m, tea.Batch(cmds...)
}

// updateEntryListScreen обрабатывает сообщения для экрана списка записей.
func (m *model) updateEntryListScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Сначала обновляем список
	m.entryList, cmd = m.entryList.Update(msg)
	cmds = append(cmds, cmd)

	// Обработка клавиш для экрана списка
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "q" {
			// Выход по 'q', если не активен режим фильтрации
			if m.entryList.FilterState() == list.Unfiltered {
				return m, tea.Quit
			}
			// TODO: Обработка Enter для выбора записи
		}
	}
	return m, tea.Batch(cmds...)
}

// handleDBOpenedMsg обрабатывает сообщение об успешном открытии базы.
func (m *model) handleDBOpenedMsg(msg dbOpenedMsg) (tea.Model, tea.Cmd) {
	m.db = msg.db
	m.err = nil
	prevState := m.state // Сохраняем предыдущее состояние
	m.state = entryListScreen
	slog.Info("База KDBX успешно открыта", "path", m.kdbxPath)

	entries := kdbx.GetAllEntries(m.db)
	slog.Debug("Записи, полученные из KDBX", "count", len(entries))

	items := make([]list.Item, len(entries))
	for i, entry := range entries {
		items[i] = entryItem{entry: entry}
	}

	// Перед установкой элементов, проверим их количество
	slog.Debug("Элементы, подготовленные для списка", "count", len(items))
	m.entryList.SetItems(items)

	// Проверим количество элементов в списке после установки
	slog.Debug("Элементы в списке после SetItems", "count", len(m.entryList.Items()))

	// Установим размер списка явно
	m.entryList.SetWidth(defaultListWidth)
	m.entryList.SetHeight(defaultListHeight)

	m.entryList.Title = fmt.Sprintf("Записи в '%s' (%d)", m.kdbxPath, len(items))

	// Явно очищаем экран при переходе на список записей
	dbOpenedCmds := []tea.Cmd{}
	if prevState != entryListScreen {
		dbOpenedCmds = append(dbOpenedCmds, tea.ClearScreen)
	}

	return m, tea.Batch(dbOpenedCmds...)
}

// View отрисовывает пользовательский интерфейс.
func (m model) View() string {
	switch m.state {
	case welcomeScreen:
		s := "Добро пожаловать в GophKeeper!\n\n"
		s += "Это безопасный менеджер паролей для командной строки,\n"
		s += "совместимый с форматом KDBX (KeePass).\n\n"
		s += "Нажмите Enter для продолжения или Ctrl+C/q для выхода.\n"
		return s
	case passwordInputScreen:
		s := "Введите мастер-пароль для открытия базы данных: " + m.kdbxPath + "\n\n"
		s += m.passwordInput.View() + "\n\n"
		if m.err != nil {
			errMsgStr := fmt.Sprintf("\nОшибка: %s\n\n(Нажмите любую клавишу для продолжения)", m.err)
			return s + errMsgStr // Возвращаем основной текст + текст ошибки
		}
		s += "(Нажмите Enter для подтверждения или Ctrl+C для выхода)\n"
		return s
	case entryListScreen:
		// Временно возвращаем простую строку для теста очистки экрана
		// return "ЭКРАН СПИСКА ЗАПИСЕЙ\n\n(Нажмите q для выхода)"
		return m.entryList.View()
	default:
		return "Неизвестное состояние!"
	}
}

// Start запускает TUI приложение.
func Start() {
	// Используем FullAltScreen для корректной работы списка
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		slog.Error("Ошибка при запуске TUI", "error", err)
		os.Exit(1)
	}
}
