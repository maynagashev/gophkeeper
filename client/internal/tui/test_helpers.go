package tui

import (
	"time"

	"github.com/tobischo/gokeepasslib/v3"

	"github.com/maynagashev/gophkeeper/models"
)

// Экспортируем типы сообщений для тестирования.
type (
	// DBOpenedMsg сообщение об успешном открытии базы данных.
	DBOpenedMsg struct {
		DB *gokeepasslib.Database
	}

	// ErrMsg сообщение об ошибке.
	ErrMsg struct {
		Err error
	}

	// DBSavedMsg сообщение об успешном сохранении базы данных.
	DBSavedMsg struct{}

	// DBSaveErrorMsg сообщение об ошибке сохранения базы данных.
	DBSaveErrorMsg struct {
		Err error
	}

	// ClearStatusMsg сообщение для очистки статусного сообщения.
	ClearStatusMsg struct{}

	// SyncStartedMsg сообщение о начале синхронизации.
	SyncStartedMsg struct{}

	// ServerMetadataMsg содержит метаданные, полученные с сервера.
	ServerMetadataMsg struct {
		Metadata *models.VaultVersion
		Found    bool
	}

	// LocalMetadataMsg содержит метаданные локального файла.
	LocalMetadataMsg struct {
		ModTime time.Time
		Found   bool
	}

	// SyncUploadSuccessMsg сигнализирует об успешной загрузке.
	SyncUploadSuccessMsg struct{}

	// SyncDownloadSuccessMsg сигнализирует об успешном скачивании.
	SyncDownloadSuccessMsg struct {
		ReloadNeeded bool
	}

	// LoginSuccessMsg сообщение об успешном входе.
	LoginSuccessMsg struct {
		Token string
	}

	// RegisterSuccessMsg сообщение об успешной регистрации.
	RegisterSuccessMsg struct{}
)

// Экспортируем функции-команды для тестирования.
//
//nolint:gochecknoglobals // Эти переменные нужны только для тестирования
var (
	OpenKdbxCmd            = openKdbxCmd
	SaveKdbxCmd            = saveKdbxCmd
	ClearStatusCmd         = clearStatusCmd
	StartSyncCmd           = startSyncCmd
	FetchServerMetadataCmd = fetchServerMetadataCmd
)
