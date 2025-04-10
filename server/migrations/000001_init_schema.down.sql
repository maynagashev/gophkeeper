-- 000001_init_schema.down.sql
-- Откат начальной схемы БД

-- Удаляем триггеры
DROP TRIGGER IF EXISTS set_vaults_timestamp ON vaults;
DROP TRIGGER IF EXISTS set_users_timestamp ON users;

-- Удаляем таблицы (в обратном порядке создания из-за внешних ключей)
DROP TABLE IF EXISTS vaults;
DROP TABLE IF EXISTS users;

-- Удаляем функцию
DROP FUNCTION IF EXISTS trigger_set_timestamp(); 