package withdraw

import (
	"time"

	"github.com/google/uuid"
)

type WithdrawDto struct {
	UserID      uuid.UUID
	OrderID     int64
	Sum         float64
	ProcessedAt *time.Time
}
