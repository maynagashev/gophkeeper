package kdbx

import (
	"errors"
	"log/slog"

	"github.com/tobischo/gokeepasslib/v3"
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
