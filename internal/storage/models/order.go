package models

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	NEW        OrderStatus = "NEW"
	PROCESSING OrderStatus = "PROCESSING"
	INVALID    OrderStatus = "INVALID"
	PROCESSED  OrderStatus = "PROCESSED"
	UNKNOWN    OrderStatus = "UNKNOWN"
)

type Order struct {
	ID          string
	UserID      uuid.UUID
	Status      OrderStatus
	CreatedAt   time.Time
	Transaction *HistoryBalanceOperation
}
