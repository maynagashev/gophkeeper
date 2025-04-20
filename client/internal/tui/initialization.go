package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/maynagashev/gophkeeper/models"
)

// Константы, используемые при инициализации.
const (
	initPasswordCharLimit = 156
	initPasswordWidth     = 20
	initPathCharLimit     = 4096
	initURLCharLimit      = 1024
	initURLWidth          = 50
	initUserCharLimit     = 128
	initUserWidth         = 30
)

// Константы для работы с данными.
const (
	bytesPerKilobyte = 1024.0 // Добавлено для mnd
)

// versionItem представляет элемент в списке версий.
type versionItem struct {
	version   models.VaultVersion
	isCurrent bool // Флаг текущей активной версии
}

func (i versionItem) Title() string {
	// Формат: ID (+ "Текущая" если активная версия)
	title := fmt.Sprintf("Версия #%d", i.version.ID)
	if i.isCurrent {
		title += " (Текущая)"
	}
	return title
}

func (i versionItem) Description() string {
	description := ""

	// Добавляем время создания, если есть
	if i.version.ContentModifiedAt != nil {
		timeStr := i.version.ContentModifiedAt.Format(time.RFC3339)
		description += fmt.Sprintf("Изменена: %s", timeStr)
	}

	// Добавляем размер, если есть
	if i.version.SizeBytes != nil {
		if description != "" {
			description += " | "
		}
		sizeKB := float64(*i.version.SizeBytes) / bytesPerKilobyte // Используем константу
		description += fmt.Sprintf("Размер: %.2f KB", sizeKB)
	}

	// Если ничего нет, просто выводим ID
	if description == "" {
		description = fmt.Sprintf("ID: %d", i.version.ID)
	}

	return description
}

func (i versionItem) FilterValue() string {
	// Фильтрация не используется для этого списка, но интерфейс требует реализации
	return i.Title()
}

// initPasswordInput инициализирует основное поле ввода пароля.
func initPasswordInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Мастер-пароль"
	ti.Focus()
	ti.CharLimit = initPasswordCharLimit
	ti.Width = initPasswordWidth
	ti.EchoMode = textinput.EchoPassword
	return ti
}

// initEntryList инициализирует основной компонент списка для записей.
func initEntryList() list.Model {
	delegate := list.NewDefaultDelegate()
	// Настраиваем цвета для лучшей видимости
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("235"))
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(lipgloss.Color("245")).
		Background(lipgloss.Color("235"))
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("212")).
		Background(lipgloss.Color("237")).
		BorderLeftForeground(lipgloss.Color("212"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("237")).
		BorderLeftForeground(lipgloss.Color("212"))

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Записи"
	l.SetShowHelp(false) // Мы переопределяем справку
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = list.DefaultStyles().Title.Bold(true)
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle
	return l
}

// initAttachmentDeleteList инициализирует список для удаления вложений.
func initAttachmentDeleteList() list.Model {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Выберите вложение для удаления"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = list.DefaultStyles().Title.Bold(true)
	return l
}

// initAttachmentPathInput инициализирует поле ввода пути к вложению.
func initAttachmentPathInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "/path/to/your/file"
	ti.CharLimit = initPathCharLimit
	ti.Width = defaultListWidth - passwordInputOffset // Предполагается, что константы доступны
	return ti
}

// initNewKdbxPasswordInputs инициализирует поля для создания нового пароля KDBX.
func initNewKdbxPasswordInputs() (textinput.Model, textinput.Model) {
	newPass1 := textinput.New()
	newPass1.Placeholder = "Новый мастер-пароль"
	newPass1.Focus()
	newPass1.CharLimit = initPasswordCharLimit
	newPass1.Width = initPasswordWidth
	newPass1.EchoMode = textinput.EchoPassword

	newPass2 := textinput.New()
	newPass2.Placeholder = "Подтвердите пароль"
	newPass2.CharLimit = initPasswordCharLimit
	newPass2.Width = initPasswordWidth
	newPass2.EchoMode = textinput.EchoPassword
	return newPass1, newPass2
}

// initSyncMenu инициализирует компонент списка для меню синхронизации/сервера.
func initSyncMenu() list.Model {
	syncMenuDelegate := list.NewDefaultDelegate()
	// Устанавливаем ненулевые размеры по умолчанию для диагностики
	defaultSyncMenuWidth := 40
	defaultSyncMenuHeight := 16
	syncMenuList := list.New([]list.Item{
		syncMenuItem{title: "Настроить URL сервера", id: "configure_url"},
		syncMenuItem{title: "Войти / Зарегистрироваться", id: "login_register"},
		syncMenuItem{title: "Синхронизировать сейчас", id: "sync_now"},
		syncMenuItem{title: "Просмотреть версии", id: "view_versions"},
		syncMenuItem{title: "Выйти на сервере", id: "logout"},
	}, syncMenuDelegate, defaultSyncMenuWidth, defaultSyncMenuHeight)
	syncMenuList.Title = ""
	syncMenuList.SetShowHelp(false)
	syncMenuList.SetShowStatusBar(false)
	syncMenuList.SetFilteringEnabled(false)
	syncMenuList.Styles.Title = list.DefaultStyles().Title.Bold(true)
	return syncMenuList
}

// initServerURLInput инициализирует поле ввода URL сервера.
func initServerURLInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = defaultServerURL // Предполагается, что константы доступны
	ti.CharLimit = initURLCharLimit
	ti.Width = initURLWidth
	return ti
}

// initLoginInputs инициализирует поля для экрана входа.
func initLoginInputs() (textinput.Model, textinput.Model) {
	loginUserInput := textinput.New()
	loginUserInput.Placeholder = "Имя пользователя"
	loginUserInput.CharLimit = initUserCharLimit
	loginUserInput.Width = initUserWidth

	loginPassInput := textinput.New()
	loginPassInput.Placeholder = "Пароль"
	loginPassInput.CharLimit = initPasswordCharLimit
	loginPassInput.Width = initUserWidth
	loginPassInput.EchoMode = textinput.EchoPassword
	return loginUserInput, loginPassInput
}

// initRegisterInputs инициализирует поля для экрана регистрации.
func initRegisterInputs() (textinput.Model, textinput.Model) {
	regUserInput := textinput.New()
	regUserInput.Placeholder = "Имя пользователя"
	regUserInput.CharLimit = initUserCharLimit
	regUserInput.Width = initUserWidth

	regPassInput := textinput.New()
	regPassInput.Placeholder = "Пароль"
	regPassInput.CharLimit = initPasswordCharLimit
	regPassInput.Width = initUserWidth
	regPassInput.EchoMode = textinput.EchoPassword
	return regUserInput, regPassInput
}

// initDocStyle инициализирует основной стиль документа.
func initDocStyle() lipgloss.Style {
	// Предполагается, что константы доступны
	return lipgloss.NewStyle().Margin(docStyleMarginVertical, docStyleMarginHorizontal)
}

// initVersionList инициализирует список для отображения версий хранилища.
func initVersionList() list.Model {
	versionDelegate := list.NewDefaultDelegate()

	// Настройка стилей для выделения текущей версии
	versionDelegate.Styles.SelectedTitle = versionDelegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("212")).
		Background(lipgloss.Color("237")).
		BorderLeftForeground(lipgloss.Color("212"))

	versionDelegate.Styles.SelectedDesc = versionDelegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("237")).
		BorderLeftForeground(lipgloss.Color("212"))

	versionList := list.New([]list.Item{}, versionDelegate, defaultListWidth, defaultListHeight)
	versionList.Title = "История версий"
	versionList.SetShowHelp(false)
	versionList.SetShowStatusBar(true)
	versionList.SetFilteringEnabled(false)
	versionList.Styles.Title = list.DefaultStyles().Title.Bold(true)

	return versionList
}

// initModel создает начальное состояние модели.
func initModel(kdbxPath string, debugMode bool) model {
	passwordInput := initPasswordInput()
	entryList := initEntryList()
	attachmentDelList := initAttachmentDeleteList()
	pathInput := initAttachmentPathInput()
	newPass1, newPass2 := initNewKdbxPasswordInputs()
	syncMenuList := initSyncMenu()
	serverURLInput := initServerURLInput()
	loginUserInput, loginPassInput := initLoginInputs()
	regUserInput, regPassInput := initRegisterInputs()
	docStyle := initDocStyle()
	versionList := initVersionList()

	return model{
		state:                     welcomeScreen,
		passwordInput:             passwordInput,
		kdbxPath:                  kdbxPath,
		entryList:                 entryList,
		attachmentList:            attachmentDelList,
		attachmentPathInput:       pathInput,
		newPasswordInput1:         newPass1,
		newPasswordInput2:         newPass2,
		newPasswordFocusedField:   0,
		loginStatus:               "Не выполнен",
		lastSyncStatus:            "Не синхронизировалось",
		syncServerMenu:            syncMenuList,
		serverURLInput:            serverURLInput,
		loginUsernameInput:        loginUserInput,
		loginPasswordInput:        loginPassInput,
		registerUsernameInput:     regUserInput,
		registerPasswordInput:     regPassInput,
		loginRegisterFocusedField: 0,
		docStyle:                  docStyle,
		debugMode:                 debugMode,
		versionList:               versionList,
	}
}
