// models/user.go
package models

import "time"

type User struct {
	ID          int
	TelegramID  int64
	Admin       bool
	Username    string
	Balance     float64
	State       State
	AvailableAt time.Time
	CreatedAt   time.Time
	ReferrerID  *int64
}
