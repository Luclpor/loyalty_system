package dto

import (
	"time"

	"github.com/google/uuid"
)

type OrderDto struct {
	OrderNumber string
	UserID      uuid.UUID
	Status      string
	Point       float64
	UploadedAt  time.Time
}
