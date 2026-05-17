package userBalance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Luclpor/loyalty_system.git/internal/dto"
	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/service/order"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto/accrualStatusOrder"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance/balanceDto"
	mockstorage "github.com/Luclpor/loyalty_system.git/internal/storage/mock"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"github.com/Luclpor/loyalty_system.git/internal/storage/postgres"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestBalanceManagerUpdateBalance(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	updater := mockstorage.NewMockBalanceUpdater(ctrl)
	manager := &BalanceManager{balanceUpdater: updater}

	userID := uuid.New()
	accrual := 120.5
	orderDto := &accrualStatusOrder.OrderDto{
		Order:   "79927398713",
		UserID:  userID,
		Accrual: &accrual,
	}
	expected := &models.HistoryBalanceOperation{UserID: userID, OrderID: orderDto.Order, AmountTransactionPoint: &accrual}

	updater.EXPECT().
		SafetyUpdateBalance(gomock.Any(), orderDto).
		Return(expected, nil)

	got, err := manager.UpdateBalance(context.Background(), orderDto)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != expected {
		t.Fatalf("unexpected history balance operation: got %#v want %#v", got, expected)
	}
}

func TestBalanceManagerUpdateBalanceReturnsError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	updater := mockstorage.NewMockBalanceUpdater(ctrl)
	manager := &BalanceManager{balanceUpdater: updater}

	expectedErr := errors.New("update failed")
	orderDto := &accrualStatusOrder.OrderDto{Order: "79927398713", UserID: uuid.New()}

	updater.EXPECT().
		SafetyUpdateBalance(gomock.Any(), orderDto).
		Return(nil, expectedErr)

	got, err := manager.UpdateBalance(context.Background(), orderDto)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if got != nil {
		t.Fatalf("expected nil result on error, got %#v", got)
	}
}

func TestBalanceManagerGetUserBalance(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	reader := mockstorage.NewMockBalanceReader(ctrl)
	manager := &BalanceManager{balanceReader: reader}

	userID := uuid.New()
	withdraw1 := 15.25
	withdraw2 := 20.75

	reader.EXPECT().
		GetUserBalance(gomock.Any(), userID).
		Return(&models.Balance{UserID: userID, Point: 180.5}, nil)
	reader.EXPECT().
		GetUserBalanceHistoryNegativeTransaction(gomock.Any(), userID).
		Return([]models.HistoryBalanceOperation{
			{AmountTransactionPoint: &withdraw1},
			{AmountTransactionPoint: &withdraw2},
			{AmountTransactionPoint: nil},
		}, nil)

	got, err := manager.GetUserBalance(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Current != 180.5 {
		t.Fatalf("unexpected current balance: got %v want 180.5", got.Current)
	}
	if got.Withdrawn != 36.0 {
		t.Fatalf("unexpected withdrawn amount: got %v want 36", got.Withdrawn)
	}
}

func TestBalanceManagerGetUserBalanceIgnoresNotFoundHistory(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	reader := mockstorage.NewMockBalanceReader(ctrl)
	manager := &BalanceManager{balanceReader: reader}

	userID := uuid.New()

	reader.EXPECT().
		GetUserBalance(gomock.Any(), userID).
		Return(&models.Balance{UserID: userID, Point: 90}, nil)
	reader.EXPECT().
		GetUserBalanceHistoryNegativeTransaction(gomock.Any(), userID).
		Return(nil, postgres.ErrorNotFoundBalanceRows)

	got, err := manager.GetUserBalance(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Current != 90 {
		t.Fatalf("unexpected current balance: got %v want 90", got.Current)
	}
	if got.Withdrawn != 0 {
		t.Fatalf("unexpected withdrawn amount: got %v want 0", got.Withdrawn)
	}
}

func TestBalanceManagerGetUserBalanceReturnsUserBalanceError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	reader := mockstorage.NewMockBalanceReader(ctrl)
	manager := &BalanceManager{balanceReader: reader}

	userID := uuid.New()
	expectedErr := errors.New("read balance failed")

	reader.EXPECT().
		GetUserBalance(gomock.Any(), userID).
		Return(nil, expectedErr)

	got, err := manager.GetUserBalance(context.Background(), userID)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if got != nil {
		t.Fatalf("expected nil result on error, got %#v", got)
	}
}

func TestBalanceManagerGetUserBalanceReturnsHistoryError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	reader := mockstorage.NewMockBalanceReader(ctrl)
	manager := &BalanceManager{balanceReader: reader}

	userID := uuid.New()
	expectedErr := errors.New("read history failed")

	reader.EXPECT().
		GetUserBalance(gomock.Any(), userID).
		Return(&models.Balance{UserID: userID, Point: 10}, nil)
	reader.EXPECT().
		GetUserBalanceHistoryNegativeTransaction(gomock.Any(), userID).
		Return(nil, expectedErr)

	got, err := manager.GetUserBalance(context.Background(), userID)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if got != nil {
		t.Fatalf("expected nil result on error, got %#v", got)
	}
}

func TestBalanceManagerWithdrawUserBalanceInvalidOrder(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	updater := mockstorage.NewMockBalanceUpdater(ctrl)
	manager := &BalanceManager{balanceUpdater: updater}

	user := &dto.UserDto{ID: uuid.New()}
	appLogger := zap.NewNop()

	got, err := manager.WithdrawUserBalance(context.Background(), &apiModel.BalanceApi{
		Order: "123",
		Sum:   10,
	}, user, appLogger)
	if !errors.Is(err, order.ErrorInvalidOrderNum) {
		t.Fatalf("expected invalid order error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil result on validation error, got %#v", got)
	}
}

func TestBalanceManagerWithdrawUserBalanceUpdaterError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	updater := mockstorage.NewMockBalanceUpdater(ctrl)
	manager := &BalanceManager{balanceUpdater: updater}

	userID := uuid.New()
	appLogger := zap.NewNop()
	expectedErr := errors.New("withdraw failed")
	api := &apiModel.BalanceApi{
		Order: "79927398713",
		Sum:   70.5,
	}

	updater.EXPECT().
		SafetyWithdrawUpdateBalance(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got *balanceDto.WithdrawUpdateBalanceDto) (*models.HistoryBalanceOperation, error) {
			if got.UserID != userID {
				t.Fatalf("unexpected user id: got %s want %s", got.UserID, userID)
			}
			if got.AmountPoint != api.Sum {
				t.Fatalf("unexpected amount: got %v want %v", got.AmountPoint, api.Sum)
			}
			if got.OrderID != api.Order {
				t.Fatalf("unexpected order id: got %q want %q", got.OrderID, api.Order)
			}
			return nil, expectedErr
		})

	got, err := manager.WithdrawUserBalance(context.Background(), api, &dto.UserDto{ID: userID}, appLogger)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if got != nil {
		t.Fatalf("expected nil result on error, got %#v", got)
	}
}

func TestBalanceManagerWithdrawUserBalanceNotEnoughBalance(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	updater := mockstorage.NewMockBalanceUpdater(ctrl)
	manager := &BalanceManager{balanceUpdater: updater}

	userID := uuid.New()
	appLogger := zap.NewNop()
	api := &apiModel.BalanceApi{
		Order: "79927398713",
		Sum:   70.5,
	}

	updater.EXPECT().
		SafetyWithdrawUpdateBalance(gomock.Any(), gomock.Any()).
		Return(nil, nil)

	got, err := manager.WithdrawUserBalance(context.Background(), api, &dto.UserDto{ID: userID}, appLogger)
	if !errors.Is(err, ErrorNotEnoughBalance) {
		t.Fatalf("expected not enough balance error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil result on insufficient balance, got %#v", got)
	}
}

func TestBalanceManagerWithdrawUserBalance(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	updater := mockstorage.NewMockBalanceUpdater(ctrl)
	manager := &BalanceManager{balanceUpdater: updater}

	userID := uuid.New()
	appLogger := zap.NewNop()
	api := &apiModel.BalanceApi{
		Order: "79927398713",
		Sum:   70.5,
	}
	withdrawSum := 70.5

	updater.EXPECT().
		SafetyWithdrawUpdateBalance(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got *balanceDto.WithdrawUpdateBalanceDto) (*models.HistoryBalanceOperation, error) {
			if got.UserID != userID {
				t.Fatalf("unexpected user id: got %s want %s", got.UserID, userID)
			}
			if got.AmountPoint != api.Sum {
				t.Fatalf("unexpected amount: got %v want %v", got.AmountPoint, api.Sum)
			}
			if got.OrderID != api.Order {
				t.Fatalf("unexpected order id: got %q want %q", got.OrderID, api.Order)
			}
			return &models.HistoryBalanceOperation{
				UserID:                 userID,
				OrderID:                api.Order,
				AmountTransactionPoint: &withdrawSum,
			}, nil
		})

	got, err := manager.WithdrawUserBalance(context.Background(), api, &dto.UserDto{ID: userID}, appLogger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.UserID != userID {
		t.Fatalf("unexpected user id: got %s want %s", got.UserID, userID)
	}
	if got.Sum != withdrawSum {
		t.Fatalf("unexpected sum: got %v want %v", got.Sum, withdrawSum)
	}
}

func TestBalanceManagerGetUserWithDraws(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	reader := mockstorage.NewMockBalanceReader(ctrl)
	manager := &BalanceManager{balanceReader: reader}

	userID := uuid.New()
	processedAt1 := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	processedAt2 := time.Date(2026, 5, 17, 12, 30, 0, 0, time.UTC)
	sum1 := 40.5
	sum2 := 15.0

	reader.EXPECT().
		GetWithDraws(gomock.Any(), userID).
		Return([]models.HistoryBalanceOperation{
			{OrderID: "79927398713", AmountTransactionPoint: &sum1, CreatedAt: processedAt1},
			{OrderID: "12345678903", AmountTransactionPoint: &sum2, CreatedAt: processedAt2},
		}, nil)

	got, err := manager.GetUserWithDraws(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected result length: got %d want 2", len(got))
	}
	if got[0].OrderID != "79927398713" || got[0].Sum != sum1 || got[0].ProcessedAt == nil || !got[0].ProcessedAt.Equal(processedAt1) {
		t.Fatalf("unexpected first withdraw: %#v", got[0])
	}
	if got[1].OrderID != "12345678903" || got[1].Sum != sum2 || got[1].ProcessedAt == nil || !got[1].ProcessedAt.Equal(processedAt2) {
		t.Fatalf("unexpected second withdraw: %#v", got[1])
	}
}

func TestBalanceManagerGetUserWithDrawsReturnsError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	reader := mockstorage.NewMockBalanceReader(ctrl)
	manager := &BalanceManager{balanceReader: reader}

	userID := uuid.New()
	expectedErr := errors.New("get withdraws failed")

	reader.EXPECT().
		GetWithDraws(gomock.Any(), userID).
		Return(nil, expectedErr)

	got, err := manager.GetUserWithDraws(context.Background(), userID)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if got != nil {
		t.Fatalf("expected nil result on error, got %#v", got)
	}
}
