package common

import (
	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto/accrualStatusOrder"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
)

func ConvertStatus(statusAccr accrualStatusOrder.AccrualSystemStatus) models.OrderStatus {
	switch statusAccr {
	case accrualStatusOrder.REGISTERED, accrualStatusOrder.PROCESSING:
		return models.PROCESSING
	case accrualStatusOrder.PROCESSED:
		return models.PROCESSED
	case accrualStatusOrder.INVALID:
		return models.INVALID
	default:
		return models.INVALID
	}
}
