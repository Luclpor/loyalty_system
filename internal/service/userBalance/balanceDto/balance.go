package balanceDto

import "github.com/google/uuid"

type BalanceDto struct {
	OrderID int
	UserID  uuid.UUID
	Point   *float64
}
