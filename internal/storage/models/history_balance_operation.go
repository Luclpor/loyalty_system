package models

import (
	"time"

	"github.com/google/uuid"
)

type HistoryBalanceOperation struct {
	ID                     int
	UserID                 uuid.UUID
	IsPositiveTransaction  bool
	AmountTransactionPoint *float64
	BalancePoint           *float64
	CreatedAt              time.Time
}
