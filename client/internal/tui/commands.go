package tui

import (
	"context"
	"errors"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tobischo/gokeepasslib/v3"

	"github.com/maynagashev/gophkeeper/client/internal/kdbx"
)

// openKdbxCmd асинхронно открывает файл базы.
func openKdbxCmd(path, password string) tea.Cmd {
	return func() tea.Msg {
		db, err := kdbx.OpenFile(path, password)
		if err != nil {
			return errMsg{err: err}
		}
		return dbOpenedMsg{db: db}
	}
}

// saveKdbxCmd асинхронно сохраняет файл базы.
func saveKdbxCmd(db *gokeepasslib.Database, path, password string) tea.Cmd {
	return func() tea.Msg {
		err := kdbx.SaveFile(db, path, password)
		if err != nil {
			return dbSaveErrorMsg{err: err}
		}
		return dbSavedMsg{}
	}
}

// clearStatusCmd возвращает команду, которая отправит clearStatusMsg через delay.
func clearStatusCmd(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(_ time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// --- Сообщения и команды для API --- //

type loginSuccessMsg struct {
	Token string
}

type LoginError struct {
	err error
}

func (e LoginError) Error() string {
	return e.err.Error()
}

// makeLoginCmd выполняет вход через API.
func (m *model) makeLoginCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		token, err := m.apiClient.Login(ctx, username, password)
		if err != nil {
			// Возвращаем исходную ошибку API клиента без добавления контекста
			return LoginError{err: err}
		}
		return loginSuccessMsg{Token: token}
	}
}

// Сообщения для регистрации.
type registerSuccessMsg struct { // Успешная регистрация не возвращает токен
}

type RegisterError struct {
	err error
}

func (e RegisterError) Error() string {
	return e.err.Error()
}

// makeRegisterCmd выполняет регистрацию через API.
func (m *model) makeRegisterCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.apiClient.Register(ctx, username, password)
		if err != nil {
			// Возвращаем исходную ошибку API клиента без добавления контекста
			return RegisterError{err: err}
		}
		return registerSuccessMsg{}
	}
}

// --- Сообщения и команды для Синхронизации --- //

// SyncError сообщает об ошибке во время процесса синхронизации.
type SyncError struct {
	err error
}

func (e SyncError) Error() string {
	return e.err.Error()
}

// startSyncCmd проверяет предусловия и запускает процесс синхронизации.
func startSyncCmd(m *model) tea.Cmd {
	return func() tea.Msg {
		slog.Info("Запуск проверки предусловий для синхронизации...")

		// 1. Проверка URL
		if m.serverURL == "" {
			err := errors.New("URL сервера не настроен")
			slog.Warn("Предусловие синхронизации не выполнено", "error", err)
			return SyncError{err: err}
		}
		// 2. Проверка токена
		if m.authToken == "" {
			err := errors.New("необходимо войти на сервер")
			slog.Warn("Предусловие синхронизации не выполнено", "error", err)
			return SyncError{err: err}
		}
		// 3. Проверка API клиента
		if m.apiClient == nil {
			err := errors.New("API клиент не инициализирован")
			slog.Error("Критическая ошибка: API клиент nil перед синхронизацией")
			return SyncError{err: err}
		}
		// 4. Проверка базы данных
		if m.db == nil {
			err := errors.New("локальная база данных не загружена")
			slog.Error("Критическая ошибка: База данных nil перед синхронизацией")
			return SyncError{err: err}
		}

		slog.Info("Предусловия синхронизации выполнены.")
		// TODO: Здесь будет запуск получения метаданных с сервера
		// Возвращаем сообщение для обновления статуса (например, "Синхронизация...")
		// Пока вернем nil, чтобы показать, что проверки пройдены
		return nil // Заменить на команду получения метаданных
	}
}
