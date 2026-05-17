package balanceDto

import "github.com/google/uuid"

type BalanceDto struct {
	OrderID string
	UserID  uuid.UUID
	Point   *float64
}
