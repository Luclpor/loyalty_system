package balanceDto

import "github.com/google/uuid"

type WithdrawUpdateBalanceDto struct {
	AmountPoint float64
	UserID      uuid.UUID
	OrderID     string
}
