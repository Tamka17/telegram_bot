// db.go
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"telegram_bot/models"
	"time"

	"github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

var dbInstance *Database

type Database struct {
	sqlDB *sql.DB
}

// InitDB - инициализатор базы данных (конструктор синглтона)
func InitDB() *Database {
	if dbInstance != nil {
		return dbInstance
	}
	dbInstance = NewDatabase()
	return dbInstance
}

// NewDatabase - создаёт новое подключение к базе данных
func NewDatabase() *Database {
	// Загрузка переменных окружения из .env
	if err := godotenv.Load(); err != nil {
		log.Println(".env файл не найден, продолжаем с системными переменными")
	} else {
		log.Println(".env файл успешно загружен в NewDatabase()")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL не задан в переменных окружения")
	}

	// Открытие соединения с базой данных
	sqlDB, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("Не удалось открыть соединение с базой данных: %v", err)
	}

	// Установка таймаута для проверки соединения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Проверка соединения
	if err := sqlDB.PingContext(ctx); err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}

	log.Println("Подключение к PostgreSQL установлено успешно!")
	return &Database{sqlDB: sqlDB}
}

// CloseDB - метод для закрытия подключения к базе данных
func CloseDB() {
	if dbInstance != nil {
		dbInstance.sqlDB.Close()
		log.Println("Соединение с базой данных закрыто.")
	}
}

// SetTaskStatus обновляет статус задания
func (db *Database) SetTaskStatus(ctx context.Context, taskID int64, status string) error {
	query := "UPDATE tasks SET status = $1, updated_at = NOW() WHERE id = $2"
	result, err := db.sqlDB.ExecContext(ctx, query, status, taskID)
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

func (db *Database) GetUserByID(ctx context.Context, telegramID int64) (*models.User, error) {
	query := "SELECT id, telegram_id, balance FROM users WHERE telegram_id = $1"

	row := db.sqlDB.QueryRowContext(ctx, query, telegramID)

	var user models.User
	err := row.Scan(&user.ID, &user.TelegramID, &user.Balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Пользователь не найден
		}
		return nil, err
	}

	return &models.User{}, nil
}

func (db *Database) SetUserBalance(ctx context.Context, telegramID int64, newBalance float64) error {
	query := "UPDATE users SET balance = $1 WHERE telegram_id = $2"

	res, err := db.sqlDB.ExecContext(ctx, query, newBalance, telegramID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("пользователь не найден")
	}

	return nil
}

// GetUserByTelegramID получает пользователя по его Telegram ID
func (db *Database) GetUserByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	user := &models.User{}
	query := `
    SELECT id, telegram_id, username, balance, state, available_at, created_at, updated_at 
              FROM users WHERE telegram_id = $1
              `
	err := db.sqlDB.QueryRowContext(ctx, query, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.Balance,
		&user.CreatedAt,
		&user.ReferrerID,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// CreateUser создает нового пользователя
func (db *Database) CreateUser(ctx context.Context, user *models.User) error {
	query := `
    INSERT INTO users (telegram_id, username, balance, state, available_at, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id
              `
	return db.sqlDB.QueryRowContext(ctx, query,
		user.TelegramID,
		user.Username,
		user.Balance,
		user.State, // Убедитесь, что поле существует в struct User
		user.AvailableAt,
	).Scan(&user.ID)
}

// SetUserState обновляет состояние пользователя по telegramID
func (db *Database) SetUserState(ctx context.Context, telegramID int64, state string) error {
	query := "UPDATE users SET state = $1, updated_at = NOW() WHERE telegram_id = $2"
	result, err := db.sqlDB.ExecContext(ctx, query, state, telegramID)
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
	query := "SELECT state FROM users WHERE telegram_id = $1"
	err := db.sqlDB.QueryRowContext(ctx, query, telegramID).Scan(&state)
	if err != nil {
		return "", err
	}
	return state, nil
}

func (db *Database) SetUserAvailableAt(ctx context.Context, telegramID int64) error {
	query := "UPDATE users SET available_at = $1, updated_at = NOW() WHERE telegram_id = $2"
	result, err := db.sqlDB.ExecContext(ctx, query, telegramID)
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
	query := "SELECT available_at FROM users WHERE telegram_id = $1"
	err := db.sqlDB.QueryRowContext(ctx, query, telegramID).Scan(&availableAt)
	if err != nil {
		return time.Time{}, err
	}
	return availableAt, nil
}

// --- Методы для заданий ---

// CreateTask создает новое задание
func (db *Database) CreateTask(ctx context.Context, task *models.Task) error {
	query := `
    INSERT INTO tasks (description, link, type, is_active, created_at, updated_at)
              VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id
              `
	return db.sqlDB.QueryRowContext(ctx, query,
		task.Description,
		task.Link,
		task.Category,
		task.IsActive,
	).Scan(&task.ID)
}

// GetTaskByID получает задание по его ID
func (db *Database) GetTaskByID(ctx context.Context, taskID int64) (*models.Task, error) {
	task := &models.Task{}
	query := `
    SELECT id, description, link, type, is_active, created_at, updated_at 
              FROM tasks WHERE id = $1
              `
	err := db.sqlDB.QueryRowContext(ctx, query, taskID).Scan(
		&task.ID,
		&task.Description,
		&task.Link,
		&task.Category,
		&task.IsActive,
		&task.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// UpdateTaskStatus обновляет статус задания
func (db *Database) UpdateTaskStatus(ctx context.Context, taskID int64, isActive string) error {
	query := "UPDATE tasks SET is_active = $1, updated_at = NOW() WHERE id = $2"
	result, err := db.sqlDB.ExecContext(ctx, query, taskID, isActive)
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
	query := `
    SELECT id, description, link, type, is_active, created_at, updated_at 
              FROM tasks 
              WHERE is_active = TRUE AND type = $1 
              ORDER BY created_at ASC LIMIT 1
              `
	err := db.sqlDB.QueryRowContext(ctx, query, taskType).Scan(
		&task.ID,
		&task.Description,
		&task.Link,
		&task.Category,
		&task.IsActive,
		&task.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// --- Методы для транзакций ---

// UpdateUserBalance обновляет баланс пользователя
func (db *Database) UpdateUserBalance(ctx context.Context, userID int64, amount float64) error {
	query := "UPDATE users SET balance = balance + $1, updated_at = NOW() WHERE id = $2"
	result, err := db.sqlDB.ExecContext(ctx, query, amount, userID)
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
func (db *Database) AssignTaskToUser(ctx context.Context, taskID int64, userID int64) error {
	var userTaskID int64

	// Выполнение запроса с RETURNING id
	query := `
        INSERT INTO user_tasks (user_id, task_id, status, created_at, updated_at) 
        VALUES ($1, $2, 'in_progress', NOW(), NOW()) RETURNING id
    `
	err := db.sqlDB.QueryRowContext(ctx, query, userID, taskID).Scan(&userTaskID)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	// Дополнительно можно использовать userTaskID, если нужно
	// fmt.Printf("Assigned task ID: %d\n", userTaskID)

	return nil
}

// --- Методы для временных данных ---

// SetTempData устанавливает временные данные для пользователя
func (db *Database) SetTempData(ctx context.Context, userID int64, key string, jsonValue interface{}) error {
	query := `
    INSERT INTO temp_data (user_id, key, value)
    VALUES ($1, $2, $3)
    ON CONFLICT (user_id, key) DO UPDATE SET value = $3
    `
	_, err := db.sqlDB.ExecContext(ctx, query, userID, key, jsonValue)
	if err != nil {
		return fmt.Errorf("не удалось установить временные данные: %w", err)
	}
	return nil
}

// GetUserReferralCount возвращает количество рефералов пользователя.
func (db *Database) GetUserReferralCount(ctx context.Context, userID int64) (int, error) {
	query := "SELECT COUNT(*) FROM users WHERE referrer_id = $1"
	var count int
	err := db.sqlDB.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetCompletedTasksCount возвращает количество выполненных заданий пользователя.
func (db *Database) GetCompletedTasksCount(ctx context.Context, userID int64) (int, error) {
	query := "SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND status = 'Completed'"
	var count int
	err := db.sqlDB.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteTempData удаляет временные данные пользователя по ключу.
func (db *Database) DeleteTempData(ctx context.Context, userID int64, key string) error {
	query := "DELETE FROM temp_data WHERE user_id = $1 AND key = $2"
	res, err := db.sqlDB.ExecContext(ctx, query, userID, key)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("временные данные не найдены")
	}
	return nil
}

// GetPendingTasks возвращает список заданий со статусом "Pending".
func (db *Database) GetPendingTasks(ctx context.Context) ([]*models.Task, error) {
	query := "SELECT id, user_id, category, description, is_active, created_at, status FROM tasks WHERE status = 'Pending'"
	rows, err := db.sqlDB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(&task.ID, &task.ID, &task.Category, &task.Description, &task.IsActive, &task.CreatedAt, &task.Status, &task.Link); err != nil {
			return nil, err
		}
		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

// Реализация общих методов, если они добавлены в интерфейс

// ExecContext выполняет общий SQL-запрос.
func (db *Database) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.sqlDB.ExecContext(ctx, query, args...)
}

// QueryRowContext выполняет запрос, возвращающий одну строку.
func (db *Database) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.sqlDB.QueryRowContext(ctx, query, args...)
}

// Query выполняет запрос, возвращающий множество строк.
func (db *Database) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.sqlDB.QueryContext(ctx, query, args...)
}

// Реализация методов для работы с временными данными

// GetTempData получает временные данные пользователя по ключу.
func (db *Database) GetTempData(ctx context.Context, userID int64, key string) (interface{}, error) {
	query := "SELECT value FROM temp_data WHERE user_id = $1 AND key = $2"
	row := db.sqlDB.QueryRowContext(ctx, query, userID, key)

	var jsonData []byte
	err := row.Scan(&jsonData)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Данные не найдены
		}
		return nil, err
	}

	// Предполагаем, что данные хранятся как строка
	var value string
	if err := json.Unmarshal(jsonData, &value); err != nil {
		return nil, err
	}

	return value, nil
}

// --- Дополнительные методы (например, создание транзакций) ---

// CreateTransaction создает новую транзакцию
func (db *Database) CreateTransaction(ctx context.Context, tx *models.Transaction) error {
	query := `
    INSERT INTO transactions (user_id, amount, description, created_at) 
              VALUES ($1, $2, $3, NOW()) RETURNING id
              `
	return db.sqlDB.QueryRowContext(ctx, query,
		tx.UserID,
		tx.Amount,
		tx.Description,
	).Scan(&tx.ID)
}

// --- Методы для администраторов ---

// Пример: Проверка, является ли пользователь администратором
func (db *Database) IsAdmin(ctx context.Context, telegramID int64) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM admins WHERE telegram_id = $1)"
	err := db.sqlDB.QueryRowContext(ctx, query, telegramID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (db *Database) SaveUserTaskScreenshot(ctx context.Context, userID int64, fileID string) error {
	query := "UPDATE tasks SET screenshot_file_id = $1 WHERE user_id = $2 AND is_completed = false"
	result, err := db.sqlDB.ExecContext(ctx, query, fileID, userID) // Убедитесь, что используете правильный объект для ExecContext
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("задание не найдено или уже завершено")
	}

	return nil
}
