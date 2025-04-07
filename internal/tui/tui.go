package tui

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tobischo/gokeepasslib/v3"

	"github.com/maynagashev/gophkeeper/internal/kdbx"
)

// Состояния (экраны) приложения.
type screenState int

const (
	welcomeScreen       screenState = iota // Приветственный экран
	passwordInputScreen                    // Экран ввода пароля
	entryListScreen                        // Экран списка записей (TODO)
	// TODO: Добавить другие экраны (детали записи и т.д.)
)

// Модель представляет состояние TUI приложения.
type model struct {
	state         screenState            // Текущее состояние (экран)
	passwordInput textinput.Model        // Поле ввода для пароля
	db            *gokeepasslib.Database // Объект открытой базы KDBX
	kdbxPath      string                 // Путь к KDBX файлу (пока захардкожен)
	err           error                  // Последняя ошибка для отображения
	// TODO: Добавить поля для списка записей, выбранной записи и т.д.
}

// initialModel создает начальное состояние модели.
func initialModel() model {
	// Создаем поле ввода для пароля
	ti := textinput.New()
	ti.Placeholder = "Мастер-пароль"
	ti.Focus() // Сразу фокусируемся на поле
	ti.CharLimit = 156
	ti.Width = 20
	ti.EchoMode = textinput.EchoPassword // Скрываем вводимые символы

	return model{
		state:         welcomeScreen, // Начинаем с приветственного экрана
		passwordInput: ti,
		kdbxPath:      "example/test.kdbx", // TODO: Сделать путь настраиваемым
	}
}

// Init - команда, выполняемая при запуске приложения.
func (m model) Init() tea.Cmd {
	// Мигание курсора в поле ввода
	return textinput.Blink
}

// Структура для сообщения об успешном открытии файла
type dbOpenedMsg struct {
	db *gokeepasslib.Database
}

// Структура для сообщения об ошибке
type errMsg struct {
	err error
}

// Команда для асинхронного открытия файла
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
	var cmd tea.Cmd

	switch msg := msg.(type) {
	// Сообщение об успешном открытии KDBX
	case dbOpenedMsg:
		m.db = msg.db
		m.err = nil               // Сбрасываем ошибку
		m.state = entryListScreen // Переходим к списку записей (пока заглушка)
		slog.Info("База KDBX успешно открыта", "path", m.kdbxPath)
		// TODO: Реализовать экран списка записей
		return m, tea.Quit // Временно выходим

	// Сообщение об ошибке
	case errMsg:
		m.err = msg.err
		slog.Error("Ошибка при работе с KDBX", "error", m.err)
		// Остаемся на экране ввода пароля, чтобы показать ошибку
		return m, nil

	// Обработка нажатия клавиш
	case tea.KeyMsg:
		switch m.state {
		case welcomeScreen:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter":
				m.state = passwordInputScreen
				return m, textinput.Blink // Начать мигание курсора при переходе
			}
		case passwordInputScreen:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				// Запускаем команду асинхронного открытия файла
				password := m.passwordInput.Value()
				return m, openKdbxCmd(m.kdbxPath, password)
			}
			// Обновляем состояние поля ввода пароля
			m.passwordInput, cmd = m.passwordInput.Update(msg)
			return m, cmd
		}
	}

	// Обновляем поле ввода пароля, если мы на соответствующем экране
	if m.state == passwordInputScreen {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
		return m, cmd
	}

	return m, nil
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
		// Отображаем ошибку, если она есть
		if m.err != nil {
			s += "Ошибка: " + m.err.Error() + "\n\n"
		}
		s += "(Нажмите Enter для подтверждения или Ctrl+C для выхода)\n"
		return s
	case entryListScreen:
		// Заглушка для экрана списка записей
		entryCount := 0
		if m.db != nil && m.db.Content != nil && m.db.Content.Root != nil {
			for _, group := range m.db.Content.Root.Groups {
				entryCount += len(group.Entries)
			}
		}
		return fmt.Sprintf("База '%s' успешно открыта!\nСодержит %d групп и %d записей.\n\n(Нажмите Ctrl+C для выхода)",
			m.kdbxPath, len(m.db.Content.Root.Groups), entryCount)
	default:
		return "Неизвестное состояние!"
	}
}

// Start запускает TUI приложение.
func Start() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		slog.Error("Ошибка при запуске TUI", "error", err)
		os.Exit(1)
	}
}
