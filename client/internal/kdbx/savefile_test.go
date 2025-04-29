package kdbx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maynagashev/gophkeeper/client/internal/kdbx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tobischo/gokeepasslib/v3"
)

// Пароль для тестирования.
const saveFileTestPassword = "password123"

func TestSaveFileFunction(t *testing.T) {
	// Создаем тестовую базу данных для сохранения
	testDB := gokeepasslib.NewDatabase()
	testDB.Credentials = gokeepasslib.NewPasswordCredentials(saveFileTestPassword)

	t.Run("Успешное_сохранение_файла", func(t *testing.T) {
		// Создаем временный файл для тестирования
		tempDir := os.TempDir()
		tempFile := filepath.Join(tempDir, "test_save.kdbx")

		// Удаляем файл после завершения теста
		defer os.Remove(tempFile)

		// Сохраняем базу данных
		saveErr := kdbx.SaveFile(testDB, tempFile, saveFileTestPassword)
		require.NoError(t, saveErr, "Ошибка при сохранении файла")

		// Проверяем, что файл был создан
		_, statErr := os.Stat(tempFile)
		require.NoError(t, statErr, "Файл не был создан после сохранения")

		// Пытаемся открыть созданный файл
		loadedDB, openErr := kdbx.OpenFile(tempFile, saveFileTestPassword)
		require.NoError(t, openErr, "Ошибка при открытии сохранённого файла")
		assert.NotNil(t, loadedDB, "Загруженная база данных не должна быть nil")
	})

	t.Run("Ошибка_при_nil_базе_данных", func(t *testing.T) {
		tempFile := filepath.Join(os.TempDir(), "nil_db_test.kdbx")

		// Сохраняем nil базу данных
		saveErr := kdbx.SaveFile(nil, tempFile, saveFileTestPassword)
		require.Error(t, saveErr, "Должна возникнуть ошибка при nil базе данных")
		assert.Contains(t, saveErr.Error(), "база данных не инициализирована",
			"Ошибка должна указывать на неинициализированную базу")
	})

	t.Run("Ошибка_при_пустом_пароле", func(t *testing.T) {
		tempFile := filepath.Join(os.TempDir(), "empty_pass_test.kdbx")

		// Создаем БД без учетных данных
		db := gokeepasslib.NewDatabase()
		db.Credentials = nil // Явно обнуляем учетные данные

		// Пытаемся сохранить с пустым паролем
		saveErr := kdbx.SaveFile(db, tempFile, "")
		require.Error(t, saveErr, "Должна возникнуть ошибка при пустом пароле")
		assert.Contains(t, saveErr.Error(), "пароль не может быть пустым",
			"Ошибка должна указывать на пустой пароль")
	})

	t.Run("Ошибка_записи_в_недоступный_каталог", func(t *testing.T) {
		// Путь к несуществующему каталогу
		invalidPath := "/non/existent/directory/test.kdbx"

		// Пытаемся сохранить в недоступный каталог
		saveErr := kdbx.SaveFile(testDB, invalidPath, saveFileTestPassword)
		require.Error(t, saveErr, "Должна возникнуть ошибка при записи в недоступный каталог")
		assert.Contains(t, saveErr.Error(), "ошибка создания/открытия файла",
			"Ошибка должна указывать на проблему с созданием файла")
	})
}
