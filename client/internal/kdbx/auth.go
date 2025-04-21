package kdbx

import (
	"errors"
	"log/slog"
	"time"

	"github.com/tobischo/gokeepasslib/v3"
	"github.com/tobischo/gokeepasslib/v3/wrappers"
)

const (
	// CustomDataKeyServerURL - ключ для хранения URL сервера в KDBX.
	CustomDataKeyServerURL = "GophKeeperServerURL"
	// CustomDataKeyAuthToken - ключ для хранения JWT токена в KDBX.
	CustomDataKeyAuthToken = "GophKeeperAuthToken" //nolint:gosec // Это имя ключа, а не сам токен
)

// setCustomDataValue обновляет или добавляет значение в слайс CustomData.
// Возвращает обновленный слайс.
func setCustomDataValue(customDataSlice []gokeepasslib.CustomData, key, value string) []gokeepasslib.CustomData {
	found := false
	for i := range customDataSlice {
		if customDataSlice[i].Key == key {
			customDataSlice[i].Value = value
			found = true
			slog.Debug("Обновлено значение CustomData", "key", key)
			break
		}
	}
	if !found {
		customDataSlice = append(customDataSlice, gokeepasslib.CustomData{
			Key:   key,
			Value: value,
		})
		slog.Debug("Добавлено новое значение CustomData", "key", key)
	}
	return customDataSlice
}

// removeCustomDataValue удаляет значение из слайса CustomData по ключу.
// Возвращает обновленный слайс.
func removeCustomDataValue(customDataSlice []gokeepasslib.CustomData, key string) []gokeepasslib.CustomData {
	newSlice := make([]gokeepasslib.CustomData, 0, len(customDataSlice))
	removed := false
	for _, item := range customDataSlice {
		if item.Key != key {
			newSlice = append(newSlice, item)
		} else {
			removed = true
		}
	}
	if removed {
		slog.Debug("Удалено значение из CustomData", "key", key)
	}
	return newSlice
}

// SaveAuthData сохраняет URL сервера и токен аутентификации
// в пользовательских данных метаданных базы KDBX.
func SaveAuthData(db *gokeepasslib.Database, serverURL, authToken string) error {
	if db == nil || db.Content == nil || db.Content.Meta == nil {
		return errors.New("база данных, ее содержимое или метаданные не инициализированы")
	}

	meta := db.Content.Meta
	initialCustomData := make([]gokeepasslib.CustomData, len(meta.CustomData)) // Копируем исходные данные
	copy(initialCustomData, meta.CustomData)
	changed := false // Флаг, что данные действительно изменились

	// Сохраняем/удаляем URL
	if serverURL != "" {
		meta.CustomData = setCustomDataValue(meta.CustomData, CustomDataKeyServerURL, serverURL)
	} else {
		meta.CustomData = removeCustomDataValue(meta.CustomData, CustomDataKeyServerURL)
	}

	// Сохраняем/удаляем токен
	if authToken != "" {
		meta.CustomData = setCustomDataValue(meta.CustomData, CustomDataKeyAuthToken, authToken)
	} else {
		meta.CustomData = removeCustomDataValue(meta.CustomData, CustomDataKeyAuthToken)
	}

	// Проверяем, изменился ли слайс CustomData
	if len(initialCustomData) != len(meta.CustomData) {
		changed = true
	} else {
		// Если длина одинаковая, проверяем содержимое
		for i := range meta.CustomData {
			if initialCustomData[i].Key != meta.CustomData[i].Key || initialCustomData[i].Value != meta.CustomData[i].Value {
				changed = true
				break
			}
		}
	}

	// Обновляем время модификации корневой группы, только если CustomData действительно изменились
	if changed {
		now := time.Now().UTC()
		// Обновляем время модификации корневой группы
		if db.Content != nil && db.Content.Root != nil && len(db.Content.Root.Groups) > 0 {
			// Получаем указатель на корневую группу
			rootGroup := &db.Content.Root.Groups[0]
			// Тип Times не указатель, поэтому присваиваем обертку напрямую
			// Создаем wrappers.TimeWrapper и присваиваем указатель на него
			modTimeWrapper := wrappers.TimeWrapper{Time: now}      // Создаем экземпляр
			rootGroup.Times.LastModificationTime = &modTimeWrapper // Присваиваем указатель
			slog.Debug("Обновлено LastModificationTime корневой группы", "newTime", now)
		} else {
			// Логируем, если Content, Root или Groups == nil/пуст
			slog.Warn("Не удалось обновить LastModificationTime корневой группы:" +
				" db.Content, Root или Groups == nil/пуст")
		}
	}

	// TODO: Как правильно установить флаг DatabaseChanged? Возможно, не нужно.
	// meta.SetDatabaseChanged(true) // Пока комментируем

	return nil
}

// LoadAuthData извлекает URL сервера и токен аутентификации
// из пользовательских данных метаданных базы KDBX.
func LoadAuthData(db *gokeepasslib.Database) (string, string, error) {
	var serverURL, authToken string
	if db == nil || db.Content == nil || db.Content.Meta == nil {
		return "", "", errors.New("база данных, ее содержимое или метаданные не инициализированы для загрузки AuthData")
	}

	meta := db.Content.Meta
	foundURL := false
	foundToken := false

	slog.Debug("Попытка загрузки Auth данных из CustomData", "count", len(meta.CustomData))

	for _, item := range meta.CustomData {
		switch item.Key {
		case CustomDataKeyServerURL:
			serverURL = item.Value
			foundURL = true
			slog.Debug("Найден URL сервера в CustomData", "key", item.Key, "value", serverURL)
		case CustomDataKeyAuthToken:
			authToken = item.Value
			foundToken = true
			slog.Debug("Найден токен в CustomData", "key", item.Key)
		}
		if foundURL && foundToken {
			break
		}
	}

	if !foundURL {
		slog.Debug("URL сервера не найден в CustomData KDBX")
	}
	if !foundToken {
		slog.Debug("Токен аутентификации не найден в CustomData KDBX")
	}

	return serverURL, authToken, nil
}
