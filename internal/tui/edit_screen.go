package tui

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

const (
	attachmentListHeightDivisor = 2 // Делитель для высоты списка вложений
)

// attachmentItem представляет элемент списка вложений для выбора/удаления.
// Реализует интерфейс list.Item.
type attachmentItem struct {
	name string
	id   int // ID из BinaryReference (Value.ID)
}

func (i attachmentItem) Title() string       { return i.name }
func (i attachmentItem) Description() string { return fmt.Sprintf("ID: %d", i.id) }
func (i attachmentItem) FilterValue() string { return i.name }

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
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Обработка нажатий клавиш делегируется отдельной функции
		return m.handleEditScreenKeys(msg)

	default:
		// Обработка других сообщений (например, обновление поля ввода)
		m.editInputs[m.focusedField], cmd = m.editInputs[m.focusedField].Update(msg)
		cmds = append(cmds, cmd)

		// Обновляем поле в editingEntry
		fieldName := m.editInputs[m.focusedField].Placeholder
		newValue := m.editInputs[m.focusedField].Value()
		found := false
		for i := range m.editingEntry.Values {
			if m.editingEntry.Values[i].Key == fieldName {
				m.editingEntry.Values[i].Value.Content = newValue
				if fieldName == fieldNamePassword {
					m.editingEntry.Values[i].Value.Protected = w.NewBoolWrapper(newValue != "")
				}
				found = true
				break
			}
		}
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
}

// handleEditScreenKeys обрабатывает нажатия клавиш на экране редактирования.
func (m *model) handleEditScreenKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc, keyBack:
		m.state = entryDetailScreen
		m.editingEntry = nil
		m.editInputs = nil
		slog.Info("Отмена редактирования, возврат к деталям записи")
		return m, tea.ClearScreen

	case "tab", "down":
		m.focusedField = (m.focusedField + 1) % numEditableFields
		cmds := m.updateFocus()
		return m, tea.Batch(cmds...)

	case "shift+tab", "up":
		m.focusedField = (m.focusedField - 1 + numEditableFields) % numEditableFields
		cmds := m.updateFocus()
		return m, tea.Batch(cmds...)

	case keyEnter:
		return m.saveEntryChanges()

	case "ctrl+o":
		slog.Info("Обработка Ctrl+O: Добавить вложение (пока не реализовано)")
		return m, nil

	case "ctrl+d":
		return m.handleAttachmentDeleteAction()

	default:
		// Если не специальная клавиша - ничего не делаем (поле ввода обновится в Update по msg)
		return m, nil
	}
}

// handleAttachmentDeleteAction обрабатывает действие удаления вложения.
func (m *model) handleAttachmentDeleteAction() (tea.Model, tea.Cmd) {
	if m.editingEntry != nil && len(m.editingEntry.Binaries) > 0 {
		slog.Info("Переход к экрану удаления вложения")
		items := make([]list.Item, len(m.editingEntry.Binaries))
		for i, binRef := range m.editingEntry.Binaries {
			items[i] = attachmentItem{name: binRef.Name, id: binRef.Value.ID}
		}
		m.attachmentList.SetItems(items)
		m.attachmentList.SetSize(defaultListWidth, defaultListHeight/attachmentListHeightDivisor)
		m.state = attachmentListDeleteScreen
		return m, tea.ClearScreen
	}
	slog.Info("Нет вложений для удаления")
	return m, nil
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

// viewEntryEditScreen отрисовывает экран редактирования записи.
func (m model) viewEntryEditScreen() string {
	if m.editingEntry == nil || len(m.editInputs) == 0 {
		return "Ошибка: Нет данных для редактирования!"
	}

	s := "Редактирование записи: " + m.editingEntry.GetTitle() + "\n\n"
	// Отображаем поля ввода
	for i, input := range m.editInputs {
		focusIndicator := "  "
		if m.focusedField == i {
			focusIndicator = "> "
		}
		s += fmt.Sprintf("%s%s: %s\n", focusIndicator, input.Placeholder, input.View())
	}

	// Отображаем вложения
	s += "\n--- Вложения ---\n"
	if len(m.editingEntry.Binaries) == 0 {
		s += "(Нет вложений)\n"
	} else {
		for i, binaryRef := range m.editingEntry.Binaries {
			// TODO: Добавить индикатор выбора для удаления?
			s += fmt.Sprintf(" [%d] %s\n", i, binaryRef.Name)
		}
	}

	return s
}
