package tui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tobischo/gokeepasslib/v3"

	"github.com/maynagashev/gophkeeper/client/internal/kdbx"
	"github.com/maynagashev/gophkeeper/models"
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

// syncStartedMsg сигнализирует об успешном начале процесса синхронизации (предусловия пройдены).
type syncStartedMsg struct{}

// serverMetadataMsg содержит метаданные, полученные с сервера.
type serverMetadataMsg struct {
	metadata *models.VaultVersion // nil если не найдено (404)
	found    bool
}

// localMetadataMsg содержит метаданные локального файла.
type localMetadataMsg struct {
	modTime time.Time // Время модификации
	found   bool      // Файл существует?
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
		// Заменяем nil на syncStartedMsg
		return syncStartedMsg{} // Заменить на команду получения метаданных ПОСЛЕ установки статуса
	}
}

// fetchServerMetadataCmd получает метаданные хранилища с сервера.
func fetchServerMetadataCmd(m *model) tea.Cmd {
	return func() tea.Msg {
		slog.Debug("Получение метаданных с сервера", "url", m.serverURL)
		ctx := context.Background() // Используем background context
		meta, err := m.apiClient.GetVaultMetadata(ctx)

		if err != nil {
			// Проверяем специфичные ошибки API клиента с помощью switch
			switch err.Error() {
			case "хранилище не найдено на сервере": // Проверка на текст ошибки
				slog.Info("Хранилище не найдено на сервере.")
				return serverMetadataMsg{metadata: nil, found: false}
			case "ошибка авторизации (невалидный или просроченный токен?)":
				slog.Warn("Ошибка авторизации при получении метаданных с сервера", "error", err)
				return SyncError{err: errors.New("ошибка авторизации")}
			default:
				// Другая ошибка (сетевая, 5xx, ошибка декодирования)
				slog.Error("Ошибка получения метаданных с сервера", "error", err)
				return SyncError{err: fmt.Errorf("ошибка сети или сервера: %w", err)}
			}
		}

		// Успешно получили метаданные
		slog.Debug("Метаданные с сервера получены", "versionId", meta.ID, "createdAt", meta.CreatedAt)
		return serverMetadataMsg{metadata: meta, found: true}
	}
}

// fetchLocalMetadataCmd получает время модификации локального файла.
func fetchLocalMetadataCmd(m *model) tea.Cmd {
	return func() tea.Msg {
		slog.Debug("Получение метаданных локального файла", "path", m.kdbxPath)
		fileInfo, err := os.Stat(m.kdbxPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				slog.Info("Локальный файл KDBX не найден.")
				return localMetadataMsg{found: false} // Файл не найден
			}
			// Другая ошибка при доступе к файлу
			slog.Error("Ошибка получения метаданных локального файла", "path", m.kdbxPath, "error", err)
			return SyncError{err: fmt.Errorf("ошибка доступа к локальному файлу: %w", err)}
		}
		// Успешно получили информацию о файле
		slog.Debug("Метаданные локального файла получены", "modTime", fileInfo.ModTime())
		return localMetadataMsg{modTime: fileInfo.ModTime(), found: true}
	}
}
