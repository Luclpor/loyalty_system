package apiModel

import (
	"time"

	"github.com/google/uuid"
)

type OrderApiModel struct {
	OrderNum string
	UserID   uuid.UUID
}

type ReadOrderApiModel struct {
	OrderNum   string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}
