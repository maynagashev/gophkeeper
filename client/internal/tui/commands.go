package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tobischo/gokeepasslib/v3"

	"github.com/maynagashev/gophkeeper/client/internal/kdbx"
)

// openKdbxCmd асинхронно открывает файл базы.
func openKdbxCmd(path, password string) tea.Cmd {
	return func() tea.Msg {
		db, err := kdbx.OpenFile(path, password)
		if err != nil {
			return errMsg{err: err}
		}
		return dbOpenedMsg{db: db}
	}
}

// saveKdbxCmd асинхронно сохраняет файл базы.
func saveKdbxCmd(db *gokeepasslib.Database, path, password string) tea.Cmd {
	return func() tea.Msg {
		err := kdbx.SaveFile(db, path, password)
		if err != nil {
			return dbSaveErrorMsg{err: err}
		}
		return dbSavedMsg{}
	}
}

// clearStatusCmd возвращает команду, которая отправит clearStatusMsg через delay.
func clearStatusCmd(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(_ time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// --- Сообщения и команды для API --- //

type loginSuccessMsg struct {
	Token string
}

type LoginError struct {
	err error
}

func (e LoginError) Error() string {
	return e.err.Error()
}

// makeLoginCmd выполняет вход через API.
func (m *model) makeLoginCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		token, err := m.apiClient.Login(ctx, username, password)
		if err != nil {
			// Возвращаем сообщение об ошибке
			return LoginError{err: fmt.Errorf("ошибка входа: %w", err)}
		}
		// Возвращаем сообщение об успехе с токеном
		return loginSuccessMsg{Token: token}
	}
}

// Сообщения для регистрации.
type registerSuccessMsg struct { // Успешная регистрация не возвращает токен
}

type RegisterError struct {
	err error
}

func (e RegisterError) Error() string {
	return e.err.Error()
}

// makeRegisterCmd выполняет регистрацию через API.
func (m *model) makeRegisterCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.apiClient.Register(ctx, username, password)
		if err != nil {
			return RegisterError{err: fmt.Errorf("ошибка регистрации: %w", err)}
		}
		return registerSuccessMsg{}
	}
}
