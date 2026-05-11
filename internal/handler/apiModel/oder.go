package apiModel

import "github.com/google/uuid"

type OrderApiModel struct {
	OrderNum string
	UserID   uuid.UUID
}
