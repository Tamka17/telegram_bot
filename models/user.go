// models/user.go
package models

import "time"

type User struct {
	ID          int
	TelegramID  int64
	Username    string
	Balance     float64
	State       string
	AvailableAt time.Time
	CreatedAt   string
	ReferrerID  *int64
}
