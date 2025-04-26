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
