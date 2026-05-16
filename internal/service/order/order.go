package order

import (
	"context"
	"errors"

	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto"
	"github.com/Luclpor/loyalty_system.git/internal/service/validator"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"github.com/Luclpor/loyalty_system.git/internal/storage/postgres"
	"github.com/google/uuid"
)

type OrderManager struct {
	accrualSystemAddress string
	NewOrderChan         chan *dto.OrderDto
	orderRep             OrderSaver
	orderReader          OrderReader
}

var (
	ErrorOrderAlreadyUploadThisUser = errors.New("order already upload this user")
	ErrorOrderAlreadyUploadSomeUser = errors.New("order already upload some user")
	ErrorInvalidOrderNum            = errors.New("invalid order number")
)

type OrderSaver interface {
	SaveOrder(ctx context.Context, orderDto *models.Order) error
	UpdateOrdersStatus(ctx context.Context, entToUpd *models.Order) (models.OrderStatus, error)
}

type OrderReader interface {
	GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]models.Order, error)
	GetOrderByID(ctx context.Context, orderID string) (*models.Order, error)
}

func NewOrderManager(address string, ordrRepository *postgres.OrderRepository) *OrderManager {
	return &OrderManager{
		NewOrderChan:         make(chan *dto.OrderDto, 2),
		orderRep:             ordrRepository,
		orderReader:          ordrRepository,
		accrualSystemAddress: address,
	}
}

func (om *OrderManager) SaveNewOrder(ctx context.Context, orderApi *apiModel.OrderApiModel) error {
	isValid := validator.ValidateLuhn(orderApi.OrderNum)
	if !isValid {
		return ErrorInvalidOrderNum
	}
	order, err := om.orderReader.GetOrderByID(ctx, orderApi.OrderNum)
	if err != nil && !errors.Is(err, postgres.ErrorNotFoundOrderRows) {
		return err
	}
	if order != nil {
		if order.UserID != orderApi.UserID {
			return ErrorOrderAlreadyUploadSomeUser
		}
		return ErrorOrderAlreadyUploadThisUser
	}
	entToAdd := &models.Order{ID: orderApi.OrderNum, UserID: orderApi.UserID, Status: models.NEW}
	err = om.orderRep.SaveOrder(ctx, entToAdd)
	if err != nil {
		return err
	}
	om.NewOrderChan <- &dto.OrderDto{OrderNumber: entToAdd.ID, UserID: entToAdd.UserID, Status: string(entToAdd.Status)}
	return nil
}

func (om *OrderManager) UpdateOrder(ctx context.Context, dto *models.Order) (models.OrderStatus, error) {
	status, err := om.orderRep.UpdateOrdersStatus(ctx, dto)
	if err != nil {
		return status, err
	}
	return status, nil
}

func (om *OrderManager) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]dto.OrderDto, error) {
	ents, err := om.orderReader.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	orders := make([]dto.OrderDto, len(ents))
	for i, ent := range ents {
		if ent.Transaction != nil && ent.Transaction.IsPositiveTransaction == false {
			continue
		}
		orders[i] = dto.OrderDto{
			OrderNumber: ent.ID,
			UserID:      ent.UserID,
			Status:      string(ent.Status),
			UploadedAt:  ent.CreatedAt,
		}
		if ent.Transaction != nil && ent.Transaction.AmountTransactionPoint != nil {
			orders[i].Point = *ent.Transaction.AmountTransactionPoint
		}
	}
	return orders, nil
}
