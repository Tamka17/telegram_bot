package models

type State string

const (
	StateNone                     State = ""
	StateAwaitingTaskCategory     State = "awaiting_task_category"
	StateAwaitingTaskDescription  State = "awaiting_task_description"
	StateAwaitingTaskScreenshot   State = "awaiting_screenshot"
	StateAwaitingCardNumder       State = "awaiting_card_number"
	StateawaitingTaskCategoryUser State = "awaiting_task_category_user"
	// Добавьте другие состояния по необходимости
)
