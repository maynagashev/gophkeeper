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
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"

	"github.com/maynagashev/gophkeeper/internal/kdbx"
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
		case keyAdd:
			// Переход к добавлению новой записи
			m.prepareAddScreen()
			m.state = entryAddScreen
			slog.Info("Переход к добавлению новой записи")
			return m, tea.ClearScreen
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
			return m.saveEntryChanges()
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

// saveEntryChanges применяет изменения из editingEntry к selectedEntry и списку.
func (m *model) saveEntryChanges() (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if m.selectedEntry == nil || m.editingEntry == nil {
		slog.Warn("Попытка сохранить изменения без выбранной или редактируемой записи")
		return m, nil // Ничего не делаем
	}

	// 1. Создаем финальную обновленную запись на основе editingEntry
	finalUpdatedEntry := deepCopyEntry(*m.editingEntry) // Используем deepCopy на всякий случай
	// Обновляем время модификации
	now := time.Now()
	finalUpdatedEntry.Times.LastModificationTime = &w.TimeWrapper{Time: now} // Создаем обертку и берем указатель

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

// updateEntryAddScreen обрабатывает сообщения для экрана добавления записи.
//
//nolint:gocognit,nestif // Сложность из-за обработки разных клавиш и навигации
func (m *model) updateEntryAddScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Обрабатываем только KeyMsg
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEsc, keyBack:
			// Отмена добавления
			m.state = entryListScreen
			m.addInputs = nil // Очищаем поля ввода
			slog.Info("Отмена добавления, возврат к списку")
			return m, tea.ClearScreen

		case "tab", "down":
			// Переход к следующему полю
			m.focusedFieldAdd = (m.focusedFieldAdd + 1) % numEditableFields
			cmds = m.updateFocusAdd()
			return m, tea.Batch(cmds...)

		case "shift+tab", "up":
			// Переход к предыдущему полю
			m.focusedFieldAdd = (m.focusedFieldAdd - 1 + numEditableFields) % numEditableFields
			cmds = m.updateFocusAdd()
			return m, tea.Batch(cmds...)

		case keyEnter:
			// Создание новой записи (пока только в памяти)
			newEntry := gokeepasslib.NewEntry()
			now := time.Now() // Получаем текущее время один раз
			// Создаем обертки и берем указатели
			creationTime := w.TimeWrapper{Time: now}
			modificationTime := w.TimeWrapper{Time: now}
			accessTime := w.TimeWrapper{Time: now}
			newEntry.Times.CreationTime = &creationTime
			newEntry.Times.LastModificationTime = &modificationTime
			newEntry.Times.LastAccessTime = &accessTime

			for _, input := range m.addInputs {
				fieldName := input.Placeholder
				newValue := input.Value()
				if newValue != "" { // Добавляем поле, только если оно не пустое
					valueData := gokeepasslib.ValueData{
						Key:   fieldName,
						Value: gokeepasslib.V{Content: newValue},
					}
					if fieldName == fieldNamePassword {
						valueData.Value.Protected = w.NewBoolWrapper(true)
					}
					newEntry.Values = append(newEntry.Values, valueData)
				}
			}

			// Добавляем newEntry в m.db (в корневую группу или первую подгруппу)
			if m.db != nil && m.db.Content != nil && m.db.Content.Root != nil {
				if len(m.db.Content.Root.Groups) > 0 {
					// Добавляем в первую группу
					m.db.Content.Root.Groups[0].Entries = append(m.db.Content.Root.Groups[0].Entries, newEntry)
				} else {
					// Если групп нет, добавляем в корневую псевдо-группу (не совсем правильно для KDBX, но для демо)
					// Правильнее было бы создать группу по умолчанию, если ее нет.
					slog.Warn("Корневая группа не найдена, добавляем запись напрямую в root (может быть некорректно)")
					// m.db.Content.Root.Entries = append(m.db.Content.Root.Entries, newEntry) // У Root нет Entries
					// Пока просто не добавляем в db если нет групп
					slog.Error("Не удалось добавить запись в m.db: нет групп")
				}
			} else {
				slog.Error("Не удалось добавить запись в m.db: база данных или Root не инициализированы")
			}

			// Добавляем newEntry в m.entryList
			newItem := entryItem{entry: newEntry}
			// Добавляем в конец списка
			insertCmd := m.entryList.InsertItem(len(m.entryList.Items()), newItem)
			// Обновляем заголовок списка
			m.entryList.Title = fmt.Sprintf("Записи в '%s' (%d)", m.kdbxPath, len(m.entryList.Items()))

			slog.Info("Новая запись добавлена", "title", newEntry.GetTitle())

			// Возвращаемся к списку
			m.state = entryListScreen
			m.addInputs = nil
			// Добавляем команду обновления списка к команде очистки экрана
			return m, tea.Batch(tea.ClearScreen, insertCmd)
		}
	} // конец if keyMsg, ok := msg.(tea.KeyMsg)

	// Если сообщение не KeyMsg или было обработано выше (кроме навигации/Enter/Esc),
	// обновляем активное поле ввода.
	var cmd tea.Cmd
	m.addInputs[m.focusedFieldAdd], cmd = m.addInputs[m.focusedFieldAdd].Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// updateFocusAdd обновляет фокус полей ввода для экрана добавления.
func (m *model) updateFocusAdd() []tea.Cmd {
	cmds := make([]tea.Cmd, len(m.addInputs))
	for i := range len(m.addInputs) {
		if i == m.focusedFieldAdd {
			cmds[i] = m.addInputs[i].Focus()
		} else {
			m.addInputs[i].Blur()
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

// handleErrorMsg обрабатывает сообщение об ошибке.
func (m *model) handleErrorMsg(msg errMsg) (tea.Model, tea.Cmd) {
	m.err = msg.err
	slog.Error("Ошибка при работе с KDBX", "error", m.err)
	m.passwordInput.Blur() // Снимаем фокус, чтобы показать ошибку
	return m, nil
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
		help = "(e - ред., Ctrl+S - сохр., Esc/b - назад)" // Уже добавлено в viewEntryDetailScreen
	case entryEditScreen:
		mainContent = m.viewEntryEditScreen()
		help = "(Tab/↑/↓ - навигация, Enter - сохр., Esc/b - отмена)" // Обновим и здесь
	case entryAddScreen:
		mainContent = m.viewEntryAddScreen()
		help = "(Enter - добавить, Ctrl+C - выход)"
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

// viewPasswordInputScreen отрисовывает экран ввода пароля.
func (m model) viewPasswordInputScreen() string {
	s := "Введите мастер-пароль для открытия базы данных: " + m.kdbxPath + "\n\n"
	s += m.passwordInput.View() + "\n\n"
	if m.err != nil {
		errMsgStr := fmt.Sprintf("\nОшибка: %s\n\n(Нажмите любую клавишу для продолжения)", m.err)
		return s + errMsgStr // Возвращаем основной текст + текст ошибки
	}
	return s
}

// viewEntryDetailScreen отрисовывает экран деталей записи.
func (m model) viewEntryDetailScreen() string {
	if m.selectedEntry == nil {
		return "Ошибка: Запись не выбрана!" // Такого не должно быть, но на всякий случай
	}

	// Определяем желаемый порядок полей
	desiredOrder := []string{"Title", "UserName", "Password", "URL", "Notes"}
	// Собираем значения в map для быстрого доступа
	valuesMap := make(map[string]gokeepasslib.ValueData)
	for _, val := range m.selectedEntry.entry.Values {
		valuesMap[val.Key] = val
	}

	s := fmt.Sprintf("Детали записи: %s\n\n", m.selectedEntry.Title())

	// Выводим поля в заданном порядке
	for _, key := range desiredOrder {
		if val, ok := valuesMap[key]; ok {
			// Пока не будем показывать пароли
			if val.Key == fieldNamePassword {
				s += fmt.Sprintf("%s: ********\n", val.Key)
			} else {
				s += fmt.Sprintf("%s: %s\n", val.Key, val.Value.Content)
			}
			// Удаляем из карты, чтобы потом вывести оставшиеся (нестандартные) поля
			delete(valuesMap, key)
		} else {
			// Если поля нет в записи, можно вывести прочерк или ничего
			s += fmt.Sprintf("%s: \n", key)
		}
	}

	// Выводим остальные (нестандартные) поля, если они есть
	if len(valuesMap) > 0 {
		s += "\n--- Дополнительные поля ---\n"
		for _, val := range m.selectedEntry.entry.Values {
			if _, existsInMap := valuesMap[val.Key]; existsInMap {
				s += fmt.Sprintf("%s: %s\n", val.Key, val.Value.Content)
			}
		}
	}

	// s += "\n(e - ред., Ctrl+S - сохр., Esc/b - назад)" // Убрали, т.к. добавляется в View
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
	// s += "\n(Tab/Shift+Tab - навигация, Esc/b - отмена)" // Убрали, т.к. добавляется в View
	// TODO: Добавить подсказку про Enter для сохранения
	return s
}

// viewEntryAddScreen отрисовывает экран добавления новой записи.
func (m model) viewEntryAddScreen() string {
	s := "Добавление новой записи\n\n"
	s += "Введите данные для новой записи:\n"
	for i, input := range m.addInputs {
		focusIndicator := "  "
		if m.focusedFieldAdd == i {
			focusIndicator = "> "
		}
		s += fmt.Sprintf("%s%s: %s\n", focusIndicator, input.Placeholder, input.View())
	}
	// s += "(Enter - добавить, Ctrl+C - выход)\n" // Убрали, т.к. добавляется в View
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

// prepareAddScreen инициализирует поля для экрана добавления.
func (m *model) prepareAddScreen() {
	m.addInputs = make([]textinput.Model, numEditableFields)
	m.focusedFieldAdd = editableFieldTitle // Начинаем с поля Title

	placeholders := map[int]string{
		editableFieldTitle:    "Title",
		editableFieldUserName: "UserName",
		editableFieldPassword: "Password",
		editableFieldURL:      "URL",
		editableFieldNotes:    "Notes",
	}

	for i := range numEditableFields {
		m.addInputs[i] = textinput.New()
		m.addInputs[i].Placeholder = placeholders[i]
		// Первое поле делаем активным
		if i == m.focusedFieldAdd {
			m.addInputs[i].Focus()
		}
	}

	// Настроим поле пароля
	m.addInputs[editableFieldPassword].EchoMode = textinput.EchoPassword
}
