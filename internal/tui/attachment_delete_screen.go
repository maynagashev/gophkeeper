package tui

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
)

// updateAttachmentListDeleteScreen обрабатывает сообщения для экрана удаления вложений.
func (m *model) updateAttachmentListDeleteScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Обновляем список вложений
	var listCmd tea.Cmd
	m.attachmentList, listCmd = m.attachmentList.Update(msg)
	cmds = append(cmds, listCmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEsc, keyBack:
			m.state = entryEditScreen
			slog.Info("Отмена удаления вложения, возврат к редактированию")
			return m, tea.ClearScreen

		case keyEnter, "d":
			return m.handleAttachmentDeleteConfirm()
		}
	}

	return m, tea.Batch(cmds...)
}

// handleAttachmentDeleteConfirm обрабатывает подтверждение удаления вложения.
func (m *model) handleAttachmentDeleteConfirm() (tea.Model, tea.Cmd) {
	selectedItem := m.attachmentList.SelectedItem()
	if selectedItem == nil {
		return m, nil // Ничего не выбрано
	}
	item, itemOk := selectedItem.(attachmentItem)
	if !itemOk {
		slog.Error("Не удалось преобразовать выбранный элемент к attachmentItem")
		return m, nil
	}

	slog.Info("Удаление ссылки на вложение", "name", item.name, "id", item.id)

	foundIndex := -1
	for i, binRef := range m.editingEntry.Binaries {
		if binRef.Value.ID == item.id {
			foundIndex = i
			break
		}
	}

	if foundIndex != -1 {
		m.editingEntry.Binaries = append(m.editingEntry.Binaries[:foundIndex], m.editingEntry.Binaries[foundIndex+1:]...)
		slog.Info("Ссылка на вложение успешно удалена из редактируемой записи")
	} else {
		slog.Warn("Не удалось найти BinaryReference для удаления в editingEntry")
	}

	m.state = entryEditScreen
	return m, tea.ClearScreen
}

// viewAttachmentListDeleteScreen отрисовывает экран удаления вложений.
func (m model) viewAttachmentListDeleteScreen() string {
	// Устанавливаем правильный размер перед отображением (если окно изменилось)
	// TODO: Лучше обрабатывать WindowSizeMsg глобально
	// m.attachmentList.SetSize(width, height)
	return m.attachmentList.View()
}
