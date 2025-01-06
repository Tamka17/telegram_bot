// models/user_task.go
package models

type UserTask struct {
	ID           int
	UserID       int
	TaskID       int
	Status       string
	Screenshots  []string
	CurrentStage int
	LastUpdated  string
}
