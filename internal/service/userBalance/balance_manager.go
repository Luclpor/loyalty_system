package userBalance

import (
	"context"

	"github.com/Luclpor/loyalty_system.git/internal/dto"
	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance/balanceDto"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance/withdraw"
	"github.com/Luclpor/loyalty_system.git/internal/service/validator"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type BalanceUpdater interface {
	BulkUpdateBalancesByUserIDs(ctx context.Context, userIDs []models.Balance) ([]models.Balance, error)
	UpdateBalance(ctx context.Context, updEntity *models.Balance) (*models.Balance, error)
}

type BalanceReader interface {
	GetUserBalanceByUserID(ctx context.Context, userID uuid.UUID) (*models.Balance, error)
	GetWithDraws(ctx context.Context, userID uuid.UUID) ([]models.HistoryBalanceOperation, error)
}

type BalanceManager struct {
	balanceUpdater BalanceUpdater
	balanceReader  BalanceReader
}

func NewBalanceManager(bu BalanceUpdater, br BalanceReader) *BalanceManager {
	return &BalanceManager{
		balanceUpdater: bu,
		balanceReader:  br,
	}
}

func (bm *BalanceManager) BulkUpdateBalance(ctx context.Context, dtos []balanceDto.BalanceDto) error {
	balModels := make([]models.Balance, 0)
	for _, dto := range dtos {
		balModels = append(balModels, models.Balance{Point: dto.Point, UserID: dto.UserID})
	}
	_, err := bm.balanceUpdater.BulkUpdateBalancesByUserIDs(ctx, balModels)
	if err != nil {
		return err
	}
	return nil
}

func (bm *BalanceManager) GetUserBalance(ctx context.Context, userID uuid.UUID) (*balanceDto.BalanceDto, error) {
	panic("implement me")
}

func (bm *BalanceManager) WithDrawUserBalance(ctx context.Context, api *apiModel.BalanceApi, user *dto.UserDto, appLogger *zap.Logger) (*withdraw.WithdrawDto, error) {
	isValid := validator.ValidateLuhn(api.Order)
	if !isValid {
		return nil, appErrors.ErrorInvalidOrderNum
	}
	ub, err := bm.balanceReader.GetUserBalanceByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if *ub.Point < float64(api.Sum) {
		return nil, appErrors.ErrorNotEnoughBalance
	}
	amTrPoint := float64(api.Sum)
	p := *ub.Point - float64(api.Sum)
	updEntity := &models.Balance{
		UserID: user.ID,
		Point:  &p,
		HistoryBalanceOperation: &models.HistoryBalanceOperation{
			UserID:                 user.ID,
			BalancePoint:           &p,
			IsPositiveTransaction:  false,
			AmountTransactionPoint: &amTrPoint,
		},
	}
	ent, err := bm.balanceUpdater.UpdateBalance(ctx, updEntity)
	if err != nil {
		appLogger.Error("error updating balance", zap.Error(err))
		return nil, err
	}
	return &withdraw.WithdrawDto{UserID: ent.UserID, Sum: *ent.HistoryBalanceOperation.AmountTransactionPoint}, nil
}

func (bm *BalanceManager) GetUserWithDraws(ctx context.Context, userID uuid.UUID) ([]withdraw.WithdrawDto, error) {
	resEnts, err := bm.balanceReader.GetWithDraws(ctx, userID)
	if err != nil {
		return nil, err
	}
	withdraws := make([]withdraw.WithdrawDto, len(resEnts))
	for _, resEnt := range resEnts {
		withdraws = append(withdraws, withdraw.WithdrawDto{UserID: userID, Sum: *resEnt.AmountTransactionPoint, ProcessedAt: &resEnt.CreatedAt})
	}
	return withdraws, nil
}
