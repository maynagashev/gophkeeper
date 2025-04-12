package tui

import (
	"fmt"
	"log/slog"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/maynagashev/gophkeeper/client/internal/kdbx"
)

// updateEntryListScreen обрабатывает сообщения для экрана списка записей.
func (m *model) updateEntryListScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Сначала обновляем список
	m.entryList, cmd = m.entryList.Update(msg)
	cmds = append(cmds, cmd)

	// Обработка клавиш для экрана списка
	//nolint:nestif // Вложенность из-за разных клавиш
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
			// Переход к добавлению новой записи (только если не Read-Only)
			if !m.readOnlyMode {
				m.prepareAddScreen()
				m.state = entryAddScreen
				slog.Info("Переход к добавлению новой записи")
				return m, tea.ClearScreen
			}
		}
	}
	return m, tea.Batch(cmds...)
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
