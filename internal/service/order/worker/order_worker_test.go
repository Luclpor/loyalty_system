package worker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto/accrualStatusOrder"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type fakePauseController struct {
	waitFn   func(ctx context.Context) error
	pauseDur []time.Duration
}

func (f *fakePauseController) Pause(dur time.Duration) {
	f.pauseDur = append(f.pauseDur, dur)
}

func (f *fakePauseController) Wait(ctx context.Context) error {
	if f.waitFn != nil {
		return f.waitFn(ctx)
	}
	return nil
}

type fakeExternalClient struct {
	getResultFn func(ctx context.Context, order dto.OrderDto, appLogger *zap.Logger) (*accrualStatusOrder.OrderDto, *time.Duration, error)
}

func (f *fakeExternalClient) getResultFromExternalSystem(ctx context.Context, order dto.OrderDto, appLogger *zap.Logger) (*accrualStatusOrder.OrderDto, *time.Duration, error) {
	return f.getResultFn(ctx, order, appLogger)
}

type fakeBalanceManager struct {
	updateBalanceFn func(ctx context.Context, order *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error)
}

func (f *fakeBalanceManager) UpdateBalance(ctx context.Context, order *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error) {
	if f.updateBalanceFn == nil {
		return nil, nil
	}
	return f.updateBalanceFn(ctx, order)
}

type fakeOrderManager struct {
	updateOrderFn func(ctx context.Context, order *models.Order) (models.OrderStatus, error)
	orderCh       chan *dto.OrderDto
}

func (f *fakeOrderManager) UpdateOrder(ctx context.Context, order *models.Order) (models.OrderStatus, error) {
	if f.updateOrderFn == nil {
		return models.UNKNOWN, nil
	}
	return f.updateOrderFn(ctx, order)
}

func (f *fakeOrderManager) OrderChan() <-chan *dto.OrderDto {
	return f.orderCh
}

func TestNewWorkerOrderInvalidURL(t *testing.T) {
	t.Parallel()

	worker, err := NewWorkerOrder("://bad-url", nil, nil, zap.NewNop())
	if err == nil {
		t.Fatal("expected error for invalid accrual url")
	}
	if worker != nil {
		t.Fatalf("expected nil worker on invalid url, got %#v", worker)
	}
}

func TestExternalRtrClientGetResultFromExternalSystem(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: got %s want GET", r.Method)
		}
		if r.URL.Path != "/api/orders/79927398713" {
			t.Fatalf("unexpected path: got %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"order":"79927398713","status":"PROCESSED","accrual":42.5}`))
	}))
	defer srv.Close()

	client, err := newRetryableHttpClient(srv.URL)
	if err != nil {
		t.Fatalf("failed to create retryable client: %v", err)
	}

	got, retryAfter, err := client.getResultFromExternalSystem(context.Background(), dto.OrderDto{
		OrderNumber: "79927398713",
		UserID:      userID,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if retryAfter != nil {
		t.Fatalf("expected no retry-after, got %v", retryAfter)
	}
	if got == nil {
		t.Fatal("expected non-nil response dto")
	}
	if got.Order != "79927398713" {
		t.Fatalf("unexpected order number: got %q", got.Order)
	}
	if got.Status != accrualStatusOrder.PROCESSED {
		t.Fatalf("unexpected status: got %q", got.Status)
	}
	if got.Accrual == nil || *got.Accrual != 42.5 {
		t.Fatalf("unexpected accrual: got %v", got.Accrual)
	}
	if got.UserID != userID {
		t.Fatalf("unexpected user id: got %s want %s", got.UserID, userID)
	}
}

func TestExternalRtrClientGetResultFromExternalSystemRetryAfter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client, err := newRetryableHttpClient(srv.URL)
	if err != nil {
		t.Fatalf("failed to create retryable client: %v", err)
	}

	got, retryAfter, err := client.getResultFromExternalSystem(context.Background(), dto.OrderDto{
		OrderNumber: "79927398713",
		UserID:      uuid.New(),
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil dto on retry-after, got %#v", got)
	}
	if retryAfter == nil || *retryAfter != 3*time.Second {
		t.Fatalf("unexpected retry-after: got %v want 3s", retryAfter)
	}
}

func TestExternalRtrClientGetResultFromExternalSystemInvalidRetryAfter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "soon")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client, err := newRetryableHttpClient(srv.URL)
	if err != nil {
		t.Fatalf("failed to create retryable client: %v", err)
	}

	got, retryAfter, err := client.getResultFromExternalSystem(context.Background(), dto.OrderDto{
		OrderNumber: "79927398713",
		UserID:      uuid.New(),
	}, zap.NewNop())
	if err == nil {
		t.Fatal("expected parse error for invalid Retry-After")
	}
	if got != nil || retryAfter != nil {
		t.Fatalf("expected nil dto and retry-after on error, got dto=%#v retry=%v", got, retryAfter)
	}
}

func TestExternalRtrClientGetResultFromExternalSystemUnexpectedStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client, err := newRetryableHttpClient(srv.URL)
	if err != nil {
		t.Fatalf("failed to create retryable client: %v", err)
	}

	got, retryAfter, err := client.getResultFromExternalSystem(context.Background(), dto.OrderDto{
		OrderNumber: "79927398713",
		UserID:      uuid.New(),
	}, zap.NewNop())
	if err == nil {
		t.Fatal("expected error for unexpected status")
	}
	if got != nil || retryAfter != nil {
		t.Fatalf("expected nil dto and retry-after on error, got dto=%#v retry=%v", got, retryAfter)
	}
}

func TestExternalRtrClientGetResultFromExternalSystemInvalidJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"order":`))
	}))
	defer srv.Close()

	client, err := newRetryableHttpClient(srv.URL)
	if err != nil {
		t.Fatalf("failed to create retryable client: %v", err)
	}

	got, retryAfter, err := client.getResultFromExternalSystem(context.Background(), dto.OrderDto{
		OrderNumber: "79927398713",
		UserID:      uuid.New(),
	}, zap.NewNop())
	if err == nil {
		t.Fatal("expected decode error for invalid json")
	}
	if got != nil || retryAfter != nil {
		t.Fatalf("expected nil dto and retry-after on error, got dto=%#v retry=%v", got, retryAfter)
	}
}

func TestHandleEvaluatingOrdersInvalidStatusSkipsManagers(t *testing.T) {
	t.Parallel()

	worker := &WorkerOrder{
		orderManager: &fakeOrderManager{updateOrderFn: func(context.Context, *models.Order) (models.OrderStatus, error) {
			t.Fatal("order manager should not be called")
			return models.UNKNOWN, nil
		}},
		balanceManager: &fakeBalanceManager{updateBalanceFn: func(context.Context, *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error) {
			t.Fatal("balance manager should not be called")
			return nil, nil
		}},
		appLogger: zap.NewNop(),
	}

	worker.HandleEvaluatingOrders(context.Background(), &accrualStatusOrder.OrderDto{
		Order:  "79927398713",
		UserID: uuid.New(),
		Status: accrualStatusOrder.AccrualSystemStatus("BROKEN"),
	})
}

func TestHandleEvaluatingOrdersUpdateOrderErrorSkipsBalance(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("update failed")
	worker := &WorkerOrder{
		orderManager: &fakeOrderManager{
			updateOrderFn: func(context.Context, *models.Order) (models.OrderStatus, error) {
				return models.UNKNOWN, expectedErr
			},
		},
		balanceManager: &fakeBalanceManager{
			updateBalanceFn: func(context.Context, *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error) {
				t.Fatal("balance manager should not be called on update error")
				return nil, nil
			},
		},
		appLogger: zap.NewNop(),
	}

	worker.HandleEvaluatingOrders(context.Background(), &accrualStatusOrder.OrderDto{
		Order:  "79927398713",
		UserID: uuid.New(),
		Status: accrualStatusOrder.PROCESSED,
	})
}

func TestHandleEvaluatingOrdersUnknownStatusSkipsBalance(t *testing.T) {
	t.Parallel()

	worker := &WorkerOrder{
		orderManager: &fakeOrderManager{
			updateOrderFn: func(_ context.Context, got *models.Order) (models.OrderStatus, error) {
				if got.Status != models.PROCESSED {
					t.Fatalf("unexpected converted status: got %s want %s", got.Status, models.PROCESSED)
				}
				return models.UNKNOWN, nil
			},
		},
		balanceManager: &fakeBalanceManager{
			updateBalanceFn: func(context.Context, *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error) {
				t.Fatal("balance manager should not be called for unknown order status")
				return nil, nil
			},
		},
		appLogger: zap.NewNop(),
	}

	worker.HandleEvaluatingOrders(context.Background(), &accrualStatusOrder.OrderDto{
		Order:  "79927398713",
		UserID: uuid.New(),
		Status: accrualStatusOrder.PROCESSED,
	})
}

func TestHandleEvaluatingOrdersProcessedUpdatesBalance(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	accrual := 55.5
	history := &models.HistoryBalanceOperation{
		UserID:                 userID,
		OrderID:                "79927398713",
		AmountTransactionPoint: &accrual,
	}
	balanceCalled := false

	worker := &WorkerOrder{
		orderManager: &fakeOrderManager{
			updateOrderFn: func(_ context.Context, got *models.Order) (models.OrderStatus, error) {
				if got.ID != "79927398713" {
					t.Fatalf("unexpected order id: got %q", got.ID)
				}
				if got.Status != models.PROCESSED {
					t.Fatalf("unexpected converted status: got %s want %s", got.Status, models.PROCESSED)
				}
				return models.PROCESSED, nil
			},
		},
		balanceManager: &fakeBalanceManager{
			updateBalanceFn: func(_ context.Context, got *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error) {
				balanceCalled = true
				if got.Order != "79927398713" {
					t.Fatalf("unexpected order in balance update: got %q", got.Order)
				}
				if got.UserID != userID {
					t.Fatalf("unexpected user id: got %s want %s", got.UserID, userID)
				}
				if got.Accrual == nil || *got.Accrual != accrual {
					t.Fatalf("unexpected accrual: got %v", got.Accrual)
				}
				return history, nil
			},
		},
		appLogger: zap.NewNop(),
	}

	worker.HandleEvaluatingOrders(context.Background(), &accrualStatusOrder.OrderDto{
		Order:   "79927398713",
		UserID:  userID,
		Status:  accrualStatusOrder.PROCESSED,
		Accrual: &accrual,
	})

	if !balanceCalled {
		t.Fatal("expected balance update to be called")
	}
}

func TestHandleEvaluatingOrdersProcessingStatusDoesNotUpdateBalance(t *testing.T) {
	t.Parallel()

	worker := &WorkerOrder{
		orderManager: &fakeOrderManager{
			updateOrderFn: func(_ context.Context, got *models.Order) (models.OrderStatus, error) {
				if got.Status != models.PROCESSING {
					t.Fatalf("unexpected converted status: got %s want %s", got.Status, models.PROCESSING)
				}
				return models.PROCESSING, nil
			},
		},
		balanceManager: &fakeBalanceManager{
			updateBalanceFn: func(context.Context, *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error) {
				t.Fatal("balance manager should not be called for processing status")
				return nil, nil
			},
		},
		appLogger: zap.NewNop(),
	}

	worker.HandleEvaluatingOrders(context.Background(), &accrualStatusOrder.OrderDto{
		Order:  "79927398713",
		UserID: uuid.New(),
		Status: accrualStatusOrder.REGISTERED,
	})
}

func TestEnqueueDelayedOrderReturnsFalseOnCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	worker := &WorkerOrder{
		chQueueOrder: make(chan delayedOrder),
	}

	ok := worker.enqueueDelayedOrder(ctx, dto.OrderDto{OrderNumber: "79927398713"}, time.Second)
	if ok {
		t.Fatal("expected enqueueDelayedOrder to return false when context is canceled")
	}
}

func TestRunDelayedJobSendsOrder(t *testing.T) {
	t.Parallel()

	worker := &WorkerOrder{}
	jobs := make(chan dto.OrderDto, 1)
	orderDto := dto.OrderDto{OrderNumber: "79927398713", UserID: uuid.New()}

	worker.runDelayedJob(context.Background(), jobs, delayedOrder{
		order: orderDto,
		delay: 10 * time.Millisecond,
	})

	select {
	case got := <-jobs:
		if got.OrderNumber != orderDto.OrderNumber {
			t.Fatalf("unexpected order number: got %q want %q", got.OrderNumber, orderDto.OrderNumber)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delayed job")
	}

	worker.wg.Wait()
}

func TestWorkerOrderExternalErrorEnqueuesDelayedOrder(t *testing.T) {
	t.Parallel()

	job := make(chan dto.OrderDto, 1)
	resultJob := make(chan *accrualStatusOrder.OrderDto, 1)
	queue := make(chan delayedOrder, 1)
	orderDto := dto.OrderDto{OrderNumber: "79927398713", UserID: uuid.New()}
	job <- orderDto
	close(job)

	worker := &WorkerOrder{
		pauseWorkerController: &fakePauseController{},
		extClient: &fakeExternalClient{
			getResultFn: func(context.Context, dto.OrderDto, *zap.Logger) (*accrualStatusOrder.OrderDto, *time.Duration, error) {
				return nil, nil, errors.New("external failure")
			},
		},
		chQueueOrder: queue,
		appLogger:    zap.NewNop(),
	}

	worker.WorkerOrder(context.Background(), job, resultJob)

	select {
	case got := <-queue:
		if got.order.OrderNumber != orderDto.OrderNumber {
			t.Fatalf("unexpected queued order: got %q want %q", got.order.OrderNumber, orderDto.OrderNumber)
		}
		if got.delay != 2*time.Second {
			t.Fatalf("unexpected delay: got %v want 2s", got.delay)
		}
	default:
		t.Fatal("expected delayed order to be enqueued")
	}

	select {
	case got := <-resultJob:
		t.Fatalf("expected no result order, got %#v", got)
	default:
	}
}

func TestWorkerOrderRetryAfterPausesAndEnqueuesDelayedOrder(t *testing.T) {
	t.Parallel()

	job := make(chan dto.OrderDto, 1)
	resultJob := make(chan *accrualStatusOrder.OrderDto, 1)
	queue := make(chan delayedOrder, 1)
	orderDto := dto.OrderDto{OrderNumber: "79927398713", UserID: uuid.New()}
	retryAfter := 4 * time.Second
	pause := &fakePauseController{}

	job <- orderDto
	close(job)

	worker := &WorkerOrder{
		pauseWorkerController: pause,
		extClient: &fakeExternalClient{
			getResultFn: func(context.Context, dto.OrderDto, *zap.Logger) (*accrualStatusOrder.OrderDto, *time.Duration, error) {
				return nil, &retryAfter, nil
			},
		},
		chQueueOrder: queue,
		appLogger:    zap.NewNop(),
	}

	worker.WorkerOrder(context.Background(), job, resultJob)

	if len(pause.pauseDur) != 1 || pause.pauseDur[0] != retryAfter {
		t.Fatalf("unexpected pause calls: got %v want [%v]", pause.pauseDur, retryAfter)
	}

	select {
	case got := <-queue:
		if got.delay != retryAfter {
			t.Fatalf("unexpected delay: got %v want %v", got.delay, retryAfter)
		}
	default:
		t.Fatal("expected delayed order to be enqueued")
	}
}

func TestWorkerOrderRegisteredOrderSendsResultAndRequeues(t *testing.T) {
	t.Parallel()

	job := make(chan dto.OrderDto, 1)
	resultJob := make(chan *accrualStatusOrder.OrderDto, 1)
	queue := make(chan delayedOrder, 1)
	orderDto := dto.OrderDto{OrderNumber: "79927398713", UserID: uuid.New()}
	respDto := &accrualStatusOrder.OrderDto{
		Order:  orderDto.OrderNumber,
		UserID: orderDto.UserID,
		Status: accrualStatusOrder.REGISTERED,
	}

	job <- orderDto
	close(job)

	worker := &WorkerOrder{
		pauseWorkerController: &fakePauseController{},
		extClient: &fakeExternalClient{
			getResultFn: func(context.Context, dto.OrderDto, *zap.Logger) (*accrualStatusOrder.OrderDto, *time.Duration, error) {
				return respDto, nil, nil
			},
		},
		chQueueOrder: queue,
		appLogger:    zap.NewNop(),
	}

	worker.WorkerOrder(context.Background(), job, resultJob)

	select {
	case got := <-resultJob:
		if got != respDto {
			t.Fatalf("unexpected result dto: got %#v want %#v", got, respDto)
		}
	default:
		t.Fatal("expected result order to be sent")
	}

	select {
	case got := <-queue:
		if got.order.OrderNumber != orderDto.OrderNumber || got.delay != 2*time.Second {
			t.Fatalf("unexpected delayed order: %#v", got)
		}
	default:
		t.Fatal("expected delayed order to be enqueued")
	}
}

func TestWorkerOrderProcessedOrderSendsResultOnly(t *testing.T) {
	t.Parallel()

	job := make(chan dto.OrderDto, 1)
	resultJob := make(chan *accrualStatusOrder.OrderDto, 1)
	queue := make(chan delayedOrder, 1)
	orderDto := dto.OrderDto{OrderNumber: "79927398713", UserID: uuid.New()}
	accrual := 25.0
	respDto := &accrualStatusOrder.OrderDto{
		Order:   orderDto.OrderNumber,
		UserID:  orderDto.UserID,
		Status:  accrualStatusOrder.PROCESSED,
		Accrual: &accrual,
	}

	job <- orderDto
	close(job)

	worker := &WorkerOrder{
		pauseWorkerController: &fakePauseController{},
		extClient: &fakeExternalClient{
			getResultFn: func(context.Context, dto.OrderDto, *zap.Logger) (*accrualStatusOrder.OrderDto, *time.Duration, error) {
				return respDto, nil, nil
			},
		},
		chQueueOrder: queue,
		appLogger:    zap.NewNop(),
	}

	worker.WorkerOrder(context.Background(), job, resultJob)

	select {
	case got := <-resultJob:
		if got != respDto {
			t.Fatalf("unexpected result dto: got %#v want %#v", got, respDto)
		}
	default:
		t.Fatal("expected result order to be sent")
	}

	select {
	case got := <-queue:
		t.Fatalf("expected no delayed order for processed status, got %#v", got)
	default:
	}
}

func TestProcessingOrdersEndToEnd(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	accrual := 64.5
	orderCh := make(chan *dto.OrderDto, 1)
	balanceUpdated := make(chan struct{}, 1)

	worker := &WorkerOrder{
		pauseWorkerController: &fakePauseController{},
		extClient: &fakeExternalClient{
			getResultFn: func(_ context.Context, got dto.OrderDto, _ *zap.Logger) (*accrualStatusOrder.OrderDto, *time.Duration, error) {
				if got.OrderNumber != "79927398713" {
					t.Fatalf("unexpected order number from worker: got %q", got.OrderNumber)
				}
				return &accrualStatusOrder.OrderDto{
					Order:   got.OrderNumber,
					UserID:  got.UserID,
					Status:  accrualStatusOrder.PROCESSED,
					Accrual: &accrual,
				}, nil, nil
			},
		},
		orderManager: &fakeOrderManager{
			orderCh: orderCh,
			updateOrderFn: func(_ context.Context, got *models.Order) (models.OrderStatus, error) {
				if got.ID != "79927398713" {
					t.Fatalf("unexpected updated order id: got %q", got.ID)
				}
				if got.Status != models.PROCESSED {
					t.Fatalf("unexpected updated status: got %s want %s", got.Status, models.PROCESSED)
				}
				return models.PROCESSED, nil
			},
		},
		balanceManager: &fakeBalanceManager{
			updateBalanceFn: func(_ context.Context, got *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error) {
				if got.UserID != userID {
					t.Fatalf("unexpected user id in balance update: got %s want %s", got.UserID, userID)
				}
				if got.Order != "79927398713" {
					t.Fatalf("unexpected order in balance update: got %q", got.Order)
				}
				select {
				case balanceUpdated <- struct{}{}:
				default:
				}
				return &models.HistoryBalanceOperation{
					UserID:                 got.UserID,
					OrderID:                got.Order,
					AmountTransactionPoint: got.Accrual,
				}, nil
			},
		},
		numWorkers: 1,
		appLogger:  zap.NewNop(),
	}

	worker.ProcessingOrders(context.Background())
	orderCh <- &dto.OrderDto{OrderNumber: "79927398713", UserID: userID}

	select {
	case <-balanceUpdated:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for end-to-end order processing")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := worker.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("expected graceful shutdown, got %v", err)
	}
}

func TestProcessingOrdersClosedOrderChannelStopsGracefully(t *testing.T) {
	t.Parallel()

	orderCh := make(chan *dto.OrderDto)
	close(orderCh)

	worker := &WorkerOrder{
		pauseWorkerController: &fakePauseController{},
		extClient: &fakeExternalClient{
			getResultFn: func(context.Context, dto.OrderDto, *zap.Logger) (*accrualStatusOrder.OrderDto, *time.Duration, error) {
				t.Fatal("external client should not be called when order channel is already closed")
				return nil, nil, nil
			},
		},
		orderManager: &fakeOrderManager{
			orderCh: orderCh,
			updateOrderFn: func(context.Context, *models.Order) (models.OrderStatus, error) {
				t.Fatal("order manager update should not be called when order channel is closed")
				return models.UNKNOWN, nil
			},
		},
		balanceManager: &fakeBalanceManager{
			updateBalanceFn: func(context.Context, *accrualStatusOrder.OrderDto) (*models.HistoryBalanceOperation, error) {
				t.Fatal("balance manager should not be called when order channel is closed")
				return nil, nil
			},
		},
		numWorkers: 1,
		appLogger:  zap.NewNop(),
	}

	worker.ProcessingOrders(context.Background())

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := worker.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("expected graceful shutdown after closed order channel, got %v", err)
	}
}
