// db.go
package database

import (
	"context"
	"log"
	"os"
	"time"
    "telegram_bot/models"
    "errors"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

var db *pgxpool.Pool

type Database struct {
    Pool *pgxpool.Pool
  }

// NewDatabase - конструктор для Database
func NewDatabase(pool *pgxpool.Pool) *Database {
    return &Database{Pool: pool}
}

var _ database.DBInterface = (*database.Database)(nil)


stmt, err := db.Pool.Prepare(ctx, "assignTask", INSERT INTO user_tasks (user_id, task_id, status, created_at, updated_at) 
VALUES ($1, $2, 'in_progress', NOW(), NOW()) RETURNING id)
if err != nil {
return fmt.Errorf("не удалось подготовить выражение: %w", err)
}
defer stmt.Unprepare(ctx)

// Использование подготовленного выражения
err = db.Pool.QueryRow(ctx, "assignTask", userID, taskID).Scan(&userTaskID)





func InitDB() *Database {
	// Загрузка переменных окружения из .env
	err := godotenv.Load()
	if err != nil {
		log.Printf(".env файл не найден, продолжаем с системными переменными")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL не задан в переменных окружения")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err = pgxpool.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v\n", err)
	}

	// Проверка соединения
	err = db.Ping(ctx)
	if err != nil {
		log.Fatalf("Не удалось проверить соединение с базой данных: %v\n", err)
	}

	log.Println("Подключение к PostgreSQL установлено успешно!")
}

func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("Соединение с базой данных закрыто.")
	}
}

// GetUserByTelegramID получает пользователя по его Telegram ID
func (db *Database) GetUserByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
    user := &models.User{}
    query := SELECT id, telegram_id, username, balance, state, available_at, created_at, updated_at 
              FROM users WHERE telegram_id = $1
    err := db.Pool.QueryRowContext(ctx, query, telegramID).Scan(
        &user.ID,
        &user.TelegramID,
        &user.Username,
        &user.Balance,
        &user.State,
        &user.AvailableAt,
        &user.CreatedAt,
        &user.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }
    return user, nil
}

// CreateUser создает нового пользователя
func (db *Database) CreateUser(ctx context.Context, user *models.User) error {
    query := INSERT INTO users (telegram_id, username, balance, state, available_at, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id
    return db.Pool.QueryRowContext(ctx, query,
        user.TelegramID,
        user.Username,
        user.Balance,
        user.State,
        user.AvailableAt,
    ).Scan(&user.ID)
}

// SetUserState обновляет состояние пользователя
func (db *Database) SetUserState(ctx context.Context, telegramID int64, state string) error {
    query := UPDATE users SET state = $1, updated_at = NOW() WHERE telegram_id = $2
    result, err := db.Pool.Exec(ctx, query, state, telegramID)
    if err != nil {
        return err
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return errors.New("пользователь не найден")
    }
    return nil
}

// GetUserState получает текущее состояние пользователя
func (db *Database) GetUserState(ctx context.Context, telegramID int64) (string, error) {
    var state string
    query := SELECT state FROM users WHERE telegram_id = $1
    err := db.Pool.QueryRowContext(ctx, query, telegramID).Scan(&state)
    if err != nil {
        return "", err
    }
    return state, nil
}

// SetUserAvailableAt устанавливает время доступности пользователя
func (db *Database) SetUserAvailableAt(ctx context.Context, telegramID int64, availableAt time.Time) error {
    query := UPDATE users SET available_at = $1, updated_at = NOW() WHERE telegram_id = $2
    result, err := db.Pool.Exec(ctx, query, availableAt, telegramID)
    if err != nil {
        return err
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
	if rowsAffected == 0 {
        return errors.New("пользователь не найден")
    }
    return nil
}

// GetUserAvailableAt получает время доступности пользователя
func (db *Database) GetUserAvailableAt(ctx context.Context, telegramID int64) (time.Time, error) {
    var availableAt time.Time
    query := SELECT available_at FROM users WHERE telegram_id = $1
    err := db.Pool.QueryRowContext(ctx, query, telegramID).Scan(&availableAt)
    if err != nil {
        return time.Time{}, err
    }
    return availableAt, nil
}

// --- Методы для заданий ---

// CreateTask создает новое задание
func (db *Database) CreateTask(ctx context.Context, task *models.Task) error {
    query := INSERT INTO tasks (description, link, type, is_active, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id
    return db.Pool.QueryRowContext(ctx, query,
        task.Description,
        task.Link,
        task.Type,
        task.IsActive,
    ).Scan(&task.ID)
}

// GetTaskByID получает задание по его ID
func (db *Database) GetTaskByID(ctx context.Context, taskID int) (*models.Task, error) {
    task := &models.Task{}
    query := 'SELECT id, description, link, type, is_active, created_at, updated_at 
              FROM tasks WHERE id = $1'
    err := db.Pool.QueryRowContext(ctx, query, taskID).Scan(
        &task.ID,
        &task.Description,
        &task.Link,
        &task.Type,
        &task.IsActive,
        &task.CreatedAt,
        &task.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }
    return task, nil
}

// UpdateTaskStatus обновляет статус задания
func (db *Database) UpdateTaskStatus(ctx context.Context, taskID int, isActive bool) error {
    query := UPDATE tasks SET is_active = $1, updated_at = NOW() WHERE id = $2
    result, err := db.Pool.Exec(ctx, query, isActive, taskID)
    if err != nil {
        return err
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return errors.New("задание не найдено")
    }
    return nil
}

// GetAvailableTaskByType получает доступное задание по типу
func (db *Database) GetAvailableTaskByType(ctx context.Context, taskType string) (*models.Task, error) {
    task := &models.Task{}
    query := SELECT id, description, link, type, is_active, created_at, updated_at 
              FROM tasks 
              WHERE is_active = TRUE AND type = $1 
              ORDER BY created_at ASC LIMIT 1
    err := db.Pool.QueryRowContext(ctx, query, taskType).Scan(
        &task.ID,
        &task.Description,
        &task.Link,
        &task.Type,
        &task.IsActive,
        &task.CreatedAt,
        &task.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }
    return task, nil
}

// --- Методы для транзакций ---

// UpdateUserBalance обновляет баланс пользователя
func (db *Database) UpdateUserBalance(ctx context.Context, userID int, amount float64) error {
    query := UPDATE users SET balance = balance + $1, updated_at = NOW() WHERE id = $2
    result, err := db.Pool.Exec(ctx, query, amount, userID)
    if err != nil {
        return err
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return errors.New("пользователь не найден")
    }
    return nil
}

// --- Методы для связывания задания с пользователем ---

// AssignTaskToUser назначает задание пользователю
func (db *Database) AssignTaskToUser(ctx context.Context, userID, taskID int) error {
    query := INSERT INTO user_tasks (user_id, task_id, status, created_at, updated_at) 
              VALUES ($1, $2, 'in_progress', NOW(), NOW()) RETURNING id
    var userTaskID int
    err := db.Pool.QueryRowContext(ctx, query, userID, taskID).Scan(&userTaskID)
    if err != nil {
        return err
    }
    return nil
}

// --- Методы для временных данных ---

// SetTempData устанавливает временные данные для пользователя
func (db *Database) SetTempData(ctx context.Context, userID int, key, value string) error {
    query := INSERT INTO temp_data (user_id, key, value, created_at, updated_at)
              VALUES ($1, $2, $3, NOW(), NOW())
              ON CONFLICT (user_id, key) 
              DO UPDATE SET value = $3, updated_at = NOW()
    _, err := db.Pool.Exec(ctx, query, userID, key, value)
    return err
}

// GetTempData получает временные данные для пользователя по ключу
func (db *Database) GetTempData(ctx context.Context, userID int, key string) (string, error) {
    var value string
    query := SELECT value FROM temp_data WHERE user_id = $1 AND key = $2
    err := db.Pool.QueryRowContext(ctx, query, userID, key).Scan(&value)
    if err != nil {
        return "", err
    }
    return value, nil
}

// --- Дополнительные методы (например, создание транзакций) ---

// CreateTransaction создает новую транзакцию
func (db *Database) CreateTransaction(ctx context.Context, tx *models.Transaction) error {
    query := INSERT INTO transactions (user_id, amount, description, created_at) 
              VALUES ($1, $2, $3, NOW()) RETURNING id
    return db.Pool.QueryRowContext(ctx, query,
        tx.UserID,
        tx.Amount,
        tx.Description,
    ).Scan(&tx.ID)
}

// --- Методы для администраторов ---

// Пример: Проверка, является ли пользователь администратором
func (db *Database) IsAdmin(ctx context.Context, telegramID int64) (bool, error) {
    var exists bool
    query := SELECT EXISTS(SELECT 1 FROM admins WHERE telegram_id = $1)
    err := db.Pool.QueryRowContext(ctx, query, telegramID).Scan(&exists)
    if err != nil {
        return false, err
    }
    return exists, nil
}

// Вставляем или обновляем данные
query := 
INSERT INTO temp_data (user_id, key, value)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, key) DO UPDATE SET value = $3

_, err = db.pool.Exec(ctx, query, userID, key, jsonValue)
if err != nil {
return fmt.Errorf("не удалось установить временные данные: %w", err)
}
