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
	m.addInputs = make([]textinput.Model, numEditableFields)
	m.focusedFieldAdd = editableFieldTitle // Начинаем с поля Title

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
		m.addInputs[i] = textinput.New()
		m.addInputs[i].Placeholder = placeholders[i]

		// Настраиваем маскирование для чувствительных полей
		switch i {
		case editableFieldPassword, editableFieldCVV, editableFieldPIN:
			m.addInputs[i].EchoMode = textinput.EchoPassword
		case editableFieldCardNumber:
			// Пока оставляем обычным
		}

		// Первое поле делаем активным
		if i == m.focusedFieldAdd {
			m.addInputs[i].Focus()
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
			// Создаем запись из введенных данных и вложений
			newEntry := createEntryFromInputs(m.db, m.addInputs, m.newEntryAttachments)
			m.newEntryAttachments = nil // Очищаем временное хранилище

			// Добавляем newEntry в m.db (в первую группу)
			if m.db != nil && m.db.Content != nil && m.db.Content.Root != nil {
				if len(m.db.Content.Root.Groups) > 0 {
					m.db.Content.Root.Groups[0].Entries = append(m.db.Content.Root.Groups[0].Entries, newEntry)
				} else {
					slog.Error("Не удалось добавить запись в m.db: нет групп")
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
			m.addInputs = nil
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

// viewEntryAddScreen отрисовывает экран добавления новой записи.
func (m model) viewEntryAddScreen() string {
	s := "Добавление новой записи\n\n"
	s += "Введите данные для новой записи:\n"
	// Отображаем все поля ввода (включая поля карты)
	for i, input := range m.addInputs {
		focusIndicator := "  "
		if m.focusedFieldAdd == i {
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
