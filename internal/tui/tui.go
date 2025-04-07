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
	w "github.com/tobischo/gokeepasslib/v3/wrappers"

	"github.com/maynagashev/gophkeeper/internal/kdbx"
)

// Состояния (экраны) приложения.
type screenState int

const (
	welcomeScreen       screenState = iota // Приветственный экран
	passwordInputScreen                    // Экран ввода пароля
	entryListScreen                        // Экран списка записей
	entryDetailScreen                      // Экран деталей записи
	entryEditScreen                        // Экран редактирования записи
	// TODO: Добавить другие экраны (детали записи и т.д.)
)

// Поля, доступные для редактирования.
const (
	editableFieldTitle = iota
	editableFieldUserName
	editableFieldPassword
	editableFieldURL
	editableFieldNotes
	numEditableFields // Количество редактируемых полей

	fieldNamePassword = "Password"
)

// Константы для TUI.
const (
	defaultListWidth    = 80 // Стандартная ширина терминала для списка
	defaultListHeight   = 24 // Стандартная высота терминала для списка
	passwordInputOffset = 4  // Отступ для поля ввода пароля

	keyEnter = "enter" // Клавиша Enter
	keyQuit  = "q"     // Клавиша выхода
	keyBack  = "b"     // Клавиша возврата
	keyEsc   = "esc"   // Клавиша Escape
	keyEdit  = "e"     // Клавиша редактирования
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
	password      string                 // Сохраненный в памяти пароль от базы (для применения изменений)
	db            *gokeepasslib.Database // Объект открытой базы KDBX
	kdbxPath      string                 // Путь к KDBX файлу (пока захардкожен)
	err           error                  // Последняя ошибка для отображения
	entryList     list.Model             // Компонент списка записей
	selectedEntry *entryItem             // Выбранная запись для детального просмотра

	// Поля для редактирования записи
	editingEntry *gokeepasslib.Entry // Копия записи, которую редактируем
	editInputs   []textinput.Model   // Поля ввода для редактирования
	focusedField int                 // Индекс активного поля ввода

	savingStatus string // Статус операции сохранения файла
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

// Структуры для сообщений о сохранении
type dbSavedMsg struct{}

type dbSaveErrorMsg struct {
	err error
}

// Команда для асинхронного сохранения файла
func saveKdbxCmd(db *gokeepasslib.Database, path, password string) tea.Cmd {
	return func() tea.Msg {
		err := kdbx.SaveFile(db, path, password)
		if err != nil {
			return dbSaveErrorMsg{err: err}
		}
		return dbSavedMsg{}
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
		switch keyMsg.String() {
		case keyQuit:
			return m, tea.Quit
		case keyEnter:
			m.state = passwordInputScreen
			m.passwordInput.Focus()
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
		} else if keyMsg.String() == keyEnter {
			password := m.passwordInput.Value()
			m.passwordInput.Blur()
			m.passwordInput.Reset()
			// Сохраняем пароль в модели перед отправкой команды
			m.password = password
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
		}
	}
	return m, tea.Batch(cmds...)
}

// updateEntryDetailScreen обрабатывает сообщения для экрана деталей записи.
func (m *model) updateEntryDetailScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEsc, keyBack:
			m.state = entryListScreen
			m.selectedEntry = nil // Сбрасываем выбранную запись
			slog.Info("Возврат к списку записей")
			return m, tea.ClearScreen
		case keyEdit:
			if m.selectedEntry != nil {
				m.prepareEditScreen()
				m.state = entryEditScreen
				slog.Info("Переход к редактированию записи", "title", m.selectedEntry.Title())
				return m, tea.ClearScreen
			}
		}
	}
	return m, nil
}

// deepCopyEntry создает глубокую копию записи gokeepasslib.Entry.
func deepCopyEntry(original gokeepasslib.Entry) gokeepasslib.Entry {
	newEntry := gokeepasslib.NewEntry()

	// Копируем UUID
	copy(newEntry.UUID[:], original.UUID[:])

	// Копируем основные поля (простые типы копируются по значению)
	newEntry.Times = original.Times
	newEntry.Tags = original.Tags             // Строки неизменяемы, можно копировать напрямую
	newEntry.CustomData = original.CustomData // Тоже карта строк, копируем
	// TODO: Добавить копирование других полей при необходимости (AutoType, History, CustomIcons)

	// Глубокое копирование среза Values
	if original.Values != nil {
		newEntry.Values = make([]gokeepasslib.ValueData, len(original.Values))
		for i, val := range original.Values {
			newValue := gokeepasslib.ValueData{
				Key:   val.Key,
				Value: gokeepasslib.V{Content: val.Value.Content, Protected: val.Value.Protected},
			}
			newEntry.Values[i] = newValue
		}
	}

	return newEntry
}

// prepareEditScreen инициализирует поля для экрана редактирования.
func (m *model) prepareEditScreen() {
	if m.selectedEntry == nil {
		return // Нечего редактировать
	}

	// Создаем глубокую копию записи для редактирования
	entryCopy := deepCopyEntry(m.selectedEntry.entry)
	m.editingEntry = &entryCopy

	m.editInputs = make([]textinput.Model, numEditableFields)
	m.focusedField = editableFieldTitle // Начинаем с поля Title

	placeholders := map[int]string{
		editableFieldTitle:    "Title",
		editableFieldUserName: "UserName",
		editableFieldPassword: "Password",
		editableFieldURL:      "URL",
		editableFieldNotes:    "Notes",
	}

	for i := range numEditableFields {
		m.editInputs[i] = textinput.New()
		m.editInputs[i].Placeholder = placeholders[i]
		m.editInputs[i].SetValue(m.editingEntry.GetContent(placeholders[i]))
		// Первое поле делаем активным
		if i == m.focusedField {
			m.editInputs[i].Focus()
		}
	}

	// Настроим поле пароля
	m.editInputs[editableFieldPassword].EchoMode = textinput.EchoPassword
}

// updateEntryEditScreen обрабатывает сообщения для экрана редактирования записи.
func (m *model) updateEntryEditScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Обрабатываем только KeyMsg
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEsc, keyBack:
			// Отмена редактирования
			m.state = entryDetailScreen
			m.editingEntry = nil // Сбрасываем редактируемую запись
			m.editInputs = nil   // Очищаем поля ввода
			slog.Info("Отмена редактирования, возврат к деталям записи")
			return m, tea.ClearScreen

		case "tab", "down":
			// Переход к следующему полю
			m.focusedField = (m.focusedField + 1) % numEditableFields
			cmds = m.updateFocus()
			return m, tea.Batch(cmds...)

		case "shift+tab", "up":
			// Переход к предыдущему полю
			m.focusedField = (m.focusedField - 1 + numEditableFields) % numEditableFields
			cmds = m.updateFocus()
			return m, tea.Batch(cmds...)

		case keyEnter:
			// Сохранение изменений
			if m.selectedEntry != nil && m.editingEntry != nil {

				// 1. Создаем финальную обновленную запись на основе editingEntry
				finalUpdatedEntry := deepCopyEntry(*m.editingEntry) // Используем deepCopy на всякий случай
				// Обновляем время модификации
				now := w.Now()
				finalUpdatedEntry.Times.LastModificationTime = &now

				// 2. Создаем новый элемент списка с обновленной записью
				newSelectedItem := entryItem{entry: finalUpdatedEntry}

				// 3. Обновляем элемент в списке list.Model
				idx := m.entryList.Index()
				updateCmd := m.entryList.SetItem(idx, newSelectedItem) // Передаем новый элемент
				cmds = append(cmds, updateCmd)

				// 4. Обновляем selectedEntry в модели, чтобы он указывал на новый элемент
				m.selectedEntry = &newSelectedItem

				// 5. Возвращаемся к деталям и очищаем состояние редактирования
				m.state = entryDetailScreen
				m.editingEntry = nil
				m.editInputs = nil
				slog.Info("Изменения сохранены, возврат к деталям записи")
				// Добавим ClearScreen к другим командам
				cmds = append(cmds, tea.ClearScreen)
				return m, tea.Batch(cmds...)
			}
		}
	} // конец if keyMsg, ok := msg.(tea.KeyMsg)

	// Если сообщение не KeyMsg или было обработано выше (кроме навигации/Enter/Esc),
	// обновляем активное поле ввода.
	var cmd tea.Cmd
	m.editInputs[m.focusedField], cmd = m.editInputs[m.focusedField].Update(msg)
	cmds = append(cmds, cmd)

	// Обновляем соответствующее поле в копии записи
	fieldName := m.editInputs[m.focusedField].Placeholder
	newValue := m.editInputs[m.focusedField].Value()

	// Ищем существующее значение или создаем новое
	found := false
	for i := range m.editingEntry.Values {
		if m.editingEntry.Values[i].Key == fieldName {
			m.editingEntry.Values[i].Value.Content = newValue
			// Обработка Protected для поля Password
			if fieldName == fieldNamePassword {
				m.editingEntry.Values[i].Value.Protected = w.NewBoolWrapper(newValue != "")
			}
			found = true
			break
		}
	}
	// Если значение не найдено, добавляем новое
	if !found {
		valueData := gokeepasslib.ValueData{
			Key:   fieldName,
			Value: gokeepasslib.V{Content: newValue},
		}
		if fieldName == fieldNamePassword {
			valueData.Value.Protected = w.NewBoolWrapper(newValue != "")
		}
		m.editingEntry.Values = append(m.editingEntry.Values, valueData)
	}

	return m, tea.Batch(cmds...)
}

// updateFocus обновляет фокус полей ввода и возвращает команды Blink.
func (m *model) updateFocus() []tea.Cmd {
	cmds := make([]tea.Cmd, len(m.editInputs))
	for i := range len(m.editInputs) {
		if i == m.focusedField {
			cmds[i] = m.editInputs[i].Focus()
		} else {
			m.editInputs[i].Blur()
		}
	}
	return cmds
}

// handleDBOpenedMsg обрабатывает сообщение об успешном открытии базы.
func (m *model) handleDBOpenedMsg(msg dbOpenedMsg) (tea.Model, tea.Cmd) {
	m.db = msg.db
	m.err = nil
	// Пароль уже сохранен в m.password при вызове openKdbxCmd
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
	var mainContent string
	switch m.state {
	case welcomeScreen:
		mainContent = m.viewWelcomeScreen()
	case passwordInputScreen:
		mainContent = m.viewPasswordInputScreen()
	case entryListScreen:
		mainContent = m.entryList.View()
	case entryDetailScreen:
		mainContent = m.viewEntryDetailScreen()
	case entryEditScreen:
		mainContent = m.viewEntryEditScreen()
	default:
		mainContent = "Неизвестное состояние!"
	}

	// Добавляем статус сохранения, если он есть
	if m.savingStatus != "" && m.state != welcomeScreen && m.state != passwordInputScreen {
		return mainContent + "\n\n" + m.savingStatus
	}
	return mainContent
}

// viewWelcomeScreen отрисовывает экран приветствия.
func (m model) viewWelcomeScreen() string {
	s := "Добро пожаловать в GophKeeper!\n\n"
	s += "Это безопасный менеджер паролей для командной строки,\n"
	s += "совместимый с форматом KDBX (KeePass).\n\n"
	s += "Нажмите Enter для продолжения или Ctrl+C/q для выхода.\n"
	return s
}

// viewPasswordInputScreen отрисовывает экран ввода пароля.
func (m model) viewPasswordInputScreen() string {
	s := "Введите мастер-пароль для открытия базы данных: " + m.kdbxPath + "\n\n"
	s += m.passwordInput.View() + "\n\n"
	if m.err != nil {
		errMsgStr := fmt.Sprintf("\nОшибка: %s\n\n(Нажмите любую клавишу для продолжения)", m.err)
		return s + errMsgStr // Возвращаем основной текст + текст ошибки
	}
	s += "(Нажмите Enter для подтверждения или Ctrl+C для выхода)\n"
	return s
}

// viewEntryDetailScreen отрисовывает экран деталей записи.
func (m model) viewEntryDetailScreen() string {
	if m.selectedEntry == nil {
		return "Ошибка: Запись не выбрана!" // Такого не должно быть, но на всякий случай
	}

	s := fmt.Sprintf("Детали записи: %s\n\n", m.selectedEntry.Title())
	for _, val := range m.selectedEntry.entry.Values {
		// Пока не будем показывать пароли
		if val.Key == "Password" {
			s += fmt.Sprintf("%s: ********\n", val.Key)
		} else {
			s += fmt.Sprintf("%s: %s\n", val.Key, val.Value.Content)
		}
	}
	s += "\n(Нажмите Esc или b для возврата к списку)"
	return s
}

// viewEntryEditScreen отрисовывает экран редактирования записи.
func (m model) viewEntryEditScreen() string {
	if m.editingEntry == nil || len(m.editInputs) == 0 {
		return "Ошибка: Нет данных для редактирования!"
	}

	s := "Редактирование записи: " + m.editingEntry.GetTitle() + "\n\n"
	for i, input := range m.editInputs {
		// Добавляем индикатор фокуса
		focusIndicator := "  "
		if m.focusedField == i {
			focusIndicator = "> " // Или другой индикатор, например, стиль
		}
		s += fmt.Sprintf("%s%s: %s\n", focusIndicator, input.Placeholder, input.View())
	}
	s += "\n(Tab/Shift+Tab - навигация, Esc/b - отмена)"
	// TODO: Добавить подсказку про Enter для сохранения
	return s
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
