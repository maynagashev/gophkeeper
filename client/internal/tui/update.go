package tui

import (
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
)

// handleWindowSizeMsg обрабатывает изменение размера окна.
func handleWindowSizeMsg(m *model, msg tea.WindowSizeMsg) {
	// Обновляем размеры компонентов
	h, v := m.docStyle.GetFrameSize() // Используем стиль из модели
	listWidth := msg.Width - h
	listHeight := msg.Height - v - helpStatusHeightOffset // Используем константу

	m.entryList.SetSize(listWidth, listHeight)
	m.passwordInput.Width = msg.Width - passwordInputOffset
	m.syncServerMenu.SetSize(listWidth, listHeight) // Обновляем размер меню синхронизации

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
		newM, cmd := m.handleDBOpenedMsg(msg)
		return newM, cmd, true
	case errMsg:
		newM := m.handleErrorMsg(msg)
		return newM, nil, true
	case dbSavedMsg:
		newM, cmd := m.setStatusMessage("Сохранено успешно!")
		return newM, cmd, true
	case dbSaveErrorMsg:
		newM, cmd := m.setStatusMessage(fmt.Sprintf("Ошибка сохранения: %v", msg.err))
		return newM, cmd, true
	case clearStatusMsg:
		m.savingStatus = ""
		m.statusTimer = nil
		return m, nil, true
	default:
		return m, nil, false // Сообщение не обработано этим хендлером
	}
}

// handleAPIMsg обрабатывает сообщения от API клиента.
func handleAPIMsg(m *model, msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case loginSuccessMsg:
		m.authToken = msg.Token
		m.loginStatus = fmt.Sprintf("Вход выполнен как %s", m.loginUsernameInput.Value()) // Используем введенное имя
		m.err = nil                                                                       // Очищаем предыдущие ошибки
		m.loginUsernameInput.SetValue("")                                                 // Очищаем поля ввода
		m.loginPasswordInput.SetValue("")
		m.state = entryListScreen // Переходим к списку записей после успешного входа
		// TODO: Сохранить токен и URL в KDBX?
		newM, cmd := m.setStatusMessage("Вход выполнен успешно!")
		return newM, cmd, true // Сообщение обработано

	case LoginError: // Используем новое имя
		m.err = msg.err // Сохраняем ошибку для отображения
		// Остаемся на экране входа (m.state не меняем)
		newM, cmd := m.setStatusMessage("Ошибка входа") // Краткий статус
		return newM, cmd, true                          // Сообщение обработано

	// --- Обработка регистрации --- //
	case registerSuccessMsg:
		m.err = nil                          // Очищаем предыдущие ошибки
		m.registerUsernameInput.SetValue("") // Очищаем поля ввода
		m.registerPasswordInput.SetValue("")
		// Переводим на экран входа после успешной регистрации
		m.state = loginScreen
		m.loginUsernameInput.Focus() // Фокусируемся на поле имени пользователя для входа
		m.loginRegisterFocusedField = 0
		newM, cmd := m.setStatusMessage("Регистрация успешна! Теперь войдите.")
		return newM, cmd, true // Сообщение обработано

	case RegisterError: // Используем новое имя
		m.err = msg.err // Сохраняем ошибку для отображения
		// Остаемся на экране регистрации (m.state не меняем)
		newM, cmd := m.setStatusMessage("Ошибка регистрации") // Краткий статус
		return newM, cmd, true                                // Сообщение обработано

	default:
		return m, nil, false // Сообщение не относится к API
	}
}

// handleGlobalKeys обрабатывает глобальные сочетания клавиш.
func handleGlobalKeys(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit, true
	case "ctrl+s":
		// Сохраняем только из списка или деталей и если не Read-Only
		if !m.readOnlyMode && (m.state == entryListScreen || m.state == entryDetailScreen) && m.db != nil {
			m.savingStatus = "Подготовка к сохранению..."
			slog.Info("Начало обновления m.db перед сохранением")

			// Проходим по всем элементам в списке интерфейса
			items := m.entryList.Items()
			updatedCount := 0
			for _, item := range items {
				if listItem, ok := item.(entryItem); ok {
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
			}
			slog.Info("Обновление m.db завершено", "updated_count", updatedCount)

			m.savingStatus = "Сохранение..."
			slog.Info("Запуск сохранения KDBX", "path", m.kdbxPath)
			// Используем сохраненный пароль
			return m, saveKdbxCmd(m.db, m.kdbxPath, m.password), true
		}
		// Если сохранение не выполнено (не тот экран или read-only),
		// то клавиша не считается обработанной глобально.
		return m, nil, false
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
