package tui

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestHandleVersionRollbackError(t *testing.T) {
	testError := errors.New("rollback error")

	t.Run("Press Escape to clear error", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		m.rollbackError = testError // Устанавливаем ошибку

		model, cmd := m.handleVersionRollbackError(keyMsg(keyEsc))

		m = toModel(t, model)
		require.NoError(t, m.rollbackError, "rollbackError should be nil after Escape")
		require.NotNil(t, cmd, "Command should not be nil")

		// Проверяем, что команда это ClearScreen (хотя ExecuteCmd не вернет тип)
		_ = s.ExecuteCmd(context.Background(), cmd)
		// Как проверить тип ClearScreen? Пока просто выполняем.
	})

	t.Run("Press Backspace to clear error", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		m.rollbackError = testError

		model, cmd := m.handleVersionRollbackError(keyMsg(keyBack))

		m = toModel(t, model)
		require.NoError(t, m.rollbackError, "rollbackError should be nil after Backspace")
		require.NotNil(t, cmd, "Command should not be nil")
		_ = s.ExecuteCmd(context.Background(), cmd)
	})

	t.Run("Press Enter to clear error", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		m.rollbackError = testError

		model, cmd := m.handleVersionRollbackError(keyMsg(keyEnter))

		m = toModel(t, model)
		require.NoError(t, m.rollbackError, "rollbackError should be nil after Enter")
		require.NotNil(t, cmd, "Command should not be nil")
		_ = s.ExecuteCmd(context.Background(), cmd)
	})

	t.Run("Press Other Key should not clear error", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		m.rollbackError = testError

		model, cmd := m.handleVersionRollbackError(keyMsg("a"))

		m = toModel(t, model)
		require.Error(t, m.rollbackError, "rollbackError should still exist")
		assert.Equal(t, testError, m.rollbackError, "rollbackError should be the same")
		assert.Nil(t, cmd, "Command should be nil for other keys")
	})
}

func TestHandleVersionListKeys(t *testing.T) {
	// Подготовка тестовых данных
	now := time.Now()
	version1Time := now.Add(-time.Hour)
	version2Time := now.Add(-2 * time.Hour)
	versions := []models.VaultVersion{
		{ID: 2, VaultID: 1, CreatedAt: now, ContentModifiedAt: &now}, // Текущая
		{ID: 1, VaultID: 1, CreatedAt: version1Time, ContentModifiedAt: &version1Time},
		{ID: 0, VaultID: 1, CreatedAt: version2Time, ContentModifiedAt: &version2Time}, // Версия с ID 0
	}

	items := []list.Item{
		versionItem{version: versions[0], isCurrent: true},
		versionItem{version: versions[1], isCurrent: false},
		versionItem{version: versions[2], isCurrent: false},
	}

	t.Run("Press Enter on non-current version", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		m.versions = versions
		m.versionList = list.New(items, list.NewDefaultDelegate(), 0, 0)
		m.versionList.Select(1) // Выбираем вторую версию (ID=1)

		model, cmd := m.handleVersionListKeys(keyMsg(keyEnter))

		m = toModel(t, model)
		require.NotNil(t, m.selectedVersionForRollback, "selectedVersionForRollback should be set")
		assert.Equal(t, int64(1), m.selectedVersionForRollback.ID, "Correct version ID should be selected")
		assert.True(t, m.confirmRollback, "confirmRollback should be true")
		require.NotNil(t, cmd, "Command should not be nil")
		_ = s.ExecuteCmd(context.Background(), cmd)
	})

	t.Run("Press Enter on current version", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		m.versions = versions
		m.versionList = list.New(items, list.NewDefaultDelegate(), 0, 0)
		m.versionList.Select(0) // Выбираем первую (текущую) версию

		model, cmd := m.handleVersionListKeys(keyMsg(keyEnter))

		m = toModel(t, model)
		assert.Nil(t, m.selectedVersionForRollback, "selectedVersionForRollback should be nil")
		assert.False(t, m.confirmRollback, "confirmRollback should be false")
		require.NotNil(t, cmd, "Command should not be nil (setStatusMessageCmd)")

		// Проверяем, что вернулась команда установки статуса
		msg := s.ExecuteCmd(context.Background(), cmd)
		_, ok := msg.(clearStatusMsg) // setStatusMessageCmd возвращает clearStatusCmd
		assert.True(t, ok, "Expected clearStatusMsg from setStatusMessageCmd")
	})

	t.Run("Press Escape", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen

		model, cmd := m.handleVersionListKeys(keyMsg(keyEsc))

		m = toModel(t, model)
		assert.Equal(t, syncServerScreen, m.state, "State should change to syncServerScreen")
		require.NotNil(t, cmd, "Command should not be nil")
		_ = s.ExecuteCmd(context.Background(), cmd)
	})

	t.Run("Press Backspace", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen

		model, cmd := m.handleVersionListKeys(keyMsg(keyBack))

		m = toModel(t, model)
		assert.Equal(t, syncServerScreen, m.state, "State should change to syncServerScreen")
		require.NotNil(t, cmd, "Command should not be nil")
		_ = s.ExecuteCmd(context.Background(), cmd)
	})

	t.Run("Press r to refresh", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		m.apiClient = &CommandsTestMockAPIClient{} // Нужен для команды
		m.authToken = "test-token"                 // Нужен для команды

		model, cmd := m.handleVersionListKeys(keyMsg("r"))

		m = toModel(t, model)
		assert.True(t, m.loadingVersions, "loadingVersions should be true")
		require.NotNil(t, cmd, "Command should be loadVersionsCmd")
		// Проверяем тип команды косвенно, выполняя ее и ожидая сообщение
		// Настроим мок для команды
		mockAPI, ok := m.apiClient.(*CommandsTestMockAPIClient)
		require.True(t, ok, "Failed to cast apiClient to mock type") // Проверяем ok
		mockAPI.On(
			"ListVersions",
			mock.Anything, // ctx
			mock.Anything, // limit
			mock.Anything, // offset
		).Return([]models.VaultVersion{}, int64(0), nil).Once()
		msg := cmd()                                                             // Выполняем команду
		_, msgOk := msg.(versionsLoadedMsg)                                      // Переименовываем ok
		assert.True(t, msgOk, "Expected versionsLoadedMsg from loadVersionsCmd") // Используем msgOk
		mockAPI.AssertExpectations(t)
	})

	t.Run("Press l to login/register", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen

		model, cmd := m.handleVersionListKeys(keyMsg("l"))

		m = toModel(t, model)
		assert.Equal(t, loginRegisterChoiceScreen, m.state, "State should change to loginRegisterChoiceScreen")
		require.NotNil(t, cmd, "Command should not be nil")
		_ = s.ExecuteCmd(context.Background(), cmd)
	})

	t.Run("Press Other Key", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		initialSelected := m.selectedVersionForRollback
		initialConfirm := m.confirmRollback

		model, cmd := m.handleVersionListKeys(keyMsg("a"))

		m = toModel(t, model)
		assert.Equal(t, versionListScreen, m.state, "State should not change")
		assert.Equal(t, initialSelected, m.selectedVersionForRollback, "selectedVersionForRollback should not change")
		assert.Equal(t, initialConfirm, m.confirmRollback, "confirmRollback should not change")
		assert.Nil(t, cmd, "Command should be nil")
	})
}

func TestViewVersionListScreen(t *testing.T) {
	// Подготовка общих данных
	now := time.Now()
	testVersion := models.VaultVersion{ID: 123, ContentModifiedAt: &now}
	testError := errors.New("test rollback error")
	items := []list.Item{
		versionItem{version: models.VaultVersion{ID: 1, ContentModifiedAt: &now}, isCurrent: true},
		versionItem{version: models.VaultVersion{ID: 2, ContentModifiedAt: &now}, isCurrent: false},
	}

	t.Run("View when loading", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		m.loadingVersions = true

		view := m.viewVersionListScreen()
		assert.Contains(t, view, "Загрузка списка версий...", "View should show loading message")
	})

	t.Run("View when confirming rollback", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		m.confirmRollback = true
		m.selectedVersionForRollback = &testVersion

		view := m.viewVersionListScreen()
		expectedConfirm := fmt.Sprintf("Вы уверены, что хотите откатиться к версии #%d?", testVersion.ID)
		expectedTime := fmt.Sprintf("Время изменения: %s", formatTime(testVersion.ContentModifiedAt))
		assert.Contains(t, view, expectedConfirm, "View should show confirmation message")
		assert.Contains(t, view, expectedTime, "View should show modification time")
		assert.Contains(t, view, "Enter - подтвердить, Esc - отменить", "View should show confirmation keys")
	})

	t.Run("View when rollback error occurred", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		m.rollbackError = testError

		view := m.viewVersionListScreen()
		expectedErrorMsg := fmt.Sprintf("Ошибка отката: %v", testError)
		assert.Contains(t, view, expectedErrorMsg, "View should show rollback error message")
		// Проверяем правильный текст подсказки при ошибке
		assert.Contains(t, view,
			"Нажмите Esc для возврата к списку версий",
			"View should show error keys")
	})

	t.Run("View when version history is empty", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		m.versions = []models.VaultVersion{}                                     // Пустой список
		m.versionList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0) // Пустой list

		view := m.viewVersionListScreen()
		assert.Contains(t, view, "История версий пуста.", "View should show empty history message")
		// Проверяем детали сообщения для пустой истории
		assert.Contains(t, view,
			"После успешной синхронизации здесь появятся версии.",
			"View should show empty history details")
	})

	t.Run("View with version list", func(t *testing.T) {
		s := NewScreenTestSuite()
		m := s.Model
		m.state = versionListScreen
		// Безопасно извлекаем версии
		var version1, version2 models.VaultVersion
		var vItem0, vItem1 versionItem
		var ok bool
		require.IsType(t, versionItem{}, items[0], "Item 0 should be versionItem")
		vItem0, ok = items[0].(versionItem)
		require.True(t, ok, "Type assertion for item 0 failed")
		version1 = vItem0.version
		require.IsType(t, versionItem{}, items[1], "Item 1 should be versionItem")
		vItem1, ok = items[1].(versionItem)
		require.True(t, ok, "Type assertion for item 1 failed")
		version2 = vItem1.version
		m.versions = []models.VaultVersion{version1, version2}
		m.versionList = list.New(items, list.NewDefaultDelegate(), 80, 20) // Увеличиваем высоту до 20

		// Получаем View от списка, так как viewVersionListScreen его возвращает
		expectedListView := m.versionList.View()
		view := m.viewVersionListScreen()

		// Проверяем, что view совпадает с View() списка
		assert.Equal(t, expectedListView, view, "View should be the list view")
		// Дополнительно проверяем наличие элементов в строке
		assert.Contains(t, view, vItem0.Title(), "View should contain title of version 1")
		assert.Contains(t, view, vItem0.Description(), "View should contain description of version 1")
		assert.Contains(t, view, vItem1.Title(), "View should contain title of version 2")
		assert.Contains(t, view, vItem1.Description(), "View should contain description of version 2")
	})
}
