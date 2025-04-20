-- 000003_add_content_mod_time.down.sql
-- Удаление колонки для хранения времени модификации контента KDBX

BEGIN;

ALTER TABLE vault_versions
DROP COLUMN IF EXISTS content_modified_at;

COMMIT; 