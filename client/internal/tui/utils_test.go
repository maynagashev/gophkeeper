//nolint:testpackage // Тесты в том же пакете для доступа к непубличным функциям
package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tobischo/gokeepasslib/v3"
)

// createTestDB создает тестовую базу данных с несколькими записями для тестирования.
func createTestDB() *gokeepasslib.Database {
	db := gokeepasslib.NewDatabase()
	db.Content = gokeepasslib.NewContent()

	// Создаем корневую группу
	rootGroup := gokeepasslib.NewGroup()
	rootGroup.Name = "Root"
	rootGroup.UUID = gokeepasslib.NewUUID()

	// Создаем подгруппу
	subGroup := gokeepasslib.NewGroup()
	subGroup.Name = "Subgroup"
	subGroup.UUID = gokeepasslib.NewUUID()

	// Создаем записи
	entry1 := gokeepasslib.NewEntry()
	entry1.UUID = gokeepasslib.NewUUID() // Для последующего поиска
	entry1.Values = append(entry1.Values, gokeepasslib.ValueData{
		Key:   "Title",
		Value: gokeepasslib.V{Content: "Entry 1"},
	})
	entry1.Values = append(entry1.Values, gokeepasslib.ValueData{
		Key:   "Password",
		Value: gokeepasslib.V{Content: "password1"},
	})

	entry2 := gokeepasslib.NewEntry()
	entry2.UUID = gokeepasslib.NewUUID()
	entry2.Values = append(entry2.Values, gokeepasslib.ValueData{
		Key:   "Title",
		Value: gokeepasslib.V{Content: "Entry 2"},
	})

	// Запись в подгруппе
	entry3 := gokeepasslib.NewEntry()
	entry3.UUID = gokeepasslib.NewUUID()
	entry3.Values = append(entry3.Values, gokeepasslib.ValueData{
		Key:   "Title",
		Value: gokeepasslib.V{Content: "Entry in Subgroup"},
	})

	// Добавляем записи в группы
	rootGroup.Entries = append(rootGroup.Entries, entry1, entry2)
	subGroup.Entries = append(subGroup.Entries, entry3)
	rootGroup.Groups = append(rootGroup.Groups, subGroup)

	// Устанавливаем корневую группу
	db.Content.Root = &gokeepasslib.RootData{
		Groups: []gokeepasslib.Group{rootGroup},
	}

	return db
}

// TestDeepCopyEntry проверяет корректность создания глубокой копии записи.
func TestDeepCopyEntry(t *testing.T) {
	// Создаем оригинальную запись с несколькими полями
	original := gokeepasslib.NewEntry()
	original.UUID = gokeepasslib.NewUUID()
	original.Values = append(original.Values, gokeepasslib.ValueData{
		Key:   "Title",
		Value: gokeepasslib.V{Content: "Test Entry"},
	})
	original.Values = append(original.Values, gokeepasslib.ValueData{
		Key:   "UserName",
		Value: gokeepasslib.V{Content: "user1"},
	})
	original.Values = append(original.Values, gokeepasslib.ValueData{
		Key:   "Password",
		Value: gokeepasslib.V{Content: "secret"},
	})

	// Добавляем теги
	original.Tags = "tag1, tag2"

	// Создаем копию
	entryCopy := DeepCopyEntry(original)

	// Проверяем, что все поля скопированы корректно
	assert.Equal(t, original.UUID, entryCopy.UUID, "UUID должны совпадать")
	assert.Equal(t, original.Tags, entryCopy.Tags, "Теги должны совпадать")

	// Проверяем, что скопированы все значения
	require.Equal(t, len(original.Values), len(entryCopy.Values), "Количество Values должно совпадать")
	for i, val := range original.Values {
		assert.Equal(t, val.Key, entryCopy.Values[i].Key, "Ключи должны совпадать")
		assert.Equal(t, val.Value.Content, entryCopy.Values[i].Value.Content, "Содержимое должно совпадать")
	}

	// Проверяем, что это действительно независимая копия
	// Меняем значение в копии и проверяем, что оригинал не изменился
	if len(entryCopy.Values) > 0 {
		entryCopy.Values[0].Value.Content = "Modified Title"
		assert.NotEqual(t, original.Values[0].Value.Content, entryCopy.Values[0].Value.Content,
			"Изменение копии не должно влиять на оригинал")
	}
}

// TestDeepCopyEntryAlias проверяет функцию-алиас deepCopyEntry.
func TestDeepCopyEntryAlias(t *testing.T) {
	// Создаем оригинальную запись с несколькими полями
	original := gokeepasslib.NewEntry()
	original.UUID = gokeepasslib.NewUUID()
	original.Values = append(original.Values, gokeepasslib.ValueData{
		Key:   "Title",
		Value: gokeepasslib.V{Content: "Test Entry for Alias"},
	})

	// Создаем копию через функцию-алиас
	entryCopy := deepCopyEntry(original)

	// Проверяем основные параметры
	assert.Equal(t, original.UUID, entryCopy.UUID, "UUID должны совпадать")
	require.Equal(t, len(original.Values), len(entryCopy.Values), "Количество Values должно совпадать")
	assert.Equal(t, original.Values[0].Value.Content, entryCopy.Values[0].Value.Content, "Содержимое должно совпадать")

	// Изменяем копию и проверяем независимость
	entryCopy.Values[0].Value.Content = "Modified in Alias Test"
	assert.NotEqual(t, original.Values[0].Value.Content, entryCopy.Values[0].Value.Content,
		"Изменение копии через алиас не должно влиять на оригинал")
}

// TestFindEntryInDB проверяет поиск записи в базе данных по UUID.
func TestFindEntryInDB(t *testing.T) {
	// Создаем тестовую базу данных
	db := createTestDB()

	// Проверяем поиск существующих записей
	t.Run("Entry_In_Root_Group", func(t *testing.T) {
		// Получаем UUID существующей записи из корневой группы
		targetUUID := db.Content.Root.Groups[0].Entries[0].UUID

		// Выполняем поиск
		entry := FindEntryInDB(db, targetUUID)

		// Проверяем результат
		require.NotNil(t, entry, "Должна быть найдена запись")
		assert.Equal(t, targetUUID, entry.UUID, "UUID найденной записи должен совпадать с искомым")

		// Проверяем, что это именно та запись, которую мы искали
		var title string
		for _, v := range entry.Values {
			if v.Key == "Title" {
				title = v.Value.Content
				break
			}
		}
		assert.Equal(t, "Entry 1", title, "Название записи должно совпадать")
	})

	t.Run("Entry_In_Subgroup", func(t *testing.T) {
		// Получаем UUID существующей записи из подгруппы
		targetUUID := db.Content.Root.Groups[0].Groups[0].Entries[0].UUID

		// Выполняем поиск
		entry := FindEntryInDB(db, targetUUID)

		// Проверяем результат
		require.NotNil(t, entry, "Должна быть найдена запись из подгруппы")
		assert.Equal(t, targetUUID, entry.UUID, "UUID найденной записи должен совпадать с искомым")

		// Проверяем, что это именно та запись, которую мы искали
		var title string
		for _, v := range entry.Values {
			if v.Key == "Title" {
				title = v.Value.Content
				break
			}
		}
		assert.Equal(t, "Entry in Subgroup", title, "Название записи должно совпадать")
	})

	t.Run("Nonexistent_Entry", func(t *testing.T) {
		// Создаем несуществующий UUID
		nonexistentUUID := gokeepasslib.NewUUID()

		// Выполняем поиск
		entry := FindEntryInDB(db, nonexistentUUID)

		// Проверяем результат
		assert.Nil(t, entry, "Запись с несуществующим UUID не должна быть найдена")
	})

	t.Run("Nil_Database", func(t *testing.T) {
		// Проверяем обработку nil базы данных
		entry := FindEntryInDB(nil, gokeepasslib.NewUUID())
		assert.Nil(t, entry, "При nil базе данных результат должен быть nil")
	})

	t.Run("Nil_Content", func(t *testing.T) {
		// Создаем базу без контента
		dbWithoutContent := &gokeepasslib.Database{Content: nil}
		entry := FindEntryInDB(dbWithoutContent, gokeepasslib.NewUUID())
		assert.Nil(t, entry, "При nil контенте результат должен быть nil")
	})

	t.Run("Nil_Root", func(t *testing.T) {
		// Создаем базу с контентом, но без корня
		dbWithoutRoot := &gokeepasslib.Database{Content: &gokeepasslib.DBContent{Root: nil}}
		entry := FindEntryInDB(dbWithoutRoot, gokeepasslib.NewUUID())
		assert.Nil(t, entry, "При nil корне результат должен быть nil")
	})
}

// TestFindEntryInGroups проверяет рекурсивный поиск записей в группах.
func TestFindEntryInGroups(t *testing.T) {
	// Создаем тестовую базу данных
	db := createTestDB()

	// Получаем слайс групп для тестирования
	groups := db.Content.Root.Groups

	t.Run("Entry_In_Root_Group", func(t *testing.T) {
		// Получаем UUID существующей записи из корневой группы
		targetUUID := db.Content.Root.Groups[0].Entries[0].UUID

		// Выполняем поиск
		entry := FindEntryInGroups(groups, targetUUID)

		// Проверяем результат
		require.NotNil(t, entry, "Должна быть найдена запись")
		assert.Equal(t, targetUUID, entry.UUID, "UUID найденной записи должен совпадать с искомым")
	})

	t.Run("Entry_In_Subgroup", func(t *testing.T) {
		// Получаем UUID существующей записи из подгруппы
		targetUUID := db.Content.Root.Groups[0].Groups[0].Entries[0].UUID

		// Выполняем поиск
		entry := FindEntryInGroups(groups, targetUUID)

		// Проверяем результат
		require.NotNil(t, entry, "Должна быть найдена запись из подгруппы")
		assert.Equal(t, targetUUID, entry.UUID, "UUID найденной записи должен совпадать с искомым")
	})

	t.Run("Nonexistent_Entry", func(t *testing.T) {
		// Создаем несуществующий UUID
		nonexistentUUID := gokeepasslib.NewUUID()

		// Выполняем поиск
		entry := FindEntryInGroups(groups, nonexistentUUID)

		// Проверяем результат
		assert.Nil(t, entry, "Запись с несуществующим UUID не должна быть найдена")
	})

	t.Run("Empty_Groups", func(t *testing.T) {
		// Проверяем пустой слайс групп
		entry := FindEntryInGroups([]gokeepasslib.Group{}, gokeepasslib.NewUUID())
		assert.Nil(t, entry, "В пустом слайсе групп не должно быть записей")
	})
}

// TestFindEntryInDBAlias проверяет функцию-алиас findEntryInDB.
func TestFindEntryInDBAlias(t *testing.T) {
	// Создаем тестовую базу данных
	db := createTestDB()

	// Получаем UUID существующей записи из корневой группы
	targetUUID := db.Content.Root.Groups[0].Entries[0].UUID

	// Выполняем поиск через алиас
	entry := findEntryInDB(db, targetUUID)

	// Проверяем результат
	require.NotNil(t, entry, "Должна быть найдена запись через алиас")
	assert.Equal(t, targetUUID, entry.UUID, "UUID найденной записи должен совпадать с искомым")

	// Проверяем nil-обработку
	nilEntry := findEntryInDB(nil, targetUUID)
	assert.Nil(t, nilEntry, "При nil базе данных результат через алиас должен быть nil")
}
