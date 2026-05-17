package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	orderdto "github.com/Luclpor/loyalty_system.git/internal/service/order/dto"
	mockstorage "github.com/Luclpor/loyalty_system.git/internal/storage/mock"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"github.com/Luclpor/loyalty_system.git/internal/storage/postgres"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestOrderManagerSaveNewOrderInvalidOrderNumber(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	saver := mockstorage.NewMockOrderSaver(ctrl)
	reader := mockstorage.NewMockOrderReader(ctrl)

	manager := &OrderManager{
		NewOrderChan: make(chan *orderdto.OrderDto, 1),
		orderRep:     saver,
		orderReader:  reader,
	}

	err := manager.SaveNewOrder(context.Background(), &apiModel.OrderApiModel{
		OrderNum: "123",
		UserID:   uuid.New(),
	})
	if !errors.Is(err, ErrorInvalidOrderNum) {
		t.Fatalf("expected invalid order error, got %v", err)
	}
}

func TestOrderManagerSaveNewOrderGetOrderError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	saver := mockstorage.NewMockOrderSaver(ctrl)
	reader := mockstorage.NewMockOrderReader(ctrl)

	manager := &OrderManager{
		NewOrderChan: make(chan *orderdto.OrderDto, 1),
		orderRep:     saver,
		orderReader:  reader,
	}

	userID := uuid.New()
	expectedErr := errors.New("repository failure")

	reader.EXPECT().
		GetOrderByID(gomock.Any(), "79927398713").
		Return(nil, expectedErr)

	err := manager.SaveNewOrder(context.Background(), &apiModel.OrderApiModel{
		OrderNum: "79927398713",
		UserID:   userID,
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestOrderManagerSaveNewOrderAlreadyUploadedByAnotherUser(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	saver := mockstorage.NewMockOrderSaver(ctrl)
	reader := mockstorage.NewMockOrderReader(ctrl)

	manager := &OrderManager{
		NewOrderChan: make(chan *orderdto.OrderDto, 1),
		orderRep:     saver,
		orderReader:  reader,
	}

	userID := uuid.New()
	reader.EXPECT().
		GetOrderByID(gomock.Any(), "79927398713").
		Return(&models.Order{ID: "79927398713", UserID: uuid.New()}, nil)

	err := manager.SaveNewOrder(context.Background(), &apiModel.OrderApiModel{
		OrderNum: "79927398713",
		UserID:   userID,
	})
	if !errors.Is(err, ErrorOrderAlreadyUploadSomeUser) {
		t.Fatalf("expected already uploaded by another user error, got %v", err)
	}
}

func TestOrderManagerSaveNewOrderAlreadyUploadedBySameUser(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	saver := mockstorage.NewMockOrderSaver(ctrl)
	reader := mockstorage.NewMockOrderReader(ctrl)

	manager := &OrderManager{
		NewOrderChan: make(chan *orderdto.OrderDto, 1),
		orderRep:     saver,
		orderReader:  reader,
	}

	userID := uuid.New()
	reader.EXPECT().
		GetOrderByID(gomock.Any(), "79927398713").
		Return(&models.Order{ID: "79927398713", UserID: userID}, nil)

	err := manager.SaveNewOrder(context.Background(), &apiModel.OrderApiModel{
		OrderNum: "79927398713",
		UserID:   userID,
	})
	if !errors.Is(err, ErrorOrderAlreadyUploadThisUser) {
		t.Fatalf("expected already uploaded by same user error, got %v", err)
	}
}

func TestOrderManagerSaveNewOrderSaveError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	saver := mockstorage.NewMockOrderSaver(ctrl)
	reader := mockstorage.NewMockOrderReader(ctrl)

	manager := &OrderManager{
		NewOrderChan: make(chan *orderdto.OrderDto, 1),
		orderRep:     saver,
		orderReader:  reader,
	}

	userID := uuid.New()
	expectedErr := errors.New("insert failed")

	reader.EXPECT().
		GetOrderByID(gomock.Any(), "79927398713").
		Return(nil, postgres.ErrorNotFoundOrderRows)
	saver.EXPECT().
		SaveOrder(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got *models.Order) error {
			if got.ID != "79927398713" {
				t.Fatalf("unexpected order id: got %q", got.ID)
			}
			if got.UserID != userID {
				t.Fatalf("unexpected user id: got %s want %s", got.UserID, userID)
			}
			if got.Status != models.NEW {
				t.Fatalf("unexpected status: got %s want %s", got.Status, models.NEW)
			}
			return expectedErr
		})

	err := manager.SaveNewOrder(context.Background(), &apiModel.OrderApiModel{
		OrderNum: "79927398713",
		UserID:   userID,
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestOrderManagerSaveNewOrder(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	saver := mockstorage.NewMockOrderSaver(ctrl)
	reader := mockstorage.NewMockOrderReader(ctrl)

	manager := &OrderManager{
		NewOrderChan: make(chan *orderdto.OrderDto, 1),
		orderRep:     saver,
		orderReader:  reader,
	}

	userID := uuid.New()

	reader.EXPECT().
		GetOrderByID(gomock.Any(), "79927398713").
		Return(nil, postgres.ErrorNotFoundOrderRows)
	saver.EXPECT().
		SaveOrder(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got *models.Order) error {
			if got.ID != "79927398713" {
				t.Fatalf("unexpected order id: got %q", got.ID)
			}
			if got.UserID != userID {
				t.Fatalf("unexpected user id: got %s want %s", got.UserID, userID)
			}
			if got.Status != models.NEW {
				t.Fatalf("unexpected status: got %s want %s", got.Status, models.NEW)
			}
			return nil
		})

	err := manager.SaveNewOrder(context.Background(), &apiModel.OrderApiModel{
		OrderNum: "79927398713",
		UserID:   userID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case got := <-manager.NewOrderChan:
		if got == nil {
			t.Fatal("expected new order dto in channel")
		}
		if got.OrderNumber != "79927398713" {
			t.Fatalf("unexpected order number in channel: got %q", got.OrderNumber)
		}
		if got.UserID != userID {
			t.Fatalf("unexpected user id in channel: got %s want %s", got.UserID, userID)
		}
		if got.Status != string(models.NEW) {
			t.Fatalf("unexpected status in channel: got %q want %q", got.Status, models.NEW)
		}
	default:
		t.Fatal("expected created order to be pushed into channel")
	}
}

func TestOrderManagerUpdateOrder(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	saver := mockstorage.NewMockOrderSaver(ctrl)
	reader := mockstorage.NewMockOrderReader(ctrl)

	manager := &OrderManager{
		orderRep:    saver,
		orderReader: reader,
	}

	orderModel := &models.Order{ID: "79927398713", Status: models.PROCESSED}

	saver.EXPECT().
		UpdateOrdersStatus(gomock.Any(), orderModel).
		Return(models.PROCESSED, nil)

	got, err := manager.UpdateOrder(context.Background(), orderModel)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != models.PROCESSED {
		t.Fatalf("unexpected status: got %s want %s", got, models.PROCESSED)
	}
}

func TestOrderManagerUpdateOrderReturnsError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	saver := mockstorage.NewMockOrderSaver(ctrl)
	reader := mockstorage.NewMockOrderReader(ctrl)

	manager := &OrderManager{
		orderRep:    saver,
		orderReader: reader,
	}

	orderModel := &models.Order{ID: "79927398713", Status: models.PROCESSED}
	expectedErr := errors.New("update failed")

	saver.EXPECT().
		UpdateOrdersStatus(gomock.Any(), orderModel).
		Return(models.UNKNOWN, expectedErr)

	got, err := manager.UpdateOrder(context.Background(), orderModel)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if got != models.UNKNOWN {
		t.Fatalf("unexpected status: got %s want %s", got, models.UNKNOWN)
	}
}

func TestOrderManagerGetUserOrdersReturnsError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	saver := mockstorage.NewMockOrderSaver(ctrl)
	reader := mockstorage.NewMockOrderReader(ctrl)

	manager := &OrderManager{
		orderRep:    saver,
		orderReader: reader,
	}

	userID := uuid.New()
	expectedErr := errors.New("read failed")

	reader.EXPECT().
		GetOrdersByUserID(gomock.Any(), userID).
		Return(nil, expectedErr)

	got, err := manager.GetUserOrders(context.Background(), userID)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if got != nil {
		t.Fatalf("expected nil result on error, got %#v", got)
	}
}

func TestOrderManagerGetUserOrders(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	saver := mockstorage.NewMockOrderSaver(ctrl)
	reader := mockstorage.NewMockOrderReader(ctrl)

	manager := &OrderManager{
		orderRep:    saver,
		orderReader: reader,
	}

	userID := uuid.New()
	now := time.Date(2026, 5, 17, 14, 30, 0, 0, time.UTC)
	point := 42.5

	reader.EXPECT().
		GetOrdersByUserID(gomock.Any(), userID).
		Return([]models.Order{
			{
				ID:        "79927398713",
				UserID:    userID,
				Status:    models.NEW,
				CreatedAt: now,
			},
			{
				ID:        "12345678903",
				UserID:    userID,
				Status:    models.PROCESSED,
				CreatedAt: now.Add(time.Minute),
				Transaction: &models.HistoryBalanceOperation{
					IsPositiveTransaction:  true,
					AmountTransactionPoint: &point,
				},
			},
			{
				ID:        "55555555555",
				UserID:    userID,
				Status:    models.PROCESSED,
				CreatedAt: now.Add(2 * time.Minute),
				Transaction: &models.HistoryBalanceOperation{
					IsPositiveTransaction:  false,
					AmountTransactionPoint: &point,
				},
			},
		}, nil)

	got, err := manager.GetUserOrders(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected result length: got %d want 2", len(got))
	}

	if got[0].OrderNumber != "79927398713" {
		t.Fatalf("unexpected first order number: got %q", got[0].OrderNumber)
	}
	if got[0].UserID != userID {
		t.Fatalf("unexpected first user id: got %s want %s", got[0].UserID, userID)
	}
	if got[0].Status != string(models.NEW) {
		t.Fatalf("unexpected first status: got %q want %q", got[0].Status, models.NEW)
	}
	if got[0].Point != 0 {
		t.Fatalf("unexpected first order points: got %v want 0", got[0].Point)
	}
	if !got[0].UploadedAt.Equal(now) {
		t.Fatalf("unexpected first uploaded time: got %s want %s", got[0].UploadedAt, now)
	}

	if got[1].OrderNumber != "12345678903" {
		t.Fatalf("unexpected second order number: got %q", got[1].OrderNumber)
	}
	if got[1].Status != string(models.PROCESSED) {
		t.Fatalf("unexpected second status: got %q want %q", got[1].Status, models.PROCESSED)
	}
	if got[1].Point != point {
		t.Fatalf("unexpected second order points: got %v want %v", got[1].Point, point)
	}
	if !got[1].UploadedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("unexpected second uploaded time: got %s want %s", got[1].UploadedAt, now.Add(time.Minute))
	}
}
