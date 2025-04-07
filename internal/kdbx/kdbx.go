package kdbx

import (
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

// TODO: Добавить функции для сохранения, добавления, редактирования, удаления записей.
