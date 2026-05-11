package order

import (
	"context"
	"errors"
	"strconv"
	"time"

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
	UploadedAt  time.Time
}

type OrderManager struct {
	accrualSystemAddress string
	NewOrderChan         chan *OrderDto
	orderRep             OrderSaver
	orderReader          OrderReader
}

type OrderSaver interface {
	SaveOrder(ctx context.Context, orderDto *models.Order) error
	UpdateOrdersStatus(ctx context.Context, dtos []models.Order) error
}

type OrderReader interface {
	GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]models.Order, error)
	GetOrderByID(ctx context.Context, orderID int64) (*models.Order, error)
}

func NewOrderManager(address string, ordrRepository *postgres.OrderRepository) *OrderManager {
	return &OrderManager{
		NewOrderChan:         make(chan *OrderDto, 2),
		orderRep:             ordrRepository,
		orderReader:          ordrRepository,
		accrualSystemAddress: address,
	}
}

func (om *OrderManager) SaveNewOrder(ctx context.Context, orderApi *apiModel.OrderApiModel) error {
	isValid := validator.ValidateLuhn(orderApi.OrderNum)
	if !isValid {
		return appErrors.ErrorInvalidOrderNum
	}
	num, _ := strconv.Atoi(orderApi.OrderNum)
	order, err := om.orderReader.GetOrderByID(ctx, int64(num))
	if err != nil && !errors.Is(err, appErrors.ErrorNotFoundRows) {
		return err
	}
	if order != nil {
		if order.UserID != orderApi.UserID {
			return appErrors.ErrorOrderAlreadyUploadSomeUser
		}
		return appErrors.ErrorOrderAlreadyUploadThisUser
	}
	entToAdd := &models.Order{ID: num, UserID: orderApi.UserID, Status: models.NEW}
	err = om.orderRep.SaveOrder(ctx, entToAdd)
	if err != nil {
		return err
	}
	om.NewOrderChan <- &OrderDto{OrderNumber: entToAdd.ID, UserID: entToAdd.UserID, Status: string(entToAdd.Status)}
	return nil
}

func (om *OrderManager) UpdateOrders(ctx context.Context, dtos []models.Order) error {
	err := om.orderRep.UpdateOrdersStatus(ctx, dtos)
	if err != nil {
		return err
	}
	return nil
}

func (om *OrderManager) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]OrderDto, error) {
	ents, err := om.orderReader.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	orders := make([]OrderDto, len(ents))
	for i, ent := range ents {
		orders[i] = OrderDto{
			OrderNumber: ent.ID,
			UserID:      ent.UserID,
			Point:       *ent.Transaction.AmountTransactionPoint,
			Status:      string(ent.Status),
			UploadedAt:  ent.CreatedAt,
		}
	}
	return orders, nil
}
