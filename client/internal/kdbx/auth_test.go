package kdbx_test

import (
	"testing"
	"time"

	"github.com/maynagashev/gophkeeper/client/internal/kdbx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tobischo/gokeepasslib/v3"
)

// Константы для тестирования auth.go.
const (
	testServerURL = "https://test-server.example.com"
	testAuthToken = "test-jwt-token-for-auth"
)

// createTestDatabase создает временную тестовую базу данных в памяти.
func createTestDatabase() *gokeepasslib.Database {
	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(testPassword)
	// Создаем объекты через API библиотеки
	db.Content = gokeepasslib.NewContent()
	// Инициализируем пустой слайс CustomData
	db.Content.Meta.CustomData = []gokeepasslib.CustomData{}

	// Добавляем корневую группу
	rootGroup := gokeepasslib.NewGroup()
	rootGroup.Name = "Root"
	rootGroup.UUID = gokeepasslib.NewUUID()
	// В TimeData могут быть nil поля, это нормально

	// Устанавливаем корневую группу
	db.Content.Root = &gokeepasslib.RootData{
		Groups: []gokeepasslib.Group{rootGroup},
	}

	return db
}

// TestSaveAuthData проверяет сохранение и обновление аутентификационных данных.
func TestSaveAuthData(t *testing.T) {
	// Тест на успешное сохранение данных
	t.Run("Success_Save_Auth_Data", func(t *testing.T) {
		db := createTestDatabase()
		require.NotNil(t, db, "База данных должна быть создана")

		// Проверяем, что изначально CustomData пуст
		require.Empty(t, db.Content.Meta.CustomData, "CustomData должен быть пустым изначально")

		// Сохраняем данные аутентификации
		err := kdbx.SaveAuthData(db, testServerURL, testAuthToken)
		require.NoError(t, err, "SaveAuthData не должен возвращать ошибку")

		// Проверяем, что данные были сохранены
		require.Len(t, db.Content.Meta.CustomData, 2, "CustomData должен содержать 2 записи")

		// Извлекаем сохраненные данные
		var foundURL, foundToken bool
		for _, item := range db.Content.Meta.CustomData {
			switch item.Key {
			case kdbx.CustomDataKeyServerURL:
				assert.Equal(t, testServerURL, item.Value, "URL сервера должен быть сохранен корректно")
				foundURL = true
			case kdbx.CustomDataKeyAuthToken:
				assert.Equal(t, testAuthToken, item.Value, "Токен должен быть сохранен корректно")
				foundToken = true
			}
		}
		assert.True(t, foundURL, "URL сервера должен быть найден в CustomData")
		assert.True(t, foundToken, "Токен должен быть найден в CustomData")

		// Проверяем обновление LastModificationTime
		rootGroup := &db.Content.Root.Groups[0]
		require.NotNil(t, rootGroup.Times.LastModificationTime, "LastModificationTime должно быть установлено")
	})

	// Тест на обновление существующих данных
	t.Run("Success_Update_Auth_Data", func(t *testing.T) {
		db := createTestDatabase()
		require.NotNil(t, db, "База данных должна быть создана")

		// Сначала добавляем начальные данные
		err := kdbx.SaveAuthData(db, testServerURL, testAuthToken)
		require.NoError(t, err, "SaveAuthData не должен возвращать ошибку")

		// Фиксируем время последней модификации
		initialModTime := time.Time{}
		if db.Content.Root.Groups[0].Times.LastModificationTime != nil {
			initialModTime = db.Content.Root.Groups[0].Times.LastModificationTime.Time
		}

		// Даем время пройти, чтобы убедиться в изменении
		time.Sleep(10 * time.Millisecond)

		// Обновляем данные
		newURL := "https://new-test-server.example.com"
		newToken := "new-test-jwt-token"
		err = kdbx.SaveAuthData(db, newURL, newToken)
		require.NoError(t, err, "SaveAuthData не должен возвращать ошибку при обновлении")

		// Проверяем, что данные обновлены
		var foundNewURL, foundNewToken bool
		for _, item := range db.Content.Meta.CustomData {
			switch item.Key {
			case kdbx.CustomDataKeyServerURL:
				assert.Equal(t, newURL, item.Value, "URL сервера должен быть обновлен")
				foundNewURL = true
			case kdbx.CustomDataKeyAuthToken:
				assert.Equal(t, newToken, item.Value, "Токен должен быть обновлен")
				foundNewToken = true
			}
		}
		assert.True(t, foundNewURL, "Новый URL сервера должен быть найден")
		assert.True(t, foundNewToken, "Новый токен должен быть найден")

		// Проверяем, что LastModificationTime изменилось
		updatedModTime := time.Time{}
		if db.Content.Root.Groups[0].Times.LastModificationTime != nil {
			updatedModTime = db.Content.Root.Groups[0].Times.LastModificationTime.Time
		}

		// Только если initialModTime не пустое время
		if !initialModTime.IsZero() {
			assert.NotEqual(t, initialModTime, updatedModTime, "LastModificationTime должно обновиться")
		}
	})

	// Тест на удаление данных (передача пустых строк)
	t.Run("Success_Remove_Auth_Data", func(t *testing.T) {
		db := createTestDatabase()
		require.NotNil(t, db, "База данных должна быть создана")

		// Сначала добавляем данные
		err := kdbx.SaveAuthData(db, testServerURL, testAuthToken)
		require.NoError(t, err, "SaveAuthData не должен возвращать ошибку")
		require.Len(t, db.Content.Meta.CustomData, 2, "CustomData должен содержать 2 записи")

		// Теперь удаляем данные, передавая пустые строки
		err = kdbx.SaveAuthData(db, "", "")
		require.NoError(t, err, "SaveAuthData не должен возвращать ошибку при удалении")

		// Проверяем, что данные удалены
		assert.Empty(t, db.Content.Meta.CustomData, "CustomData должен быть пустым после удаления")
	})

	// Тест с некорректной базой данных
	t.Run("Error_With_Nil_Database", func(t *testing.T) {
		err := kdbx.SaveAuthData(nil, testServerURL, testAuthToken)
		require.Error(t, err, "SaveAuthData должен возвращать ошибку при nil базе данных")
		assert.Contains(t, err.Error(), "не инициализированы", "Ошибка должна указывать на проблему инициализации")
	})

	// Тест с nil Content
	t.Run("Error_With_Nil_Content", func(t *testing.T) {
		db := &gokeepasslib.Database{Content: nil}
		err := kdbx.SaveAuthData(db, testServerURL, testAuthToken)
		require.Error(t, err, "SaveAuthData должен возвращать ошибку при nil Content")
		assert.Contains(t, err.Error(), "не инициализированы", "Ошибка должна указывать на проблему инициализации")
	})

	// Тест с nil Meta
	t.Run("Error_With_Nil_Meta", func(t *testing.T) {
		db := &gokeepasslib.Database{Content: &gokeepasslib.DBContent{Meta: nil}}
		err := kdbx.SaveAuthData(db, testServerURL, testAuthToken)
		require.Error(t, err, "SaveAuthData должен возвращать ошибку при nil Meta")
		assert.Contains(t, err.Error(), "не инициализированы", "Ошибка должна указывать на проблему инициализации")
	})
}

// TestLoadAuthData проверяет загрузку аутентификационных данных.
func TestLoadAuthData(t *testing.T) {
	// Тест на успешную загрузку данных
	t.Run("Success_Load_Auth_Data", func(t *testing.T) {
		db := createTestDatabase()
		require.NotNil(t, db, "База данных должна быть создана")

		// Сохраняем данные аутентификации
		err := kdbx.SaveAuthData(db, testServerURL, testAuthToken)
		require.NoError(t, err, "SaveAuthData не должен возвращать ошибку")

		// Загружаем данные
		url, token, err := kdbx.LoadAuthData(db)
		require.NoError(t, err, "LoadAuthData не должен возвращать ошибку")
		assert.Equal(t, testServerURL, url, "Загруженный URL должен соответствовать сохраненному")
		assert.Equal(t, testAuthToken, token, "Загруженный токен должен соответствовать сохраненному")
	})

	// Тест на загрузку отсутствующих данных
	t.Run("Success_Load_Empty_Auth_Data", func(t *testing.T) {
		db := createTestDatabase()
		require.NotNil(t, db, "База данных должна быть создана")

		// CustomData пуст, данных нет
		url, token, err := kdbx.LoadAuthData(db)
		require.NoError(t, err, "LoadAuthData не должен возвращать ошибку для пустых данных")
		assert.Empty(t, url, "URL должен быть пустым")
		assert.Empty(t, token, "Токен должен быть пустым")
	})

	// Тест с некорректной базой данных
	t.Run("Error_With_Nil_Database", func(t *testing.T) {
		url, token, err := kdbx.LoadAuthData(nil)
		require.Error(t, err, "LoadAuthData должен возвращать ошибку при nil базе данных")
		assert.Empty(t, url, "URL должен быть пустым при ошибке")
		assert.Empty(t, token, "Токен должен быть пустым при ошибке")
		assert.Contains(t, err.Error(), "не инициализированы", "Ошибка должна указывать на проблему инициализации")
	})

	// Тест с nil Content
	t.Run("Error_With_Nil_Content", func(t *testing.T) {
		db := &gokeepasslib.Database{Content: nil}
		url, token, err := kdbx.LoadAuthData(db)
		require.Error(t, err, "LoadAuthData должен возвращать ошибку при nil Content")
		assert.Empty(t, url, "URL должен быть пустым при ошибке")
		assert.Empty(t, token, "Токен должен быть пустым при ошибке")
		assert.Contains(t, err.Error(), "не инициализированы", "Ошибка должна указывать на проблему инициализации")
	})

	// Тест с nil Meta
	t.Run("Error_With_Nil_Meta", func(t *testing.T) {
		db := &gokeepasslib.Database{Content: &gokeepasslib.DBContent{Meta: nil}}
		url, token, err := kdbx.LoadAuthData(db)
		require.Error(t, err, "LoadAuthData должен возвращать ошибку при nil Meta")
		assert.Empty(t, url, "URL должен быть пустым при ошибке")
		assert.Empty(t, token, "Токен должен быть пустым при ошибке")
		assert.Contains(t, err.Error(), "не инициализированы", "Ошибка должна указывать на проблему инициализации")
	})
}
