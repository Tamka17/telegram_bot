// models/transaction.go
package models

type Transaction struct {
	ID          int
	UserID      int
	Amount      float64
	Description string
	CreatedAt   string
}
