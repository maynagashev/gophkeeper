-- 000001_init_schema.up.sql
-- Создание начальной схемы БД

-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Триггер для обновления updated_at в users
CREATE TRIGGER set_users_timestamp
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Таблица метаданных хранилищ
CREATE TABLE IF NOT EXISTS vaults (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    object_key TEXT UNIQUE NOT NULL,
    checksum VARCHAR(64) NULL, -- Может быть NULL, если еще не посчитана
    size_bytes BIGINT NULL,     -- Может быть NULL
    last_modified_server TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE -- Удаляем метаданные при удалении пользователя
);

-- Индекс для быстрого поиска хранилищ по пользователю
CREATE INDEX IF NOT EXISTS idx_vaults_user_id ON vaults(user_id);

-- Триггер для обновления updated_at в vaults
CREATE TRIGGER set_vaults_timestamp
BEFORE UPDATE ON vaults
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp(); 