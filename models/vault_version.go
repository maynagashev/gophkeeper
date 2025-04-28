package models

import "time"

// VaultVersion представляет конкретную версию файла хранилища KDBX.
// Содержит метаданные версии и ключ для доступа к файлу в S3/MinIO.
type VaultVersion struct {
	ID        int64     `db:"id" json:"id"`                 // Уникальный ID версии
	VaultID   int64     `db:"vault_id" json:"vault_id"`     // ID основного хранилища
	ObjectKey string    `db:"object_key" json:"object_key"` // Ключ файла в S3/MinIO
	Checksum  *string   `db:"checksum" json:"checksum"`     // Контрольная сумма (SHA256) файла
	SizeBytes *int64    `db:"size_bytes" json:"size"`       // Размер файла в байтах
	CreatedAt time.Time `db:"created_at" json:"created_at"` // Время создания этой версии на сервере
	// Время последнего изменения *контента* KDBX (из Root.LastModificationTime)
	// Передается клиентом при загрузке.
	ContentModifiedAt *time.Time `db:"content_modified_at" json:"content_modified_at,omitempty"`
}
