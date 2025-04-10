package tui

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

// prepareAddScreen инициализирует поля для экрана добавления.
func (m *model) prepareAddScreen() {
	m.editInputs = make([]textinput.Model, numEditableFields)
	m.focusedField = editableFieldTitle // Начинаем с поля Title
	// Сбрасываем вложения, которые могли остаться от предыдущего добавления
	m.newEntryAttachments = nil

	// Используем константы имен полей как плейсхолдеры
	placeholders := map[int]string{
		editableFieldTitle:          fieldNameTitle,
		editableFieldUserName:       fieldNameUserName,
		editableFieldPassword:       fieldNamePassword,
		editableFieldURL:            fieldNameURL,
		editableFieldNotes:          fieldNameNotes,
		editableFieldCardNumber:     fieldNameCardNumber,
		editableFieldCardHolderName: fieldNameCardHolderName,
		editableFieldExpiryDate:     fieldNameExpiryDate,
		editableFieldCVV:            fieldNameCVV,
		editableFieldPIN:            fieldNamePIN,
	}

	for i := range numEditableFields {
		m.editInputs[i] = textinput.New()
		m.editInputs[i].Placeholder = placeholders[i]

		// Настраиваем маскирование для чувствительных полей
		switch i {
		case editableFieldPassword, editableFieldCVV, editableFieldPIN:
			m.editInputs[i].EchoMode = textinput.EchoPassword
		case editableFieldCardNumber:
			// Пока оставляем обычным
		}

		// Первое поле делаем активным
		if i == m.focusedField {
			m.editInputs[i].Focus()
		}
	}
}

// createEntryFromInputs создает новую запись gokeepasslib.Entry на основе
// данных из полей ввода (inputs) и временных вложений (attachments).
// Также добавляет бинарные данные в базу (db).
func createEntryFromInputs(db *gokeepasslib.Database, inputs []textinput.Model, attachments []struct {
	Name    string
	Content []byte
}) gokeepasslib.Entry {
	newEntry := gokeepasslib.NewEntry()
	now := time.Now()
	creationTime := w.TimeWrapper{Time: now}
	modificationTime := w.TimeWrapper{Time: now}
	accessTime := w.TimeWrapper{Time: now}
	newEntry.Times.CreationTime = &creationTime
	newEntry.Times.LastModificationTime = &modificationTime
	newEntry.Times.LastAccessTime = &accessTime

	// Добавляем обычные поля
	for _, input := range inputs {
		fieldName := input.Placeholder
		newValue := input.Value()
		if newValue != "" {
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

	// Добавляем вложения
	if db != nil && len(attachments) > 0 {
		for _, att := range attachments {
			binary := db.AddBinary(att.Content)
			if binary == nil {
				slog.Error("Не удалось добавить бинарные данные в базу", "name", att.Name)
				continue
			}
			binaryRef := binary.CreateReference(att.Name)
			newEntry.Binaries = append(newEntry.Binaries, binaryRef)
			slog.Info("Добавлено вложение к новой записи", "name", att.Name, "binary_id", binary.ID)
		}
	}

	return newEntry
}

// updateEntryAddScreen обрабатывает сообщения для экрана добавления записи.
//
//nolint:nestif // Сложность из-за обработки разных клавиш и навигации
func (m *model) updateEntryAddScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Если в режиме ReadOnly, сразу возвращаемся к списку
	if m.readOnlyMode {
		slog.Warn("Попытка доступа к экрану добавления в режиме Read-Only.")
		m.state = entryListScreen
		return m, tea.ClearScreen
	}

	var cmds []tea.Cmd

	// Обрабатываем только KeyMsg
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEsc, keyBack:
			// Отмена добавления
			m.state = entryListScreen
			m.editInputs = nil // Очищаем поля ввода
			m.newEntryAttachments = nil
			slog.Info("Отмена добавления, возврат к списку")
			return m, tea.ClearScreen

		case keyTab, keyDown:
			// Переход к следующему полю
			m.focusedField = (m.focusedField + 1) % numEditableFields
			cmds = m.updateFocus()
			return m, tea.Batch(cmds...)

		case keyShiftTab, keyUp:
			// Переход к предыдущему полю
			m.focusedField = (m.focusedField - 1 + numEditableFields) % numEditableFields
			cmds = m.updateFocus()
			return m, tea.Batch(cmds...)

		case keyEnter:
			// Создаем запись из введенных данных и вложений
			newEntry := createEntryFromInputs(m.db, m.editInputs, m.newEntryAttachments)
			m.newEntryAttachments = nil // Очищаем временное хранилище

			// Добавляем newEntry в m.db (в первую группу)
			if m.db != nil && m.db.Content != nil && m.db.Content.Root != nil {
				if len(m.db.Content.Root.Groups) > 0 {
					m.db.Content.Root.Groups[0].Entries = append(m.db.Content.Root.Groups[0].Entries, newEntry)
				} else {
					// Если нет групп, создаем первую
					newGroup := gokeepasslib.NewGroup()
					newGroup.Name = "General"
					newGroup.Entries = append(newGroup.Entries, newEntry)
					m.db.Content.Root.Groups = append(m.db.Content.Root.Groups, newGroup)
					slog.Warn("Группы не найдены, создана новая группа 'General'")
				}
			} else {
				slog.Error("Не удалось добавить запись в m.db: база данных или Root не инициализированы")
			}

			// Добавляем newEntry в m.entryList
			newItem := entryItem{entry: newEntry}
			insertCmd := m.entryList.InsertItem(len(m.entryList.Items()), newItem)
			m.entryList.Title = fmt.Sprintf("Записи в '%s' (%d)", m.kdbxPath, len(m.entryList.Items()))
			slog.Info("Новая запись добавлена", "title", newEntry.GetTitle())

			// Возвращаемся к списку
			m.state = entryListScreen
			m.editInputs = nil
			return m, tea.Batch(tea.ClearScreen, insertCmd)

		case "ctrl+o": // Добавить вложение
			slog.Info("Переход к экрану ввода пути для добавления вложения")
			m.previousScreenState = m.state // Запоминаем текущий экран (entryAddScreen)
			m.state = attachmentPathInputScreen
			m.attachmentPathInput.Reset()
			m.attachmentPathInput.Focus()
			m.attachmentError = nil // Сбрасываем предыдущую ошибку
			// Добавляем очистку экрана
			return m, tea.Batch(textinput.Blink, tea.ClearScreen)
		}
	} // конец if keyMsg, ok := msg.(tea.KeyMsg)

	// Если сообщение не KeyMsg или было обработано выше (кроме навигации/Enter/Esc),
	// обновляем активное поле ввода.
	var cmd tea.Cmd
	m.editInputs[m.focusedField], cmd = m.editInputs[m.focusedField].Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// updateFocus обновляет фокус полей ввода для экрана добавления И редактирования.
// Переименована из updateFocusAdd.
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

// viewEntryAddScreen отрисовывает экран добавления новой записи.
func (m *model) viewEntryAddScreen() string {
	s := "Добавление новой записи\n\n"
	s += "Введите данные для новой записи:\n"
	// Отображаем все поля ввода (включая поля карты)
	for i, input := range m.editInputs {
		focusIndicator := "  "
		if m.focusedField == i {
			focusIndicator = "> "
		}
		s += fmt.Sprintf("%s%s: %s\n", focusIndicator, input.Placeholder, input.View())
	}

	// Отображаем добавляемые вложения
	s += "\n--- Вложения для добавления ---\n"
	if len(m.newEntryAttachments) == 0 {
		s += "(Нет вложений)\n"
	} else {
		for i, att := range m.newEntryAttachments {
			s += fmt.Sprintf(" [%d] %s (%d байт)\n", i, att.Name, len(att.Content))
		}
	}

	// s += "(Enter - добавить, Ctrl+C - выход)\n" // Убрали, т.к. добавляется в View
	return s
}
