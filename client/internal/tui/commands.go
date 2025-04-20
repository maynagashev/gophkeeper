package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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

// syncUploadSuccessMsg сигнализирует об успешной загрузке хранилища на сервер.
type syncUploadSuccessMsg struct{}

// syncDownloadSuccessMsg сигнализирует об успешном скачивании хранилища с сервера.
// Включает флаг необходимости перезагрузки базы данных.
type syncDownloadSuccessMsg struct {
	reloadNeeded bool // Обычно true, чтобы обновить TUI
}

// (Может использоваться после скачивания или других операций, изменяющих файл извне TUI).
type dbReloadNeededMsg struct{}

const defaultFilePerm = 0600

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

// uploadVaultCmd загружает локальный файл KDBX на сервер.
func uploadVaultCmd(m *model) tea.Cmd {
	return func() tea.Msg {
		// Шаг 1: Убедиться, что все изменения из TUI применены к m.db (как в Ctrl+S)
		slog.Info("Подготовка к загрузке: обновление m.db из TUI...")
		applyUIChangesToDB(m)
		slog.Info("Обновление m.db завершено.")

		var err error // Объявляем err один раз

		// Шаг 2: Заблокировать и сохранить m.db во временный буфер для загрузки
		if err = m.db.LockProtectedEntries(); err != nil {
			slog.Warn("Не удалось заблокировать поля перед сохранением в буфер", "error", err)
		}
		buf := new(bytes.Buffer)
		encoder := gokeepasslib.NewEncoder(buf)
		if err = encoder.Encode(m.db); err != nil {
			slog.Error("Ошибка кодирования KDBX в буфер перед загрузкой", "error", err)
			// Разблокируем обратно в случае ошибки кодирования
			if unlockErr := m.db.UnlockProtectedEntries(); unlockErr != nil {
				slog.Warn("Не удалось разблокировать поля после ошибки кодирования", "error", unlockErr)
			}
			return SyncError{err: fmt.Errorf("ошибка подготовки данных для загрузки: %w", err)}
		}
		// Разблокируем поля после успешного кодирования для дальнейшей работы
		if err = m.db.UnlockProtectedEntries(); err != nil {
			slog.Warn("Не удалось разблокировать поля после кодирования в буфер", "error", err)
			// Продолжаем, так как данные уже в буфере
		}

		dataSize := int64(buf.Len())

		// Шаг 3: Вызвать API для загрузки
		slog.Info("Запуск загрузки KDBX на сервер...")
		ctx := context.Background()
		err = m.apiClient.UploadVault(ctx, buf, dataSize)
		if err != nil {
			slog.Error("Ошибка загрузки KDBX на сервер", "error", err)
			return SyncError{err: fmt.Errorf("ошибка загрузки на сервер: %w", err)}
		}

		slog.Info("Загрузка KDBX на сервер успешно завершена.")
		return syncUploadSuccessMsg{}
	}
}

// downloadVaultCmd скачивает файл KDBX с сервера и перезаписывает локальный.
func downloadVaultCmd(m *model) tea.Cmd {
	return func() tea.Msg {
		slog.Info("Запуск скачивания KDBX с сервера...")
		ctx := context.Background()
		reader, _, err := m.apiClient.DownloadVault(ctx) // Метаданные пока не используем
		if err != nil {
			slog.Error("Ошибка скачивания KDBX с сервера", "error", err)
			return SyncError{err: fmt.Errorf("ошибка скачивания с сервера: %w", err)}
		}
		defer reader.Close()

		// Шаг 2: Создать/открыть локальный файл для записи (перезапись)
		slog.Debug("Открытие локального файла для записи", "path", m.kdbxPath)
		file, err := os.OpenFile(m.kdbxPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultFilePerm)
		if err != nil {
			slog.Error("Ошибка открытия локального файла для записи", "path", m.kdbxPath, "error", err)
			return SyncError{err: fmt.Errorf("ошибка записи локального файла: %w", err)}
		}
		defer file.Close()

		// Шаг 3: Скопировать данные из ответа в файл
		_, err = io.Copy(file, reader)
		if err != nil {
			slog.Error("Ошибка копирования данных в локальный файл", "error", err)
			return SyncError{err: fmt.Errorf("ошибка сохранения скачанного файла: %w", err)}
		}

		slog.Info("Скачивание KDBX с сервера и сохранение локально завершено.")
		// Отправляем сообщение об успехе и необходимости перезагрузки
		return syncDownloadSuccessMsg{reloadNeeded: true}
	}
}

// applyUIChangesToDB применяет изменения из компонентов TUI (например, списка) к m.db.
// Эта функция должна быть похожа на логику в handleGlobalKeys для Ctrl+S.
func applyUIChangesToDB(m *model) {
	// TODO: Избежать дублирования кода с handleGlobalKeys
	items := m.entryList.Items()
	updatedCount := 0
	for _, item := range items {
		if listItem, ok := item.(entryItem); ok {
			dbEntryPtr := findEntryInDB(m.db, listItem.entry.UUID)
			if dbEntryPtr != nil {
				entryToSave := deepCopyEntry(listItem.entry)
				*dbEntryPtr = entryToSave
				updatedCount++
			} else {
				slog.Warn("Запись из списка не найдена в m.db при подготовке к загрузке", "uuid", listItem.entry.UUID)
			}
		}
	}
	slog.Debug("Применено изменений из UI в m.db перед загрузкой", "updated_count", updatedCount)
}
