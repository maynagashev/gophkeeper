package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tobischo/gokeepasslib/v3"

	"github.com/maynagashev/gophkeeper/client/internal/api"
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

// NewSyncError создает новый экземпляр SyncError для тестирования.
func NewSyncError(err error) SyncError {
	return SyncError{err: err}
}

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

// TestModel представляет собой интерфейс для тестирования команд,
// который предоставляет доступ к необходимым методам модели.
type TestModel interface {
	SetAPIClient(client api.Client)
	SetAuthToken(token string)
	SetServerURL(url string)
	SetDB(db *gokeepasslib.Database)
	MakeLoginCmd(username, password string) tea.Cmd
	MakeRegisterCmd(username, password string) tea.Cmd
}

// testModel реализует интерфейс TestModel для тестирования.
type testModel struct {
	model
}

// NewTestModel создает новую модель для тестирования.
func NewTestModel() TestModel {
	return &testModel{}
}

// SetAPIClient устанавливает API клиент в модель.
func (m *testModel) SetAPIClient(client api.Client) {
	m.apiClient = client
}

// SetAuthToken устанавливает токен авторизации в модель.
func (m *testModel) SetAuthToken(token string) {
	m.authToken = token
}

// SetServerURL устанавливает URL сервера в модель.
func (m *testModel) SetServerURL(url string) {
	m.serverURL = url
}

// SetDB устанавливает базу данных в модель.
func (m *testModel) SetDB(db *gokeepasslib.Database) {
	m.db = db
}

// MakeLoginCmd возвращает команду для входа.
func (m *testModel) MakeLoginCmd(username, password string) tea.Cmd {
	return m.makeLoginCmd(username, password)
}

// MakeRegisterCmd возвращает команду для регистрации.
func (m *testModel) MakeRegisterCmd(username, password string) tea.Cmd {
	return m.makeRegisterCmd(username, password)
}
