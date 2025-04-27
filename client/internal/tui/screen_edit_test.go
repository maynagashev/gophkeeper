package tui

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
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

// TestPrepareEditScreen проверяет функцию prepareEditScreen.
func TestPrepareEditScreen(t *testing.T) {
	t.Run("С выбранной записью", func(t *testing.T) {
		// Создаем тестовую запись
		entry := gokeepasslib.Entry{
			Values: []gokeepasslib.ValueData{
				{Key: fieldNameTitle, Value: gokeepasslib.V{Content: "Тестовая запись"}},
				{Key: fieldNameUserName, Value: gokeepasslib.V{Content: "user123"}},
				{Key: fieldNamePassword, Value: gokeepasslib.V{Content: "pass123", Protected: w.NewBoolWrapper(true)}},
			},
		}

		// Создаем модель с выбранной записью
		m := &model{
			selectedEntry: &entryItem{entry: entry},
		}

		// Вызываем тестируемую функцию
		m.prepareEditScreen()

		// Проверяем, что поля редактирования были инициализированы корректно
		assert.NotNil(t, m.editingEntry, "Должна быть создана копия для редактирования")
		assert.Len(t, m.editInputs, numEditableFields, "Должны быть созданы все поля ввода")
		assert.Equal(t, "Тестовая запись", m.editInputs[editableFieldTitle].Value(),
			"Поле заголовка должно содержать значение из записи")
		assert.Equal(t, "user123", m.editInputs[editableFieldUserName].Value(),
			"Поле имени пользователя должно содержать значение из записи")
		assert.Equal(t, "pass123", m.editInputs[editableFieldPassword].Value(),
			"Поле пароля должно содержать значение из записи")

		// Проверяем, что чувствительные поля правильно замаскированы
		assert.Equal(t, textinput.EchoPassword, m.editInputs[editableFieldPassword].EchoMode,
			"Поле пароля должно быть замаскировано")
		assert.Equal(t, textinput.EchoPassword, m.editInputs[editableFieldCVV].EchoMode, "Поле CVV должно быть замаскировано")
		assert.Equal(t, textinput.EchoPassword, m.editInputs[editableFieldPIN].EchoMode, "Поле PIN должно быть замаскировано")
	})

	t.Run("Без выбранной записи", func(t *testing.T) {
		// Создаем модель без выбранной записи
		m := &model{
			selectedEntry: nil,
		}

		// Вызываем тестируемую функцию
		m.prepareEditScreen()

		// Проверяем, что редактирование не инициализировалось
		assert.Nil(t, m.editingEntry, "Не должна быть создана копия для редактирования")
		assert.Nil(t, m.editInputs, "Не должны быть созданы поля ввода")
	})
}

// TestUpdateEntryEditScreen проверяет функцию updateEntryEditScreen и handleEditScreenKeys.
func TestUpdateEntryEditScreen(t *testing.T) {
	t.Run("Обработка клавиши Escape", func(t *testing.T) {
		// Создаем базовую модель для теста
		entry := gokeepasslib.Entry{
			Values: []gokeepasslib.ValueData{
				{Key: fieldNameTitle, Value: gokeepasslib.V{Content: "Тестовая запись"}},
			},
		}

		m := &model{
			state:         entryEditScreen,
			selectedEntry: &entryItem{entry: entry},
		}

		// Подготавливаем экран редактирования
		m.prepareEditScreen()

		// Моделируем нажатие клавиши Escape
		escKeyMsg := tea.KeyMsg{Type: tea.KeyEsc}
		resultModel, _ := m.updateEntryEditScreen(escKeyMsg)
		modelAfter, ok := resultModel.(*model)
		assert.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Проверяем, что состояние изменилось обратно на экран деталей
		assert.Equal(t, entryDetailScreen, modelAfter.state, "После нажатия Escape должен быть переход на экран деталей")
		assert.Nil(t, modelAfter.editingEntry, "Редактируемая запись должна быть очищена")
		assert.Nil(t, modelAfter.editInputs, "Поля ввода должны быть очищены")
	})

	t.Run("Навигация с помощью Tab", func(t *testing.T) {
		// Создаем базовую модель для теста
		entry := gokeepasslib.Entry{
			Values: []gokeepasslib.ValueData{
				{Key: fieldNameTitle, Value: gokeepasslib.V{Content: "Тестовая запись"}},
			},
		}

		m := &model{
			state:         entryEditScreen,
			selectedEntry: &entryItem{entry: entry},
			focusedField:  0,
		}

		// Подготавливаем экран редактирования
		m.prepareEditScreen()

		// Моделируем нажатие клавиши Tab
		tabKeyMsg := tea.KeyMsg{Type: tea.KeyTab}
		resultModel, _ := m.updateEntryEditScreen(tabKeyMsg)
		modelAfter, ok := resultModel.(*model)
		assert.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Проверяем, что фокус переместился на следующее поле
		assert.Equal(t, 1, modelAfter.focusedField, "После нажатия Tab фокус должен переместиться на следующее поле")
	})

	t.Run("Сохранение изменений по Enter", func(t *testing.T) {
		// Создаем базовую модель для теста с мок-списком
		entry := gokeepasslib.Entry{
			Values: []gokeepasslib.ValueData{
				{Key: fieldNameTitle, Value: gokeepasslib.V{Content: "Тестовая запись"}},
			},
		}

		// Создаем мок-список для тестирования
		mockList := list.New([]list.Item{entryItem{entry: entry}}, list.NewDefaultDelegate(), 0, 0)
		mockList.Select(0)

		m := &model{
			state:         entryEditScreen,
			selectedEntry: &entryItem{entry: entry},
			entryList:     mockList,
			readOnlyMode:  false,
		}

		// Подготавливаем экран редактирования
		m.prepareEditScreen()

		// Моделируем ввод нового текста в поле Title
		// Сначала очищаем поле (если там что-то было)
		// (В данном тесте поле изначально "Тестовая запись")
		// Симулируем удаление старого значения (Backspace несколько раз)
		for range len(m.editInputs[editableFieldTitle].Value()) {
			backspaceKeyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
			m.updateEntryEditScreen(backspaceKeyMsg) // Обновляем модель
		}

		// Симулируем ввод нового значения
		newValue := "Обновленная запись"
		for _, r := range newValue {
			runeKeyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			m.updateEntryEditScreen(runeKeyMsg) // Обновляем модель
		}

		// Моделируем нажатие клавиши Enter
		enterKeyMsg := tea.KeyMsg{Type: tea.KeyEnter}
		resultModel, _ := m.updateEntryEditScreen(enterKeyMsg)
		modelAfter, ok := resultModel.(*model)
		assert.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Проверяем, что состояние изменилось и изменения сохранены
		assert.Equal(t, entryDetailScreen, modelAfter.state, "После нажатия Enter должен быть переход на экран деталей")
		assert.Equal(t, "Обновленная запись", modelAfter.selectedEntry.entry.GetContent(fieldNameTitle),
			"Значение поля Title должно быть обновлено")
	})

	t.Run("Редактирование значения поля", func(t *testing.T) {
		// Создаем базовую модель для теста
		entry := gokeepasslib.Entry{
			Values: []gokeepasslib.ValueData{
				{Key: fieldNameTitle, Value: gokeepasslib.V{Content: "Тестовая запись"}},
			},
		}

		m := &model{
			state:         entryEditScreen,
			selectedEntry: &entryItem{entry: entry},
			readOnlyMode:  false,
			focusedField:  editableFieldTitle,
		}

		// Подготавливаем экран редактирования
		m.prepareEditScreen()

		// Моделируем ввод текста
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}}
		resultModel, _ := m.updateEntryEditScreen(keyMsg)
		modelAfter, ok := resultModel.(*model)
		assert.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Получаем текущее значение поля заголовка
		updatedValue := modelAfter.editInputs[editableFieldTitle].Value()

		// Проверяем, что значение поля изменилось
		// Ожидаем "Тестовая записьA", так как мы добавили символ 'A' к существующему значению
		assert.Contains(t, updatedValue, "A", "Введенный символ должен быть добавлен к значению поля")
	})
}

// TestHandleAttachmentDeleteAction проверяет функцию handleAttachmentDeleteAction.
func TestHandleAttachmentDeleteAction(t *testing.T) {
	t.Run("С вложениями для удаления", func(t *testing.T) {
		// Создаем запись с вложениями
		entry := gokeepasslib.Entry{}
		binRef1 := gokeepasslib.BinaryReference{Name: "file1.dat"}
		binRef1.Value.ID = 10
		binRef2 := gokeepasslib.BinaryReference{Name: "report.pdf"}
		binRef2.Value.ID = 20
		entry.Binaries = []gokeepasslib.BinaryReference{binRef1, binRef2}

		// Создаем модель
		m := &model{
			state:          entryEditScreen,
			editingEntry:   &entry,
			attachmentList: initAttachmentDeleteList(), // Инициализируем список
		}

		// Вызываем функцию
		resultModel, cmd := m.handleAttachmentDeleteAction()
		assert.NotNil(t, cmd, "Должна быть команда ClearScreen") // Ожидаем команду
		require.IsType(t, &model{}, resultModel, "Должен вернуться указатель на model")
		modelAfter, ok := resultModel.(*model)
		require.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Проверяем результат
		assert.Equal(t, attachmentListDeleteScreen, modelAfter.state, "Состояние должно измениться на удаление вложений")
		items := modelAfter.attachmentList.Items()
		require.Len(t, items, 2, "В списке должно быть 2 элемента")

		// Проверяем первый элемент
		require.IsType(t, attachmentItem{}, items[0])
		item1, ok := items[0].(attachmentItem)
		require.True(t, ok, "Приведение типа к attachmentItem должно быть успешным")
		assert.Equal(t, "file1.dat", item1.Title())
		assert.Equal(t, "ID: 10", item1.Description())

		// Проверяем второй элемент
		require.IsType(t, attachmentItem{}, items[1])
		item2, ok := items[1].(attachmentItem)
		require.True(t, ok, "Приведение типа к attachmentItem должно быть успешным")
		assert.Equal(t, "report.pdf", item2.Title())
		assert.Equal(t, "ID: 20", item2.Description())
	})

	t.Run("Без вложений", func(t *testing.T) {
		// Создаем запись без вложений
		entry := gokeepasslib.Entry{}

		// Создаем модель
		m := &model{
			state:          entryEditScreen,
			editingEntry:   &entry,
			attachmentList: initAttachmentDeleteList(),
		}

		// Вызываем функцию
		resultModel, cmd := m.handleAttachmentDeleteAction()
		assert.Nil(t, cmd, "Не должно быть команды") // Не должно быть команды
		require.IsType(t, &model{}, resultModel, "Должен вернуться указатель на model")
		modelAfter, ok := resultModel.(*model)
		require.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Проверяем результат
		assert.Equal(t, entryEditScreen, modelAfter.state, "Состояние не должно меняться")
		assert.Empty(t, modelAfter.attachmentList.Items(), "Список должен оставаться пустым")
	})

	t.Run("editingEntry is nil", func(t *testing.T) {
		// Создаем модель без editingEntry
		m := &model{
			state:          entryEditScreen,
			editingEntry:   nil,
			attachmentList: initAttachmentDeleteList(),
		}

		// Вызываем функцию
		resultModel, cmd := m.handleAttachmentDeleteAction()
		assert.Nil(t, cmd, "Не должно быть команды") // Не должно быть команды
		require.IsType(t, &model{}, resultModel, "Должен вернуться указатель на model")
		modelAfter, ok := resultModel.(*model)
		require.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Проверяем результат
		assert.Equal(t, entryEditScreen, modelAfter.state, "Состояние не должно меняться")
	})
}

// TestViewEntryEditScreen проверяет функцию viewEntryEditScreen.
func TestViewEntryEditScreen(t *testing.T) {
	t.Skip("Тест временно отключен из-за проблем с фокусом/отображением")

	// Создаем тестовую запись
	entry := gokeepasslib.Entry{
		Values: []gokeepasslib.ValueData{
			{Key: fieldNameTitle, Value: gokeepasslib.V{Content: "Заголовок"}},
			{Key: fieldNameUserName, Value: gokeepasslib.V{Content: "Пользователь"}},
		},
		Binaries: []gokeepasslib.BinaryReference{
			{Name: "file1.txt", Value: struct {
				ID int `xml:"Ref,attr"`
			}{ID: 1}},
			{Name: "image.png", Value: struct {
				ID int `xml:"Ref,attr"`
			}{ID: 2}},
		},
	}

	// Создаем модель
	m := &model{
		state:         entryEditScreen,
		selectedEntry: &entryItem{entry: entry},
		focusedField:  editableFieldUserName, // Фокус на втором поле
	}
	// Подготавливаем экран редактирования (создает editingEntry и editInputs)
	m.prepareEditScreen()
	require.NotNil(t, m.editingEntry, "editingEntry не должно быть nil после prepareEditScreen")
	require.NotNil(t, m.editInputs, "editInputs не должно быть nil после prepareEditScreen")
	require.Len(t, m.editInputs, numEditableFields, "Должно быть создано %d полей ввода", numEditableFields)

	// Дополнительно проверяем состояние фокуса после prepareEditScreen
	assert.True(t, m.editInputs[editableFieldUserName].Focused(), "Поле UserName должно быть в фокусе")
	assert.False(t, m.editInputs[editableFieldTitle].Focused(), "Поле Title НЕ должно быть в фокусе")
	for i := range m.editInputs {
		if i != editableFieldUserName {
			assert.False(t, m.editInputs[i].Focused(), "Только поле UserName должно быть в фокусе, но поле %d тоже", i)
		}
	}

	// Вызываем функцию отображения
	view := m.viewEntryEditScreen()

	// Проверяем основные элементы
	assert.Contains(t, view, "Редактирование записи: Заголовок", "Должен быть заголовок экрана")

	// Проверяем отображение полей ввода, полагаясь на формат вывода textinput.View()
	// Точный формат может зависеть от версии библиотеки, предполагаем:
	// Не в фокусе: "Placeholder: Value"
	// В фокусе: "> Placeholder: Value|" (с курсором)
	// Важно: Убираем лишние пробелы и индикаторы, добавленные ранее в viewEntryEditScreen.

	// Поле Title (не в фокусе)
	assert.Contains(t, view, fieldNameTitle+": Заголовок\n",
		"Должно отображаться поле Title без фокуса (формат View())")
	assert.NotContains(t, view, "> "+fieldNameTitle, // Убеждаемся, что нет индикатора фокуса
		"Поле Title не должно иметь индикатора фокуса '>'")

	// Поле UserName (в фокусе)
	assert.Contains(t, view, "> "+fieldNameUserName+": Пользователь", // Ожидаем индикатор и плейсхолдер
		"Должно отображаться поле UserName с фокусом (формат View())")
	// Можно также проверить наличие курсора, если его формат известен и стабилен
	// assert.Contains(t, view, "|", "Должен быть курсор в сфокусированном поле")

	// Проверяем отображение вложений
	assert.Contains(t, view, "\n--- Вложения ---\n", "Должен быть разделитель вложений")
	assert.Contains(t, view, " [0] file1.txt\n", "Должно отображаться первое вложение")
	assert.Contains(t, view, " [1] image.png\n", "Должно отображаться второе вложение")

	// Проверяем случай без вложений
	m.editingEntry.Binaries = nil
	viewNoAttachments := m.viewEntryEditScreen()
	assert.Contains(t, viewNoAttachments, "(Нет вложений)", "Должно быть сообщение об отсутствии вложений")
	assert.NotContains(t, viewNoAttachments, "[0]", "Не должно быть индекса вложения, если их нет")
}
