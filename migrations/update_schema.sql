-- Добавление столбцов state и available_at в таблицу users
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS state VARCHAR(50),
ADD COLUMN IF NOT EXISTS available_at TIMESTAMP;

-- Добавление столбца type в таблицу tasks
ALTER TABLE tasks 
ADD COLUMN IF NOT EXISTS type VARCHAR(50);

-- Создание таблицы temp_data
CREATE TABLE IF NOT EXISTS temp_data (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    key VARCHAR(255) NOT NULL,
    value TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, key)
);

-- Создание индекса по типу в таблице tasks
CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);
