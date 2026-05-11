package order

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/service/validator"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"github.com/Luclpor/loyalty_system.git/internal/storage/postgres"
	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
	"github.com/google/uuid"
)

type OrderDto struct {
	OrderNumber int
	UserID      uuid.UUID
	Status      string
	Point       float64
}

type OrderManager struct {
	accrualSystemAddress string
	orderRep             OrderSaver
}

type OrderSaver interface {
	SaveOrder(ctx context.Context, orderDto *models.Order) error
	UpdateOrderStatus(ctx context.Context, dtos []models.Order) error
}

func NewOrderManager(address string, ordrRepository *postgres.OrderRepository) *OrderManager {
	return &OrderManager{
		orderRep:             ordrRepository,
		accrualSystemAddress: address,
	}
}

func (om *OrderManager) SaveNewOrder(ctx context.Context, orderApi *apiModel.OrderApiModel) error {
	isValid := validator.ValidateLuhn(orderApi.OrderNum)
	if !isValid {
		return appErrors.ErrorInvalidOrderNum
	}
	num, _ := strconv.Atoi(orderApi.OrderNum)
	err := om.orderRep.SaveOrder(ctx, &OrderDto{OrderNumber: num, UserID: orderApi.UserID, Status: "NEW"})
	if err != nil {
		return err
	}
	return nil
}

func (om *OrderManager) UpdateOrders(ctx context.Context, dtos []models.Order) error {
	err := om.UpdateOrders(ctx, dtos)
	if err != nil {
		return err
	}
	return nil
}
