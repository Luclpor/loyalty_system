package userBalance

import (
	"context"
	"errors"

	"github.com/Luclpor/loyalty_system.git/internal/dto"
	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/service/order"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto/accrualStatusOrder"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance/balanceDto"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance/withdraw"
	"github.com/Luclpor/loyalty_system.git/internal/service/validator"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"github.com/Luclpor/loyalty_system.git/internal/storage/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type BalanceUpdater interface {
	SafetyUpdateBalance(ctx context.Context, entToUpdate *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error)
	SafetyWithdrawUpdateBalance(ctx context.Context, wdDto *balanceDto.WithdrawUpdateBalanceDto) (*models.HistoryBalanceOperation, error)
}

type BalanceReader interface {
	GetUserBalance(ctx context.Context, userID uuid.UUID) (*models.Balance, error)
	GetUserBalanceHistoryNegativeTransaction(ctx context.Context, userID uuid.UUID) ([]models.HistoryBalanceOperation, error)
	GetWithDraws(ctx context.Context, userID uuid.UUID) ([]models.HistoryBalanceOperation, error)
	GetUsersBalances(ctx context.Context, userIDs []uuid.UUID) ([]models.Balance, error)
}

type HistoryBalancer interface {
	CreateNewBalanceHistoryTransaction(ctx context.Context, tx pgx.Tx, hisBalanceEntity *models.HistoryBalanceOperation) (*models.HistoryBalanceOperation, error)
}

type BalanceManager struct {
	balanceUpdater BalanceUpdater
	balanceReader  BalanceReader
	hisBalancer    HistoryBalancer
}

var (
	ErrorNotEnoughBalance error = errors.New("not enough balance")
)

func NewBalanceManager(br *postgres.BalanceRepository) *BalanceManager {
	return &BalanceManager{
		balanceUpdater: br,
		balanceReader:  br,
		hisBalancer:    br,
	}
}

func (bm *BalanceManager) UpdateBalance(ctx context.Context, order *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error) {
	his, err := bm.balanceUpdater.SafetyUpdateBalance(ctx, order)
	if err != nil {
		return nil, err
	}
	return his, nil
}

func (bm *BalanceManager) GetUserBalance(ctx context.Context, userID uuid.UUID) (*apiModel.ReadBalanceApi, error) {
	userBalance, err := bm.balanceReader.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	operations, err := bm.balanceReader.GetUserBalanceHistoryNegativeTransaction(ctx, userID)
	if err != nil && !errors.Is(err, postgres.ErrorNotFoundBalanceRows) {
		return nil, err
	}
	var allWithdrawn float64 = 0
	for _, op := range operations {
		if op.AmountTransactionPoint != nil {
			allWithdrawn += *op.AmountTransactionPoint
		}
	}
	return &apiModel.ReadBalanceApi{Current: userBalance.Point, Withdrawn: allWithdrawn}, nil
}

func (bm *BalanceManager) WithdrawUserBalance(ctx context.Context, api *apiModel.BalanceApi, user *dto.UserDto, appLogger *zap.Logger) (*withdraw.WithdrawDto, error) {
	isValid := validator.ValidateLuhn(api.Order)
	if !isValid {
		return nil, order.ErrorInvalidOrderNum
	}
	hb, err := bm.balanceUpdater.SafetyWithdrawUpdateBalance(ctx, &balanceDto.WithdrawUpdateBalanceDto{UserID: user.ID, AmountPoint: api.Sum, OrderID: api.Order})
	if err != nil {
		appLogger.Error("error updating balance", zap.Error(err))
		return nil, err
	}
	if hb == nil {
		return nil, ErrorNotEnoughBalance
	}
	return &withdraw.WithdrawDto{UserID: user.ID, Sum: *hb.AmountTransactionPoint}, nil
}

func (bm *BalanceManager) GetUserWithDraws(ctx context.Context, userID uuid.UUID) ([]apiModel.ReadWithdrawApi, error) {
	resEnts, err := bm.balanceReader.GetWithDraws(ctx, userID)
	if err != nil {
		return nil, err
	}
	withdraws := make([]apiModel.ReadWithdrawApi, len(resEnts))
	for i, resEnt := range resEnts {
		withdraws[i] = apiModel.ReadWithdrawApi{OrderID: resEnt.OrderID, Sum: *resEnt.AmountTransactionPoint, ProcessedAt: &resEnt.CreatedAt}
	}
	return withdraws, nil
}
