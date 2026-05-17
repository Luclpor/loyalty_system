package handler

import (
	"context"

	"github.com/Luclpor/loyalty_system.git/internal/dto"
	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	orderdto "github.com/Luclpor/loyalty_system.git/internal/service/order/dto"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance/withdraw"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type userContextGetter interface {
	GetUserFromContext(ctx context.Context) (*dto.UserDto, error)
}

type orderManagerService interface {
	SaveNewOrder(ctx context.Context, orderApi *apiModel.OrderApiModel) error
	GetUserOrders(ctx context.Context, userID uuid.UUID) ([]orderdto.OrderDto, error)
}

type balanceManagerService interface {
	WithdrawUserBalance(ctx context.Context, api *apiModel.BalanceApi, user *dto.UserDto, appLogger *zap.Logger) (*withdraw.WithdrawDto, error)
	GetUserWithDraws(ctx context.Context, userID uuid.UUID) ([]apiModel.ReadWithdrawApi, error)
	GetUserBalance(ctx context.Context, userID uuid.UUID) (*apiModel.ReadBalanceApi, error)
}
