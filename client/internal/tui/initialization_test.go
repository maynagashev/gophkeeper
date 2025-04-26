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
