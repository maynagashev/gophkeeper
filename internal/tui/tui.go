package tui

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

// Update обрабатывает входящие сообщения.
//
//nolint:gocognit,funlen // Снизим сложность и длину в будущем рефакторинге
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
		return m.handleErrorMsg(msg)

	case dbSavedMsg:
		m.savingStatus = "Сохранено успешно!"
		slog.Info("База KDBX успешно сохранена", "path", m.kdbxPath)
		// Можно добавить таймер для скрытия сообщения через пару секунд
		return m, nil

	case dbSaveErrorMsg:
		m.savingStatus = fmt.Sprintf("Ошибка сохранения: %v", msg.err)
		slog.Error("Ошибка сохранения KDBX", "path", m.kdbxPath, "error", msg.err)
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
	switch m.state {
	case welcomeScreen:
		return m.updateWelcomeScreen(msg)
	case passwordInputScreen:
		return m.updatePasswordInputScreen(msg)
	case entryListScreen:
		return m.updateEntryListScreen(msg)
	case entryDetailScreen:
		return m.updateEntryDetailScreen(msg)
	case entryEditScreen:
		return m.updateEntryEditScreen(msg)
	case entryAddScreen:
		return m.updateEntryAddScreen(msg)
	default:
		// Для неизвестных состояний возвращаем модель без изменений и команд
		return m, nil
	}

	// Возвращаем модель и собранные команды
	// return m, tea.Batch(cmds...)
}

// View отрисовывает пользовательский интерфейс.
func (m model) View() string {
	var mainContent string
	var help string

	switch m.state {
	case welcomeScreen:
		mainContent = m.viewWelcomeScreen()
		help = "(Enter - продолжить, Ctrl+C/q - выход)"
	case passwordInputScreen:
		mainContent = m.viewPasswordInputScreen()
		help = "(Enter - подтвердить, Ctrl+C - выход)"
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
	default:
		mainContent = "Неизвестное состояние!"
	}

	// Добавляем статус сохранения, если он есть
	statusLine := ""
	if m.savingStatus != "" && m.state != welcomeScreen && m.state != passwordInputScreen {
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
	// Используем FullAltScreen для корректной работы списка
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		slog.Error("Ошибка при запуске TUI", "error", err)
		os.Exit(1)
	}
}
