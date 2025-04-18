package tui

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/gofrs/flock"
	"github.com/maynagashev/gophkeeper/client/internal/api"
	"github.com/tobischo/gokeepasslib/v3"
)

// Состояния (экраны) приложения.
type screenState int

const (
	welcomeScreen              screenState = iota // Приветственный экран
	passwordInputScreen                           // Экран ввода пароля
	entryListScreen                               // Экран списка записей
	entryDetailScreen                             // Экран деталей записи
	entryEditScreen                               // Экран редактирования записи
	entryAddScreen                                // Экран добавления новой записи
	attachmentListDeleteScreen                    // Экран выбора вложения для удаления
	attachmentPathInputScreen                     // Экран ввода пути к добавляемому вложению
	newKdbxPasswordScreen                         // Экран ввода пароля для нового KDBX файла
	// Новые состояния для синхронизации и сервера.
	syncServerScreen          screenState = iota // Экран "Синхронизация и Сервер"
	serverURLInputScreen                         // Экран ввода URL сервера
	loginRegisterChoiceScreen                    // Экран выбора "Войти или Зарегистрироваться?"
	loginScreen                                  // Экран ввода данных для входа
	registerScreen                               // Экран ввода данных для регистрации
)

// Поля, доступные для редактирования.
const (
	// Стандартные поля.
	editableFieldTitle = iota
	editableFieldUserName
	editableFieldPassword
	editableFieldURL
	editableFieldNotes
	// Поля карты.
	editableFieldCardNumber
	editableFieldCardHolderName
	editableFieldExpiryDate
	editableFieldCVV
	editableFieldPIN
	// Конец списка.
	numEditableFields // Общее количество редактируемых полей
)

// Имена полей (используются как плейсхолдеры и ключи в KDBX).
const (
	fieldNameTitle          = "Title"
	fieldNameUserName       = "UserName"
	fieldNamePassword       = "Password"
	fieldNameURL            = "URL"
	fieldNameNotes          = "Notes"
	fieldNameCardNumber     = "CardNumber"
	fieldNameCardHolderName = "CardHolderName"
	fieldNameExpiryDate     = "ExpiryDate"
	fieldNameCVV            = "CVV"
	fieldNamePIN            = "PIN"
)

// Константы для TUI.
const (
	defaultListWidth    = 80 // Стандартная ширина терминала для списка
	defaultListHeight   = 24 // Стандартная высота терминала для списка
	passwordInputOffset = 4  // Отступ для поля ввода пароля

	keyEnter    = "enter" // Клавиша Enter
	keyQuit     = "q"     // Клавиша выхода
	keyBack     = "b"     // Клавиша возврата
	keyEsc      = "esc"   // Клавиша Escape
	keyEdit     = "e"     // Клавиша редактирования
	keyAdd      = "a"     // Клавиша добавления
	keyTab      = "tab"
	keyShiftTab = "shift+tab"
	keyUp       = "up"
	keyDown     = "down"
)

const numNewPasswordFields = 2 // Количество полей на экране создания пароля

// entryItem представляет элемент списка записей.
// Реализует интерфейс list.Item.
type entryItem struct {
	entry gokeepasslib.Entry
}

func (i entryItem) Title() string {
	// Пытаемся получить значение поля "Title"
	title := i.entry.GetTitle()
	if title == "" {
		// Если Title пустой, используем Username
		title = i.entry.GetContent("UserName")
	}
	if title == "" {
		// Если и Username пустой, используем UUID
		title = hex.EncodeToString(i.entry.UUID[:])
	}
	return title
}

func (i entryItem) Description() string {
	// В описании можно показать Username или URL
	username := i.entry.GetContent("UserName")
	url := i.entry.GetContent("URL")
	var desc string // Объявляем переменную без инициализации
	switch {
	case username != "" && url != "":
		desc = fmt.Sprintf("User: %s | URL: %s", username, url)
	case username != "":
		desc = fmt.Sprintf("User: %s", username)
	case url != "":
		desc = fmt.Sprintf("URL: %s", url)
	default:
		desc = ""
	}

	// Добавляем индикатор наличия вложений
	if len(i.entry.Binaries) > 0 {
		if desc != "" {
			desc += " " // Добавляем пробел, если описание уже есть
		}
		desc += fmt.Sprintf("[A:%d]", len(i.entry.Binaries)) // Показываем количество вложений
	}

	return desc
}

func (i entryItem) FilterValue() string { return i.Title() }

// Структура для сообщения об успешном открытии файла.
type dbOpenedMsg struct {
	db *gokeepasslib.Database
}

// Структура для сообщения об ошибке.
type errMsg struct {
	err error
}

// Структуры для сообщений о сохранении.
type dbSavedMsg struct{}

type dbSaveErrorMsg struct {
	err error
}

// model представляет состояние TUI приложения.
type model struct {
	state               screenState
	kdbxPath            string // Путь к файлу KDBX
	password            string // Сохраненный мастер-пароль
	db                  *gokeepasslib.Database
	fileLock            *flock.Flock        // Объект блокировки файла
	lockAcquired        bool                // Флаг: удалось ли получить блокировку
	readOnlyMode        bool                // Флаг: приложение в режиме только для чтения
	passwordInput       textinput.Model     // Поле ввода пароля для существующего файла
	entryList           list.Model          // Список записей
	selectedEntry       *entryItem          // Выбранная запись для просмотра/редактирования
	detailScroll        int                 //nolint:unused // Задел на будущее для скроллинга деталей
	editInputs          []textinput.Model   // Поля ввода для редактирования
	focusedField        int                 // Индекс активного поля при редактировании/добавлении
	editingEntry        *gokeepasslib.Entry // Копия записи при редактировании/добавлении
	attachmentList      list.Model          // Список вложений для выбора/удаления
	attachmentPathInput textinput.Model     // Поле ввода пути для добавления вложения
	attachmentError     error               // Ошибка при добавлении вложения
	previousScreenState screenState         // Предыдущее состояние (для возврата)
	savingStatus        string              // Статус сохранения (отображается внизу)
	statusTimer         *time.Timer         // Таймер для очистки статуса сохранения
	width               int                 //nolint:unused // Потенциально для адаптивной верстки
	height              int                 //nolint:unused // Потенциально для адаптивной верстки
	listMutex           sync.Mutex          //nolint:unused // Задел на будущее для синхронизации

	// Поля для создания нового KDBX
	newPasswordInput1       textinput.Model // Первое поле ввода нового пароля
	newPasswordInput2       textinput.Model // Второе поле для подтверждения пароля
	newPasswordFocusedField int             // 0 или 1, указывает на активное поле
	confirmPasswordError    string          // Сообщение об ошибке несовпадения паролей

	// Поле для временного хранения вложений при добавлении
	newEntryAttachments []struct {
		Name    string
		Content []byte
	}
	// Поля для подтверждения удаления вложения
	confirmationPrompt string          // Текст запроса подтверждения
	itemToDelete       *attachmentItem // Вложение, выбранное для удаления
	err                error           // Последняя ошибка для отображения

	// -- Новые поля для интеграции с сервером --
	apiClient                 api.Client      // Клиент для взаимодействия с API
	serverURL                 string          // URL сервера
	authToken                 string          // JWT токен аутентификации
	loginStatus               string          // Статус входа ("Не выполнен", "Выполнен как...")
	lastSyncStatus            string          // Статус последней синхронизации
	syncServerMenu            list.Model      // Меню действий на экране синхронизации
	serverURLInput            textinput.Model // Поле для ввода URL сервера
	loginUsernameInput        textinput.Model // Поле для ввода имени пользователя при входе
	loginPasswordInput        textinput.Model // Поле для ввода пароля при входе
	registerUsernameInput     textinput.Model // Поле для ввода имени пользователя при регистрации
	registerPasswordInput     textinput.Model // Поле для ввода пароля при регистрации
	loginRegisterFocusedField int             // Индекс активного поля на экранах входа/регистрации/URL
	docStyle                  lipgloss.Style  // Общий стиль для обрамления View
}

// Сообщение для очистки статуса.
type clearStatusMsg struct{}
