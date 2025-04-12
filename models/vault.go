package models

import "time"

// Vault представляет основную запись о хранилище KDBX пользователя.
// Содержит ссылку на текущую активную версию метаданных и файла.
type Vault struct {
	ID               int64     `db:"id" json:"id"`
	UserID           int64     `db:"user_id" json:"user_id"`
	CurrentVersionID *int64    `db:"current_version_id" json:"current_version_id,omitempty"` // может быть NULL
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}
