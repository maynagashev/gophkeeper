//nolint:testpackage // Это тесты в том же пакете для доступа к приватным компонентам
package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/tobischo/gokeepasslib/v3"
)

// attachmentListItem реализует list.Item для вложений (для тестов).
type attachmentListItem struct {
	Name string
	UUID gokeepasslib.UUID // Оставим UUID для логики теста, хоть его нет в BinaryReference
}

func (i attachmentListItem) Title() string       { return i.Name }
func (i attachmentListItem) Description() string { return "Attachment" }
func (i attachmentListItem) FilterValue() string { return i.Name }

// TestUpdateAttachmentListDeleteScreen проверяет обработку сообщений на экране удаления вложений.
func TestUpdateAttachmentListDeleteScreen(t *testing.T) {
	t.Run("Возврат к редактированию в режиме ReadOnly", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.readOnlyMode = true
		s.Model.state = attachmentListDeleteScreen
		s.Model.attachmentList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

		model, cmd := s.Model.updateAttachmentListDeleteScreen(tea.KeyMsg{Type: tea.KeyEnter})
		m := toModel(t, model)

		assert.Equal(t, entryEditScreen, m.state)
		assert.NotNil(t, cmd)
	})

	t.Run("Обработка подтверждения удаления", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.state = attachmentListDeleteScreen
		s.Model.confirmationPrompt = "Удалить вложение 'test.txt'? (y/n)"
		s.Model.itemToDelete = &attachmentItem{
			name: "test.txt",
			id:   123,
		}
		s.Model.editingEntry = &gokeepasslib.Entry{
			Binaries: []gokeepasslib.BinaryReference{
				{
					Value: struct {
						ID int `xml:"Ref,attr"`
					}{ID: 123},
				},
			},
		}
		s.Model.attachmentList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

		// Симулируем нажатие 'y' для подтверждения
		model, cmd := s.Model.updateAttachmentListDeleteScreen(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m := toModel(t, model)

		assert.Equal(t, entryEditScreen, m.state)
		assert.Empty(t, m.confirmationPrompt)
		assert.Nil(t, m.itemToDelete)
		assert.NotNil(t, cmd)
		assert.Empty(t, m.editingEntry.Binaries)
	})

	t.Run("Отмена подтверждения удаления", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.state = attachmentListDeleteScreen
		s.Model.confirmationPrompt = "Удалить вложение 'test.txt'? (y/n)"
		s.Model.itemToDelete = &attachmentItem{
			name: "test.txt",
			id:   123,
		}
		s.Model.attachmentList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

		// Симулируем нажатие 'n' для отмены
		model, cmd := s.Model.updateAttachmentListDeleteScreen(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		m := toModel(t, model)

		assert.Empty(t, m.confirmationPrompt)
		assert.Nil(t, m.itemToDelete)
		assert.Nil(t, cmd)
		assert.Equal(t, attachmentListDeleteScreen, m.state)
	})

	t.Run("Возврат к редактированию по ESC", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.state = attachmentListDeleteScreen
		s.Model.attachmentList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

		model, cmd := s.Model.updateAttachmentListDeleteScreen(tea.KeyMsg{Type: tea.KeyEsc})
		m := toModel(t, model)

		assert.Equal(t, entryEditScreen, m.state)
		assert.NotNil(t, cmd)
	})

	t.Run("Выбор вложения для удаления", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.state = attachmentListDeleteScreen
		s.Model.attachmentList = list.New([]list.Item{
			attachmentItem{name: "test.txt", id: 123},
		}, list.NewDefaultDelegate(), 0, 0)

		// Симулируем нажатие Enter для выбора вложения
		model, _ := s.Model.updateAttachmentListDeleteScreen(tea.KeyMsg{Type: tea.KeyEnter})
		m := toModel(t, model)

		assert.NotEmpty(t, m.confirmationPrompt)
		assert.Contains(t, m.confirmationPrompt, "test.txt")
		assert.NotNil(t, m.itemToDelete)
		assert.Equal(t, attachmentListDeleteScreen, m.state)
	})

	t.Run("Выбор несуществующего вложения", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.state = attachmentListDeleteScreen
		s.Model.attachmentList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

		// Симулируем нажатие Enter без выбранного элемента
		model, cmd := s.Model.updateAttachmentListDeleteScreen(tea.KeyMsg{Type: tea.KeyEnter})
		m := toModel(t, model)

		assert.Empty(t, m.confirmationPrompt)
		assert.Nil(t, m.itemToDelete)
		assert.Equal(t, attachmentListDeleteScreen, m.state)
		assert.Nil(t, cmd)
	})

	t.Run("Обработка неизвестной клавиши", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.state = attachmentListDeleteScreen
		s.Model.attachmentList = list.New([]list.Item{
			attachmentItem{name: "test.txt", id: 123},
		}, list.NewDefaultDelegate(), 0, 0)

		// Симулируем нажатие неизвестной клавиши
		model, _ := s.Model.updateAttachmentListDeleteScreen(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		m := toModel(t, model)

		assert.Equal(t, attachmentListDeleteScreen, m.state)
	})
}

func TestPerformAttachmentDelete(t *testing.T) {
	t.Run("Успешное удаление вложения", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.itemToDelete = &attachmentItem{
			name: "test.txt",
			id:   123,
		}
		s.Model.editingEntry = &gokeepasslib.Entry{
			Binaries: []gokeepasslib.BinaryReference{
				{
					Value: struct {
						ID int `xml:"Ref,attr"`
					}{ID: 123},
				},
			},
		}

		model, cmd := s.Model.performAttachmentDelete()
		m := toModel(t, model)

		assert.Equal(t, entryEditScreen, m.state)
		assert.Empty(t, m.confirmationPrompt)
		assert.Nil(t, m.itemToDelete)
		assert.Empty(t, m.editingEntry.Binaries)
		assert.Equal(t, "Вложение 'test.txt' удалено.", m.savingStatus)
		assert.NotNil(t, cmd)
	})

	t.Run("Попытка удаления без выбранного элемента", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.itemToDelete = nil

		model, cmd := s.Model.performAttachmentDelete()
		m := toModel(t, model)

		assert.Empty(t, m.confirmationPrompt)
		assert.Nil(t, cmd)
	})

	t.Run("Вложение не найдено в записи", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.itemToDelete = &attachmentItem{
			name: "test.txt",
			id:   123,
		}
		s.Model.editingEntry = &gokeepasslib.Entry{
			Binaries: []gokeepasslib.BinaryReference{
				{
					Value: struct {
						ID int `xml:"Ref,attr"`
					}{ID: 456}, // Другой ID
				},
			},
		}

		model, cmd := s.Model.performAttachmentDelete()
		m := toModel(t, model)

		assert.Equal(t, entryEditScreen, m.state)
		assert.Empty(t, m.confirmationPrompt)
		assert.Nil(t, m.itemToDelete)
		assert.Len(t, m.editingEntry.Binaries, 1)
		assert.Equal(t, "Не удалось удалить вложение 'test.txt'.", m.savingStatus)
		assert.NotNil(t, cmd)
	})
}

func TestViewAttachmentListDeleteScreen(t *testing.T) {
	t.Run("Отображение промпта подтверждения", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.confirmationPrompt = "Удалить вложение 'test.txt'? (y/n)"

		view := s.Model.viewAttachmentListDeleteScreen()
		assert.Equal(t, "Удалить вложение 'test.txt'? (y/n)", view)
	})

	t.Run("Отображение списка вложений", func(t *testing.T) {
		s := NewScreenTestSuite()
		s.Model.confirmationPrompt = ""
		s.Model.attachmentList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

		view := s.Model.viewAttachmentListDeleteScreen()
		assert.NotEmpty(t, view)
	})
}
