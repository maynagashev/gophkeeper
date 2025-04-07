package kdbx

import (
	"errors"
	"fmt"
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
	return nil, errors.New("функция CreateDatabase еще не реализована")
}
