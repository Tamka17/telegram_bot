// models/user.go
package models

type User struct {
	ID         int
	TelegramID int64
	Username   string
	Balance    float64
	CreatedAt  string
	ReferrerID *int64
}
