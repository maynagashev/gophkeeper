// игнорируем ошибки приведения типов
//
//nolint:testpackage,errcheck // Тесты в том же пакете для доступа к непубличным функциям,
package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/client/internal/api"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/stretchr/testify/require"
	"github.com/tobischo/gokeepasslib/v3"
	"github.com/tobischo/gokeepasslib/v3/wrappers"
)

// Создаем тестовую модель для тестов.
func createTestModelForUpdate() *model {
	m := &model{
		state:    entryListScreen,
		password: "test_password",
		kdbxPath: "/tmp/test.kdbx",

		// Инициализируем текстовые поля, необходимые для тестов
		passwordInput:         textinput.New(),
		loginUsernameInput:    textinput.New(),
		loginPasswordInput:    textinput.New(),
		registerUsernameInput: textinput.New(),
		registerPasswordInput: textinput.New(),
	}

	// Инициализируем поля для предотвращения паники
	m.loginRegisterFocusedField = 0
	m.serverURLInput = textinput.New()
	m.serverURL = "https://example.com"

	// Фокусируемся на полях ввода, чтобы инициализировать их курсоры
	m.passwordInput.Focus()
	m.loginUsernameInput.Focus()
	m.loginPasswordInput.Focus()
	m.registerUsernameInput.Focus()
	m.registerPasswordInput.Focus()
	m.serverURLInput.Focus()
	// Убираем фокус с остальных, чтобы избежать неожиданного поведения в тестах
	m.loginUsernameInput.Blur()
	m.loginPasswordInput.Blur()
	m.registerUsernameInput.Blur()
	m.registerPasswordInput.Blur()
	m.serverURLInput.Blur()

	// Инициализируем список версий
	m.versionList = initVersionList() // Добавляем инициализацию списка версий

	return m
}

// TestHandleDBSavedMsg проверяет обработку сообщения об успешном сохранении базы данных.
func TestHandleDBSavedMsg(t *testing.T) {
	m := createTestModelForUpdate()

	// Проверяем обработку
	newM, cmd := handleDBSavedMsg(m)

	// Проверяем результаты
	require.NotNil(t, cmd, "Должна быть возвращена команда")
	require.Contains(t, newM.(*model).savingStatus, "успешно", "Статус должен содержать сообщение об успехе")
}

// TestHandleDBSaveErrorMsg проверяет обработку сообщения об ошибке сохранения.
func TestHandleDBSaveErrorMsg(t *testing.T) {
	m := createTestModelForUpdate()
	testErr := errors.New("тестовая ошибка")
	msg := dbSaveErrorMsg{err: testErr}

	// Проверяем обработку
	newM, cmd := handleDBSaveErrorMsg(m, msg)

	// Проверяем результаты
	require.NotNil(t, cmd, "Должна быть возвращена команда")
	require.Contains(t, newM.(*model).savingStatus, "ошибка", "Статус должен содержать сообщение об ошибке")
	require.Contains(t, newM.(*model).savingStatus, testErr.Error(), "Текст ошибки должен быть включен в статус")
}

// TestHandleClearStatusMsg проверяет обработку сообщения очистки статуса.
func TestHandleClearStatusMsg(t *testing.T) {
	m := createTestModelForUpdate()
	m.savingStatus = "Тестовый статус"
	m.statusTimer = time.NewTimer(time.Second)

	// Проверяем обработку
	newM, cmd := handleClearStatusMsg(m)

	// Проверяем результаты
	require.NotNil(t, cmd, "Должна быть возвращена команда")
	require.Equal(t, "", newM.(*model).savingStatus, "Статус должен быть очищен")
	require.Nil(t, newM.(*model).statusTimer, "Таймер должен быть обнулен")
}

// TestHandleSyncErrorMsg проверяет обработку ошибок синхронизации.
func TestHandleSyncErrorMsg(t *testing.T) {
	t.Run("ОбычнаяОшибка", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.isSyncing = true
		testErr := errors.New("тестовая ошибка синхронизации")
		msg := SyncError{err: testErr}

		// Проверяем обработку
		newM, cmd := handleSyncErrorMsg(m, msg)

		// Проверяем результаты
		require.NotNil(t, cmd, "Должна быть возвращена команда")
		require.False(t, newM.(*model).isSyncing, "Флаг isSyncing должен быть сброшен")
		require.Contains(t, newM.(*model).savingStatus, "ошибка", "Статус должен содержать сообщение об ошибке")
		require.Contains(t, newM.(*model).savingStatus, testErr.Error(), "Текст ошибки должен быть включен в статус")
		require.Equal(t, entryListScreen, newM.(*model).state, "Состояние не должно меняться для обычной ошибки")
	})

	t.Run("ОшибкаАвторизации", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.isSyncing = true
		testErr := api.ErrAuthorization
		msg := SyncError{err: testErr}

		// Проверяем обработку
		newM, cmd := handleSyncErrorMsg(m, msg)

		// Проверяем результаты
		require.NotNil(t, cmd, "Должна быть возвращена команда")
		require.False(t, newM.(*model).isSyncing, "Флаг isSyncing должен быть сброшен")
		require.Equal(t, loginRegisterChoiceScreen, newM.(*model).state, "Состояние должно измениться на экран входа")
		require.Contains(t, newM.(*model).savingStatus, "Пожалуйста, войдите", "Статус должен содержать сообщение о входе")
	})
}

// TestHandleSyncStartedMsg проверяет обработку начала синхронизации.
func TestHandleSyncStartedMsg(t *testing.T) {
	m := createTestModelForUpdate()

	// Проверяем обработку
	newM, cmd := handleSyncStartedMsg(m)

	// Проверяем результаты
	require.NotNil(t, cmd, "Должна быть возвращена команда")
	require.True(t, newM.(*model).isSyncing, "Флаг isSyncing должен быть установлен")
	require.False(t, newM.(*model).receivedServerMeta, "Флаг receivedServerMeta должен быть сброшен")
	require.False(t, newM.(*model).receivedLocalMeta, "Флаг receivedLocalMeta должен быть сброшен")
	require.Contains(t, newM.(*model).savingStatus, "метаданн", "Статус должен содержать сообщение о метаданных")
}

// TestHandleServerMetadataMsg проверяет обработку сообщения с метаданными сервера.
func TestHandleServerMetadataMsg(t *testing.T) {
	t.Run("Не в состоянии синхронизации", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.isSyncing = false
		msg := serverMetadataMsg{}

		newM, cmd := handleServerMetadataMsg(m, msg)

		require.Same(t, m, newM, "Модель не должна меняться")
		require.Nil(t, cmd, "Команда должна быть nil")
		require.False(t, m.receivedServerMeta, "receivedServerMeta должен остаться false")
	})

	t.Run("В состоянии синхронизации, локальные метаданные не получены", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.isSyncing = true
		m.receivedLocalMeta = false
		meta := &models.VaultVersion{ID: 1}
		msg := serverMetadataMsg{metadata: meta, found: true}

		newM, cmd := handleServerMetadataMsg(m, msg)

		updatedModel := newM.(*model)
		require.True(t, updatedModel.receivedServerMeta, "receivedServerMeta должен стать true")
		require.Same(t, meta, updatedModel.serverMeta, "serverMeta должен обновиться")
		require.True(t, updatedModel.serverMetaFound, "serverMetaFound должен обновиться")
		require.Nil(t, cmd, "Команда должна быть nil, так как локальные метаданные еще не получены")
	})

	t.Run("В состоянии синхронизации, локальные метаданные уже получены", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.isSyncing = true
		m.receivedLocalMeta = true // Локальные метаданные уже есть
		m.localMetaFound = true
		m.localMetaModTime = time.Now().Add(-time.Hour) // Устанавливаем какое-то время
		meta := &models.VaultVersion{ID: 1, ContentModifiedAt: &m.localMetaModTime}
		msg := serverMetadataMsg{metadata: meta, found: true}

		newM, cmd := handleServerMetadataMsg(m, msg)
		updatedModel := newM.(*model)

		// Проверяем, что processMetadataResults был вызван (по побочным эффектам)
		require.False(t, updatedModel.receivedServerMeta, "receivedServerMeta должен быть сброшен в processMetadataResults")
		require.False(t, updatedModel.receivedLocalMeta, "receivedLocalMeta должен быть сброшен в processMetadataResults")
		require.False(t, updatedModel.isSyncing, "isSyncing должен быть сброшен в processMetadataResults")
		require.NotNil(t, cmd, "Должна быть возвращена команда из processMetadataResults")
	})
}

// TestHandleLocalMetadataMsg проверяет обработку сообщения с локальными метаданными.
func TestHandleLocalMetadataMsg(t *testing.T) {
	t.Run("Не в состоянии синхронизации", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.isSyncing = false
		msg := localMetadataMsg{}

		newM, cmd := handleLocalMetadataMsg(m, msg)

		require.Same(t, m, newM, "Модель не должна меняться")
		require.Nil(t, cmd, "Команда должна быть nil")
		require.False(t, m.receivedLocalMeta, "receivedLocalMeta должен остаться false")
	})

	t.Run("В состоянии синхронизации, метаданные сервера не получены", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.isSyncing = true
		m.receivedServerMeta = false
		modTime := time.Now()
		msg := localMetadataMsg{modTime: modTime, found: true}

		newM, cmd := handleLocalMetadataMsg(m, msg)

		updatedModel := newM.(*model)
		require.True(t, updatedModel.receivedLocalMeta, "receivedLocalMeta должен стать true")
		require.Equal(t, modTime, updatedModel.localMetaModTime, "localMetaModTime должен обновиться")
		require.True(t, updatedModel.localMetaFound, "localMetaFound должен обновиться")
		require.Nil(t, cmd, "Команда должна быть nil, так как метаданные сервера еще не получены")
	})

	t.Run("В состоянии синхронизации, метаданные сервера уже получены", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.isSyncing = true
		m.receivedServerMeta = true // Метаданные сервера уже есть
		m.serverMetaFound = true
		serverTime := time.Now().Add(-time.Hour)
		m.serverMeta = &models.VaultVersion{ID: 1, ContentModifiedAt: &serverTime}
		modTime := time.Now()
		msg := localMetadataMsg{modTime: modTime, found: true}

		newM, cmd := handleLocalMetadataMsg(m, msg)

		updatedModel := newM.(*model)

		// Проверяем, что processMetadataResults был вызван (по побочным эффектам)
		require.False(t, updatedModel.receivedServerMeta, "receivedServerMeta должен быть сброшен в processMetadataResults")
		require.False(t, updatedModel.receivedLocalMeta, "receivedLocalMeta должен быть сброшен в processMetadataResults")
		require.False(t, updatedModel.isSyncing, "isSyncing должен быть сброшен в processMetadataResults")
		require.NotNil(t, cmd, "Должна быть возвращена команда из processMetadataResults")
	})
}

// TestHandleSyncUploadSuccessMsg проверяет обработку сообщения об успешной загрузке.
func TestHandleSyncUploadSuccessMsg(t *testing.T) {
	m := createTestModelForUpdate()

	newM, cmd := handleSyncUploadSuccessMsg(m)

	updatedModel := newM.(*model)
	// Разбиваем строку для линтера
	expectedStatus := "завершена (загружено)"
	require.Contains(t, updatedModel.savingStatus, expectedStatus,
		"Статус должен содержать сообщение об успешной загрузке")
	require.NotNil(t, cmd, "Должна быть возвращена команда (Batch)")
	// TODO: Проверить, что время последней синхронизации обновлено, когда это будет реализовано
}

// TestHandleVersionMsg проверяет обработку сообщений, связанных с версиями.
func TestHandleVersionMsg(t *testing.T) {
	t.Run("versionsLoadedMsg", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.loadingVersions = true // Имитируем состояние загрузки
		versions := []models.VaultVersion{{ID: 1}, {ID: 2}}
		currentID := int64(2)
		msg := versionsLoadedMsg{versions: versions, currentVersionID: currentID}

		newM, cmd, handled := handleVersionMsg(m, msg)

		require.True(t, handled, "Сообщение должно быть обработано")
		require.NotNil(t, cmd, "Должна быть возвращена команда (от списка)")
		updatedModel := newM.(*model)
		require.False(t, updatedModel.loadingVersions, "Флаг loadingVersions должен быть сброшен")
		require.Len(t, updatedModel.versions, 2, "Список версий должен обновиться")
		require.Equal(t, versions, updatedModel.versions, "Содержимое списка версий должно обновиться")
		// Проверяем, что текущая версия отмечена правильно в списке TUI (не напрямую в m.versions)
		require.NotNil(t, updatedModel.versionList, "Список версий TUI должен быть инициализирован")
		items := updatedModel.versionList.Items()
		require.Len(t, items, 2, "В списке TUI должно быть 2 элемента")
		foundCurrent := false
		for _, item := range items {
			vItem, ok := item.(versionItem)
			require.True(t, ok, "Элемент списка должен быть типа versionItem")
			if vItem.version.ID == currentID {
				require.True(t, vItem.isCurrent, "Версия с currentID должна быть отмечена как текущая")
				foundCurrent = true
			} else {
				require.False(t, vItem.isCurrent, "Другие версии не должны быть отмечены как текущие")
			}
		}
		require.True(t, foundCurrent, "Текущая версия должна быть найдена в списке TUI")
	})

	t.Run("versionsLoadErrorMsg", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.loadingVersions = true // Имитируем состояние загрузки
		testErr := errors.New("ошибка загрузки версий")
		msg := versionsLoadErrorMsg{err: testErr}

		newM, cmd, handled := handleVersionMsg(m, msg)

		require.True(t, handled, "Сообщение должно быть обработано")
		require.NotNil(t, cmd, "Должна быть возвращена команда статуса")
		updatedModel := newM.(*model)
		require.False(t, updatedModel.loadingVersions, "Флаг loadingVersions должен быть сброшен")
		expectedStatusPart := "Ошибка загрузки версий: ошибка загрузки версий" // Исправляем на "списка"
		require.Contains(t, updatedModel.savingStatus, expectedStatusPart,
			"Статус должен содержать сообщение об ошибке")
	})

	t.Run("rollbackSuccessMsg", func(t *testing.T) {
		m := createTestModelForUpdate()
		versionID := int64(123)
		msg := rollbackSuccessMsg{versionID: versionID}

		newM, cmd, handled := handleVersionMsg(m, msg)

		require.True(t, handled, "Сообщение должно быть обработано")
		require.NotNil(t, cmd, "Должна быть возвращена команда статуса и обновления версий")
		updatedModel := newM.(*model)
		expectedStatus := "Откат к версии #123 успешен. Загрузка..." // Исправляем ожидаемый статус
		require.Contains(t, updatedModel.savingStatus, expectedStatus,
			"Статус должен содержать сообщение об успехе")
		require.NoError(t, updatedModel.rollbackError, "Ошибка отката должна быть сброшена")
	})

	t.Run("rollbackErrorMsg", func(t *testing.T) {
		m := createTestModelForUpdate()
		testErr := errors.New("ошибка отката")
		msg := rollbackErrorMsg{err: testErr}

		newM, cmd, handled := handleVersionMsg(m, msg)

		require.True(t, handled, "Сообщение должно быть обработано")
		require.NotNil(t, cmd, "Должна быть возвращена команда статуса")
		updatedModel := newM.(*model)
		require.ErrorIs(t, updatedModel.rollbackError, testErr, "Ошибка отката должна быть установлена в модели")
	})

	t.Run("НеизвестноеСообщение", func(t *testing.T) {
		m := createTestModelForUpdate()
		msg := "неизвестное сообщение" // Не тот тип

		newM, cmd, handled := handleVersionMsg(m, msg)

		require.False(t, handled, "Неизвестное сообщение не должно быть обработано")
		require.Nil(t, cmd, "Команда должна быть nil")
		require.Same(t, m, newM, "Модель не должна измениться")
	})
}

// TestHandleSyncDownloadSuccessMsg проверяет обработку успешного скачивания.
func TestHandleSyncDownloadSuccessMsg(t *testing.T) {
	t.Run("Без перезагрузки", func(t *testing.T) {
		m := createTestModelForUpdate()
		msg := syncDownloadSuccessMsg{reloadNeeded: false}

		newM, cmd := handleSyncDownloadSuccessMsg(m, msg)

		updatedModel := newM.(*model)
		require.Contains(t, updatedModel.savingStatus, "Синхронизация завершена (скачано)", "Статус должен обновиться")

		// Проверяем, что команда - это просто команда статуса (без openKdbxCmd)
		isBatch := false
		if _, ok := cmd().(tea.BatchMsg); ok {
			isBatch = true
		}
		require.False(t, isBatch, "Команда не должна быть BatchMsg, если перезагрузка не нужна")
		// Мы не можем точно проверить тип команды статуса, т.к. setStatusMessage возвращает func() tea.Msg
		require.NotNil(t, cmd, "Должна быть возвращена команда статуса")
	})

	t.Run("С перезагрузкой", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.kdbxPath = "/path/to/test.kdbx" // Устанавливаем путь для команды
		m.password = "testpass"           // Устанавливаем пароль для команды
		msg := syncDownloadSuccessMsg{reloadNeeded: true}

		newM, cmd := handleSyncDownloadSuccessMsg(m, msg)

		updatedModel := newM.(*model)
		require.Contains(t, updatedModel.savingStatus, "Синхронизация завершена (скачано)", "Статус должен обновиться")

		// Проверяем, что команда - это BatchMsg
		require.NotNil(t, cmd, "Должна быть возвращена команда BatchMsg")
		batchCmd, ok := cmd().(tea.BatchMsg)
		require.True(t, ok, "Команда должна быть BatchMsg")
		require.Len(t, batchCmd, 2, "BatchMsg должен содержать 2 команды (статус и открытие)")
	})
}

// TestCanSave проверяет логику разрешения сохранения.
func TestCanSave(t *testing.T) {
	t.Run("РазрешеноСохранение", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.state = entryListScreen
		m.readOnlyMode = false
		m.db = &gokeepasslib.Database{} // Используем настоящую структуру

		result := m.canSave()
		require.True(t, result, "Сохранение должно быть разрешено")
	})

	t.Run("РежимТолькоЧтение", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.state = entryListScreen
		m.readOnlyMode = true
		m.db = &gokeepasslib.Database{} // Используем настоящую структуру

		result := m.canSave()
		require.False(t, result, "Сохранение не должно быть разрешено в режиме только чтение")
	})

	t.Run("НетБазыДанных", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.state = entryListScreen
		m.readOnlyMode = false
		m.db = nil // База данных не загружена

		result := m.canSave()
		require.False(t, result, "Сохранение не должно быть разрешено без базы данных")
	})

	t.Run("НеподходящееСостояние", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.state = passwordInputScreen // Не entryListScreen или entryDetailScreen
		m.readOnlyMode = false
		m.db = &gokeepasslib.Database{} // Используем настоящую структуру

		result := m.canSave()
		require.False(t, result, "Сохранение не должно быть разрешено в неподходящем состоянии")
	})
}

// TestUpdateRootModTime проверяет обновление времени модификации корня.
func TestUpdateRootModTime(t *testing.T) {
	t.Run("УспешноеОбновление", func(t *testing.T) {
		// Создаем реальную базу данных с корневой группой
		mockTime := wrappers.TimeWrapper{Time: time.Now().Add(-24 * time.Hour)}
		rootGroup := gokeepasslib.Group{
			Times: gokeepasslib.TimeData{
				LastModificationTime: &mockTime,
			},
		}

		db := &gokeepasslib.Database{
			Content: &gokeepasslib.DBContent{
				Root: &gokeepasslib.RootData{
					Groups: []gokeepasslib.Group{rootGroup},
				},
			},
		}

		m := createTestModelForUpdate()
		m.db = db

		// Сохраняем исходное время для сравнения
		originalTime := mockTime.Time

		// Вызываем тестируемый метод
		m.updateRootModTime()

		// Проверяем результат
		updatedTime := m.db.Content.Root.Groups[0].Times.LastModificationTime.Time
		require.NotEqual(t, originalTime, updatedTime, "Время модификации должно быть обновлено")
		require.True(t, updatedTime.After(originalTime), "Новое время должно быть позже старого")
	})

	t.Run("ПустаяБазаДанных", func(_ *testing.T) {
		m := createTestModelForUpdate()
		m.db = nil

		// Вызываем тестируемый метод, не должно быть паники
		m.updateRootModTime()
	})

	t.Run("НетКорневойГруппы", func(_ *testing.T) {
		db := &gokeepasslib.Database{
			Content: &gokeepasslib.DBContent{
				Root: &gokeepasslib.RootData{
					Groups: []gokeepasslib.Group{}, // Пустой массив групп
				},
			},
		}

		m := createTestModelForUpdate()
		m.db = db

		// Вызываем тестируемый метод, не должно быть паники
		m.updateRootModTime()
	})
}

// TestHandleGlobalKeys проверяет обработку глобальных клавиш.
func TestHandleGlobalKeys(t *testing.T) {
	t.Run("CtrlC", func(t *testing.T) {
		m := createTestModelForUpdate()
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}

		// Проверяем обработку
		_, cmd, handled := handleGlobalKeys(m, keyMsg)

		// Проверяем результаты
		require.True(t, handled, "Ctrl+C должен быть обработан")
		require.NotNil(t, cmd, "Должна быть возвращена команда tea.Quit")
	})

	t.Run("CtrlS_РазрешеноСохранение", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.state = entryListScreen
		m.db = &gokeepasslib.Database{} // Используем настоящую структуру
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlS}

		// Проверяем обработку
		newM, cmd, handled := handleGlobalKeys(m, keyMsg)

		// Проверяем результаты
		require.True(t, handled, "Ctrl+S должен быть обработан")
		require.NotNil(t, cmd, "Должна быть возвращена команда сохранения")
		require.Contains(t, newM.(*model).savingStatus, "охранени", "Статус должен содержать сообщение о сохранении")
	})

	t.Run("CtrlS_ЗапрещеноСохранение", func(t *testing.T) {
		m := createTestModelForUpdate()
		m.state = passwordInputScreen // Неподходящее состояние
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlS}

		// Проверяем обработку
		_, cmd, handled := handleGlobalKeys(m, keyMsg)

		// Проверяем результаты
		require.False(t, handled, "Ctrl+S не должен быть обработан, если сохранение запрещено")
		require.Nil(t, cmd, "Команда должна быть nil, если сохранение запрещено")
	})

	t.Run("ДругаяКлавиша", func(t *testing.T) {
		m := createTestModelForUpdate()
		keyMsg := tea.KeyMsg{Type: tea.KeyEnter}

		// Проверяем обработку
		_, _, handled := handleGlobalKeys(m, keyMsg)

		// Проверяем результаты
		require.False(t, handled, "Клавиша Enter не должна быть обработана глобальным обработчиком")
	})
}

// TestUpdateDBFromList проверяет обновление базы данных из списка записей.
func TestUpdateDBFromList(t *testing.T) {
	// Создаем базовую модель для тестирования
	m := createTestModelForUpdate()

	// Создаем тестовую базу данных
	db := gokeepasslib.NewDatabase()
	db.Content = &gokeepasslib.DBContent{
		Root: &gokeepasslib.RootData{
			Groups: []gokeepasslib.Group{
				{
					Name: "TestGroup",
					Entries: []gokeepasslib.Entry{
						{
							UUID: gokeepasslib.NewUUID(),
							Values: []gokeepasslib.ValueData{
								{
									Key:   "Title",
									Value: gokeepasslib.V{Content: "Original Entry"},
								},
							},
						},
					},
				},
			},
		},
	}
	m.db = db

	t.Run("ОбновлениеСуществующейЗаписи", func(t *testing.T) {
		// Получаем UUID существующей записи
		entryUUID := m.db.Content.Root.Groups[0].Entries[0].UUID

		// Создаем модифицированную запись для списка
		modifiedEntry := gokeepasslib.Entry{
			UUID: entryUUID,
			Values: []gokeepasslib.ValueData{
				{
					Key:   "Title",
					Value: gokeepasslib.V{Content: "Modified Entry"},
				},
				{
					Key:   "UserName",
					Value: gokeepasslib.V{Content: "test_user"},
				},
			},
		}

		// Создаем список с модифицированной записью
		m.entryList = list.New([]list.Item{entryItem{entry: modifiedEntry}}, list.NewDefaultDelegate(), 0, 0)

		// Вызываем тестируемый метод
		updatedCount := m.updateDBFromList()

		// Проверяем результаты
		require.Equal(t, 1, updatedCount, "Должна быть обновлена одна запись")

		// Проверяем, что запись в базе данных обновилась
		dbEntry := &m.db.Content.Root.Groups[0].Entries[0]

		// Находим значение поля Title
		var titleValue, userNameValue string
		for _, val := range dbEntry.Values {
			if val.Key == "Title" {
				titleValue = val.Value.Content
			}
			if val.Key == "UserName" {
				userNameValue = val.Value.Content
			}
		}

		require.Equal(t, "Modified Entry", titleValue, "Заголовок должен быть обновлен")
		require.Equal(t, "test_user", userNameValue, "Имя пользователя должно быть добавлено")
	})

	t.Run("ЗаписьНеНайденаВБазе", func(t *testing.T) {
		// Создаем запись с несуществующим UUID
		nonExistentEntry := gokeepasslib.Entry{
			UUID: gokeepasslib.NewUUID(), // Новый UUID, который не существует в базе
			Values: []gokeepasslib.ValueData{
				{
					Key:   "Title",
					Value: gokeepasslib.V{Content: "New Entry"},
				},
			},
		}

		// Создаем список с новой записью
		m.entryList = list.New([]list.Item{entryItem{entry: nonExistentEntry}}, list.NewDefaultDelegate(), 0, 0)

		// Вызываем тестируемый метод
		updatedCount := m.updateDBFromList()

		// Проверяем результаты
		require.Equal(t, 0, updatedCount, "Не должно быть обновленных записей")

		// Проверяем, что база данных не изменилась (осталась только одна запись)
		require.Len(t, m.db.Content.Root.Groups[0].Entries, 1, "Количество записей не должно измениться")
	})

	t.Run("ПустойСписок", func(t *testing.T) {
		// Создаем пустой список
		m.entryList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

		// Вызываем тестируемый метод
		updatedCount := m.updateDBFromList()

		// Проверяем результаты
		require.Equal(t, 0, updatedCount, "При пустом списке не должно быть обновленных записей")
	})
}

// TestHandleErrorMsg проверяет обработку общего сообщения об ошибке.
func TestHandleErrorMsg(t *testing.T) {
	m := createTestModelForUpdate()
	testErr := errors.New("общая тестовая ошибка")
	msg := errMsg{err: testErr}

	// Проверяем обработку
	newM := handleErrorMsg(m, msg)

	// Проверяем результаты
	require.Equal(t, testErr, newM.(*model).err, "Поле err должно содержать ошибку из сообщения")
}
