package kdbx

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tobischo/gokeepasslib/v3"
)

// Константы для тестирования.
const (
	testPassword    = "test"
	invalidPassword = "wrongpassword"
)

// Вспомогательная функция для получения пути относительно корня проекта.
func getProjectPath(relPath string) string {
	_, filename, _, _ := runtime.Caller(0)
	// Путь к директории пакета kdbx
	kdbxDir := path.Dir(filename)
	// Поднимаемся на два уровня вверх (internal/kdbx -> /), чтобы получить корень проекта
	projectRoot := filepath.Join(kdbxDir, "..", "..")
	return filepath.Join(projectRoot, relPath)
}

var (
	testFilePath    = getProjectPath("example/test.kdbx") // Абсолютный путь к тестовому файлу
	nonExistentPath = getProjectPath("testdata/nonexistent.kdbx")
)

// skipIfTestFileNotExist пропускает тест, если тестовый файл не существует.
func skipIfTestFileNotExist(t *testing.T) {
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Skipf("Тестовый файл не найден по пути %s, пропускаем тест", testFilePath)
	}
}

func TestOpenFile(t *testing.T) {
	// Проверяем доступность тестового файла
	skipIfTestFileNotExist(t)

	// Тест 1: Успешное открытие файла с правильным паролем
	t.Run("Success_With_Correct_Password", func(t *testing.T) {
		db, err := OpenFile(testFilePath, testPassword)
		require.NoError(t, err, "Ошибка при открытии файла с правильным паролем")
		require.NotNil(t, db, "База данных не должна быть nil")

		// Дополнительная проверка: бд должна содержать какие-то записи
		entries := GetAllEntries(db)
		assert.NotEmpty(t, entries, "База данных должна содержать как минимум одну запись")
	})

	// Тест 2: Ошибка при неправильном пароле
	t.Run("Error_With_Incorrect_Password", func(t *testing.T) {
		db, err := OpenFile(testFilePath, invalidPassword)
		require.Error(t, err, "Должна возникнуть ошибка при неправильном пароле")
		assert.Nil(t, db, "База данных должна быть nil при ошибке")
		assert.Contains(t, err.Error(), "ошибка дешифрования", "Ошибка должна указывать на проблему с дешифрованием")
	})

	// Тест 3: Ошибка при несуществующем файле
	t.Run("Error_With_Nonexistent_File", func(t *testing.T) {
		db, err := OpenFile(nonExistentPath, testPassword)
		require.Error(t, err, "Должна возникнуть ошибка при несуществующем файле")
		assert.Nil(t, db, "База данных должна быть nil при ошибке")
		assert.Contains(t, err.Error(), "ошибка открытия файла", "Ошибка должна указывать на проблему с открытием файла")
	})
}

func TestGetAllEntries(t *testing.T) {
	// Проверяем доступность тестового файла
	skipIfTestFileNotExist(t)

	// Тест 1: Успешное получение записей из базы данных
	t.Run("Success_Get_Entries", func(t *testing.T) {
		// Сначала открываем базу данных
		db, err := OpenFile(testFilePath, testPassword)
		require.NoError(t, err, "Ошибка при открытии тестового файла")

		// Получаем все записи
		entries := GetAllEntries(db)
		assert.NotEmpty(t, entries, "Список записей не должен быть пустым")
	})

	// Тест 2: Корректная обработка nil-значений
	t.Run("Handle_Nil_Database", func(t *testing.T) {
		entries := GetAllEntries(nil)
		assert.Empty(t, entries, "При nil базе данных должен возвращаться пустой список")

		// Тест с nil Content
		db := &gokeepasslib.Database{Content: nil}
		entries = GetAllEntries(db)
		assert.Empty(t, entries, "При nil Content должен возвращаться пустой список")

		// Тест с nil Root
		db.Content = &gokeepasslib.DBContent{Root: nil}
		entries = GetAllEntries(db)
		assert.Empty(t, entries, "При nil Root должен возвращаться пустой список")
	})
}

// TestCreateDatabase тестирует функцию создания новой базы данных KDBX.
// Эта функция ещё не реализована, но тест служит спецификацией для её реализации.
func TestCreateDatabase(t *testing.T) {
	// Создаем временный файл для тестирования
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "test_create_db.kdbx")

	// Удаляем файл после завершения теста
	defer os.Remove(tempFile)

	// Проверяем, что функция еще не реализована (должна вернуть ошибку)
	// После реализации функции этот тест нужно будет изменить
	t.Run("Create_New_Database", func(t *testing.T) {
		// Создаем новую базу данных с паролем
		newPassword := "new_password"
		_, err := CreateDatabase(tempFile, newPassword)

		// Пока функция не реализована, ожидаем ошибку
		// После реализации функции, этот assert нужно будет заменить на require.NoError
		assert.Error(t, err, "Функция CreateDatabase еще не реализована")

		// После реализации:
		// require.NoError(t, err, "Ошибка при создании новой базы данных")

		// Проверяем, что файл был создан
		// _, err = os.Stat(tempFile)
		// assert.NoError(t, err, "Файл не был создан")

		// Пробуем открыть файл
		// db, err := OpenFile(tempFile, newPassword)
		// assert.NoError(t, err, "Ошибка при открытии созданного файла")
		// assert.NotNil(t, db, "База данных не должна быть nil")
	})
}
