package tui

import (
	"os"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tobischo/gokeepasslib/v3"
)

func TestUpdateAttachmentPathInputScreen(t *testing.T) {
	// Подготавливаем модель
	input := textinput.New()
	input.Focus()

	m := &model{
		state:               attachmentPathInputScreen,
		previousScreenState: entryAddScreen,
		attachmentPathInput: input,
	}

	// Проверяем базовое обновление
	var updatedModel tea.Model
	updatedModel, _ = m.updateAttachmentPathInputScreen(nil)
	newM, ok := updatedModel.(*model)
	require.True(t, ok, "Model type assertion failed")
	m = newM // Обновляем ссылку на модель
	assert.NotNil(t, m)

	// Обработка клавиши ESC - возврат к предыдущему экрану
	updatedModel, _ = m.updateAttachmentPathInputScreen(tea.KeyMsg{Type: tea.KeyEsc})
	newM, ok = updatedModel.(*model)
	require.True(t, ok, "Model type assertion failed")
	m = newM // Обновляем ссылку на модель
	assert.Equal(t, entryAddScreen, m.state, "Expected state to be entryAddScreen after ESC")

	// Сбрасываем состояние для следующего теста
	m.state = attachmentPathInputScreen

	// Обработка обычного ввода
	m.attachmentPathInput.SetValue("")
	m.attachmentPathInput.Focus()
	var cmd tea.Cmd
	updatedModel, cmd = m.updateAttachmentPathInputScreen(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		cmd() // Выполняем команду обновления (если есть)
	}
	newM, ok = updatedModel.(*model)
	require.True(t, ok, "Model type assertion failed")
	m = newM // Обновляем ссылку на модель
	assert.Equal(t, "a", m.attachmentPathInput.Value(), "Expected attachmentPathInput to contain 'a'")
}

func TestHandleAttachmentPathConfirm(t *testing.T) {
	// Создаем временный файл для тестирования
	tmpFile, err := os.CreateTemp("", "test_attachment_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Записываем тестовые данные
	testData := []byte("test attachment data")
	_, writeErr := tmpFile.Write(testData)
	require.NoError(t, writeErr)
	closeErr := tmpFile.Close()
	require.NoError(t, closeErr)

	// Инициализируем базу данных KDBX4
	db := gokeepasslib.NewDatabase(
		gokeepasslib.WithDatabaseKDBXVersion4(),
	)
	db.Content.Root = gokeepasslib.NewRootData()
	db.Content.Root.Groups = []gokeepasslib.Group{
		gokeepasslib.NewGroup(),
	}
	db.Content.Root.Groups[0].Name = "TestGroup"

	// Создаем запись для тестирования
	entry := &entryItem{
		entry: gokeepasslib.Entry{
			Values: []gokeepasslib.ValueData{
				{Key: "Title", Value: gokeepasslib.V{Content: "Test Entry"}},
			},
		},
	}

	// Подготавливаем модель
	pathInput := textinput.New()
	pathInput.Focus()

	// Тест с несуществующим файлом
	m := &model{
		state:               attachmentPathInputScreen,
		attachmentPathInput: pathInput,
		previousScreenState: entryEditScreen,
		db:                  db,
		selectedEntry:       entry,
		editingEntry:        &entry.entry,
	}
	m.attachmentPathInput.SetValue("nonexistent.txt")
	// Обновляем модель с новым значением перед симуляцией Enter
	m.attachmentPathInput, _ = m.attachmentPathInput.Update(tea.KeyMsg{}) // Используем пустое KeyMsg
	// Симулируем нажатие Enter
	var updatedModel tea.Model
	updatedModel, _ = m.updateAttachmentPathInputScreen(tea.KeyMsg{Type: tea.KeyEnter})
	newM, ok := updatedModel.(*model)
	require.True(t, ok, "Model type assertion failed")
	m = newM // Обновляем ссылку на модель
	require.Error(t, m.attachmentError, "Expected error for nonexistent file")
	assert.Contains(t, m.attachmentError.Error(), "ошибка чтения файла", "Error should mention file read error")
	assert.Equal(t, attachmentPathInputScreen, m.state, "Should stay on input screen after error")

	// Успешное добавление вложения в режиме редактирования
	// Сбрасываем состояние перед следующим тестом
	m.state = attachmentPathInputScreen
	m.attachmentError = nil
	// m.attachmentPathInput = pathInput - Больше не нужно, так как мы всегда обновляем m.attachmentPathInput
	m.attachmentPathInput.Reset() // Очищаем поле ввода перед установкой нового значения
	m.attachmentPathInput.SetValue(tmpFile.Name())
	m.attachmentPathInput.Focus() // Убедимся, что поле ввода в фокусе
	// Обновляем модель с новым значением перед симуляцией Enter
	m.attachmentPathInput, _ = m.attachmentPathInput.Update(tea.KeyMsg{}) // Используем пустое KeyMsg
	// Симулируем нажатие Enter
	var cmd tea.Cmd
	updatedModel, cmd = m.updateAttachmentPathInputScreen(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		cmd() // Выполняем команду очистки экрана (если есть)
	}
	newM, ok = updatedModel.(*model)
	require.True(t, ok, "Model type assertion failed")
	m = newM // Обновляем ссылку на модель
	assert.Equal(t, entryEditScreen, m.state, "Should return to edit screen after success")
	require.NoError(t, m.attachmentError, "Expected no error after successful attachment")
	assert.Contains(t, m.savingStatus, "Вложение", "Should show success message")

	// Тест с неверным контекстом
	m.state = attachmentPathInputScreen // Сбрасываем состояние
	m.attachmentError = nil             // Сбрасываем ошибку
	m.previousScreenState = syncServerScreen
	// Обновляем модель с текущим значением перед симуляцией Enter
	m.attachmentPathInput, _ = m.attachmentPathInput.Update(tea.KeyMsg{})
	// Симулируем нажатие Enter
	updatedModel, _ = m.updateAttachmentPathInputScreen(tea.KeyMsg{Type: tea.KeyEnter})
	newM, ok = updatedModel.(*model)
	require.True(t, ok, "Model type assertion failed")
	m = newM // Обновляем ссылку на модель
	require.Error(t, m.attachmentError, "Expected error for invalid context")
	assert.Contains(t, m.attachmentError.Error(), "неизвестный контекст", "Error should mention invalid context")
}

func TestViewAttachmentPathInputScreen(t *testing.T) {
	m := &model{
		state:               attachmentPathInputScreen,
		previousScreenState: entryAddScreen,
		attachmentPathInput: textinput.New(),
	}

	view := m.viewAttachmentPathInputScreen()
	assert.Contains(t, view, "Введите полный путь к файлу")
}
