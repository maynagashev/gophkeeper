-- 000003_add_content_mod_time.up.sql
-- Добавление колонки для хранения времени модификации контента KDBX

BEGIN;

ALTER TABLE vault_versions
ADD COLUMN content_modified_at TIMESTAMPTZ NULL;

COMMENT ON COLUMN vault_versions.content_modified_at IS 'Время последнего изменения контента KDBX (из Root.LastModificationTime на момент загрузки)';

COMMIT; 