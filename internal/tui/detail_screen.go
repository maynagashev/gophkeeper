package tui

import (
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tobischo/gokeepasslib/v3"
)

// updateEntryDetailScreen обрабатывает сообщения для экрана деталей записи.
func (m *model) updateEntryDetailScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEsc, keyBack:
			m.state = entryListScreen
			m.selectedEntry = nil // Сбрасываем выбранную запись
			slog.Info("Возврат к списку записей")
			return m, tea.ClearScreen
		case keyEdit:
			if m.selectedEntry != nil {
				m.prepareEditScreen()
				m.state = entryEditScreen
				slog.Info("Переход к редактированию записи", "title", m.selectedEntry.Title())
				return m, tea.ClearScreen
			}
		}
	}
	return m, nil
}

// renderStandardFields форматирует стандартные поля записи для отображения.
// Он также удаляет отображенные поля из переданной карты valuesMap.
func renderStandardFields(_ gokeepasslib.Entry, valuesMap map[string]gokeepasslib.ValueData) string {
	s := ""
	desiredOrder := []string{"Title", "UserName", "Password", "URL", "Notes"}
	// Не создаем новую карту, используем переданную

	for _, key := range desiredOrder {
		if val, ok := valuesMap[key]; ok {
			if val.Key == fieldNamePassword {
				s += fmt.Sprintf("%s: ********\n", val.Key)
			} else {
				s += fmt.Sprintf("%s: %s\n", val.Key, val.Value.Content)
			}
			// Удаляем из карты, чтобы она осталась для renderCustomFields
			delete(valuesMap, key)
		} else {
			s += fmt.Sprintf("%s: \n", key)
		}
	}
	return s
}

// renderCustomFields форматирует дополнительные (нестандартные) поля записи.
func renderCustomFields(valuesMap map[string]gokeepasslib.ValueData) string {
	if len(valuesMap) == 0 {
		return ""
	}
	s := "\n--- Дополнительные поля ---\n"
	// Перебираем оставшиеся в карте поля
	for _, val := range valuesMap {
		s += fmt.Sprintf("%s: %s\n", val.Key, val.Value.Content)
	}
	return s
}

// getBinariesFromDB возвращает срез бинарных данных из базы данных,
// учитывая версию формата KDBX.
// Возвращает срез и bool (true, если путь к данным корректен, даже если срез пуст).
func getBinariesFromDB(db *gokeepasslib.Database) ([]gokeepasslib.Binary, bool) {
	if db == nil || db.Content == nil || db.Header == nil {
		slog.Warn("Cannot get binaries: db, db.Content, or db.Header is nil")
		return nil, false
	}

	if db.Header.IsKdbx4() {
		// KDBX v4
		if db.Content.InnerHeader == nil {
			slog.Warn("Cannot get KDBX4 binaries: db.Content.InnerHeader is nil")
			return nil, false // Структура некорректна
		}
		return db.Content.InnerHeader.Binaries, true
	}

	// KDBX v3 (если не KDBX4)
	if db.Content.Meta == nil {
		slog.Warn("Cannot get KDBX3 binaries: db.Content.Meta is nil")
		return nil, false // Структура некорректна
	}
	return db.Content.Meta.Binaries, true
}

// renderAttachments форматирует список вложений для отображения.
func renderAttachments(db *gokeepasslib.Database, entry gokeepasslib.Entry) string {
	if len(entry.Binaries) == 0 {
		return ""
	}
	s := "\n--- Вложения ---\n"
	binaries, pathOk := getBinariesFromDB(db)

	// Определяем путь к бинарным данным в зависимости от версии KDBX
	if !pathOk {
		// Если структура БД некорректна, просто выводим имена без размера
		for _, binaryRef := range entry.Binaries {
			s += fmt.Sprintf("- %s (размер неизвестен)\n", binaryRef.Name)
		}
		return s
	}

	if len(binaries) > 0 {
		// Создаем map длин, если бинарные данные найдены
		binaryLens := make(map[int]int)
		for _, bin := range binaries {
			binaryLens[bin.ID] = len(bin.Content)
		}
		// Выводим вложения с размерами
		for _, binaryRef := range entry.Binaries {
			if binaryLen, ok := binaryLens[binaryRef.Value.ID]; ok {
				s += fmt.Sprintf("- %s (%d байт)\n", binaryRef.Name, binaryLen)
			} else {
				s += fmt.Sprintf("- %s (данные не найдены!)\n", binaryRef.Name)
				slog.Warn("Binary content not found in DB for reference", "name", binaryRef.Name, "ref_id", binaryRef.Value.ID)
			}
		}
	} else {
		// Если путь к данным корректен, но самих данных нет (например, пустая база)
		slog.Warn("No binary content found at the expected path, although DB structure seems ok.")
		for _, binaryRef := range entry.Binaries {
			s += fmt.Sprintf("- %s (размер неизвестен)\n", binaryRef.Name)
		}
	}
	return s
}

// viewEntryDetailScreen отрисовывает экран деталей записи.
func (m model) viewEntryDetailScreen() string {
	if m.selectedEntry == nil {
		return "Ошибка: Запись не выбрана!"
	}

	s := fmt.Sprintf("Детали записи: %s\n\n", m.selectedEntry.Title())

	// Собираем все значения в map для дальнейшего использования
	valuesMap := make(map[string]gokeepasslib.ValueData)
	for _, val := range m.selectedEntry.entry.Values {
		valuesMap[val.Key] = val
	}

	// Отображаем стандартные поля (и удаляем их из map)
	s += renderStandardFields(m.selectedEntry.entry, valuesMap) // Передаем map

	// Отображаем дополнительные поля (оставшиеся в map)
	s += renderCustomFields(valuesMap)

	// Отображаем вложения
	s += renderAttachments(m.db, m.selectedEntry.entry)

	return s
}
