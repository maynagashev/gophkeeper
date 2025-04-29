package tui

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

// TestScreenState_String проверяет строковое представление screenState.
func TestScreenState_String(t *testing.T) {
	tests := []struct {
		state screenState
		want  string
	}{
		{welcomeScreen, "welcomeScreen"},
		{passwordInputScreen, "passwordInputScreen"},
		{newKdbxPasswordScreen, "newKdbxPasswordScreen"},
		{entryListScreen, "entryListScreen"},
		{entryDetailScreen, "entryDetailScreen"},
		{entryEditScreen, "entryEditScreen"},
		{entryAddScreen, "entryAddScreen"},
		{attachmentListDeleteScreen, "attachmentListDeleteScreen"},
		{attachmentPathInputScreen, "attachmentPathInputScreen"},
		{syncServerScreen, "syncServerScreen"},
		{serverURLInputScreen, "serverURLInputScreen"},
		{loginRegisterChoiceScreen, "loginRegisterChoiceScreen"},
		{loginScreen, "loginScreen"},
		{registerScreen, "registerScreen"},
		{versionListScreen, "versionListScreen"},
		{screenState(999), "unknownScreen(999)"}, // Неизвестное состояние
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

// --- Тесты для entryItem ---

// createTestEntry создает тестовую запись gokeepasslib.Entry.
func createTestEntry(title, username, url string, withAttachment bool) gokeepasslib.Entry {
	e := gokeepasslib.NewEntry()
	e.Values = append(e.Values,
		gokeepasslib.ValueData{Key: "Title", Value: gokeepasslib.V{Content: title, Protected: w.NewBoolWrapper(false)}},
		gokeepasslib.ValueData{Key: "UserName", Value: gokeepasslib.V{Content: username, Protected: w.NewBoolWrapper(false)}},
		gokeepasslib.ValueData{Key: "URL", Value: gokeepasslib.V{Content: url, Protected: w.NewBoolWrapper(false)}},
	)
	if withAttachment {
		// Используем конструктор NewBinaryReference.
		binaryRef := gokeepasslib.NewBinaryReference("test.txt", 0)
		e.Binaries = append(e.Binaries, binaryRef)
	}
	return e
}

// TestEntryItem_Title проверяет метод Title для entryItem.
func TestEntryItem_Title(t *testing.T) {
	entryWithTitle := createTestEntry("My Login", "user", "url", false)
	entryWithUsername := createTestEntry("", "user@example.com", "url", false)
	entryOnlyURL := createTestEntry("", "", "http://example.com", false)
	emptyEntry := createTestEntry("", "", "", false)
	uuidStr := hex.EncodeToString(emptyEntry.UUID[:])

	tests := []struct {
		name string
		item entryItem
		want string
	}{
		{"С Title", entryItem{entry: entryWithTitle}, "My Login"},
		{"Без Title, с Username", entryItem{entry: entryWithUsername}, "user@example.com"},
		{"Только URL (используется UUID)", entryItem{entry: entryOnlyURL}, hex.EncodeToString(entryOnlyURL.UUID[:])},
		{"Пустая запись (используется UUID)", entryItem{entry: emptyEntry}, uuidStr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.item.Title())
		})
	}
}

// TestEntryItem_Description проверяет метод Description для entryItem.
func TestEntryItem_Description(t *testing.T) {
	entryFull := createTestEntry("Title", "user", "url", true)
	entryUserOnly := createTestEntry("Title", "user", "", false)
	entryURLOnly := createTestEntry("Title", "", "url", false)
	entryEmpty := createTestEntry("Title", "", "", false)
	entryUserAttach := createTestEntry("Title", "user", "", true)
	entryURLAttach := createTestEntry("Title", "", "url", true)
	entryEmptyAttach := createTestEntry("Title", "", "", true)

	tests := []struct {
		name string
		item entryItem
		want string
	}{
		{"Полная", entryItem{entry: entryFull}, "User: user | URL: url [A:1]"},
		{"Только User", entryItem{entry: entryUserOnly}, "User: user"},
		{"Только URL", entryItem{entry: entryURLOnly}, "URL: url"},
		{"Пустая", entryItem{entry: entryEmpty}, ""},
		{"User + Attach", entryItem{entry: entryUserAttach}, "User: user [A:1]"},
		{"URL + Attach", entryItem{entry: entryURLAttach}, "URL: url [A:1]"},
		{"Пустая + Attach", entryItem{entry: entryEmptyAttach}, "[A:1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.item.Description())
		})
	}
}

// TestEntryItem_FilterValue проверяет метод FilterValue для entryItem.
func TestEntryItem_FilterValue(t *testing.T) {
	// FilterValue просто возвращает Title, поэтому тесты аналогичны TestEntryItem_Title
	entryWithTitle := createTestEntry("My Login", "user", "url", false)
	entryWithUsername := createTestEntry("", "user@example.com", "url", false)
	emptyEntry := createTestEntry("", "", "", false)
	uuidStr := hex.EncodeToString(emptyEntry.UUID[:])

	tests := []struct {
		name string
		item entryItem
		want string
	}{
		{"С Title", entryItem{entry: entryWithTitle}, "My Login"},
		{"Без Title, с Username", entryItem{entry: entryWithUsername}, "user@example.com"},
		{"Пустая запись (используется UUID)", entryItem{entry: emptyEntry}, uuidStr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.item.FilterValue())
		})
	}
}
