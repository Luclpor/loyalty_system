package userBalance

import (
	"context"
	"errors"
	"strconv"

	"github.com/Luclpor/loyalty_system.git/internal/dto"
	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance/balanceDto"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance/withdraw"
	"github.com/Luclpor/loyalty_system.git/internal/service/validator"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"github.com/Luclpor/loyalty_system.git/internal/storage/postgres"
	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type BalanceUpdater interface {
	BulkUpdateBalances(ctx context.Context, userIDs []models.Balance) ([]models.Balance, error)
	UpdateBalance(ctx context.Context, updEntity *models.Balance) (*models.Balance, error)
}

type BalanceReader interface {
	GetUserBalance(ctx context.Context, userID uuid.UUID) (*models.Balance, error)
	GetUserBalanceHistoryNegativeTransaction(ctx context.Context, userID uuid.UUID) ([]models.HistoryBalanceOperation, error)
	GetWithDraws(ctx context.Context, userID uuid.UUID) ([]models.HistoryBalanceOperation, error)
	GetUsersBalances(ctx context.Context, userIDs []uuid.UUID) ([]models.Balance, error)
}

type BalanceManager struct {
	balanceUpdater BalanceUpdater
	balanceReader  BalanceReader
}

func NewBalanceManager(br *postgres.BalanceRepository) *BalanceManager {
	return &BalanceManager{
		balanceUpdater: br,
		balanceReader:  br,
	}
}

func (bm *BalanceManager) BulkUpdateBalance(ctx context.Context, dtos []balanceDto.BalanceDto) error {
	userBalanceDtos := make(map[uuid.UUID]balanceDto.BalanceDto, 0)
	userIDs := make([]uuid.UUID, len(dtos))
	for i, d := range dtos {
		userBalanceDtos[d.UserID] = d
		userIDs[i] = d.UserID
	}
	userBalances, err := bm.balanceReader.GetUsersBalances(ctx, userIDs)
	if err != nil {
		return err
	}
	for i, userBalance := range userBalances {
		d := userBalanceDtos[userBalance.UserID]
		p := userBalance.Point
		if d.Point != nil {
			p = *d.Point + userBalance.Point
		}
		userBalances[i].Point = p
		userBalances[i].HistoryBalanceOperation = &models.HistoryBalanceOperation{
			UserID:                 userBalance.UserID,
			IsPositiveTransaction:  true,
			AmountTransactionPoint: d.Point,
			BalancePoint:           p,
			OrderID:                d.OrderID,
		}
	}
	_, err = bm.balanceUpdater.BulkUpdateBalances(ctx, userBalances)
	if err != nil {
		return err
	}
	return nil
}

func (bm *BalanceManager) GetUserBalance(ctx context.Context, userID uuid.UUID) (*apiModel.ReadBalanceApi, error) {
	userBalance, err := bm.balanceReader.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	operations, err := bm.balanceReader.GetUserBalanceHistoryNegativeTransaction(ctx, userID)
	if err != nil && !errors.Is(err, appErrors.ErrorNotFoundRows) {
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

func (bm *BalanceManager) WithDrawUserBalance(ctx context.Context, api *apiModel.BalanceApi, user *dto.UserDto, appLogger *zap.Logger) (*withdraw.WithdrawDto, error) {
	isValid := validator.ValidateLuhn(api.Order)
	if !isValid {
		return nil, appErrors.ErrorInvalidOrderNum
	}
	ub, err := bm.balanceReader.GetUserBalance(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if ub.Point < float64(api.Sum) {
		return nil, appErrors.ErrorNotEnoughBalance
	}
	orderNum, _ := strconv.Atoi(api.Order)
	amTrPoint := float64(api.Sum)
	p := ub.Point - float64(api.Sum)
	updEntity := &models.Balance{
		UserID: user.ID,
		Point:  p,
		HistoryBalanceOperation: &models.HistoryBalanceOperation{
			UserID:                 user.ID,
			BalancePoint:           p,
			IsPositiveTransaction:  false,
			AmountTransactionPoint: &amTrPoint,
			OrderID:                int64(orderNum),
		},
	}
	ent, err := bm.balanceUpdater.UpdateBalance(ctx, updEntity)
	if err != nil {
		appLogger.Error("error updating balance", zap.Error(err))
		return nil, err
	}
	return &withdraw.WithdrawDto{UserID: ent.UserID, Sum: *ent.HistoryBalanceOperation.AmountTransactionPoint}, nil
}

func (bm *BalanceManager) GetUserWithDraws(ctx context.Context, userID uuid.UUID) ([]apiModel.ReadWithdrawApi, error) {
	resEnts, err := bm.balanceReader.GetWithDraws(ctx, userID)
	if err != nil {
		return nil, err
	}
	withdraws := make([]apiModel.ReadWithdrawApi, len(resEnts))
	for i, resEnt := range resEnts {
		withdraws[i] = apiModel.ReadWithdrawApi{OrderID: strconv.FormatInt(resEnt.OrderID, 10), Sum: *resEnt.AmountTransactionPoint, ProcessedAt: &resEnt.CreatedAt}
	}
	return withdraws, nil
}
