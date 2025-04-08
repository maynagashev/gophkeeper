package tui

import (
	"encoding/hex"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/tobischo/gokeepasslib/v3"
)

// Состояния (экраны) приложения
type screenState int

const (
	welcomeScreen       screenState = iota // Приветственный экран
	passwordInputScreen                    // Экран ввода пароля
	entryListScreen                        // Экран списка записей
	entryDetailScreen                      // Экран деталей записи
	entryEditScreen                        // Экран редактирования записи
	entryAddScreen                         // Экран добавления новой записи
)

// Поля, доступные для редактирования
const (
	editableFieldTitle = iota
	editableFieldUserName
	editableFieldPassword
	editableFieldURL
	editableFieldNotes
	numEditableFields // Количество редактируемых полей

	fieldNamePassword = "Password"
)

// Константы для TUI
const (
	defaultListWidth    = 80 // Стандартная ширина терминала для списка
	defaultListHeight   = 24 // Стандартная высота терминала для списка
	passwordInputOffset = 4  // Отступ для поля ввода пароля

	keyEnter = "enter" // Клавиша Enter
	keyQuit  = "q"     // Клавиша выхода
	keyBack  = "b"     // Клавиша возврата
	keyEsc   = "esc"   // Клавиша Escape
	keyEdit  = "e"     // Клавиша редактирования
	keyAdd   = "a"     // Клавиша добавления
)

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
	switch {
	case username != "" && url != "":
		return fmt.Sprintf("User: %s | URL: %s", username, url)
	case username != "":
		return fmt.Sprintf("User: %s", username)
	case url != "":
		return fmt.Sprintf("URL: %s", url)
	default:
		return ""
	}
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

// model представляет состояние TUI приложения
type model struct {
	state         screenState            // Текущее состояние (экран)
	passwordInput textinput.Model        // Поле ввода для пароля
	password      string                 // Сохраненный в памяти пароль от базы
	db            *gokeepasslib.Database // Объект открытой базы KDBX
	kdbxPath      string                 // Путь к KDBX файлу
	err           error                  // Последняя ошибка для отображения
	entryList     list.Model             // Компонент списка записей
	selectedEntry *entryItem             // Выбранная запись для детального просмотра

	// Поля для редактирования записи
	editingEntry *gokeepasslib.Entry // Копия записи, которую редактируем
	editInputs   []textinput.Model   // Поля ввода для редактирования
	focusedField int                 // Индекс активного поля ввода

	// Поля для добавления записи
	addInputs       []textinput.Model // Поля ввода для новой записи
	focusedFieldAdd int               // Индекс активного поля ввода

	savingStatus string // Статус операции сохранения файла
}
