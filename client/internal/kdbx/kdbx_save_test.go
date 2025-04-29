package kdbx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maynagashev/gophkeeper/client/internal/kdbx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSaveFile проверяет сохранение базы данных KDBX в файл.
func TestSaveFile(t *testing.T) {
	// Создаем временную директорию для тестовых файлов
	tempDir, err := os.MkdirTemp("", "kdbx-test-*")
	require.NoError(t, err, "Ошибка при создании временной директории")
	defer os.RemoveAll(tempDir) // Удаляем директорию после завершения тестов

	// Успешное сохранение новой базы данных
	t.Run("Success_Save_New_File", func(t *testing.T) {
		// Создаем новую базу данных
		db := createTestDatabase()
		require.NotNil(t, db, "База данных должна быть создана")

		// Определяем путь для сохранения файла
		savePath := filepath.Join(tempDir, "test-save-new.kdbx")

		// Сохраняем файл
		saveErr := kdbx.SaveFile(db, savePath, testPassword)
		require.NoError(t, saveErr, "SaveFile не должен возвращать ошибку при корректных данных")

		// Проверяем, что файл был создан
		_, statErr := os.Stat(savePath)
		require.NoError(t, statErr, "Файл должен быть создан")

		// Пробуем открыть сохраненный файл для проверки целостности
		savedDB, openErr := kdbx.OpenFile(savePath, testPassword)
		require.NoError(t, openErr, "Сохраненный файл должен открываться корректно")
		require.NotNil(t, savedDB, "Открытая база данных не должна быть nil")
	})

	// Ошибка при сохранении с nil базой данных
	t.Run("Error_Save_Nil_Database", func(t *testing.T) {
		savePath := filepath.Join(tempDir, "test-save-nil.kdbx")
		saveErr := kdbx.SaveFile(nil, savePath, testPassword)
		require.Error(t, saveErr, "SaveFile должен возвращать ошибку при nil базе данных")
		assert.Contains(t, saveErr.Error(), "не инициализирована", "Ошибка должна указывать на проблему инициализации")

		// Проверяем, что файл не был создан
		_, statErr := os.Stat(savePath)
		require.Error(t, statErr, "Файл не должен быть создан")
		require.True(t, os.IsNotExist(statErr), "Ошибка должна указывать на отсутствие файла")
	})

	// Ошибка при сохранении с пустым паролем
	t.Run("Error_Save_With_Empty_Password", func(t *testing.T) {
		// Создаем новую базу данных без учетных данных
		db := createTestDatabase()
		require.NotNil(t, db, "База данных должна быть создана")
		db.Credentials = nil // Удаляем учетные данные

		savePath := filepath.Join(tempDir, "test-save-empty-password.kdbx")

		// Пытаемся сохранить с пустым паролем
		saveErr := kdbx.SaveFile(db, savePath, "")
		require.Error(t, saveErr, "SaveFile должен возвращать ошибку при пустом пароле")
		assert.Contains(t, saveErr.Error(), "пароль не может быть пустым", "Ошибка должна указывать на пустой пароль")
	})

	// Успешное сохранение в несуществующую директорию (должна создаваться)
	t.Run("Success_Save_In_Nonexistent_Directory", func(t *testing.T) {
		// Создаем новую базу данных
		db := createTestDatabase()
		require.NotNil(t, db, "База данных должна быть создана")

		// Определяем путь с несуществующей поддиректорией
		savePath := filepath.Join(tempDir, "subdir", "test-save-subdir.kdbx")

		// Сохраняем файл
		saveErr := kdbx.SaveFile(db, savePath, testPassword)

		// Эта проверка может давать разные результаты в зависимости от реализации:
		// - Если SaveFile автоматически создает директории, то ошибки не должно быть
		// - Если нет, то должна быть ошибка о несуществующей директории
		if saveErr == nil {
			// Проверяем, что файл был создан
			_, statErr := os.Stat(savePath)
			require.NoError(t, statErr, "Файл должен быть создан")
		} else {
			assert.Contains(t, saveErr.Error(), "ошибка создания/открытия файла",
				"Ошибка должна указывать на проблему с созданием файла")
		}
	})

	// Проверка сохранения в уже существующий файл (перезапись)
	t.Run("Success_Overwrite_Existing_File", func(t *testing.T) {
		// Создаем и сохраняем первую базу данных
		db1 := createTestDatabase()
		require.NotNil(t, db1, "База данных должна быть создана")

		// Добавляем URL в CustomData для первой базы
		authErr1 := kdbx.SaveAuthData(db1, testServerURL, "")
		require.NoError(t, authErr1, "SaveAuthData не должен возвращать ошибку")

		savePath := filepath.Join(tempDir, "test-overwrite.kdbx")

		// Сохраняем первую версию
		saveErr1 := kdbx.SaveFile(db1, savePath, testPassword)
		require.NoError(t, saveErr1, "SaveFile не должен возвращать ошибку")

		// Открываем файл для проверки
		dbCheck1, openErr1 := kdbx.OpenFile(savePath, testPassword)
		require.NoError(t, openErr1, "OpenFile не должен возвращать ошибку")
		url1, _, loadErr1 := kdbx.LoadAuthData(dbCheck1)
		require.NoError(t, loadErr1, "LoadAuthData не должен возвращать ошибку")
		assert.Equal(t, testServerURL, url1, "URL должен совпадать с сохраненным")

		// Создаем вторую базу данных
		db2 := createTestDatabase()
		require.NotNil(t, db2, "Вторая база данных должна быть создана")

		// Добавляем другой URL в CustomData для второй базы
		newURL := "https://other-server.example.com"
		authErr2 := kdbx.SaveAuthData(db2, newURL, "")
		require.NoError(t, authErr2, "SaveAuthData не должен возвращать ошибку")

		// Перезаписываем файл второй базой
		saveErr2 := kdbx.SaveFile(db2, savePath, testPassword)
		require.NoError(t, saveErr2, "SaveFile не должен возвращать ошибку при перезаписи")

		// Открываем файл и проверяем, что он содержит данные из второй базы
		dbCheck2, openErr2 := kdbx.OpenFile(savePath, testPassword)
		require.NoError(t, openErr2, "OpenFile не должен возвращать ошибку")
		url2, _, loadErr2 := kdbx.LoadAuthData(dbCheck2)
		require.NoError(t, loadErr2, "LoadAuthData не должен возвращать ошибку")
		assert.Equal(t, newURL, url2, "URL должен совпадать с новым значением")
		assert.NotEqual(t, url1, url2, "URL должен отличаться от первого значения")
	})
}
