package balanceDto

import "github.com/google/uuid"

type BalanceDto struct {
	OrderID int64
	UserID  uuid.UUID
	Point   *float64
}
