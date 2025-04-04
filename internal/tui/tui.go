package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Модель представляет состояние TUI приложения.
// На начальном этапе она будет очень простой.
type model struct {
	// TODO: Добавить поля для хранения состояния (например, текущий экран, данные)
}

// initialModel создает начальное состояние модели.
func initialModel() model {
	return model{
		// TODO: Инициализировать начальное состояние
	}
}

// Init - команда, выполняемая при запуске приложения.
// Пока не требуется никаких начальных команд.
func (m model) Init() tea.Cmd {
	return nil
}

// Update обрабатывает входящие сообщения (события клавиатуры, мыши, команды).
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Обработка нажатия клавиш
	case tea.KeyMsg:
		switch msg.String() {
		// Выход из приложения по Ctrl+C
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	// Если сообщение не обработано, возвращаем текущую модель и nil команду
	return m, nil
}

// View отрисовывает пользовательский интерфейс.
func (m model) View() string {
	// TODO: Реализовать отображение разных экранов (приветствие, ввод пароля, список записей)
	// Приветственное сообщение с кратким описанием
	s := "Добро пожаловать в GophKeeper!\n\n"
	s += "Это безопасный менеджер паролей для командной строки,\n"
	s += "совместимый с форматом KDBX (KeePass).\n\n"
	s += "Нажмите Ctrl+C или q для выхода.\n"
	return s
}

// Start запускает TUI приложение.
func Start() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Ошибка при запуске TUI: %v", err)
		os.Exit(1)
	}
}
