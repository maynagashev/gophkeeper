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

// viewEntryDetailScreen отрисовывает экран деталей записи.
func (m model) viewEntryDetailScreen() string {
	if m.selectedEntry == nil {
		return "Ошибка: Запись не выбрана!" // Такого не должно быть, но на всякий случай
	}

	// Определяем желаемый порядок полей
	desiredOrder := []string{"Title", "UserName", "Password", "URL", "Notes"}
	// Собираем значения в map для быстрого доступа
	valuesMap := make(map[string]gokeepasslib.ValueData)
	for _, val := range m.selectedEntry.entry.Values {
		valuesMap[val.Key] = val
	}

	s := fmt.Sprintf("Детали записи: %s\n\n", m.selectedEntry.Title())

	// Выводим поля в заданном порядке
	for _, key := range desiredOrder {
		if val, ok := valuesMap[key]; ok {
			// Пока не будем показывать пароли
			if val.Key == fieldNamePassword {
				s += fmt.Sprintf("%s: ********\n", val.Key)
			} else {
				s += fmt.Sprintf("%s: %s\n", val.Key, val.Value.Content)
			}
			// Удаляем из карты, чтобы потом вывести оставшиеся (нестандартные) поля
			delete(valuesMap, key)
		} else {
			// Если поля нет в записи, можно вывести прочерк или ничего
			s += fmt.Sprintf("%s: \n", key)
		}
	}

	// Выводим остальные (нестандартные) поля, если они есть
	if len(valuesMap) > 0 {
		s += "\n--- Дополнительные поля ---\n"
		for _, val := range m.selectedEntry.entry.Values {
			if _, existsInMap := valuesMap[val.Key]; existsInMap {
				s += fmt.Sprintf("%s: %s\n", val.Key, val.Value.Content)
			}
		}
	}

	// s += "\n(e - ред., Ctrl+S - сохр., Esc/b - назад)" // Убрали, т.к. добавляется в View
	return s
}
