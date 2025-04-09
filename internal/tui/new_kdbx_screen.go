package tui

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tobischo/gokeepasslib/v3"
)

// var errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // Красный цвет для ошибок (перенесено в view)

// updateNewKdbxPasswordScreen обрабатывает сообщения для экрана ввода пароля нового KDBX.
//
//nolint:gocognit,funlen // Сложность и длина будут снижены при рефакторинге
func (m *model) updateNewKdbxPasswordScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	//nolint:nestif // Сложность вложенности будет снижена при рефакторинге
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c", keyEsc: // Выход из приложения, если мы еще не создали файл
			return m, tea.Quit

		case keyTab, keyShiftTab, keyUp, keyDown:
			m.newPasswordFocusedField = (m.newPasswordFocusedField + 1) % numNewPasswordFields // Переключаем фокус
			m.confirmPasswordError = ""                                                        // Сбрасываем ошибку

			if m.newPasswordFocusedField == 0 {
				m.newPasswordInput1.Focus()
				m.newPasswordInput2.Blur()
				cmd = textinput.Blink
			} else {
				m.newPasswordInput1.Blur()
				m.newPasswordInput2.Focus()
				cmd = textinput.Blink
			}
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)

		case keyEnter:
			pass1 := m.newPasswordInput1.Value()
			pass2 := m.newPasswordInput2.Value()

			if pass1 == "" {
				m.confirmPasswordError = "Пароль не может быть пустым!"
				return m, nil
			}
			if pass1 != pass2 {
				m.confirmPasswordError = "Пароли не совпадают!"
				m.newPasswordInput1.SetValue("") // Очищаем поля
				m.newPasswordInput2.SetValue("")
				m.newPasswordInput1.Focus() // Возвращаем фокус на первое поле
				m.newPasswordInput2.Blur()
				m.newPasswordFocusedField = 0
				return m, textinput.Blink
			}

			// Пароли совпадают, создаем новую базу
			m.confirmPasswordError = "" // Сбрасываем ошибку
			m.password = pass1          // Сохраняем пароль в модели

			slog.Info("Пароли совпадают, создаем новую базу KDBX", "path", m.kdbxPath)
			// Создаем пустую базу данных
			m.db = gokeepasslib.NewDatabase()
			// По умолчанию gokeepasslib.NewDatabase() создает KDBX 4
			m.db.Content.Meta.DatabaseName = "Новая база" // Можно сделать имя по умолчанию
			// Устанавливаем пароль
			m.db.Credentials = gokeepasslib.NewPasswordCredentials(m.password)

			// Сохраняем новую базу
			file, err := os.Create(m.kdbxPath)
			if err != nil {
				slog.Error("Ошибка создания файла KDBX", "path", m.kdbxPath, "error", err)
				m.confirmPasswordError = fmt.Sprintf("Ошибка создания файла: %v", err)
				return m, nil
			}
			defer file.Close() // Используем defer для гарантированного закрытия

			encoder := gokeepasslib.NewEncoder(file)
			if errEncode := encoder.Encode(m.db); errEncode != nil {
				slog.Error("Ошибка записи в новый файл KDBX", "path", m.kdbxPath, "error", errEncode)
				m.confirmPasswordError = fmt.Sprintf("Ошибка записи в файл: %v", errEncode)
				return m, nil
			}

			slog.Info("Новый файл KDBX успешно создан и сохранен", "path", m.kdbxPath)
			// Переходим к списку (он будет пуст)
			m.state = entryListScreen
			// Инициализируем список (может быть пустым)
			m.entryList.SetItems([]list.Item{}) // Убедимся, что список пуст
			m.entryList.Title = fmt.Sprintf("Записи (%s)", m.kdbxPath)
			return m, tea.ClearScreen // Очищаем экран перед показом списка
		}

		// Обновляем активное поле ввода, если это не спец. клавиша
		if m.newPasswordFocusedField == 0 {
			m.newPasswordInput1, cmd = m.newPasswordInput1.Update(msg)
		} else {
			m.newPasswordInput2, cmd = m.newPasswordInput2.Update(msg)
		}
		cmds = append(cmds, cmd)
	} // else if msg.(type) is not tea.KeyMsg, do nothing? Maybe return m, nil?

	return m, tea.Batch(cmds...)
}

// viewNewKdbxPasswordScreen отрисовывает экран ввода пароля для нового KDBX.
func (m *model) viewNewKdbxPasswordScreen() string {
	// Определяем стиль ошибки локально
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	s := "Создание нового файла KDBX: " + m.kdbxPath + "\n\n"
	s += m.newPasswordInput1.View() + "\n"
	s += m.newPasswordInput2.View() + "\n\n"

	if m.confirmPasswordError != "" {
		s += errorStyle.Render(m.confirmPasswordError) + "\n"
	}

	return s
}
