package tui

import (
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Состояния (экраны) приложения.
type screenState int

const (
	welcomeScreen       screenState = iota // Приветственный экран
	passwordInputScreen                    // Экран ввода пароля
	// TODO: Добавить другие экраны (список записей, детали записи и т.д.)
)

// Модель представляет состояние TUI приложения.
type model struct {
	state         screenState     // Текущее состояние (экран)
	passwordInput textinput.Model // Поле ввода для пароля
	// TODO: Добавить поля для хранения данных KDBX
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
	}
}

// Init - команда, выполняемая при запуске приложения.
func (m model) Init() tea.Cmd {
	// Мигание курсора в поле ввода
	return textinput.Blink
}

// Update обрабатывает входящие сообщения (события клавиатуры, мыши, команды).
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Обработка нажатия клавиш
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch m.state {
		case welcomeScreen:
			switch keyMsg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter":
				// Переход на экран ввода пароля
				m.state = passwordInputScreen
				return m, nil // Дополнительных команд не требуется
			}
		case passwordInputScreen:
			switch keyMsg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				// TODO: Проверить пароль и открыть KDBX файл
				slog.Debug("Password entered")
				// TODO: Перейти на экран списка записей при успехе
				return m, tea.Quit // Пока выходим после ввода пароля
			}
			// Обновляем состояние поля ввода пароля
			m.passwordInput, cmd = m.passwordInput.Update(keyMsg)
			return m, cmd
		}
	}

	// Если сообщение не обработано для текущего состояния,
	// возможно, оно предназначено для компонента (например, textinput)
	// Обновляем поле ввода пароля, если мы на соответствующем экране
	if m.state == passwordInputScreen {
		// Мы должны обновить поле ввода с исходным сообщением 'msg',
		// а не только с 'keyMsg', так как могут быть другие типы сообщений,
		// которые textinput может обрабатывать.
		// Однако, если мы уже обработали KeyMsg выше, то повторное обновление здесь
		// может быть не нужно или даже вредно, если KeyMsg уже обновил состояние.
		// Текущая логика обновления в конце может быть немного запутанной.
		// Давайте пока оставим этот блок как есть, но возможно, его нужно будет пересмотреть.
		m.passwordInput, cmd = m.passwordInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View отрисовывает пользовательский интерфейс.
func (m model) View() string {
	switch m.state {
	case welcomeScreen:
		// Приветственное сообщение с кратким описанием
		s := "Добро пожаловать в GophKeeper!\n\n"
		s += "Это безопасный менеджер паролей для командной строки,\n"
		s += "совместимый с форматом KDBX (KeePass).\n\n"
		s += "Нажмите Enter для продолжения или Ctrl+C/q для выхода.\n"
		return s
	case passwordInputScreen:
		// Экран ввода пароля
		s := "Введите мастер-пароль для открытия базы данных:\n\n"
		s += m.passwordInput.View() // Отображаем поле ввода
		s += "\n\n(Нажмите Enter для подтверждения или Ctrl+C для выхода)\n"
		return s
	default:
		return "Неизвестное состояние!"
	}
}

// Start запускает TUI приложение.
func Start() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		slog.Error("Error starting TUI", "error", err)
		os.Exit(1)
	}
}
