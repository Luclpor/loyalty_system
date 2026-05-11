package accrualStatusOrder

import (
	"github.com/google/uuid"
)

type AccrualSystemStatus string

const (
	REGISTERED AccrualSystemStatus = "REGISTERED"
	INVALID    AccrualSystemStatus = "INVALID"
	PROCESSING AccrualSystemStatus = "PROCESSING"
	PROCESSED  AccrualSystemStatus = "PROCESSED"
)

type OrderDto struct {
	Order   string `json:"order"`
	UserID  uuid.UUID
	Status  AccrualSystemStatus `json:"status"`
	Accrual *float64            `json:"accrual"`
}
