// models/task.go
package models

type Task struct {
	ID               int
	Category         string
	Description      string
	Link             string
	IsActive         bool
	CreatedAt        string
	Status           string
	ScreenshotFileID string
}
