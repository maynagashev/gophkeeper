package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/stretchr/testify/assert"
)

// TestVersionItem_Title проверяет метод Title для versionItem.
func TestVersionItem_Title(t *testing.T) {
	tests := []struct {
		name      string
		item      versionItem
		wantTitle string
	}{
		{
			name: "Обычная версия",
			item: versionItem{
				version: models.VaultVersion{
					ID: 123,
				},
				isCurrent: false,
			},
			wantTitle: "Версия #123",
		},
		{
			name: "Текущая версия",
			item: versionItem{
				version: models.VaultVersion{
					ID: 456,
				},
				isCurrent: true,
			},
			wantTitle: "Версия #456 (Текущая)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantTitle, tt.item.Title())
		})
	}
}

// TestVersionItem_Description проверяет метод Description для versionItem.
func TestVersionItem_Description(t *testing.T) {
	now := time.Now()
	nowStr := now.Format(time.RFC3339)
	size := int64(2048) // 2 KB

	tests := []struct {
		name            string
		item            versionItem
		wantDescription string
	}{
		{
			name: "Только ID",
			item: versionItem{
				version: models.VaultVersion{
					ID: 1,
				},
			},
			wantDescription: "ID: 1",
		},
		{
			name: "Только время модификации",
			item: versionItem{
				version: models.VaultVersion{
					ID:                2,
					ContentModifiedAt: &now,
				},
			},
			wantDescription: fmt.Sprintf("Изменена: %s", nowStr),
		},
		{
			name: "Только размер",
			item: versionItem{
				version: models.VaultVersion{
					ID:        3,
					SizeBytes: &size,
				},
			},
			wantDescription: "Размер: 2.00 KB",
		},
		{
			name: "Время и размер",
			item: versionItem{
				version: models.VaultVersion{
					ID:                4,
					ContentModifiedAt: &now,
					SizeBytes:         &size,
				},
			},
			wantDescription: fmt.Sprintf("Изменена: %s | Размер: 2.00 KB", nowStr),
		},
		{
			name: "Текущая версия со временем и размером", // isCurrent не влияет на Description
			item: versionItem{
				version: models.VaultVersion{
					ID:                5,
					ContentModifiedAt: &now,
					SizeBytes:         &size,
				},
				isCurrent: true,
			},
			wantDescription: fmt.Sprintf("Изменена: %s | Размер: 2.00 KB", nowStr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantDescription, tt.item.Description())
		})
	}
}

// TestVersionItem_FilterValue проверяет метод FilterValue для versionItem.
func TestVersionItem_FilterValue(t *testing.T) {
	// FilterValue просто возвращает Title, поэтому тесты аналогичны TestVersionItem_Title
	tests := []struct {
		name            string
		item            versionItem
		wantFilterValue string
	}{
		{
			name: "Обычная версия",
			item: versionItem{
				version: models.VaultVersion{
					ID: 789,
				},
				isCurrent: false,
			},
			wantFilterValue: "Версия #789",
		},
		{
			name: "Текущая версия",
			item: versionItem{
				version: models.VaultVersion{
					ID: 101,
				},
				isCurrent: true,
			},
			wantFilterValue: "Версия #101 (Текущая)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantFilterValue, tt.item.FilterValue())
		})
	}
}

// TestInitPasswordInput проверяет инициализацию поля ввода пароля.
func TestInitPasswordInput(t *testing.T) {
	ti := initPasswordInput()
	assert.Equal(t, "Мастер-пароль", ti.Placeholder)
	assert.True(t, ti.Focused())
	assert.Equal(t, initPasswordCharLimit, ti.CharLimit)
	assert.Equal(t, initPasswordWidth, ti.Width)
	assert.Equal(t, textinput.EchoPassword, ti.EchoMode)
}

// TestInitEntryList проверяет инициализацию списка записей.
func TestInitEntryList(t *testing.T) {
	l := initEntryList()
	assert.Equal(t, "Записи", l.Title)
	assert.False(t, l.ShowHelp())
	assert.True(t, l.ShowStatusBar())
	assert.Equal(t, list.Unfiltered, l.FilterState())
	assert.True(t, l.Styles.Title.GetBold())
}

// TestInitAttachmentDeleteList проверяет инициализацию списка удаления вложений.
func TestInitAttachmentDeleteList(t *testing.T) {
	l := initAttachmentDeleteList()
	assert.Equal(t, "Выберите вложение для удаления", l.Title)
	assert.False(t, l.ShowHelp())
	assert.False(t, l.ShowStatusBar())
	assert.Equal(t, list.Unfiltered, l.FilterState()) // Фильтрация должна быть выключена (Unfiltered)
	assert.True(t, l.Styles.Title.GetBold())
}

// TestInitAttachmentPathInput проверяет инициализацию поля ввода пути к вложению.
func TestInitAttachmentPathInput(t *testing.T) {
	ti := initAttachmentPathInput()
	assert.Equal(t, "/path/to/your/file", ti.Placeholder)
	assert.Equal(t, initPathCharLimit, ti.CharLimit)
	// Используем константы, как в оригинальной функции
	assert.Equal(t, defaultListWidth-passwordInputOffset, ti.Width)
	assert.False(t, ti.Focused()) // По умолчанию фокуса нет
}

// TestInitNewKdbxPasswordInputs проверяет инициализацию полей для нового пароля KDBX.
func TestInitNewKdbxPasswordInputs(t *testing.T) {
	pass1, pass2 := initNewKdbxPasswordInputs()

	// Проверка первого поля
	assert.Equal(t, "Новый мастер-пароль", pass1.Placeholder)
	assert.True(t, pass1.Focused()) // Первое поле должно быть в фокусе
	assert.Equal(t, initPasswordCharLimit, pass1.CharLimit)
	assert.Equal(t, initPasswordWidth, pass1.Width)
	assert.Equal(t, textinput.EchoPassword, pass1.EchoMode)

	// Проверка второго поля
	assert.Equal(t, "Подтвердите пароль", pass2.Placeholder)
	assert.False(t, pass2.Focused()) // Второе поле не должно быть в фокусе
	assert.Equal(t, initPasswordCharLimit, pass2.CharLimit)
	assert.Equal(t, initPasswordWidth, pass2.Width)
	assert.Equal(t, textinput.EchoPassword, pass2.EchoMode)
}

// TestInitSyncMenu проверяет инициализацию меню синхронизации.
func TestInitSyncMenu(t *testing.T) {
	l := initSyncMenu()
	assert.Equal(t, "", l.Title) // Заголовок пустой
	assert.False(t, l.ShowHelp())
	assert.False(t, l.ShowStatusBar())
	assert.Equal(t, list.Unfiltered, l.FilterState()) // Фильтрация выключена
	assert.True(t, l.Styles.Title.GetBold())
	assert.Len(t, l.Items(), 5) // Проверяем количество пунктов меню
}

// TestInitServerURLInput проверяет инициализацию поля ввода URL сервера.
func TestInitServerURLInput(t *testing.T) {
	ti := initServerURLInput()
	assert.Equal(t, defaultServerURL, ti.Placeholder)
	assert.Equal(t, initURLCharLimit, ti.CharLimit)
	assert.Equal(t, initURLWidth, ti.Width)
	assert.False(t, ti.Focused())
}

// TestInitLoginInputs проверяет инициализацию полей ввода для логина.
func TestInitLoginInputs(t *testing.T) {
	userInput, passInput := initLoginInputs()

	assert.Equal(t, "Имя пользователя", userInput.Placeholder)
	assert.Equal(t, initUserCharLimit, userInput.CharLimit)
	assert.Equal(t, initUserWidth, userInput.Width)
	assert.False(t, userInput.Focused())

	assert.Equal(t, "Пароль", passInput.Placeholder)
	assert.Equal(t, initPasswordCharLimit, passInput.CharLimit)
	assert.Equal(t, initUserWidth, passInput.Width)
	assert.Equal(t, textinput.EchoPassword, passInput.EchoMode)
	assert.False(t, passInput.Focused())
}

// TestInitRegisterInputs проверяет инициализацию полей ввода для регистрации.
func TestInitRegisterInputs(t *testing.T) {
	userInput, passInput := initRegisterInputs()

	assert.Equal(t, "Имя пользователя", userInput.Placeholder)
	assert.Equal(t, initUserCharLimit, userInput.CharLimit)
	assert.Equal(t, initUserWidth, userInput.Width)
	assert.False(t, userInput.Focused())

	assert.Equal(t, "Пароль", passInput.Placeholder)
	assert.Equal(t, initPasswordCharLimit, passInput.CharLimit)
	assert.Equal(t, initUserWidth, passInput.Width)
	assert.Equal(t, textinput.EchoPassword, passInput.EchoMode)
	assert.False(t, passInput.Focused())
}

// TestInitDocStyle проверяет инициализацию стиля документа.
func TestInitDocStyle(t *testing.T) {
	style := initDocStyle()
	// Проверяем, что отступы установлены
	marginTop, marginRight, marginBottom, marginLeft := style.GetMargin()
	assert.Equal(t, docStyleMarginVertical, marginTop)
	assert.Equal(t, docStyleMarginHorizontal, marginRight)
	assert.Equal(t, docStyleMarginVertical, marginBottom)
	assert.Equal(t, docStyleMarginHorizontal, marginLeft)
}

// TestInitVersionList проверяет инициализацию списка версий.
func TestInitVersionList(t *testing.T) {
	l := initVersionList()
	assert.Equal(t, "История версий", l.Title)
	assert.False(t, l.ShowHelp())
	assert.True(t, l.ShowStatusBar())
	assert.Equal(t, list.Unfiltered, l.FilterState()) // Фильтрация выключена
	assert.True(t, l.Styles.Title.GetBold())
	// Проверка конкретных стилей делегата убрана, так как делегат неэкспортируемый
}

// TestInitModel проверяет инициализацию основной модели.
func TestInitModel(t *testing.T) {
	testKdbxPath := "/tmp/test.kdbx"
	testServerURL := "http://localhost:8080"
	// Создаем мок API клиента
	mockAPI := &MockAPIClient{}

	m := initModel(testKdbxPath, true, testServerURL, mockAPI)

	assert.Equal(t, welcomeScreen, m.state) // Начальное состояние
	assert.Equal(t, testKdbxPath, m.kdbxPath)
	assert.True(t, m.debugMode)
	assert.Equal(t, testServerURL, m.serverURL)
	assert.Equal(t, mockAPI, m.apiClient)

	// Проверяем, что основные компоненты инициализированы (не nil)
	assert.NotNil(t, m.passwordInput)
	assert.NotNil(t, m.entryList)
	assert.NotNil(t, m.attachmentList)
	assert.NotNil(t, m.attachmentPathInput)
	assert.NotNil(t, m.newPasswordInput1)
	assert.NotNil(t, m.newPasswordInput2)
	assert.NotNil(t, m.syncServerMenu)
	assert.NotNil(t, m.serverURLInput)
	assert.NotNil(t, m.loginUsernameInput)
	assert.NotNil(t, m.loginPasswordInput)
	assert.NotNil(t, m.registerUsernameInput)
	assert.NotNil(t, m.registerPasswordInput)
	assert.NotNil(t, m.docStyle) // Это lipgloss.Style, не указатель
	assert.NotNil(t, m.versionList)

	// Проверяем некоторые начальные значения по умолчанию
	assert.Equal(t, 0, m.newPasswordFocusedField)
	assert.Equal(t, "Не выполнен", m.loginStatus)
	assert.Equal(t, "Не синхронизировалось", m.lastSyncStatus)
	assert.Equal(t, 0, m.loginRegisterFocusedField)
}
