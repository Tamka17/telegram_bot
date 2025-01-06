// models/temp_data.go
package models

import "time"

type TempData struct {
	ID        int
	UserID    int
	Key       string
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
