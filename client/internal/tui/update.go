package tui

import (
	"fmt"
	"log/slog"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/client/internal/api"
)

// Update обрабатывает входящие сообщения.
//
//nolint:gocognit,funlen,gocyclo // TODO: Рефакторить роутинг и длину функции
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd // Собираем команды

	switch msg := msg.(type) {
	// == Глобальные сообщения (не зависят от экрана) ==
	case tea.WindowSizeMsg:
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

		return m, nil

	case dbOpenedMsg:
		return m.handleDBOpenedMsg(msg)

	case errMsg:
		return m.handleErrorMsg(msg)

	case dbSavedMsg:
		return m.setStatusMessage("Сохранено успешно!")

	case dbSaveErrorMsg:
		return m.setStatusMessage(fmt.Sprintf("Ошибка сохранения: %v", msg.err))

	case clearStatusMsg:
		m.savingStatus = ""
		m.statusTimer = nil
		return m, nil

	// Обработка нажатия клавиш делегируется состоянию
	case tea.KeyMsg:
		// Глобальные команды (работают на всех экранах, кроме ввода пароля?)
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
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
				return m, saveKdbxCmd(m.db, m.kdbxPath, m.password)
			}
		}
		// Если не глобальная команда, передаем дальше в обработчик текущего экрана
	}

	// == Обновление компонентов в зависимости от состояния ==
	var updatedModel tea.Model = m // По умолчанию возвращаем текущую модель
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
		// Обрабатываем Esc и Enter, остальное передаем в textinput
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case keyEsc:
				m.state = syncServerScreen
				return m, nil
			case keyEnter:
				newURL := m.serverURLInput.Value()
				if newURL == "" {
					newURL = m.serverURLInput.Placeholder // Используем плейсхолдер если пусто
				}
				// TODO: Добавить валидацию URL?
				m.serverURL = newURL
				// Сбрасываем статус, т.к. URL изменился
				m.loginStatus = "Не выполнен"
				m.authToken = ""
				m.apiClient = api.NewHTTPClient(newURL) // Пересоздаем клиент с новым URL
				slog.Info("URL сервера обновлен", "url", newURL)
				// Переходим к выбору логина/регистрации
				m.state = loginRegisterChoiceScreen
				return m, nil
			}
		}
		// Обновляем поле ввода
		newInput, cmd := m.serverURLInput.Update(msg)
		m.serverURLInput = newInput
		stateCmd = cmd
	case loginRegisterChoiceScreen:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "r", "R":
				m.state = registerScreen
				m.registerUsernameInput.Focus()
				m.loginRegisterFocusedField = 0
				return m, textinput.Blink
			case "l", "L":
				m.state = loginScreen
				m.loginUsernameInput.Focus()
				m.loginRegisterFocusedField = 0
				return m, textinput.Blink
			case keyEsc, keyBack:
				m.state = entryListScreen
				return m, nil
			}
		}
	case loginScreen:
		var focusedInput *textinput.Model
		if m.loginRegisterFocusedField == 0 {
			focusedInput = &m.loginUsernameInput
		} else {
			focusedInput = &m.loginPasswordInput
		}
		*focusedInput, stateCmd = focusedInput.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == keyEsc {
				m.state = loginRegisterChoiceScreen
				return m, nil
			}
		}
	case registerScreen:
		var focusedInput *textinput.Model
		if m.loginRegisterFocusedField == 0 {
			focusedInput = &m.registerUsernameInput
		} else {
			focusedInput = &m.registerPasswordInput
		}
		*focusedInput, stateCmd = focusedInput.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == keyEsc {
				m.state = loginRegisterChoiceScreen
				return m, nil
			}
		}
	default:
		// Неизвестное состояние - возвращаем как есть
		// updatedModel = m // Уже присвоено по умолчанию
	}
	cmds = append(cmds, stateCmd)

	// Кастуем тип обратно к *model перед возвратом
	finalModel, ok := updatedModel.(*model)
	if !ok {
		// Это не должно произойти, если все update... функции возвращают *model
		slog.Error("Ошибка каста модели в *model")
		return m, tea.Quit // Выход в случае серьезной ошибки
	}

	return finalModel, tea.Batch(cmds...)
}
