package tui

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAttachmentItem_Title проверяет метод Title для attachmentItem.
func TestAttachmentItem_Title(t *testing.T) {
	testCases := []struct {
		name     string
		item     attachmentItem
		expected string
	}{
		{
			name:     "Обычное имя файла",
			item:     attachmentItem{name: "test.txt", id: 1},
			expected: "test.txt",
		},
		{
			name:     "Пустое имя",
			item:     attachmentItem{name: "", id: 2},
			expected: "",
		},
		{
			name:     "Имя с пробелами",
			item:     attachmentItem{name: "файл с пробелами.doc", id: 3},
			expected: "файл с пробелами.doc",
		},
		{
			name:     "Специальные символы",
			item:     attachmentItem{name: "!@#$%^&*().txt", id: 4},
			expected: "!@#$%^&*().txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.item.Title()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

// TestAttachmentItem_Description проверяет метод Description для attachmentItem.
func TestAttachmentItem_Description(t *testing.T) {
	testCases := []struct {
		name     string
		item     attachmentItem
		expected string
	}{
		{
			name:     "Положительный ID",
			item:     attachmentItem{name: "test.txt", id: 123},
			expected: "ID: 123",
		},
		{
			name:     "Нулевой ID",
			item:     attachmentItem{name: "empty.txt", id: 0},
			expected: "ID: 0",
		},
		{
			name:     "Отрицательный ID",
			item:     attachmentItem{name: "negative.txt", id: -5},
			expected: "ID: -5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.item.Description()
			assert.Equal(t, tc.expected, actual)

			// Дополнительно проверяем, что Description действительно форматирует ID как ожидается
			expected := fmt.Sprintf("ID: %d", tc.item.id)
			assert.Equal(t, expected, actual)
		})
	}
}

// TestAttachmentItem_FilterValue проверяет метод FilterValue для attachmentItem.
func TestAttachmentItem_FilterValue(t *testing.T) {
	testCases := []struct {
		name     string
		item     attachmentItem
		expected string
	}{
		{
			name:     "Обычное имя файла",
			item:     attachmentItem{name: "test.txt", id: 1},
			expected: "test.txt",
		},
		{
			name:     "Пустое имя",
			item:     attachmentItem{name: "", id: 2},
			expected: "",
		},
		{
			name:     "Имя с пробелами",
			item:     attachmentItem{name: "файл для поиска.pdf", id: 3},
			expected: "файл для поиска.pdf",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.item.FilterValue()
			assert.Equal(t, tc.expected, actual)

			// Проверяем, что FilterValue возвращает то же значение, что и Title
			assert.Equal(t, tc.item.Title(), tc.item.FilterValue())
		})
	}
}
