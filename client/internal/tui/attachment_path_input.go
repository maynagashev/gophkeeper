package tui

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateAttachmentPathInputScreen обрабатывает ввод пути к файлу вложения.
func (m *model) updateAttachmentPathInputScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	// Обновляем поле ввода пути
	m.attachmentPathInput, cmd = m.attachmentPathInput.Update(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEsc:
			// Отмена ввода, возврат на предыдущий экран
			slog.Info("Отмена ввода пути вложения")
			m.state = m.previousScreenState
			m.attachmentPathInput.Blur()
			m.attachmentError = nil // Очищаем ошибку при отмене
			// Нужно вернуть фокус на предыдущий компонент?
			// Зависит от previousScreenState. Пока просто ClearScreen.
			return m, tea.ClearScreen

		case keyEnter:
			return m.handleAttachmentPathConfirm()
		}
	}

	return m, tea.Batch(cmds...)
}

// handleAttachmentPathConfirm обрабатывает подтверждение ввода пути и чтение файла.
func (m *model) handleAttachmentPathConfirm() (tea.Model, tea.Cmd) {
	filePath := m.attachmentPathInput.Value()
	slog.Info("Попытка добавить вложение из файла", "path", filePath)

	content, err := os.ReadFile(filePath)
	if err != nil {
		m.attachmentError = fmt.Errorf("ошибка чтения файла: %w", err)
		slog.Error("Ошибка чтения файла для вложения", "path", filePath, "error", err)
		return m, nil // Остаемся на экране ввода
	}

	fileName := filepath.Base(filePath)
	slog.Info("Файл успешно прочитан", "name", fileName, "size", len(content))

	// Добавляем вложение
	//nolint:exhaustive // Игнорируем exhaustive, т.к. другие состояния не должны сюда приводить
	switch m.previousScreenState {
	case entryEditScreen:
		if m.editingEntry != nil && m.db != nil {
			binary := m.db.AddBinary(content)
			if binary == nil {
				m.attachmentError = fmt.Errorf("не удалось добавить бинарные данные '%s' в базу", fileName)
				slog.Error("Ошибка при добавлении бинарных данных в базу (edit screen)", "name", fileName)
				return m, nil // Остаемся на экране ввода
			}
			binaryRef := binary.CreateReference(fileName)
			m.editingEntry.Binaries = append(m.editingEntry.Binaries, binaryRef)
			slog.Info("Вложение добавлено к редактируемой записи", "name", fileName, "binary_id", binary.ID)
		} else {
			slog.Error("Не удалось добавить вложение: editingEntry или db is nil")
			m.attachmentError = errors.New("внутренняя ошибка: нет контекста для добавления вложения")
			return m, nil // Остаемся на экране ввода
		}
	case entryAddScreen:
		m.newEntryAttachments = append(m.newEntryAttachments, struct {
			Name    string
			Content []byte
		}{fileName, content})
		slog.Info("Вложение добавлено во временный список для новой записи", "name", fileName)
	default:
		// Этого не должно происходить
		slog.Error("Неожиданный предыдущий экран при добавлении вложения", "state", m.previousScreenState)
		m.attachmentError = fmt.Errorf("внутренняя ошибка: неизвестный контекст %v", m.previousScreenState)
		return m, nil // Остаемся на экране ввода
	}

	// Возвращаемся на предыдущий экран
	m.state = m.previousScreenState
	m.attachmentPathInput.Blur()
	m.attachmentError = nil                                            // Очищаем ошибку после успеха
	m.savingStatus = fmt.Sprintf("Вложение '%s' добавлено.", fileName) // Устанавливаем статус
	return m, tea.ClearScreen
}

// viewAttachmentPathInputScreen отрисовывает экран ввода пути к файлу вложения.
func (m *model) viewAttachmentPathInputScreen() string {
	var b strings.Builder
	b.WriteString("Введите полный путь к файлу для добавления вложения:\n")
	b.WriteString(m.attachmentPathInput.View())
	b.WriteString("\n\n")

	if m.attachmentError != nil {
		b.WriteString(fmt.Sprintf("Ошибка: %s\n", m.attachmentError))
	}

	return b.String()
}
