package database

import (
	"context"
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
	GetAvailableTaskByType(ctx context.Context, taskType string) ([]*models.Task, error)
	AssignTaskToUser(ctx context.Context, taskID, userID int64) error
	SetUserState(ctx context.Context, userID int64, state string) error
	GetUserState(ctx context.Context, userID int64) (string, error)
}
