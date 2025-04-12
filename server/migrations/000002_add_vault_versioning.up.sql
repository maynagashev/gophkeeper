-- 000002_add_vault_versioning.up.sql
-- Добавление версионирования для хранилищ

BEGIN; -- Начинаем транзакцию

-- 1. Создание таблицы для хранения версий
CREATE TABLE IF NOT EXISTS vault_versions (
    id SERIAL PRIMARY KEY,
    vault_id INTEGER NOT NULL,
    object_key TEXT NOT NULL UNIQUE, -- Ключ объекта в MinIO, должен быть уникальным
    checksum VARCHAR(64) NULL,
    size_bytes BIGINT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_vault
        FOREIGN KEY(vault_id)
        REFERENCES vaults(id)
        ON DELETE CASCADE -- Удаляем версии при удалении основного хранилища
);

-- Индекс для быстрого поиска версий по хранилищу
CREATE INDEX IF NOT EXISTS idx_vault_versions_vault_id ON vault_versions(vault_id);

-- 2. Изменение таблицы vaults
-- Добавляем колонку для ссылки на текущую версию
ALTER TABLE vaults
ADD COLUMN current_version_id INTEGER NULL;

-- Добавляем внешний ключ с ON DELETE SET NULL
-- (Отдельно, чтобы не блокировать таблицу надолго, если она большая)
ALTER TABLE vaults
ADD CONSTRAINT fk_current_version
    FOREIGN KEY(current_version_id)
    REFERENCES vault_versions(id)
    ON DELETE SET NULL; -- Если версию удалят, поле станет NULL

-- Удаляем старые колонки, связанные с файлом (теперь они в vault_versions)
ALTER TABLE vaults
DROP COLUMN IF EXISTS object_key,
DROP COLUMN IF EXISTS checksum,
DROP COLUMN IF EXISTS size_bytes,
DROP COLUMN IF EXISTS last_modified_server;

-- Оставляем updated_at в vaults, т.к. сама запись хранилища (ссылка на версию) может обновляться

COMMIT; -- Завершаем транзакцию 