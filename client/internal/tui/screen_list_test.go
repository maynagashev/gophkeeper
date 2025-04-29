package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maynagashev/gophkeeper/client/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

// Вспомогательная функция для создания временной тестовой базы данных KDBX.
func createTestKdbx(t *testing.T, password string, numEntries int) (string, *gokeepasslib.Database) {
	t.Helper()
	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(password)
	db.Content.Meta.DatabaseName = "Test DB"
	db.Content.Meta.RecycleBinEnabled = w.NewBoolWrapper(false) // Отключаем корзину для простоты

	// Создаем корневую группу
	rootGroup := gokeepasslib.NewGroup()
	rootGroup.Name = "Root"
	db.Content.Root = &gokeepasslib.RootData{Groups: []gokeepasslib.Group{rootGroup}}

	// Добавляем записи
	for i := range numEntries {
		entry := gokeepasslib.NewEntry()
		entry.Values = append(entry.Values, gokeepasslib.ValueData{
			Key:   "Title",
			Value: gokeepasslib.V{Content: fmt.Sprintf("Entry %d", i+1)},
		})
		entry.Values = append(entry.Values, gokeepasslib.ValueData{
			Key:   "UserName",
			Value: gokeepasslib.V{Content: fmt.Sprintf("user%d", i+1)},
		})
		db.Content.Root.Groups[0].Entries = append(db.Content.Root.Groups[0].Entries, entry)
	}

	// Сохраняем во временный файл
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.kdbx")
	file, err := os.Create(tmpFile)
	require.NoError(t, err)
	defer file.Close()

	encoder := gokeepasslib.NewEncoder(file)
	err = encoder.Encode(db)
	require.NoError(t, err)

	return tmpFile, db
}

// TestUpdateEntryListScreen проверяет функцию updateEntryListScreen.
func TestUpdateEntryListScreen(t *testing.T) {
	// Создаем тестовую базу данных
	_, testDB := createTestKdbx(t, "password", 3)

	t.Run("Обработка dbOpenedMsg", func(t *testing.T) {
		// Создаем начальную модель
		initialModel := &model{
			state:     entryListScreen,
			entryList: initEntryList(), // Инициализируем список
		}

		// Создаем сообщение об открытии БД
		msg := dbOpenedMsg{db: testDB}

		// Вызываем Update
		updatedModel, _ := initialModel.Update(msg)
		require.IsType(t, &model{}, updatedModel, "Должен вернуться указатель на model")
		m, ok := updatedModel.(*model)
		require.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Проверки
		assert.Equal(t, entryListScreen, m.state, "Состояние должно остаться entryListScreen")
		assert.NotNil(t, m.db, "База данных должна быть установлена в модели")
		assert.Len(t, m.entryList.Items(), 3, "Список должен содержать 3 элемента")

		// Проверяем содержимое первого элемента
		items := m.entryList.Items()
		require.IsType(t, entryItem{}, items[0], "Элемент должен быть типа entryItem")
		firstItem, ok := items[0].(entryItem)
		require.True(t, ok, "Приведение типа к entryItem должно быть успешным")
		assert.Equal(t, "Entry 1", firstItem.Title(), "Заголовок первого элемента")
		assert.Contains(t, firstItem.Description(), "User: user1", "Описание первого элемента")
	})

	t.Run("Навигация по списку (вниз)", func(t *testing.T) {
		// Создаем модель с уже открытой БД
		initialModel := &model{
			state:     entryListScreen,
			db:        testDB,
			entryList: initEntryList(),
		}
		// Заполняем список
		initialModel.Update(dbOpenedMsg{db: testDB})

		// Моделируем нажатие клавиши "вниз"
		keyDownMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}} // Или tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := initialModel.Update(keyDownMsg)
		m, ok := updatedModel.(*model)
		require.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Проверка
		assert.Equal(t, 1, m.entryList.Index(), "Индекс должен сместиться на 1")
	})

	t.Run("Навигация по списку (вверх)", func(t *testing.T) {
		// Создаем модель, выбрав второй элемент
		initialModel := &model{
			state:     entryListScreen,
			db:        testDB,
			entryList: initEntryList(),
		}
		initialModel.Update(dbOpenedMsg{db: testDB})
		initialModel.entryList.Select(1) // Выбираем второй элемент (индекс 1)

		// Моделируем нажатие клавиши "вверх"
		keyUpMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}} // Или tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := initialModel.Update(keyUpMsg)
		m, ok := updatedModel.(*model)
		require.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Проверка
		assert.Equal(t, 0, m.entryList.Index(), "Индекс должен вернуться на 0")
	})

	t.Run("Переход к деталям по Enter", func(t *testing.T) {
		// Создаем модель с открытой БД
		initialModel := &model{
			state:     entryListScreen,
			db:        testDB,
			entryList: initEntryList(),
		}
		initialModel.Update(dbOpenedMsg{db: testDB})
		initialModel.entryList.Select(0) // Выбираем первый элемент

		// Моделируем нажатие Enter
		enterKeyMsg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := initialModel.Update(enterKeyMsg)
		m, ok := updatedModel.(*model)
		require.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Проверки
		assert.Equal(t, entryDetailScreen, m.state, "Состояние должно измениться на entryDetailScreen")
		assert.NotNil(t, m.selectedEntry, "Должна быть выбрана запись")
		assert.Equal(t, "Entry 1", m.selectedEntry.Title(), "Должна быть выбрана первая запись")
	})

	t.Run("Переход к добавлению по 'a'", func(t *testing.T) {
		// Создаем модель с открытой БД
		initialModel := &model{
			state:        entryListScreen,
			db:           testDB,
			entryList:    initEntryList(),
			readOnlyMode: false, // Убедимся, что не в read-only
		}
		initialModel.Update(dbOpenedMsg{db: testDB})

		// Моделируем нажатие 'a'
		addKeyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		updatedModel, _ := initialModel.Update(addKeyMsg)
		m, ok := updatedModel.(*model)
		require.True(t, ok, "Приведение типа к *model должно быть успешным")

		// Проверки
		assert.Equal(t, entryAddScreen, m.state, "Состояние должно измениться на entryAddScreen")
		assert.NotNil(t, m.editingEntry, "Должна быть создана пустая запись для добавления")
		assert.Len(t, m.editInputs, numEditableFields, "Должны быть инициализированы поля ввода для добавления")
		assert.Equal(t, editableFieldTitle, m.focusedField, "Фокус должен быть на поле Title")
	})

	// TODO: Добавить тесты для _handleAuthLoadErrorMsg, _handleAuthLoadSuccessMsg,
	// TODO: клавиш 'e' (редактирование), 's' (сохранение), 'b' (назад), 'q' (выход), '/' (фильтр)
}

// TestHandleAuthLoadError проверяет функцию _handleAuthLoadError.
func TestHandleAuthLoadError(t *testing.T) {
	t.Run("URL не из флага", func(t *testing.T) {
		initialModel := &model{
			serverURL: "http://existing.url",
			apiClient: api.NewHTTPClient("http://existing.url"),
			authToken: "existing_token",
		}
		errLoad := errors.New("test auth load error")

		initialModel._handleAuthLoadError(errLoad, false)

		assert.Equal(t, "", initialModel.serverURL, "URL сервера должен быть сброшен")
		assert.Nil(t, initialModel.apiClient, "API клиент должен быть сброшен")
		assert.Equal(t, "", initialModel.authToken, "Токен должен быть сброшен")
		assert.Contains(t, initialModel.loginStatus, statusNotLoggedIn)
		assert.Contains(t, initialModel.loginStatus, "(ошибка загрузки)")
	})

	t.Run("URL из флага", func(t *testing.T) {
		initialModel := &model{
			serverURL: "http://flag.url",
			apiClient: api.NewHTTPClient("http://flag.url"),
			authToken: "existing_token",
		}
		errLoad := errors.New("test auth load error")

		initialModel._handleAuthLoadError(errLoad, true)

		assert.Equal(t, "http://flag.url", initialModel.serverURL, "URL сервера не должен меняться")
		assert.NotNil(t, initialModel.apiClient, "API клиент не должен меняться")
		assert.Equal(t, "", initialModel.authToken, "Токен должен быть сброшен")
		assert.Contains(t, initialModel.loginStatus, statusNotLoggedIn)
		assert.Contains(t, initialModel.loginStatus, "(ошибка загрузки)")
	})
}

// TestHandleAuthLoadSuccess проверяет функцию _handleAuthLoadSuccess.
func TestHandleAuthLoadSuccess(t *testing.T) {
	t.Run("URL не из флага, URL и токен загружены", func(t *testing.T) {
		initialModel := &model{}
		loadedURL := "http://loaded.url"
		loadedToken := "loaded_token"

		initialModel._handleAuthLoadSuccess(loadedURL, loadedToken, false)

		assert.Equal(t, loadedURL, initialModel.serverURL, "URL сервера должен быть установлен")
		assert.NotNil(t, initialModel.apiClient, "API клиент должен быть создан")
		assert.Equal(t, loadedToken, initialModel.authToken, "Токен должен быть установлен")
		assert.Contains(t, initialModel.loginStatus, "Вход выполнен")
	})

	t.Run("URL не из флага, URL пустой, токен загружен", func(t *testing.T) {
		initialModel := &model{}
		loadedURL := ""
		loadedToken := "loaded_token"

		initialModel._handleAuthLoadSuccess(loadedURL, loadedToken, false)

		assert.Equal(t, "", initialModel.serverURL, "URL сервера должен быть пустым")
		assert.Nil(t, initialModel.apiClient, "API клиент не должен быть создан")
		assert.Equal(t, loadedToken, initialModel.authToken, "Токен должен быть установлен")
		assert.Contains(t, initialModel.loginStatus, "Вход выполнен")
	})

	t.Run("URL не из флага, URL загружен, токен пустой", func(t *testing.T) {
		initialModel := &model{}
		loadedURL := "http://loaded.url"
		loadedToken := ""

		initialModel._handleAuthLoadSuccess(loadedURL, loadedToken, false)

		assert.Equal(t, loadedURL, initialModel.serverURL, "URL сервера должен быть установлен")
		assert.NotNil(t, initialModel.apiClient, "API клиент должен быть создан")
		assert.Equal(t, "", initialModel.authToken, "Токен должен быть пустым")
		assert.Equal(t, statusNotLoggedIn, initialModel.loginStatus)
	})

	t.Run("URL из флага, токен загружен", func(t *testing.T) {
		initialModel := &model{
			serverURL: "http://flag.url",
			apiClient: api.NewHTTPClient("http://flag.url"), // Клиент уже есть
		}
		loadedURL := "http://ignored.url"
		loadedToken := "loaded_token"

		initialModel._handleAuthLoadSuccess(loadedURL, loadedToken, true)

		assert.Equal(t, "http://flag.url", initialModel.serverURL, "URL сервера не должен меняться")
		assert.NotNil(t, initialModel.apiClient, "API клиент не должен меняться")
		assert.Equal(t, loadedToken, initialModel.authToken, "Токен должен быть установлен")
		assert.Contains(t, initialModel.loginStatus, "Вход выполнен")
	})

	t.Run("URL из флага, токен пустой", func(t *testing.T) {
		initialModel := &model{
			serverURL: "http://flag.url",
			apiClient: api.NewHTTPClient("http://flag.url"),
		}
		loadedURL := "http://ignored.url"
		loadedToken := ""

		initialModel._handleAuthLoadSuccess(loadedURL, loadedToken, true)

		assert.Equal(t, "http://flag.url", initialModel.serverURL, "URL сервера не должен меняться")
		assert.NotNil(t, initialModel.apiClient, "API клиент не должен меняться")
		assert.Equal(t, "", initialModel.authToken, "Токен должен быть пустым")
		assert.Equal(t, statusNotLoggedIn, initialModel.loginStatus)
	})
}
