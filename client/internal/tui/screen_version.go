package tui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/client/internal/api"
	"github.com/maynagashev/gophkeeper/models"
)

// --- Сообщения для работы с версиями --- //

// Константы для работы с версиями.
const (
	defaultVersionListLimit = 50 // Добавлено для mnd
)

// versionsLoadedMsg сообщает о завершении загрузки списка версий.
type versionsLoadedMsg struct {
	versions         []models.VaultVersion
	currentVersionID int64
}

// versionsLoadErrorMsg сообщает об ошибке при загрузке списка версий.
type versionsLoadErrorMsg struct {
	err error
}

// rollbackSuccessMsg сообщает об успешном откате к выбранной версии.
type rollbackSuccessMsg struct {
	versionID int64
}

// rollbackErrorMsg сообщает об ошибке при откате к выбранной версии.
type rollbackErrorMsg struct {
	err error
}

// --- Команды для работы с версиями --- //

// loadVersionsCmd загружает список версий с сервера.
func loadVersionsCmd(m *model) tea.Cmd {
	return func() tea.Msg {
		if m.apiClient == nil {
			return versionsLoadErrorMsg{err: errors.New("API клиент не инициализирован")}
		}

		if m.authToken == "" {
			return versionsLoadErrorMsg{err: errors.New("требуется авторизация")}
		}

		ctx := context.Background()
		// Вызываем обновленную ListVersions
		versions, currentID, err := m.apiClient.ListVersions(ctx, defaultVersionListLimit, 0)
		if err != nil {
			// Проверяем на ошибку авторизации
			if errors.Is(err, api.ErrAuthorization) {
				slog.Error("Ошибка загрузки списка версий: ошибка авторизации")
				return versionsLoadErrorMsg{err: api.ErrAuthorization} // Возвращаем именно ErrAuthorization
			}
			slog.Error("Ошибка загрузки списка версий", "error", err)
			return versionsLoadErrorMsg{err: err}
		}

		slog.Info("Список версий успешно загружен", "count", len(versions), "current_id", currentID)
		return versionsLoadedMsg{versions: versions, currentVersionID: currentID}
	}
}

// rollbackToVersionCmd выполняет откат к выбранной версии.
func rollbackToVersionCmd(m *model, versionID int64) tea.Cmd {
	return func() tea.Msg {
		if m.apiClient == nil {
			return rollbackErrorMsg{err: errors.New("API клиент не инициализирован")}
		}

		if m.authToken == "" {
			return rollbackErrorMsg{err: errors.New("требуется авторизация")}
		}

		ctx := context.Background()
		err := m.apiClient.RollbackToVersion(ctx, versionID)
		if err != nil {
			slog.Error("Ошибка отката к версии", "version_id", versionID, "error", err)
			return rollbackErrorMsg{err: err}
		}

		slog.Info("Успешный откат к версии", "version_id", versionID)
		return rollbackSuccessMsg{versionID: versionID}
	}
}

// --- Функции обработки экрана версий --- //

// handleVersionRollbackConfirm обрабатывает ввод в режиме подтверждения отката.
func (m *model) handleVersionRollbackConfirm(keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyMsg.String() {
	case keyEnter:
		// Подтверждение отката
		if m.selectedVersionForRollback != nil {
			m.confirmRollback = false
			m.rollbackError = nil
			return m, rollbackToVersionCmd(m, m.selectedVersionForRollback.ID)
		}
	case keyEsc, keyBack:
		// Отмена отката
		m.confirmRollback = false
		m.selectedVersionForRollback = nil
		return m, nil
	}
	return m, nil
}

// handleVersionRollbackError обрабатывает ввод в режиме отображения ошибки отката.
func (m *model) handleVersionRollbackError(keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if keyMsg.String() == keyEsc || keyMsg.String() == keyBack || keyMsg.String() == keyEnter {
		m.rollbackError = nil
		return m, nil
	}
	return m, nil
}

// handleVersionListKeys обрабатывает основные клавиши на экране списка версий.
func (m *model) handleVersionListKeys(keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyMsg.String() {
	case keyEnter:
		// Выбор версии для отката
		selectedItem := m.versionList.SelectedItem()
		if item, itemOk := selectedItem.(versionItem); itemOk { // Используем itemOk
			if item.isCurrent {
				// Нельзя откатиться к текущей версии
				return m.setStatusMessage("Это уже текущая версия")
			}
			m.selectedVersionForRollback = &item.version
			m.confirmRollback = true
			return m, nil
		}
	case keyEsc, keyBack:
		// Возврат к экрану синхронизации
		m.state = syncServerScreen
		return m, tea.ClearScreen
	case "r":
		// Обновление списка версий
		m.loadingVersions = true
		return m, loadVersionsCmd(m)
	case "l":
		// Переход к экрану логина/регистрации
		m.state = loginRegisterChoiceScreen
		return m, tea.ClearScreen
	}
	return m, nil // Клавиша не обработана здесь
}

// viewVersionListScreen отображает экран со списком версий.
func (m *model) viewVersionListScreen() string {
	if m.loadingVersions {
		return "Загрузка списка версий..."
	}

	if m.confirmRollback && m.selectedVersionForRollback != nil {
		// Показываем экран подтверждения отката
		confirmMsg := fmt.Sprintf(
			"Вы уверены, что хотите откатиться к версии #%d?\n\n"+
				"Время изменения: %s\n\n"+
				"ВНИМАНИЕ: После отката вам потребуется перезагрузить локальный файл.\n\n"+
				"Enter - подтвердить, Esc - отменить",
			m.selectedVersionForRollback.ID,
			formatTime(m.selectedVersionForRollback.ContentModifiedAt),
		)
		return confirmMsg
	}

	if m.rollbackError != nil {
		// Показываем ошибку отката
		return fmt.Sprintf("Ошибка отката: %v\n\nНажмите Esc для возврата к списку версий", m.rollbackError)
	}

	if len(m.versions) == 0 {
		return "История версий пуста.\n\nПосле успешной синхронизации здесь появятся версии."
	}

	// Обычный показ списка версий
	return m.versionList.View()
}

// updateVersionListScreen обрабатывает сообщения для экрана списка версий.
func (m *model) updateVersionListScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Обработка сообщений клавиатуры
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Если показывается экран подтверждения
		if m.confirmRollback {
			return m.handleVersionRollbackConfirm(keyMsg)
		}

		// Если показывается ошибка отката
		if m.rollbackError != nil {
			return m.handleVersionRollbackError(keyMsg)
		}

		// Стандартная обработка клавиш для списка версий
		model, keyCmd := m.handleVersionListKeys(keyMsg)
		// Если клавиша была обработана в handleVersionListKeys, она вернет команду
		if keyCmd != nil {
			return model, keyCmd
		}
		// Если клавиша не была обработана выше, передаем ее списку
	}

	// Обработка обновлений списка (скроллинг и т.д.)
	m.versionList, cmd = m.versionList.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// handleVersionsLoadedMsg обрабатывает сообщение о загруженных версиях.
func handleVersionsLoadedMsg(m *model, msg versionsLoadedMsg) (tea.Model, tea.Cmd) {
	m.loadingVersions = false
	m.versions = msg.versions

	// Преобразуем модели в версии для списка и отмечаем текущую
	var items []list.Item
	currentVersionID := msg.currentVersionID // Используем ID из сообщения

	// Пытаемся определить текущую версию, если ID из сообщения 0 или его нет (для обратной совместимости?)
	if currentVersionID == 0 && m.serverMeta != nil {
		currentVersionID = m.serverMeta.ID
		slog.Warn("currentVersionID из API = 0, используем ID из serverMeta", "serverMeta.ID", currentVersionID)
	}

	for _, v := range m.versions {
		items = append(items, versionItem{
			version:   v,
			isCurrent: v.ID == currentVersionID,
		})
	}

	// Обновляем список и получаем команду от него
	cmd := m.versionList.SetItems(items) // Не игнорируем команду

	// Добавляем ClearScreen для очистки артефактов
	return m, tea.Batch(cmd, tea.ClearScreen)
}

// handleVersionsLoadErrorMsg обрабатывает ошибку загрузки версий.
func handleVersionsLoadErrorMsg(m *model, msg versionsLoadErrorMsg) (tea.Model, tea.Cmd) {
	m.loadingVersions = false
	// Проверяем, является ли ошибка ошибкой авторизации
	if errors.Is(msg.err, api.ErrAuthorization) {
		// Не меняем состояние здесь, т.к. мы уже на экране версий,
		// но даем пользователю понять, что делать (нажать 'l')
		newM, statusCmd := m.setStatusMessage("Ошибка авторизации. Токен истек? Попробуйте войти заново (L).")
		return newM, tea.Batch(statusCmd, tea.ClearScreen) // Добавляем ClearScreen
	}
	// Иначе показываем общую ошибку
	// Возвращаем результат setStatusMessage, который включает команду
	newM, statusCmd := m.setStatusMessage(fmt.Sprintf("Ошибка загрузки версий: %v", msg.err))
	// Добавляем ClearScreen и сюда
	return newM, tea.Batch(statusCmd, tea.ClearScreen)
}

// handleRollbackSuccessMsg обрабатывает успешный откат к версии.
func handleRollbackSuccessMsg(m *model, msg rollbackSuccessMsg) (tea.Model, tea.Cmd) {
	// После успешного отката нужно обновить список версий и скачать новую версию
	newM, statusCmd := m.setStatusMessage(fmt.Sprintf("Откат к версии #%d выполнен. Синхронизация...", msg.versionID))

	// После успешного отката выполняем синхронизацию для загрузки обновленной версии
	// Добавляем ClearScreen перед синхронизацией
	return newM, tea.Batch(statusCmd, tea.ClearScreen, startSyncCmd(m))
}

// handleRollbackErrorMsg обрабатывает ошибку отката.
func handleRollbackErrorMsg(m *model, msg rollbackErrorMsg) (tea.Model, tea.Cmd) {
	// Проверяем, является ли ошибка ошибкой авторизации
	if errors.Is(msg.err, api.ErrAuthorization) {
		m.state = loginRegisterChoiceScreen // Переходим на экран выбора входа/регистрации
		newM, statusCmd := m.setStatusMessage("Сессия истекла. Пожалуйста, войдите снова (L).")
		return newM, tea.Batch(statusCmd, tea.ClearScreen)
	}
	// Иначе устанавливаем ошибку для отображения на экране версий
	m.rollbackError = msg.err
	return m, nil // Явно возвращаем nil команду
}

// Вспомогательная функция для форматирования времени.
func formatTime(t *time.Time) string {
	if t == nil {
		return "неизвестно"
	}
	return t.Format("2006-01-02 15:04:05")
}
