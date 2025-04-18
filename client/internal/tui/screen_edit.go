package tui

import (
	"fmt"
	"log/slog"
	"strings"
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
		placeholder := placeholders[i]
		m.editInputs[i] = textinput.New()
		m.editInputs[i].Placeholder = placeholder
		// Получаем текущее значение из редактируемой записи
		m.editInputs[i].SetValue(m.editingEntry.GetContent(placeholder))

		// Настраиваем маскирование для чувствительных полей
		switch i {
		case editableFieldPassword, editableFieldCVV, editableFieldPIN:
			m.editInputs[i].EchoMode = textinput.EchoPassword
		case editableFieldCardNumber:
			// TODO: Может быть, использовать EchoPassword или спец. режим?
			// Пока оставим обычным текстом
		}

		// Первое поле делаем активным
		if i == m.focusedField {
			m.editInputs[i].Focus()
		}
	}
}

// updateEntryEditScreen обрабатывает сообщения для экрана редактирования записи.
func (m *model) updateEntryEditScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Обработка нажатий клавиш делегируется отдельной функции
		return m.handleEditScreenKeys(msg)

	default:
		// Другие сообщения (не KeyMsg) на этом экране пока не обрабатываются
		// (Логика обновления поля перенесена в handleEditScreenKeys)
		return m, nil
	}
}

// handleEditScreenKeys обрабатывает нажатия клавиш на экране редактирования.
//
//nolint:gocognit,funlen // Сложность и длина будут снижены при рефакторинге
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
		if !m.readOnlyMode {
			return m.saveEntryChanges()
		}
		return m, nil

	case "ctrl+o":
		if !m.readOnlyMode {
			slog.Info("Переход к экрану ввода пути для добавления вложения")
			m.previousScreenState = m.state
			m.state = attachmentPathInputScreen
			m.attachmentPathInput.Reset()
			m.attachmentPathInput.Focus()
			m.attachmentError = nil
			return m, tea.Batch(textinput.Blink, tea.ClearScreen)
		}
		return m, nil

	case "ctrl+d":
		if !m.readOnlyMode {
			return m.handleAttachmentDeleteAction()
		}
		return m, nil

	default:
		//nolint:nestif // Вложенность из-за readOnlyMode
		if !m.readOnlyMode {
			var cmds []tea.Cmd
			var cmd tea.Cmd

			m.editInputs[m.focusedField], cmd = m.editInputs[m.focusedField].Update(msg)
			cmds = append(cmds, cmd)

			fieldName := m.editInputs[m.focusedField].Placeholder
			newValue := m.editInputs[m.focusedField].Value()
			found := false
			for i := range m.editingEntry.Values {
				if m.editingEntry.Values[i].Key == fieldName {
					m.editingEntry.Values[i].Value.Content = newValue
					if fieldName == fieldNamePassword || fieldName == fieldNameCVV || fieldName == fieldNamePIN {
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
				if fieldName == fieldNamePassword || fieldName == fieldNameCVV || fieldName == fieldNamePIN {
					valueData.Value.Protected = w.NewBoolWrapper(newValue != "")
				}
				m.editingEntry.Values = append(m.editingEntry.Values, valueData)
			}
			return m, tea.Batch(cmds...)
		}
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

// viewEntryEditScreen отрисовывает экран редактирования записи.
func (m *model) viewEntryEditScreen() string {
	var s strings.Builder
	s.WriteString("Редактирование записи: " + m.editingEntry.GetTitle() + "\n\n")
	// Отображаем все поля ввода (включая поля карты)
	for i, input := range m.editInputs {
		focusIndicator := "  "
		if m.focusedField == i {
			focusIndicator = "> "
		}
		s.WriteString(fmt.Sprintf("%s%s: %s\n", focusIndicator, input.Placeholder, input.View()))
	}

	// Отображаем вложения
	s.WriteString("\n--- Вложения ---\n")
	if len(m.editingEntry.Binaries) == 0 {
		s.WriteString("(Нет вложений)\n")
	} else {
		for i, binaryRef := range m.editingEntry.Binaries {
			// TODO: Добавить индикатор выбора для удаления?
			s.WriteString(fmt.Sprintf(" [%d] %s\n", i, binaryRef.Name))
		}
	}

	return s.String()
}
