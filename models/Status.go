package models

// TaskStatus представляет возможные статусы задачи
type Status string

const (
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)
