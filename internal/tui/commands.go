package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tobischo/gokeepasslib/v3"

	"github.com/maynagashev/gophkeeper/internal/kdbx"
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
