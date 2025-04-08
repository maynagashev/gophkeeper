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
