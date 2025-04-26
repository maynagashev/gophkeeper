package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/stretchr/testify/assert"
	"github.com/tobischo/gokeepasslib/v3"
)

// TestPrepareAddScreen проверяет инициализацию полей для экрана добавления.
func TestPrepareAddScreen(t *testing.T) {
	// Создаем модель с минимальным состоянием для вызова prepareAddScreen
	m := &model{}

	m.prepareAddScreen()

	// Проверяем количество созданных полей
	assert.Len(t, m.editInputs, numEditableFields)

	// Проверяем, что первое поле (Title) в фокусе
	assert.Equal(t, editableFieldTitle, m.focusedField)
	assert.True(t, m.editInputs[editableFieldTitle].Focused(), "Поле Title должно быть в фокусе")

	// Проверяем плейсхолдеры и режим Echo
	assert.Equal(t, fieldNameTitle, m.editInputs[editableFieldTitle].Placeholder)
	assert.Equal(t, fieldNameUserName, m.editInputs[editableFieldUserName].Placeholder)
	assert.Equal(t, fieldNamePassword, m.editInputs[editableFieldPassword].Placeholder)
	assert.Equal(t, textinput.EchoPassword, m.editInputs[editableFieldPassword].EchoMode)
	assert.Equal(t, fieldNameURL, m.editInputs[editableFieldURL].Placeholder)
	assert.Equal(t, fieldNameNotes, m.editInputs[editableFieldNotes].Placeholder)
	assert.Equal(t, fieldNameCardNumber, m.editInputs[editableFieldCardNumber].Placeholder)
	assert.Equal(t, fieldNameCardHolderName, m.editInputs[editableFieldCardHolderName].Placeholder)
	assert.Equal(t, fieldNameExpiryDate, m.editInputs[editableFieldExpiryDate].Placeholder)
	assert.Equal(t, fieldNameCVV, m.editInputs[editableFieldCVV].Placeholder)
	assert.Equal(t, textinput.EchoPassword, m.editInputs[editableFieldCVV].EchoMode)
	assert.Equal(t, fieldNamePIN, m.editInputs[editableFieldPIN].Placeholder)
	assert.Equal(t, textinput.EchoPassword, m.editInputs[editableFieldPIN].EchoMode)

	// Проверяем, что временные вложения сброшены
	assert.Nil(t, m.newEntryAttachments)
}

// TestCreateEntryFromInputs проверяет создание записи из полей ввода.
func TestCreateEntryFromInputs(t *testing.T) {
	// Создаем тестовые поля ввода
	inputs := make([]textinput.Model, numEditableFields)
	inputs[editableFieldTitle] = textinput.New()
	inputs[editableFieldTitle].Placeholder = fieldNameTitle
	inputs[editableFieldTitle].SetValue("Test Entry")

	inputs[editableFieldUserName] = textinput.New()
	inputs[editableFieldUserName].Placeholder = fieldNameUserName
	inputs[editableFieldUserName].SetValue("testuser")

	inputs[editableFieldPassword] = textinput.New()
	inputs[editableFieldPassword].Placeholder = fieldNamePassword
	inputs[editableFieldPassword].SetValue("secret")

	inputs[editableFieldNotes] = textinput.New()
	inputs[editableFieldNotes].Placeholder = fieldNameNotes
	inputs[editableFieldNotes].SetValue("Some notes")

	// Пустое поле URL для проверки, что оно не добавляется
	inputs[editableFieldURL] = textinput.New()
	inputs[editableFieldURL].Placeholder = fieldNameURL
	inputs[editableFieldURL].SetValue("")

	// Остальные поля не инициализируем значениями
	for i := editableFieldCardNumber; i < numEditableFields; i++ {
		inputs[i] = textinput.New()
		inputs[i].Placeholder = "placeholder" // Важно, чтобы плейсхолдер был
	}

	// Создаем тестовые вложения
	attachments := []struct {
		Name    string
		Content []byte
	}{
		{Name: "file1.txt", Content: []byte("content1")},
		{Name: "image.png", Content: []byte("content2")},
	}

	// Создаем минимальную базу данных для AddBinary
	db := gokeepasslib.NewDatabase(gokeepasslib.WithDatabaseKDBXVersion4())
	db.Content.Root = &gokeepasslib.RootData{
		Groups: []gokeepasslib.Group{gokeepasslib.NewGroup()},
	}

	// Вызываем тестируемую функцию
	entry := createEntryFromInputs(db, inputs, attachments)

	// Проверяем значения полей
	assert.Equal(t, "Test Entry", entry.GetTitle())
	assert.Equal(t, "testuser", entry.GetContent(fieldNameUserName))
	assert.Equal(t, "secret", entry.GetPassword()) // GetPassword должен работать для Protected поля
	assert.Equal(t, "Some notes", entry.GetContent(fieldNameNotes))
	// Проверяем, что пустое поле URL не было добавлено
	assert.Nil(t, entry.Get(fieldNameURL))

	// Проверяем, что поле Password защищено
	passwordValue := entry.Get(fieldNamePassword)
	assert.NotNil(t, passwordValue)
	assert.True(t, passwordValue.Value.Protected.Bool, "Поле Password должно быть Protected")

	// Проверяем наличие и имена вложений
	assert.Len(t, entry.Binaries, 2, "Должно быть 2 вложения")
	assert.Equal(t, "file1.txt", entry.Binaries[0].Name)
	assert.Equal(t, "image.png", entry.Binaries[1].Name)

	// Проверяем, что бинарные данные добавлены в базу
	assert.Len(t, db.Content.InnerHeader.Binaries, 2, "Бинарные данные должны быть добавлены в InnerHeader")
	bin1 := db.FindBinary(entry.Binaries[0].Value.ID)
	bin2 := db.FindBinary(entry.Binaries[1].Value.ID)
	assert.NotNil(t, bin1)
	assert.NotNil(t, bin2)
	assert.Equal(t, []byte("content1"), bin1.Content)
	assert.Equal(t, []byte("content2"), bin2.Content)

	// Проверяем временные метки (хотя бы что они не нулевые)
	assert.NotNil(t, entry.Times.CreationTime)
	assert.NotNil(t, entry.Times.LastModificationTime)
	assert.NotNil(t, entry.Times.LastAccessTime)
}
