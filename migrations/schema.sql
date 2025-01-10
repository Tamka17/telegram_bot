-- migrations/schema.sql

-- Создание таблицы пользователей
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    balance DECIMAL(10, 2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Добавление столбцов в таблицу users, если они не существуют
ALTER TABLE users 
    ADD COLUMN IF NOT EXISTS state VARCHAR(50),
    ADD COLUMN IF NOT EXISTS available_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS referrer_id INTEGER;

-- Добавление внешнего ключа к столбцу referrer_id в таблице users
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_referrer_id_fkey;
ALTER TABLE users
    ADD CONSTRAINT users_referrer_id_fkey FOREIGN KEY (referrer_id) REFERENCES users(id);

INSERT INTO users (telegram_id, admin, created_at)
VALUES
(790745265, TRUE, NOW()),
(884539153, TRUE, NOW()),
(908077320, TRUE, NOW())
ON CONFLICT (telegram_id) DO UPDATE SET admin = TRUE;
-- Создание индекса на столбец referrer_id в таблице users
CREATE INDEX IF NOT EXISTS idx_users_referrer_id ON users(referrer_id);

-- Создание таблицы заданий
CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL PRIMARY KEY,
    description TEXT NOT NULL,
    link VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Добавление столбцов в таблицу tasks, если они не существуют
ALTER TABLE tasks 
    ADD COLUMN IF NOT EXISTS user_id INTEGER,
    ADD COLUMN IF NOT EXISTS type VARCHAR(50),
    ADD COLUMN IF NOT EXISTS status VARCHAR(50);

-- Добавление внешнего ключа к столбцу user_id в таблице tasks
ALTER TABLE tasks
    DROP CONSTRAINT IF EXISTS tasks_user_id_fkey;
ALTER TABLE tasks
    ADD CONSTRAINT tasks_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

-- Создание индекса на столбцы user_id и status в таблице tasks
CREATE INDEX IF NOT EXISTS idx_tasks_user_id_status ON tasks(user_id, status);

-- Создание таблицы пользовательских заданий
CREATE TABLE IF NOT EXISTS user_tasks (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    task_id INTEGER REFERENCES tasks(id),
    status VARCHAR(50) DEFAULT 'pending',
    screenshots JSONB,
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
