package worker

import (
	"context"
	"time"

	orderdto "github.com/Luclpor/loyalty_system.git/internal/service/order/dto"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto/accrualStatusOrder"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"go.uber.org/zap"
)

type pauseController interface {
	Pause(dur time.Duration)
	Wait(ctx context.Context) error
}

type externalResultClient interface {
	getResultFromExternalSystem(ctx context.Context, order orderdto.OrderDto, appLogger *zap.Logger) (*accrualStatusOrder.OrderDto, *time.Duration, error)
}

type balanceManagerService interface {
	UpdateBalance(ctx context.Context, order *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error)
}

type orderManagerService interface {
	UpdateOrder(ctx context.Context, dto *models.Order) (models.OrderStatus, error)
	OrderChan() <-chan *orderdto.OrderDto
}
