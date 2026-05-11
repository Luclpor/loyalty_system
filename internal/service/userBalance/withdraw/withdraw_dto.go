package withdraw

import (
	"time"

	"github.com/google/uuid"
)

type WithdrawDto struct {
	UserID      uuid.UUID
	Sum         float64
	ProcessedAt *time.Time
}
