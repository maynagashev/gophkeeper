package models

import "time"

// Vault представляет метаданные хранилища KDBX.
// Тэги `db` используются для маппинга с полями БД с помощью sqlx.
// Тэги `json` используются для (де)сериализации JSON.
type Vault struct {
	ID                 int64     `db:"id" json:"id"`
	UserID             int64     `db:"user_id" json:"user_id"`
	ObjectKey          string    `db:"object_key" json:"object_key"`
	Checksum           *string   `db:"checksum" json:"checksum,omitempty"`     // Указатель, т.к. может быть NULL
	SizeBytes          *int64    `db:"size_bytes" json:"size_bytes,omitempty"` // Указатель, т.к. может быть NULL
	LastModifiedServer time.Time `db:"last_modified_server" json:"last_modified_server"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
}
