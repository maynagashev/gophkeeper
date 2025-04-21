package tui

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	// Убедимся, что импорты на месте.
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/client/internal/api"
	"github.com/maynagashev/gophkeeper/client/internal/kdbx"
	"github.com/tobischo/gokeepasslib/v3/wrappers"
)

// handleWindowSizeMsg обрабатывает изменение размера окна.
func handleWindowSizeMsg(m *model, msg tea.WindowSizeMsg) {
	// Обновляем размеры компонентов
	h, v := m.docStyle.GetFrameSize() // Используем стиль из модели
	listWidth := msg.Width - h
	// Высота для основного списка записей
	entryListHeight := msg.Height - v - helpStatusHeightOffset // Используем константу

	// Высота для меню синхронизации
	// Статус занимает 3 строки + 1 строка разделитель = 4
	const statusHeight = 4
	syncMenuHeight := msg.Height - v - statusHeight

	m.entryList.SetSize(listWidth, entryListHeight)
	m.passwordInput.Width = msg.Width - passwordInputOffset
	m.syncServerMenu.SetSize(listWidth, syncMenuHeight) // Используем новую высоту

	// TODO: Обновить размеры других полей ввода по необходимости
	m.serverURLInput.Width = listWidth - passwordInputOffset
	m.loginUsernameInput.Width = listWidth - passwordInputOffset
	m.loginPasswordInput.Width = listWidth - passwordInputOffset
	m.registerUsernameInput.Width = listWidth - passwordInputOffset
	m.registerPasswordInput.Width = listWidth - passwordInputOffset
}

// handleDBMsg обрабатывает сообщения, связанные с базой данных или статусом.
func handleDBMsg(m *model, msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case dbOpenedMsg:
		newM, cmd := handleDBOpenedMsg(m, msg)
		return newM, cmd, true
	case errMsg:
		newM := handleErrorMsg(m, msg)
		return newM, nil, true
	case dbSavedMsg:
		newM, cmd := handleDBSavedMsg(m)
		return newM, cmd, true
	case dbSaveErrorMsg:
		newM, cmd := handleDBSaveErrorMsg(m, msg)
		return newM, cmd, true
	case clearStatusMsg:
		newM, cmd := handleClearStatusMsg(m)
		return newM, cmd, true
	case SyncError:
		newM, cmd := handleSyncErrorMsg(m, msg)
		return newM, cmd, true
	case syncStartedMsg:
		newM, cmd := handleSyncStartedMsg(m)
		return newM, cmd, true
	case serverMetadataMsg:
		newM, cmd := handleServerMetadataMsg(m, msg)
		return newM, cmd, true
	case localMetadataMsg:
		newM, cmd := handleLocalMetadataMsg(m, msg)
		return newM, cmd, true
	case syncUploadSuccessMsg:
		newM, cmd := handleSyncUploadSuccessMsg(m)
		return newM, cmd, true
	case syncDownloadSuccessMsg:
		newM, cmd := handleSyncDownloadSuccessMsg(m, msg)
		return newM, cmd, true
	default:
		return m, nil, false // Не обработали сообщение этого типа
	}
}

// --- Обработчики для handleDBMsg ---

func handleDBOpenedMsg(m *model, msg dbOpenedMsg) (tea.Model, tea.Cmd) {
	return m.handleDBOpenedMsg(msg)
}

func handleErrorMsg(m *model, msg errMsg) tea.Model {
	return m.handleErrorMsg(msg)
}

func handleDBSavedMsg(m *model) (tea.Model, tea.Cmd) {
	// Добавляем tea.ClearScreen в Batch после установки статуса
	newM, statusCmd := m.setStatusMessage("Сохранено успешно!")
	return newM, tea.Batch(statusCmd, tea.ClearScreen)
}

func handleDBSaveErrorMsg(m *model, msg dbSaveErrorMsg) (tea.Model, tea.Cmd) {
	return m.setStatusMessage(fmt.Sprintf("Ошибка сохранения: %v", msg.err))
}

func handleClearStatusMsg(m *model) (tea.Model, tea.Cmd) {
	m.savingStatus = ""
	m.statusTimer = nil // Обнуляем таймер (хотя он и так не используется для отмены)
	// Возвращаем команду для принудительной перерисовки
	return m, tea.ClearScreen
}

func handleSyncErrorMsg(m *model, msg SyncError) (tea.Model, tea.Cmd) {
	m.isSyncing = false
	// Проверяем на ошибку авторизации
	if errors.Is(msg.err, api.ErrAuthorization) {
		m.state = loginRegisterChoiceScreen // Переходим на экран выбора входа/регистрации
		// Можно добавить очистку экрана, если нужно
		newM, statusCmd := m.setStatusMessage("Сессия истекла. Пожалуйста, войдите снова (L).")
		return newM, tea.Batch(statusCmd, tea.ClearScreen)
	}
	// Иначе показываем общую ошибку синхронизации
	// Добавляем ClearScreen и здесь
	newM, statusCmd := m.setStatusMessage(fmt.Sprintf("Ошибка синхронизации: %v", msg.err))
	return newM, tea.Batch(statusCmd, tea.ClearScreen)
}

func handleSyncStartedMsg(m *model) (tea.Model, tea.Cmd) {
	m.isSyncing = true
	m.receivedServerMeta = false
	m.receivedLocalMeta = false
	newM, statusCmd := m.setStatusMessage("Получение метаданных...")
	fetchCmds := tea.Batch(fetchServerMetadataCmd(m), fetchLocalMetadataCmd(m))
	return newM, tea.Batch(statusCmd, fetchCmds)
}

func handleServerMetadataMsg(m *model, msg serverMetadataMsg) (tea.Model, tea.Cmd) {
	if !m.isSyncing {
		return m, nil
	}
	m.serverMeta = msg.metadata
	m.serverMetaFound = msg.found
	m.receivedServerMeta = true
	slog.Debug("Получено сообщение serverMetadataMsg", "found", msg.found)
	if m.receivedLocalMeta {
		newM, cmd := m.processMetadataResults()
		return newM, cmd
	}
	return m, nil
}

func handleLocalMetadataMsg(m *model, msg localMetadataMsg) (tea.Model, tea.Cmd) {
	if !m.isSyncing {
		return m, nil
	}
	m.localMetaModTime = msg.modTime
	m.localMetaFound = msg.found
	m.receivedLocalMeta = true
	slog.Debug("Получено сообщение localMetadataMsg", "found", msg.found)
	if m.receivedServerMeta {
		newM, cmd := m.processMetadataResults()
		return newM, cmd
	}
	return m, nil
}

func handleSyncUploadSuccessMsg(m *model) (tea.Model, tea.Cmd) {
	// TODO: Обновить время последней синхронизации в m
	newM, statusCmd := m.setStatusMessage("Синхронизация завершена (загружено)")
	// Добавляем ClearScreen
	return newM, tea.Batch(statusCmd, tea.ClearScreen)
}

func handleSyncDownloadSuccessMsg(m *model, msg syncDownloadSuccessMsg) (tea.Model, tea.Cmd) {
	newM, statusCmd := m.setStatusMessage("Синхронизация завершена (скачано), перезагрузка...")
	if msg.reloadNeeded {
		openCmd := openKdbxCmd(m.kdbxPath, m.password)
		return newM, tea.Batch(statusCmd, openCmd)
	}
	return newM, statusCmd
}

// processMetadataResults обрабатывает ситуацию, когда получены и локальные, и серверные метаданные.
func (m *model) processMetadataResults() (tea.Model, tea.Cmd) {
	slog.Info("Получены метаданные сервера и локального файла. Запуск сравнения...")

	// Определяем время модификации контента с сервера
	// (Используем новое поле ContentModifiedAt)
	var serverModTime time.Time
	if m.serverMeta != nil && m.serverMeta.ContentModifiedAt != nil {
		serverModTime = *m.serverMeta.ContentModifiedAt
	}

	slog.Debug("Данные для сравнения",
		"serverFound", m.serverMetaFound,
		"serverMetaTime", serverModTime, // Теперь это ContentModifiedAt
		"localFound", m.localMetaFound,
		"localMetaTime", m.localMetaModTime, // Это local db.Root.LastModificationTime
	)

	// Сбрасываем флаги получения
	m.receivedServerMeta = false
	m.receivedLocalMeta = false
	m.isSyncing = false // Завершаем состояние синхронизации

	var cmd tea.Cmd
	var statusMsg string

	switch {
	// Случай 1: Хранилища нет на сервере (404)
	case !m.serverMetaFound:
		if m.localMetaFound {
			slog.Info("Хранилища нет на сервере, локальное есть. Загрузка на сервер.")
			statusMsg = "Загрузка на сервер..."
			cmd = uploadVaultCmd(m) // Команда загрузки
		} else {
			slog.Info("Нет ни локального хранилища, ни на сервере. Нечего синхронизировать.")
			statusMsg = "Нечего синхронизировать."
			cmd = nil // Ничего не делаем
		}
	// Случай 2: Хранилище есть на сервере
	case m.serverMetaFound:
		if !m.localMetaFound {
			// Случай 3: Локального файла нет, но на сервере есть
			slog.Info("Локального файла нет, но есть на сервере. Скачивание с сервера.")
			statusMsg = "Скачивание с сервера..."
			cmd = downloadVaultCmd(m) // Команда скачивания
		} else {
			// Обе версии существуют, сравниваем время
			// Переписываем if-else на switch для gocritic
			switch {
			case m.localMetaModTime.After(serverModTime):
				slog.Info("Локальная версия новее. Загрузка на сервер.")
				statusMsg = "Загрузка на сервер..."
				cmd = uploadVaultCmd(m) // Команда загрузки
			case serverModTime.After(m.localMetaModTime):
				slog.Info("Серверная версия новее. Скачивание с сервера.")
				statusMsg = "Скачивание с сервера..."
				cmd = downloadVaultCmd(m) // Команда скачивания
			default: // Времена равны
				slog.Info("Версии идентичны. Синхронизация не требуется.")
				statusMsg = "Уже синхронизировано."
				cmd = nil // Ничего не делаем
			}
		}
	}

	// Устанавливаем статус и возвращаем команду
	newM, statusCmd := m.setStatusMessage(statusMsg)
	// Объединяем команду установки статуса с командой загрузки/скачивания (если она есть)
	finalCmd := tea.Batch(statusCmd, cmd)

	return newM, finalCmd
}

// handleVersionMsg обрабатывает сообщения, связанные с версиями.
func handleVersionMsg(m *model, msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case versionsLoadedMsg:
		newM, cmd := handleVersionsLoadedMsg(m, msg)
		return newM, cmd, true
	case versionsLoadErrorMsg:
		newM, cmd := handleVersionsLoadErrorMsg(m, msg)
		return newM, cmd, true
	case rollbackSuccessMsg:
		newM, cmd := handleRollbackSuccessMsg(m, msg)
		return newM, cmd, true
	case rollbackErrorMsg:
		newM, cmd := handleRollbackErrorMsg(m, msg)
		return newM, cmd, true
	default:
		return m, nil, false // Не обработали сообщение этого типа
	}
}

// handleAPIMsg обрабатывает сообщения от API клиента.
func handleAPIMsg(m *model, msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case loginSuccessMsg:
		m.authToken = msg.Token
		m.loginStatus = fmt.Sprintf("Вход выполнен как %s", m.loginUsernameInput.Value())
		m.err = nil
		m.loginUsernameInput.SetValue("")
		m.loginPasswordInput.SetValue("")

		// Устанавливаем токен в существующем API клиенте
		if m.apiClient != nil {
			m.apiClient.SetAuthToken(m.authToken)
			slog.Debug("Установлен токен в API клиенте после успешного входа")
		} else {
			slog.Error("API клиент nil при попытке установить токен после входа")
		}

		// Сохраняем Auth данные в KDBX (в памяти)
		if m.db != nil {
			errSave := kdbx.SaveAuthData(m.db, m.serverURL, m.authToken)
			if errSave != nil {
				slog.Error("Ошибка сохранения Auth данных в KDBX (в памяти)", "error", errSave)
				m.err = fmt.Errorf("ошибка сохранения данных сессии: %w", errSave)
				m.state = loginScreen // Остаемся на экране входа для показа ошибки
				newM, statusCmd := m.setStatusMessage("Ошибка сохранения сессии")
				return newM, tea.Batch(statusCmd, tea.ClearScreen), true // Возвращаемся при ошибке
			}
			// Если ошибки не было
			slog.Info("Auth данные успешно обновлены в KDBX (в памяти)")
			m.state = entryListScreen // Переходим к списку записей
		} else {
			slog.Error("Попытка сохранить Auth данные в KDBX, но m.db is nil")
			m.state = entryListScreen // Переходим к списку, но без сохранения данных сессии
		}

		// Возвращаем команды только после успешного сохранения (или если db был nil)
		newM, statusCmd := m.setStatusMessage("Вход выполнен успешно!")
		return newM, tea.Batch(statusCmd, tea.ClearScreen), true

	case LoginError:
		m.err = msg.err
		newM, statusCmd := m.setStatusMessage("Ошибка входа")
		// Добавляем очистку экрана, чтобы перерисовать с ошибкой чисто
		return newM, tea.Batch(statusCmd, tea.ClearScreen), true

	// --- Обработка регистрации --- //
	case registerSuccessMsg:
		m.err = nil
		m.registerUsernameInput.SetValue("")
		m.registerPasswordInput.SetValue("")
		m.state = loginScreen
		m.loginUsernameInput.Focus()
		m.loginRegisterFocusedField = 0
		newM, statusCmd := m.setStatusMessage("Регистрация успешна! Теперь войдите.")
		// Добавляем команду очистки экрана
		return newM, tea.Batch(statusCmd, tea.ClearScreen), true

	case RegisterError:
		m.err = msg.err
		newM, statusCmd := m.setStatusMessage("Ошибка регистрации")
		// Добавляем очистку экрана, чтобы перерисовать с ошибкой чисто
		return newM, tea.Batch(statusCmd, tea.ClearScreen), true

	default:
		return m, nil, false
	}
}

// canSave checks if the application is in a state where saving is allowed.
func (m *model) canSave() bool {
	return !m.readOnlyMode && m.db != nil && (m.state == entryListScreen || m.state == entryDetailScreen)
}

// updateDBFromList updates the in-memory database (m.db) with data from the TUI list (m.entryList).
func (m *model) updateDBFromList() int {
	slog.Info("Начало обновления m.db перед сохранением")
	items := m.entryList.Items()
	updatedCount := 0
	for _, item := range items {
		listItem, ok := item.(entryItem)
		if !ok {
			continue // Skip if not an entryItem
		}
		// Находим соответствующую запись в m.db по UUID
		dbEntryPtr := findEntryInDB(m.db, listItem.entry.UUID)
		if dbEntryPtr != nil {
			// Обновляем найденную запись данными из элемента списка
			// Создаем копию перед присваиванием, чтобы не менять listItem
			entryToSave := deepCopyEntry(listItem.entry)
			*dbEntryPtr = entryToSave
			updatedCount++
		} else {
			slog.Warn("Запись из списка не найдена в m.db", "uuid", listItem.entry.UUID)
		}
	}
	slog.Info("Обновление m.db завершено", "updated_count", updatedCount)
	return updatedCount
}

// updateRootModTime updates the LastModificationTime of the root group.
func (m *model) updateRootModTime() {
	if m.db == nil || m.db.Content == nil || m.db.Content.Root == nil {
		slog.Warn("Не удалось обновить LastModificationTime: db, Content или Root is nil")
		return
	}

	now := time.Now().UTC()
	modTimeWrapper := wrappers.TimeWrapper{Time: now}

	// Обновляем время модификации первой (корневой) группы
	if len(m.db.Content.Root.Groups) > 0 {
		rootGroup := &m.db.Content.Root.Groups[0]
		rootGroup.Times.LastModificationTime = &modTimeWrapper // Присваиваем указатель
		slog.Debug("Обновлено LastModificationTime корневой группы перед сохранением", "newTime", now)
	} else {
		slog.Warn("Не удалось обновить LastModificationTime: нет корневой группы")
	}
}

// handleSaveKeyPress handles the logic for the Ctrl+S key press.
func (m *model) handleSaveKeyPress() (tea.Model, tea.Cmd) {
	if !m.canSave() {
		return m, nil // Not in a state to save, or read-only
	}

	m.savingStatus = "Подготовка к сохранению..."

	// Update m.db from the list UI
	m.updateDBFromList()

	// Update the root modification time
	m.updateRootModTime()

	m.savingStatus = "Сохранение..."
	slog.Info("Запуск сохранения KDBX", "path", m.kdbxPath)
	// Use the stored password
	return m, saveKdbxCmd(m.db, m.kdbxPath, m.password)
}

// handleGlobalKeys обрабатывает глобальные сочетания клавиш.
func handleGlobalKeys(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit, true
	case "ctrl+s":
		// Delegate saving logic to a separate function
		updatedModel, cmd := m.handleSaveKeyPress()
		// Return true only if saving was actually attempted (canSave was true)
		return updatedModel, cmd, m.canSave()
	default:
		// Клавиша не является глобальной
		return m, nil, false
	}
}

// Update обрабатывает входящие сообщения.
//
//nolint:funlen // TODO: Рефакторить роутинг и длину функции (убрали gocyclo)
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd         // Собираем команды для батчинга
	var cmd tea.Cmd            // Команда от обработчика
	var handled bool           // Флаг: сообщение было обработано глобальным хендлером
	var updatedModel tea.Model // Модель по умолчанию - текущая (убираем `= m`)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Обработка изменения размера окна
		handleWindowSizeMsg(m, msg)
		return m, nil // Возвращаем nil команду здесь, так как команда не нужна

	case tea.KeyMsg:
		// Обработка глобальных клавиш
		updatedModel, cmd, handled = handleGlobalKeys(m, msg)
		if handled {
			return updatedModel, cmd
		}
		// Если не глобальная клавиша, передаем дальше для обработки по состоянию

	default:
		// Сначала пытаемся обработать сообщения API
		updatedModel, cmd, handled = handleAPIMsg(m, msg)
		if handled {
			return updatedModel, cmd
		}

		// Затем пытаемся обработать сообщения БД/статуса
		updatedModel, cmd, handled = handleDBMsg(m, msg)
		if handled {
			return updatedModel, cmd
		}

		// Затем пытаемся обработать сообщения версий
		updatedModel, cmd, handled = handleVersionMsg(m, msg)
		if handled {
			return updatedModel, cmd
		}
	}

	// == Обработка сообщения в зависимости от текущего состояния ==
	// (Вызывается, только если сообщение не было обработано глобально)
	var stateCmd tea.Cmd
	switch m.state {
	case welcomeScreen:
		updatedModel, stateCmd = m.updateWelcomeScreen(msg)
	case passwordInputScreen:
		updatedModel, stateCmd = m.updatePasswordInputScreen(msg)
	case newKdbxPasswordScreen:
		updatedModel, stateCmd = m.updateNewKdbxPasswordScreen(msg)
	case entryListScreen:
		updatedModel, stateCmd = m.updateEntryListScreen(msg)
	case entryDetailScreen:
		updatedModel, stateCmd = m.updateEntryDetailScreen(msg)
	case entryEditScreen:
		updatedModel, stateCmd = m.updateEntryEditScreen(msg)
	case entryAddScreen:
		updatedModel, stateCmd = m.updateEntryAddScreen(msg)
	case attachmentListDeleteScreen:
		updatedModel, stateCmd = m.updateAttachmentListDeleteScreen(msg)
	case attachmentPathInputScreen:
		updatedModel, stateCmd = m.updateAttachmentPathInputScreen(msg)
	case syncServerScreen:
		updatedModel, stateCmd = m.updateSyncServerScreen(msg)
	case serverURLInputScreen:
		updatedModel, stateCmd = m.updateServerURLInputScreen(msg)
	case loginRegisterChoiceScreen:
		updatedModel, stateCmd = m.updateLoginRegisterChoiceScreen(msg)
	case loginScreen:
		updatedModel, stateCmd = m.updateLoginScreen(msg)
	case registerScreen:
		updatedModel, stateCmd = m.updateRegisterScreen(msg)
	case versionListScreen:
		updatedModel, stateCmd = m.updateVersionListScreen(msg)
	default:
		// Неизвестное состояние - ничего не делаем, updatedModel остается nil?
		// Это нужно обработать: если updatedModel не был присвоен,
		// нужно вернуть исходную модель m.
		updatedModel = m // Присваиваем m, если ни один case не сработал
		// stateCmd остается nil
	}
	cmds = append(cmds, stateCmd) // Добавляем команду от обработчика состояния

	// Кастуем тип обратно к *model перед возвратом
	finalModel, ok := updatedModel.(*model)
	if !ok {
		// Это не должно произойти, если все update... функции возвращают *model
		// Или если updatedModel не был присвоен в default
		slog.Error("Ошибка каста модели в *model или updatedModel не был присвоен")
		return m, tea.Quit // Выход в случае серьезной ошибки
	}

	return finalModel, tea.Batch(cmds...)
}
