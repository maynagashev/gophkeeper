-- 000002_add_vault_versioning.down.sql
-- Откат версионирования хранилищ
-- ВНИМАНИЕ: Этот откат приведет к потере данных о версиях!

BEGIN;

-- 1. Восстановление (примерное) таблицы vaults
-- Добавляем обратно колонки (без данных)
ALTER TABLE vaults
ADD COLUMN object_key TEXT NULL,
ADD COLUMN checksum VARCHAR(64) NULL,
ADD COLUMN size_bytes BIGINT NULL,
ADD COLUMN last_modified_server TIMESTAMPTZ NULL;
-- Сделаем object_key снова UNIQUE и NOT NULL, если это требуется
-- ALTER TABLE vaults ADD CONSTRAINT vaults_object_key_key UNIQUE (object_key);
-- ALTER TABLE vaults ALTER COLUMN object_key SET NOT NULL;
-- (Пропускаем для простоты отката)

-- Удаляем внешний ключ и колонку current_version_id
ALTER TABLE vaults DROP CONSTRAINT IF EXISTS fk_current_version;
ALTER TABLE vaults DROP COLUMN IF EXISTS current_version_id;

-- 2. Удаление таблицы vault_versions
DROP TABLE IF EXISTS vault_versions;

COMMIT; 