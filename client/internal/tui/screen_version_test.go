package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Вспомогательная функция для создания KeyMsg.
func keyMsg(key string) tea.KeyMsg {
	// Для простых клавиш тип не важен, важен String()
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func TestHandleVersionRollbackConfirm(t *testing.T) {
	versionID := int64(123)
	now := time.Now() // Создаем переменную для времени
	selectedVersion := &models.VaultVersion{
		ID:                versionID,
		VaultID:           1,
		ObjectKey:         "testkey",
		CreatedAt:         now,  // Используем переменную
		ContentModifiedAt: &now, // Передаем адрес переменной
	}

	t.Run("Press Enter to Confirm", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		ctxForTest := context.Background()
		m.state = versionListScreen
		m.confirmRollback = true
		m.selectedVersionForRollback = selectedVersion
		m.authToken = "test-token" // Добавляем токен для теста
		// Настроим мок, чтобы rollbackToVersionCmd вернул success
		// Используем ctxForTest, так как теперь он передается в команду
		s.Mocks.APIClient.On("RollbackToVersion", ctxForTest, versionID).Return(nil).Once()

		// Передаем контекст в handleVersionRollbackConfirm
		model, cmd := m.handleVersionRollbackConfirm(ctxForTest, keyMsg(keyEnter)) // Используем keyMsg

		// Проверяем состояние
		m = toModel(t, model)
		assert.False(t, m.confirmRollback, "confirmRollback should be false after Enter")
		require.NoError(t, m.rollbackError, "rollbackError should be nil after successful confirm")

		// Проверяем команду
		require.NotNil(t, cmd)

		// Выполняем команду (батч), передавая контекст
		msg := s.ExecuteCmd(ctxForTest, cmd)

		// Проверяем, что ExecuteCmd вернул BatchMsg
		batchCmds, ok := msg.(tea.BatchMsg)
		require.True(t, ok, "ExecuteCmd should return tea.BatchMsg, got %T", msg)

		// Ищем rollbackSuccessMsg среди результатов выполнения команд батча
		found := false
		for _, itemCmd := range batchCmds {
			if itemCmd == nil { // Пропускаем nil команды, если такие есть
				continue
			}
			itemMsg := itemCmd() // Выполняем команду из батча
			if _, successOk := itemMsg.(rollbackSuccessMsg); successOk {
				found = true
				// Не выходим из цикла, чтобы убедиться, что AssertExpectations вызывается после всех команд
			}
		}
		require.True(t, found, "rollbackSuccessMsg not found in the results of the BatchMsg commands")

		// Проверяем, что мок был вызван ПОСЛЕ выполнения команд батча.
		s.Mocks.APIClient.AssertExpectations(t)
	})

	t.Run("Press Escape to Cancel", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		ctxForTest := context.Background()
		m.state = versionListScreen
		m.confirmRollback = true
		m.selectedVersionForRollback = selectedVersion

		// Передаем контекст в handleVersionRollbackConfirm
		model, cmd := m.handleVersionRollbackConfirm(ctxForTest, keyMsg(keyEsc)) // Используем keyMsg

		m = toModel(t, model)
		assert.False(t, m.confirmRollback, "confirmRollback should be false after Escape")
		assert.Nil(t, m.selectedVersionForRollback, "selectedVersionForRollback should be nil after Escape")

		// Проверяем команду ClearScreen
		require.NotNil(t, cmd)
		_ = s.ExecuteCmd(ctxForTest, cmd)
	})

	t.Run("Press Backspace to Cancel", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		ctxForTest := context.Background()
		m.state = versionListScreen
		m.confirmRollback = true
		m.selectedVersionForRollback = selectedVersion

		// Используем 'b' как keyBack
		// Передаем контекст в handleVersionRollbackConfirm
		model, cmd := m.handleVersionRollbackConfirm(ctxForTest, keyMsg(keyBack)) // Используем keyMsg

		m = toModel(t, model)
		assert.False(t, m.confirmRollback, "confirmRollback should be false after Backspace")
		assert.Nil(t, m.selectedVersionForRollback, "selectedVersionForRollback should be nil after Backspace")

		// Проверяем команду ClearScreen
		require.NotNil(t, cmd)
		_ = s.ExecuteCmd(ctxForTest, cmd)
	})

	t.Run("Press Other Key", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		ctxForTest := context.Background()
		m.state = versionListScreen
		m.confirmRollback = true
		m.selectedVersionForRollback = selectedVersion

		// Передаем контекст в handleVersionRollbackConfirm
		model, cmd := m.handleVersionRollbackConfirm(ctxForTest, keyMsg("a")) // Используем keyMsg

		m = toModel(t, model)
		assert.True(t, m.confirmRollback, "confirmRollback should remain true")
		assert.Equal(t, selectedVersion, m.selectedVersionForRollback, "selectedVersionForRollback should remain")
		assert.Nil(t, cmd, "Command should be nil for other keys")
	})

	t.Run("Press Enter with nil selectedVersion", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		ctxForTest := context.Background()
		m.state = versionListScreen
		m.confirmRollback = true
		m.selectedVersionForRollback = nil // Устанавливаем nil

		// Передаем контекст в handleVersionRollbackConfirm
		model, cmd := m.handleVersionRollbackConfirm(ctxForTest, keyMsg(keyEnter)) // Используем keyMsg

		m = toModel(t, model)
		assert.True(t, m.confirmRollback, "confirmRollback should remain true if selectedVersion is nil")
		assert.Nil(t, m.selectedVersionForRollback, "selectedVersionForRollback should remain nil")
		assert.Nil(t, cmd, "Command should be nil if selectedVersion is nil")
	})
}
