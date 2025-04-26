//nolint:testpackage // Это файл с вспомогательными функциями для тестов в том же пакете
package tui

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tobischo/gokeepasslib/v3"
)

// TestVaultVersion модель версии хранилища для тестов
type TestVaultVersion struct {
	ID                int64
	CreatedAt         time.Time
	SizeBytes         int64
	ContentModifiedAt time.Time
	ServerModifiedAt  time.Time
}

// TestEntry модель записи для тестов
type TestEntry struct {
	UUID     string
	Title    string
	Username string
	Password string
	URL      string
	Notes    string
}

func (e TestEntry) GetTitle() string {
	return e.Title
}

func (e TestEntry) GetUsername() string {
	return e.Username
}

func (e TestEntry) GetPassword() string {
	return e.Password
}

func (e TestEntry) GetURL() string {
	return e.URL
}

func (e TestEntry) GetNotes() string {
	return e.Notes
}

func (e TestEntry) GetUUID() string {
	return e.UUID
}

// ScreenTestSuite содержит общую инфраструктуру для тестирования экранов
type ScreenTestSuite struct {
	Model *model
	Mocks struct {
		APIClient *ScreenTestMockAPIClient
	}
}

// ScreenTestMockAPIClient - мок для API клиента
type ScreenTestMockAPIClient struct {
	mock.Mock
}

// Login мокирует метод Login
func (m *ScreenTestMockAPIClient) Login(ctx context.Context, username, password string) (string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.Error(1)
}

// Register мокирует метод Register
func (m *ScreenTestMockAPIClient) Register(ctx context.Context, username, password string) error {
	args := m.Called(ctx, username, password)
	return args.Error(0)
}

// GetVaultMetadata мокирует метод GetVaultMetadata
func (m *ScreenTestMockAPIClient) GetVaultMetadata(ctx context.Context) (*TestVaultVersion, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TestVaultVersion), args.Error(1)
}

// UploadVault мокирует метод UploadVault
func (m *ScreenTestMockAPIClient) UploadVault(ctx context.Context, data io.Reader, size int64, contentModifiedAt time.Time) error {
	args := m.Called(ctx, data, size, contentModifiedAt)
	return args.Error(0)
}

// DownloadVault мокирует метод DownloadVault
func (m *ScreenTestMockAPIClient) DownloadVault(ctx context.Context) (io.ReadCloser, *TestVaultVersion, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	if args.Get(1) == nil {
		return args.Get(0).(io.ReadCloser), nil, args.Error(2)
	}
	return args.Get(0).(io.ReadCloser), args.Get(1).(*TestVaultVersion), args.Error(2)
}

// ListVersions мокирует метод ListVersions
func (m *ScreenTestMockAPIClient) ListVersions(ctx context.Context, limit, offset int) ([]TestVaultVersion, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]TestVaultVersion), args.Get(1).(int64), args.Error(2)
}

// RollbackToVersion мокирует метод RollbackToVersion
func (m *ScreenTestMockAPIClient) RollbackToVersion(ctx context.Context, versionID int64) error {
	args := m.Called(ctx, versionID)
	return args.Error(0)
}

// SetAuthToken мокирует метод SetAuthToken
func (m *ScreenTestMockAPIClient) SetAuthToken(token string) {
	m.Called(token)
}

// NewScreenTestSuite создает новую тестовую среду для экранов
func NewScreenTestSuite() *ScreenTestSuite {
	s := &ScreenTestSuite{}

	// Инициализируем модель
	s.Model = &model{
		// Базовые поля модели
		db:           nil,
		kdbxPath:     "/tmp/test.kdbx",
		readOnlyMode: false,
		password:     "test-password",

		// Текстовые поля
		loginUsernameInput:    textinput.New(),
		loginPasswordInput:    textinput.New(),
		registerUsernameInput: textinput.New(),
		registerPasswordInput: textinput.New(),
		serverURLInput:        textinput.New(),

		// Компоненты UI
		entryList: list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
	}

	// Инициализируем моки
	s.Mocks.APIClient = new(ScreenTestMockAPIClient)
	s.Model.apiClient = s.Mocks.APIClient

	return s
}

// WithServerURL устанавливает URL сервера в модели
func (s *ScreenTestSuite) WithServerURL(url string) *ScreenTestSuite {
	s.Model.serverURL = url
	return s
}

// WithAuthToken устанавливает токен авторизации в модели
func (s *ScreenTestSuite) WithAuthToken(token string) *ScreenTestSuite {
	s.Model.authToken = token
	return s
}

// WithDatabase устанавливает базу данных в модели
func (s *ScreenTestSuite) WithDatabase(db *gokeepasslib.Database) *ScreenTestSuite {
	s.Model.db = db
	return s
}

// WithState устанавливает состояние экрана в модели
func (s *ScreenTestSuite) WithState(state screenState) *ScreenTestSuite {
	s.Model.state = state
	return s
}

// SimulateKeyPress симулирует нажатие клавиши
func (s *ScreenTestSuite) SimulateKeyPress(key tea.KeyType) (tea.Model, tea.Cmd) {
	msg := tea.KeyMsg{Type: key}
	return s.Model.Update(msg)
}

// SimulateKeyRune симулирует ввод символа
func (s *ScreenTestSuite) SimulateKeyRune(r rune) (tea.Model, tea.Cmd) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
	return s.Model.Update(msg)
}

// ExecuteCmd выполняет команду и возвращает результат
func (s *ScreenTestSuite) ExecuteCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// AssertViewContains проверяет, что View() модели содержит указанный текст
func (s *ScreenTestSuite) AssertViewContains(t *testing.T, substring string) {
	view := s.Model.View()
	assert.Contains(t, view, substring, "View должен содержать указанный текст")
}

// AssertState проверяет, что состояние модели соответствует ожидаемому
func (s *ScreenTestSuite) AssertState(t *testing.T, expected screenState) {
	assert.Equal(t, expected, s.Model.state, "Состояние модели должно быть %s, но получено %s", expected, s.Model.state)
}

// toModel безопасно приводит tea.Model к *model
func toModel(t *testing.T, m tea.Model) *model {
	result, ok := m.(*model)
	require.True(t, ok, "Модель должна быть типа *model")
	return result
}

// CaptureView выполняет команду и возвращает результат View()
func (s *ScreenTestSuite) CaptureView(cmd tea.Cmd) string {
	if cmd != nil {
		msg := cmd()
		newModel, _ := s.Model.Update(msg)
		s.Model = toModel(nil, newModel) // Игнорируем ошибку, так как мы контролируем типы
	}
	return s.Model.View()
}

// CaptureOutput выполняет последовательность команд и возвращает финальный View
func (s *ScreenTestSuite) CaptureOutput(cmds ...tea.Cmd) string {
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		msg := cmd()
		newModel, _ := s.Model.Update(msg)
		s.Model = toModel(nil, newModel)
	}
	return s.Model.View()
}

// CreateBasicTestDB создает простую тестовую базу данных
func CreateBasicTestDB() *gokeepasslib.Database {
	db := gokeepasslib.NewDatabase()
	db.Content = gokeepasslib.NewContent()

	rootGroup := gokeepasslib.Group{
		Name: "Root",
	}

	// Создаем пару записей для тестирования
	entry1 := gokeepasslib.NewEntry()
	entry1.Values = append(entry1.Values, gokeepasslib.ValueData{
		Key:   "Title",
		Value: gokeepasslib.V{Content: "Тестовая запись 1"},
	})
	entry1.Values = append(entry1.Values, gokeepasslib.ValueData{
		Key:   "UserName",
		Value: gokeepasslib.V{Content: "user1"},
	})
	entry1.Values = append(entry1.Values, gokeepasslib.ValueData{
		Key:   "Password",
		Value: gokeepasslib.V{Content: "pass1"},
	})

	entry2 := gokeepasslib.NewEntry()
	entry2.Values = append(entry2.Values, gokeepasslib.ValueData{
		Key:   "Title",
		Value: gokeepasslib.V{Content: "Тестовая запись 2"},
	})

	rootGroup.Entries = append(rootGroup.Entries, entry1, entry2)
	db.Content.Root = &gokeepasslib.RootData{
		Groups: []gokeepasslib.Group{rootGroup},
	}

	return db
}

// MockResponse создает ответ для мока API клиента
type MockResponse struct {
	Success bool
	Token   string
	Error   error
}

// SetupMockAPILogin настраивает мок API клиента для метода Login
func (s *ScreenTestSuite) SetupMockAPILogin(username, password string, response MockResponse) *ScreenTestSuite {
	if response.Success {
		s.Mocks.APIClient.On("Login", mock.Anything, username, password).
			Return(response.Token, nil).Once()
	} else {
		s.Mocks.APIClient.On("Login", mock.Anything, username, password).
			Return("", response.Error).Once()
	}
	return s
}

// SetupMockAPIRegister настраивает мок API клиента для метода Register
func (s *ScreenTestSuite) SetupMockAPIRegister(username, password string, response MockResponse) *ScreenTestSuite {
	if response.Success {
		s.Mocks.APIClient.On("Register", mock.Anything, username, password).
			Return(nil).Once()
	} else {
		s.Mocks.APIClient.On("Register", mock.Anything, username, password).
			Return(response.Error).Once()
	}
	return s
}

// RenderScreen выполняет рендеринг View() для тестирования отображения
func (s *ScreenTestSuite) RenderScreen() string {
	var buf bytes.Buffer

	// Получаем View текущей модели
	view := s.Model.View()

	// Очищаем ANSI-последовательности для упрощения тестирования
	// Это простая очистка, для более точной можно использовать библиотеки
	cleanView := strings.ReplaceAll(view, "\033[", "")
	cleanView = strings.ReplaceAll(cleanView, "\r", "")

	buf.WriteString(cleanView)
	return buf.String()
}
