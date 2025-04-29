package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

// TestUpdateEntryAddScreen проверяет обновление экрана добавления записи.
func TestUpdateEntryAddScreen(t *testing.T) {
	suite := NewScreenTestSuite()
	suite.WithState(entryAddScreen)
	suite.Model.prepareAddScreen() // Вызываем вручную, т.к. стандартный initModel не вызывается для AddScreen

	// Проверяем начальный фокус на Title
	assert.Equal(t, editableFieldTitle, suite.Model.focusedField)
	assert.True(t, suite.Model.editInputs[editableFieldTitle].Focused())

	// 1. Навигация Tab/Down
	newModel, _ := suite.SimulateKeyPress(tea.KeyTab)
	suite.Model = toModel(t, newModel)
	assert.Equal(t, editableFieldUserName, suite.Model.focusedField, "Фокус должен перейти на UserName")
	assert.True(t, suite.Model.editInputs[editableFieldUserName].Focused())
	assert.False(t, suite.Model.editInputs[editableFieldTitle].Focused())

	newModel, _ = suite.SimulateKeyPress(tea.KeyDown)
	suite.Model = toModel(t, newModel)
	assert.Equal(t, editableFieldPassword, suite.Model.focusedField, "Фокус должен перейти на Password")
	assert.True(t, suite.Model.editInputs[editableFieldPassword].Focused())
	assert.False(t, suite.Model.editInputs[editableFieldUserName].Focused())

	// 2. Навигация Shift+Tab/Up
	newModel, _ = suite.SimulateKeyPress(tea.KeyShiftTab)
	suite.Model = toModel(t, newModel)
	assert.Equal(t, editableFieldUserName, suite.Model.focusedField, "Фокус должен вернуться на UserName")
	assert.True(t, suite.Model.editInputs[editableFieldUserName].Focused())
	assert.False(t, suite.Model.editInputs[editableFieldPassword].Focused())

	newModel, _ = suite.SimulateKeyPress(tea.KeyUp)
	suite.Model = toModel(t, newModel)
	assert.Equal(t, editableFieldTitle, suite.Model.focusedField, "Фокус должен вернуться на Title")
	assert.True(t, suite.Model.editInputs[editableFieldTitle].Focused())
	assert.False(t, suite.Model.editInputs[editableFieldUserName].Focused())

	// 3. Ввод текста в поле Title
	newModel, _ = suite.SimulateKeyRune('T')
	suite.Model = toModel(t, newModel)
	newModel, _ = suite.SimulateKeyRune('e')
	suite.Model = toModel(t, newModel)
	newModel, _ = suite.SimulateKeyRune('s')
	suite.Model = toModel(t, newModel)
	newModel, _ = suite.SimulateKeyRune('t')
	suite.Model = toModel(t, newModel)
	assert.Equal(t, "Test", suite.Model.editInputs[editableFieldTitle].Value())

	// 4. Нажатие Esc (отмена)
	newModel, _ = suite.SimulateKeyPress(tea.KeyEsc)
	suite.Model = toModel(t, newModel)
	suite.AssertState(t, entryListScreen)
	assert.Nil(t, suite.Model.editInputs, "Поля ввода должны быть очищены")
	assert.Nil(t, suite.Model.newEntryAttachments, "Временные вложения должны быть очищены")

	// 5. Нажатие Enter (добавление) - сначала снова перейдем на экран
	suite = NewScreenTestSuite() // Пересоздаем, чтобы сбросить состояние
	suite.WithState(entryAddScreen)
	suite.Model.prepareAddScreen()
	// Заполняем обязательное поле Title
	suite.Model.editInputs[editableFieldTitle].SetValue("New Entry Title")
	newModel, _ = suite.SimulateKeyPress(tea.KeyEnter)
	suite.Model = toModel(t, newModel)
	suite.AssertState(t, entryListScreen)
	assert.Nil(t, suite.Model.editInputs, "Поля ввода должны быть очищены")
	assert.Len(t, suite.Model.entryList.Items(), 1, "В списке должна появиться одна запись")
	// Проверяем тип элемента и получаем его
	rawItem := suite.Model.entryList.Items()[0]
	item, ok := rawItem.(entryItem)
	assert.True(t, ok, "Элемент списка должен быть типа entryItem")
	assert.Equal(t, "New Entry Title", item.entry.GetTitle())

	// 6. Переход к добавлению вложения (Ctrl+O)
	suite = NewScreenTestSuite() // Пересоздаем
	suite.WithState(entryAddScreen)
	suite.Model.prepareAddScreen()
	newModel, _ = suite.SimulateKeyPress(tea.KeyCtrlO)
	suite.Model = toModel(t, newModel)
	suite.AssertState(t, attachmentPathInputScreen)
	assert.Equal(t, entryAddScreen, suite.Model.previousScreenState, "Предыдущее состояние должно быть сохранено")
	assert.True(t, suite.Model.attachmentPathInput.Focused(), "Поле ввода пути должно быть в фокусе")
}

// TestViewEntryAddScreen проверяет функцию viewEntryAddScreen.
func TestViewEntryAddScreen(t *testing.T) {
	t.Run("Без вложений", func(t *testing.T) {
		// Создаем модель и подготавливаем экран добавления
		m := &model{
			state: entryAddScreen,
		}
		m.prepareAddScreen() // Инициализирует editInputs и editingEntry

		// Получаем отрисованный вид
		view := m.viewEntryAddScreen()

		// Проверяем основные элементы
		assert.Contains(t, view, "Добавление новой записи", "Должен отображаться заголовок")
		assert.Contains(t, view, "> Title: > Title", "Поле Title должно быть с фокусом")
		assert.Contains(t, view, "  UserName: > UserName", "Поле UserName должно быть без фокуса")

		// Проверяем отображение вложений
		assert.Contains(t, view, "--- Вложения для добавления ---", "Должен быть раздел вложений")
		assert.Contains(t, view, "(Нет вложений)", "Должно отображаться сообщение об отсутствии вложений")
	})

	t.Run("С вложениями", func(t *testing.T) {
		// Создаем модель и подготавливаем экран добавления
		m := &model{
			state: entryAddScreen,
		}
		m.prepareAddScreen()
		// Устанавливаем вложения ПОСЛЕ вызова prepareAddScreen
		m.newEntryAttachments = []struct {
			Name    string
			Content []byte
		}{
			{Name: "doc.txt", Content: []byte("тест")},
			{Name: "image.jpg", Content: make([]byte, 1024)},
		}

		// Получаем отрисованный вид
		view := m.viewEntryAddScreen()

		// Проверяем основные элементы
		assert.Contains(t, view, "Добавление новой записи", "Должен отображаться заголовок")
		assert.Contains(t, view, "> Title: > Title", "Поле Title должно быть с фокусом")

		// Проверяем отображение вложений
		assert.Contains(t, view, "--- Вложения для добавления ---", "Должен быть раздел вложений")
		assert.Contains(t, view, "[0] doc.txt (8 байт)", "Должно отображаться первое вложение с размером (кириллица)")
		assert.Contains(t, view, "[1] image.jpg (1024 байт)", "Должно отображаться второе вложение с размером")
		assert.NotContains(t, view, "(Нет вложений)", "Не должно быть сообщения об отсутствии вложений")
	})
}
