package kdbx

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/tobischo/gokeepasslib/v3"
)

// TODO: Реализация работы с KDBX файлами

// OpenFile открывает и дешифрует KDBX файл по указанному пути и паролю.
// Возвращает объект базы данных или ошибку.
func OpenFile(filePath string, password string) (*gokeepasslib.Database, error) {
	// Открываем файл
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла '%s': %w", filePath, err)
	}
	defer file.Close()

	// Создаем новую базу данных для декодирования
	db := gokeepasslib.NewDatabase()
	// Устанавливаем учетные данные для дешифровки
	db.Credentials = gokeepasslib.NewPasswordCredentials(password)

	// Декодируем (дешифруем) файл
	err = gokeepasslib.NewDecoder(file).Decode(db)
	if err != nil {
		return nil, fmt.Errorf("ошибка дешифрования файла '%s': %w", filePath, err)
	}

	// Разблокируем защищенные значения (пароли и т.д.)
	err = db.UnlockProtectedEntries()
	if err != nil {
		return nil, fmt.Errorf("ошибка разблокировки защищенных полей: %w", err)
	}

	return db, nil
}

// GetAllEntries рекурсивно обходит все группы и возвращает плоский список всех записей.
func GetAllEntries(db *gokeepasslib.Database) []gokeepasslib.Entry {
	var entries []gokeepasslib.Entry
	if db == nil || db.Content == nil || db.Content.Root == nil {
		return entries
	}
	collectEntries(&entries, db.Content.Root.Groups)
	return entries
}

// collectEntries - вспомогательная рекурсивная функция для сбора записей.
func collectEntries(entries *[]gokeepasslib.Entry, groups []gokeepasslib.Group) {
	for _, group := range groups {
		*entries = append(*entries, group.Entries...)
		collectEntries(entries, group.Groups)
	}
}

// TODO: Добавить функции для сохранения, добавления, редактирования, удаления записей.

// CreateDatabase создает новую базу данных KDBX с указанным паролем.
// Пока это только заглушка для тестов, которая всегда возвращает ошибку.
func CreateDatabase(_ string, _ string) (*gokeepasslib.Database, error) {
	return nil, fmt.Errorf("функция CreateDatabase еще не реализована")
}

// SaveFile кодирует и сохраняет базу данных KDBX в указанный файл.
func SaveFile(db *gokeepasslib.Database, filePath string, password string) error {
	if db == nil {
		return fmt.Errorf("база данных не инициализирована (nil)")
	}

	// Устанавливаем учетные данные, если их нет (нужны для сохранения)
	if db.Credentials == nil {
		if password == "" {
			return fmt.Errorf("пароль не может быть пустым при сохранении")
		}
		db.Credentials = gokeepasslib.NewPasswordCredentials(password)
	}

	// Важно: перед сохранением нужно заблокировать защищенные поля!
	db.LockProtectedEntries()

	// Открываем файл для записи (перезаписываем существующий)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("ошибка создания/открытия файла '%s' для записи: %w", filePath, err)
	}
	defer file.Close()

	// Кодируем и записываем БД в файл
	keepassEncoder := gokeepasslib.NewEncoder(file)
	if err := keepassEncoder.Encode(db); err != nil {
		return fmt.Errorf("ошибка кодирования и записи БД в файл '%s': %w", filePath, err)
	}

	// Разблокируем обратно после сохранения (если нужно продолжить работу)
	// TODO: Решить, нужно ли это делать здесь или после вызова SaveFile
	err = db.UnlockProtectedEntries()
	if err != nil {
		slog.Warn("Не удалось разблокировать поля после сохранения", "error", err)
		// Не возвращаем ошибку, так как сохранение прошло успешно
	}

	return nil // Сохранение успешно
}
