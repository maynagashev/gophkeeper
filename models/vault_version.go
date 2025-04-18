package models

import "time"

// VaultVersion представляет конкретную версию файла хранилища и его метаданные.
type VaultVersion struct {
	ID        int64     `db:"id" json:"id"`
	VaultID   int64     `db:"vault_id" json:"vault_id"`
	ObjectKey string    `db:"object_key" json:"object_key"`
	Checksum  *string   `db:"checksum" json:"checksum,omitempty"`
	SizeBytes *int64    `db:"size_bytes" json:"size_bytes,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
