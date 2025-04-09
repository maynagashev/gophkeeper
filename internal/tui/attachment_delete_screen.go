package tui

import (
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
)

// updateAttachmentListDeleteScreen обрабатывает сообщения для экрана удаления вложений.
func (m *model) updateAttachmentListDeleteScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Если есть активный запрос на подтверждение, обрабатываем его
	if m.confirmationPrompt != "" {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case keyEnter, "y", "d": // Подтверждение
				return m.performAttachmentDelete()
			case keyEsc, keyBack, "n": // Отмена
				m.confirmationPrompt = ""
				m.itemToDelete = nil
				return m, nil // Остаемся на экране, убираем промпт
			}
		}
		// Игнорируем другие сообщения, пока активен промпт
		return m, nil
	}

	// Обновляем список вложений (если нет активного промпта)
	var listCmd tea.Cmd
	m.attachmentList, listCmd = m.attachmentList.Update(msg)
	cmds = append(cmds, listCmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEsc, keyBack:
			// Отмена удаления, возврат к редактированию
			m.state = entryEditScreen
			slog.Info("Отмена удаления вложения, возврат к редактированию")
			return m, tea.ClearScreen

		case keyEnter, "d":
			selectedItem := m.attachmentList.SelectedItem()
			if selectedItem != nil {
				if item, itemOk := selectedItem.(attachmentItem); itemOk {
					// Запрашиваем подтверждение
					m.itemToDelete = &item
					m.confirmationPrompt = fmt.Sprintf("Удалить вложение '%s'? (y/n)", item.name)
					return m, nil // Остаемся на экране, показываем промпт
				}
				// Если не itemOk, выводим ошибку и остаемся
				slog.Error("Не удалось преобразовать выбранный элемент к attachmentItem")
			}
			// Если selectedItem == nil, тоже остаемся
			return m, nil
		}
	}

	return m, tea.Batch(cmds...)
}

// performAttachmentDelete выполняет фактическое удаление вложения после подтверждения.
func (m *model) performAttachmentDelete() (tea.Model, tea.Cmd) {
	if m.itemToDelete == nil {
		slog.Error("Попытка удаления без выбранного itemToDelete")
		m.confirmationPrompt = ""
		return m, nil
	}

	itemName := m.itemToDelete.name
	itemID := m.itemToDelete.id
	slog.Info("Подтверждено удаление ссылки на вложение", "name", itemName, "id", itemID)

	// Находим и удаляем BinaryReference из среза m.editingEntry.Binaries
	foundIndex := -1
	for i, binRef := range m.editingEntry.Binaries {
		if binRef.Value.ID == itemID {
			foundIndex = i
			break
		}
	}

	if foundIndex != -1 {
		m.editingEntry.Binaries = append(m.editingEntry.Binaries[:foundIndex], m.editingEntry.Binaries[foundIndex+1:]...)
		slog.Info("Ссылка на вложение успешно удалена из редактируемой записи")
		m.savingStatus = fmt.Sprintf("Вложение '%s' удалено.", itemName) // Устанавливаем статус
	} else {
		slog.Warn("Не удалось найти BinaryReference для удаления в editingEntry")
		m.savingStatus = fmt.Sprintf("Не удалось удалить вложение '%s'.", itemName) // Устанавливаем статус ошибки
	}

	// Сбрасываем состояние подтверждения и возвращаемся к экрану редактирования
	m.confirmationPrompt = ""
	m.itemToDelete = nil
	m.state = entryEditScreen
	// Возвращаем ClearScreen, чтобы статус был виден на чистом экране редактирования
	return m, tea.ClearScreen
}

// viewAttachmentListDeleteScreen отрисовывает экран удаления вложений.
func (m model) viewAttachmentListDeleteScreen() string {
	var s string
	if m.confirmationPrompt != "" {
		// Отображаем только промпт подтверждения
		s = m.confirmationPrompt
	} else {
		// Отображаем список вложений
		// TODO: Устанавливать размер списка корректно
		s = m.attachmentList.View()
	}
	return s
}
