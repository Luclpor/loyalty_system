package common

import (
	"errors"

	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto/accrualStatusOrder"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
)

func ConvertStatus(statusAccr accrualStatusOrder.AccrualSystemStatus) (models.OrderStatus, error) {
	switch statusAccr {
	case accrualStatusOrder.REGISTERED, accrualStatusOrder.PROCESSING:
		return models.PROCESSING, nil
	case accrualStatusOrder.PROCESSED:
		return models.PROCESSED, nil
	case accrualStatusOrder.INVALID:
		return models.INVALID, nil
	default:
		return models.INVALID, errors.New("invalid accrual status")
	}
}
