// models/task.go
package models

import (
	"time"
)

type Task struct {
	ID               int
	UserID           int
	Category         Category
	Description      string
	IsActive         bool
	CreatedAt        time.Time
	Status           Status
	Link             string
	ScreenshotFileID string
}
