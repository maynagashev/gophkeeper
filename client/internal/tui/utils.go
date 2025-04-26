package tui

import (
	"github.com/tobischo/gokeepasslib/v3"
)

// DeepCopyEntry создает глубокую копию записи.
func DeepCopyEntry(original gokeepasslib.Entry) gokeepasslib.Entry {
	newEntry := gokeepasslib.NewEntry()

	// Копируем UUID
	copy(newEntry.UUID[:], original.UUID[:])

	// Копируем основные поля
	newEntry.Times = original.Times
	newEntry.Tags = original.Tags
	newEntry.CustomData = original.CustomData

	// Глубокое копирование Values
	if original.Values != nil {
		newEntry.Values = make([]gokeepasslib.ValueData, len(original.Values))
		for i, val := range original.Values {
			newValue := gokeepasslib.ValueData{
				Key:   val.Key,
				Value: gokeepasslib.V{Content: val.Value.Content, Protected: val.Value.Protected},
			}
			newEntry.Values[i] = newValue
		}
	}

	// Копируем срез ссылок на бинарные данные (BinaryReference)
	// Сами структуры BinaryReference содержат простые типы (Name, Value.ID),
	// поэтому достаточно поверхностного копирования среза.
	if original.Binaries != nil {
		newEntry.Binaries = make([]gokeepasslib.BinaryReference, len(original.Binaries))
		copy(newEntry.Binaries, original.Binaries)
	}

	// TODO: Добавить копирование History, если оно будет редактироваться

	return newEntry
}

// deepCopyEntry - алиас для совместимости со старым кодом.
// Использует публичную функцию DeepCopyEntry.
func deepCopyEntry(original gokeepasslib.Entry) gokeepasslib.Entry {
	return DeepCopyEntry(original)
}

// FindEntryInDB ищет запись по UUID в базе данных.
func FindEntryInDB(db *gokeepasslib.Database, uuid gokeepasslib.UUID) *gokeepasslib.Entry {
	if db == nil || db.Content == nil || db.Content.Root == nil {
		return nil
	}
	return FindEntryInGroups(db.Content.Root.Groups, uuid)
}

// findEntryInDB - алиас для совместимости со старым кодом.
// Использует публичную функцию FindEntryInDB.
func findEntryInDB(db *gokeepasslib.Database, uuid gokeepasslib.UUID) *gokeepasslib.Entry {
	return FindEntryInDB(db, uuid)
}

// FindEntryInGroups рекурсивно ищет запись по UUID.
func FindEntryInGroups(groups []gokeepasslib.Group, uuid gokeepasslib.UUID) *gokeepasslib.Entry {
	for i := range groups {
		group := &groups[i]
		// Поиск в текущей группе
		for j := range group.Entries {
			entry := &group.Entries[j]
			if entry.UUID == uuid {
				return entry
			}
		}
		// Поиск в подгруппах
		if entry := FindEntryInGroups(group.Groups, uuid); entry != nil {
			return entry
		}
	}
	return nil
}
