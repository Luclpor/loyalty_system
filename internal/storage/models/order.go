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
)

type Order struct {
	ID        int
	UserID    uuid.UUID
	Status    OrderStatus
	CreatedAt time.Time
}
