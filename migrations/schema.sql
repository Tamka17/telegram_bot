-- migrations/schema.sql

-- Создание таблицы пользователей
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    balance DECIMAL(10, 2) DEFAULT 0,
    state VARCHAR(50), -- Добавлено для отслеживания состояния пользователя
    available_at TIMESTAMP, -- Добавлено для хранения времени доступности
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы заданий
CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL PRIMARY KEY,
    description TEXT NOT NULL,
    link VARCHAR(255),
    type VARCHAR(50), -- Добавлено для классификации типов заданий
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы пользовательских заданий
CREATE TABLE IF NOT EXISTS user_tasks (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    task_id INTEGER REFERENCES tasks(id),
    status VARCHAR(50) DEFAULT 'pending',
    screenshots JSONB, -- Хранение ссылок на скриншоты
    current_stage INTEGER DEFAULT 1,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы транзакций
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    amount DECIMAL(10, 2),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы администраторов
CREATE TABLE IF NOT EXISTS admins (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL
);

-- Создание таблицы для временных данных
CREATE TABLE IF NOT EXISTS temp_data (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    key VARCHAR(255) NOT NULL,
    value TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, key)
);

-- Добавление столбцов state и available_at в таблицу users
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS state VARCHAR(50),
ADD COLUMN IF NOT EXISTS available_at TIMESTAMP;
ADD COLUMN referrer_id BIGINT;

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

CREATE INDEX idx_users_referrer_id ON users(referrer_id);
CREATE INDEX idx_tasks_user_id_status ON tasks(user_id, status);
    

