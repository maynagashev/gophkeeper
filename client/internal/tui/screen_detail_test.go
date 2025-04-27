package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

// Хелпер createTestEntry объявлен в model_test.go

func TestMaskCardNumber(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Short number", "123456", "********"},
		{"Exact min length", "123456789", "1234********6789"},
		{"Standard length", "1234567890123456", "1234********3456"},
		{"Long number", "12345678901234567890", "1234********7890"},
		{"Empty string", "", "********"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := maskCardNumber(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestRenderStandardFields(t *testing.T) {
	// Используем хелпер для базовых полей
	testEntry := createTestEntry("Test Title", "TestUser", "http://example.com", false)
	// Добавляем остальные поля вручную
	testEntry.Values = append(testEntry.Values,
		gokeepasslib.ValueData{
			Key:   fieldNamePassword,
			Value: gokeepasslib.V{Content: "SecretPass", Protected: w.NewBoolWrapper(true)},
		},
		gokeepasslib.ValueData{Key: fieldNameNotes, Value: gokeepasslib.V{Content: "Some notes"}},
		gokeepasslib.ValueData{Key: fieldNameCardNumber, Value: gokeepasslib.V{Content: "1234567890123456"}},
		gokeepasslib.ValueData{Key: fieldNameCardHolderName, Value: gokeepasslib.V{Content: "Test Holder"}},
		gokeepasslib.ValueData{Key: fieldNameExpiryDate, Value: gokeepasslib.V{Content: "12/25"}},
		gokeepasslib.ValueData{
			Key:   fieldNameCVV,
			Value: gokeepasslib.V{Content: "123", Protected: w.NewBoolWrapper(true)},
		},
		gokeepasslib.ValueData{
			Key:   fieldNamePIN,
			Value: gokeepasslib.V{Content: "4321", Protected: w.NewBoolWrapper(true)},
		},
		gokeepasslib.ValueData{Key: "CustomField1", Value: gokeepasslib.V{Content: "CustomValue1"}}, // Дополнительное поле
	)

	valuesMap := make(map[string]gokeepasslib.ValueData)
	for _, val := range testEntry.Values {
		valuesMap[val.Key] = val
	}

	expectedOrder := []string{
		"Title: Test Title\n",
		"UserName: TestUser\n",
		"Password: ********\n", // Маскируется
		"URL: http://example.com\n",
		"Notes: Some notes\n",
		"CardNumber: 1234********3456\n", // Маскируется
		"CardHolderName: Test Holder\n",
		"ExpiryDate: 12/25\n",
		"CVV: ********\n", // Маскируется
		"PIN: ********\n", // Маскируется
	}

	actual := renderStandardFields(testEntry, valuesMap)

	// Проверяем, что все стандартные поля присутствуют в ожидаемом порядке
	for _, expectedLine := range expectedOrder {
		assert.Contains(t, actual, expectedLine)
	}

	// Проверяем, что стандартные поля были удалены из map
	expectedRemainingKeys := []string{"CustomField1"}
	actualRemainingKeys := []string{}
	for k := range valuesMap {
		actualRemainingKeys = append(actualRemainingKeys, k)
	}
	assert.ElementsMatch(t, expectedRemainingKeys, actualRemainingKeys, "Standard fields were not removed from the map")

	// Проверяем случай с пустым map
	emptyMap := make(map[string]gokeepasslib.ValueData)
	assert.Equal(t,
		"",
		renderStandardFields(gokeepasslib.Entry{}, emptyMap),
		"Rendering empty map should return empty string",
	)
}

func TestRenderCustomFields(t *testing.T) {
	valuesMap := map[string]gokeepasslib.ValueData{
		"CustomField1": {Key: "CustomField1", Value: gokeepasslib.V{Content: "Value1"}},
		"AnotherField": {Key: "AnotherField", Value: gokeepasslib.V{Content: "Value2"}},
	}

	// Порядок не гарантирован, поэтому проверяем содержание
	actual := renderCustomFields(valuesMap)
	assert.Contains(t, actual, "\n--- Дополнительные поля ---\n")
	assert.Contains(t, actual, "CustomField1: Value1\n")
	assert.Contains(t, actual, "AnotherField: Value2\n")

	// Проверяем случай с пустым map
	emptyMap := make(map[string]gokeepasslib.ValueData)
	assert.Equal(t, "", renderCustomFields(emptyMap), "Rendering empty map should return empty string")
}

// TODO: Добавить тесты для getBinariesFromDB, renderAttachments, viewEntryDetailScreen, updateEntryDetailScreen

func TestViewEntryDetailScreen(t *testing.T) {
	// Создаем тестовую модель с выбранной записью
	testModel := &model{}

	// Создаем базу данных версии 4 для тестирования вложений
	db := gokeepasslib.NewDatabase(gokeepasslib.WithDatabaseKDBXVersion4())

	// Добавляем бинарные данные в InnerHeader
	db.Content.InnerHeader.Binaries = []gokeepasslib.Binary{
		{ID: 0, Content: []byte("test content")},
	}
	testModel.db = db

	// Создаем тестовую запись с различными полями и вложением
	entry := createTestEntry("Тестовая запись", "testuser", "http://example.com", true)

	// Добавляем стандартные и дополнительные поля
	entry.Values = append(entry.Values,
		gokeepasslib.ValueData{
			Key:   fieldNamePassword,
			Value: gokeepasslib.V{Content: "SecretPass", Protected: w.NewBoolWrapper(true)},
		},
		gokeepasslib.ValueData{Key: fieldNameNotes, Value: gokeepasslib.V{Content: "Тестовые заметки"}},
		gokeepasslib.ValueData{Key: fieldNameCardNumber, Value: gokeepasslib.V{Content: "1234567890123456"}},
		gokeepasslib.ValueData{Key: fieldNameCardHolderName, Value: gokeepasslib.V{Content: "Иван Иванов"}},
		gokeepasslib.ValueData{Key: fieldNameExpiryDate, Value: gokeepasslib.V{Content: "12/25"}},
		gokeepasslib.ValueData{
			Key:   fieldNameCVV,
			Value: gokeepasslib.V{Content: "123", Protected: w.NewBoolWrapper(true)},
		},
		gokeepasslib.ValueData{
			Key:   fieldNamePIN,
			Value: gokeepasslib.V{Content: "4321", Protected: w.NewBoolWrapper(true)},
		},
		gokeepasslib.ValueData{Key: "ДополнительноеПоле", Value: gokeepasslib.V{Content: "ДополнительноеЗначение"}},
	)

	// Устанавливаем запись в модель
	testModel.selectedEntry = &entryItem{entry: entry}

	// Получаем результат отрисовки
	result := testModel.viewEntryDetailScreen()

	// Проверяем наличие ожидаемых элементов в результате
	assert.Contains(t, result, "Детали записи: Тестовая запись")

	// Проверяем стандартные поля
	assert.Contains(t, result, "Title: Тестовая запись")
	assert.Contains(t, result, "UserName: testuser")
	assert.Contains(t, result, "Password: ********")
	assert.Contains(t, result, "URL: http://example.com")
	assert.Contains(t, result, "Notes: Тестовые заметки")
	assert.Contains(t, result, "CardNumber: 1234********3456")
	assert.Contains(t, result, "CardHolderName: Иван Иванов")
	assert.Contains(t, result, "ExpiryDate: 12/25")
	assert.Contains(t, result, "CVV: ********")
	assert.Contains(t, result, "PIN: ********")

	// Проверяем дополнительные поля
	assert.Contains(t, result, "--- Дополнительные поля ---")
	assert.Contains(t, result, "ДополнительноеПоле: ДополнительноеЗначение")

	// Проверяем вложения
	assert.Contains(t, result, "--- Вложения ---")
	assert.Contains(t, result, "- test.txt (12 байт)") // "test content" = 12 байт

	// Проверяем случай с отсутствующей записью
	testModel.selectedEntry = nil
	assert.Equal(t, "Ошибка: запись не выбрана.", testModel.viewEntryDetailScreen())
}

// TestUpdateEntryDetailScreen проверяет функцию updateEntryDetailScreen.
func TestUpdateEntryDetailScreen(t *testing.T) {
	suite := NewScreenTestSuite()

	// Подготавливаем модель для экрана деталей
	entry := gokeepasslib.Entry{
		Values: []gokeepasslib.ValueData{
			{Key: fieldNameTitle, Value: gokeepasslib.V{Content: "Detail Test"}},
		},
	}
	initialModel := suite.Model
	initialModel.state = entryDetailScreen
	initialModel.selectedEntry = &entryItem{entry: entry}
	initialModel.readOnlyMode = false // Режим редактирования разрешен
	suite.Model = initialModel

	t.Run("Переход к редактированию по 'e'", func(t *testing.T) {
		// Моделируем нажатие 'e'
		newModel, _ := suite.SimulateKeyRune('e')
		suite.Model = toModel(t, newModel)

		// Проверки
		suite.AssertState(t, entryEditScreen)
		assert.NotNil(t, suite.Model.editingEntry, "Должна быть создана копия для редактирования")
		assert.Equal(t, "Detail Test", suite.Model.editingEntry.GetTitle(), "Заголовок редактируемой записи")
		assert.Len(t, suite.Model.editInputs, numEditableFields, "Должны быть созданы поля ввода")
		assert.True(t, suite.Model.editInputs[editableFieldTitle].Focused(), "Фокус должен быть на поле Title")
	})

	// Сбрасываем состояние для следующего теста
	initialModel = suite.Model
	initialModel.state = entryDetailScreen
	initialModel.selectedEntry = &entryItem{entry: entry}
	initialModel.editingEntry = nil // Убираем состояние редактирования
	initialModel.editInputs = nil
	suite.Model = initialModel

	t.Run("Возврат к списку по 'b'", func(t *testing.T) {
		// Моделируем нажатие 'b'
		newModel, _ := suite.SimulateKeyRune('b')
		suite.Model = toModel(t, newModel)

		// Проверка
		suite.AssertState(t, entryListScreen)
		assert.Nil(t, suite.Model.selectedEntry, "Выбранная запись должна быть сброшена")
	})

	// TODO: Добавить тесты для других клавиш, если они будут добавлены
	// (например, копирование полей)
}
