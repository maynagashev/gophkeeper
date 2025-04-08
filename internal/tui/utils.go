package tui

import (
	"github.com/tobischo/gokeepasslib/v3"
)

// deepCopyEntry создает глубокую копию записи.
func deepCopyEntry(original gokeepasslib.Entry) gokeepasslib.Entry {
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

	return newEntry
}

// findEntryInDB ищет запись по UUID в базе данных.
func findEntryInDB(db *gokeepasslib.Database, uuid gokeepasslib.UUID) *gokeepasslib.Entry {
	if db == nil || db.Content == nil || db.Content.Root == nil {
		return nil
	}
	return findEntryInGroups(db.Content.Root.Groups, uuid)
}

// findEntryInGroups рекурсивно ищет запись по UUID.
func findEntryInGroups(groups []gokeepasslib.Group, uuid gokeepasslib.UUID) *gokeepasslib.Entry {
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
		if entry := findEntryInGroups(group.Groups, uuid); entry != nil {
			return entry
		}
	}
	return nil
}
