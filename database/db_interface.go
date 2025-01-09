package database

import (
	"context"
	"database/sql"
	"time"

	"telegram_bot/models"
)

// DBInterface определяет методы для взаимодействия с базой данных
type DBInterface interface {
	GetUserAvailableAt(ctx context.Context, telegramID int64) (time.Time, error)
	CreateTask(ctx context.Context, task *models.Task) error
	SetUserAvailableAt(ctx context.Context, telegramID int64) error
	GetTaskByID(ctx context.Context, taskID int64) (*models.Task, error)
	UpdateUserBalance(ctx context.Context, userID int64, balance float64) error
	UpdateTaskStatus(ctx context.Context, taskID int64, status string) error
	SetTempData(ctx context.Context, userID int64, key string, value interface{}) error
	GetTempData(ctx context.Context, userID int64, key string) (interface{}, error)
	GetAvailableTaskByType(ctx context.Context, taskType string) (*models.Task, error)
	AssignTaskToUser(ctx context.Context, taskID, userID int64) error
	SetUserState(ctx context.Context, userID int64, state string) error
	GetUserState(ctx context.Context, userID int64) (string, error)

	GetUserByID(ctx context.Context, telegramID int64) (*models.User, error)
	SetUserBalance(ctx context.Context, telegramID int64, newBalance float64) error

	SetTaskStatus(ctx context.Context, taskID int64, status string) error
	GetPendingTasks(ctx context.Context) ([]*models.Task, error)
	DeleteTempData(ctx context.Context, userID int64, key string) error

	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	GetUserReferralCount(ctx context.Context, userID int64) (int, error)
	GetCompletedTasksCount(ctx context.Context, userID int64) (int, error)

	SaveUserTaskScreenshot(ctx context.Context, userID int64, fileID string) error
}
