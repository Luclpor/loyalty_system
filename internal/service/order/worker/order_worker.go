package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/Luclpor/loyalty_system.git/internal/service/order"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/common"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto/accrualStatusOrder"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

type WorkerOrder struct {
	pauseWorkerController pauseWorkerController
	extClient             *externalRtrClient
	chQueueOrder          chan delayedOrder
	numWorkers            int
	accrualSystemAddress  string
	balanceManager        *userBalance.BalanceManager
	orderManager          *order.OrderManager
	appLogger             *zap.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type delayedOrder struct {
	order dto.OrderDto
	delay time.Duration
}

type externalRtrClient struct {
	client  *retryablehttp.Client
	baseURL *url.URL
}

func newRetryableHttpClient(baseUrl string) (*externalRtrClient, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	client := retryablehttp.NewClient()
	client.RetryMax = 5
	client.RetryWaitMin = 500 * time.Millisecond
	client.RetryWaitMax = 10 * time.Second
	client.HTTPClient.Timeout = time.Second * 10
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			return true, nil
		}

		if resp == nil {
			return false, nil
		}

		switch resp.StatusCode {
		case http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true, nil
		default:
			return false, nil
		}
	}
	return &externalRtrClient{
		baseURL: u,
		client:  client,
	}, nil
}

func NewWorkerOrder(accrualSysAddress string, balanceManager *userBalance.BalanceManager, orderManager *order.OrderManager, appLogger *zap.Logger) (*WorkerOrder, error) {
	cl, err := newRetryableHttpClient(accrualSysAddress)
	if err != nil {
		return nil, err
	}
	return &WorkerOrder{
		extClient:            cl,
		accrualSystemAddress: accrualSysAddress,
		numWorkers:           5,
		balanceManager:       balanceManager,
		appLogger:            appLogger,
		orderManager:         orderManager,
	}, nil
}

func (p *WorkerOrder) WorkerOrder(ctx context.Context, job chan dto.OrderDto, resultJob chan *accrualStatusOrder.OrderDto) {
	for {
		select {
		case <-ctx.Done():
			return
		case j, ok := <-job:
			if !ok {
				return
			}
			if err := p.pauseWorkerController.Wait(ctx); err != nil {
				p.appLogger.Error("worker pause interrupted", zap.Error(err))
				continue
			}
			respDto, retryAfter, err := p.extClient.getResultFromExternalSystem(
				ctx,
				j,
				p.appLogger,
			)
			if err != nil {
				p.appLogger.Error("Error getting result from external system", zap.Error(err))
				if !p.enqueueDelayedOrder(ctx, j, 2*time.Second) {
					return
				}
				continue
			}
			if retryAfter != nil {
				p.pauseWorkerController.Pause(*retryAfter)
				if !p.enqueueDelayedOrder(ctx, j, *retryAfter) {
					return
				}

				continue
			}

			if respDto == nil {
				continue
			}
			select {
			case resultJob <- respDto:
			case <-ctx.Done():
				return
			}

			if respDto.Status == accrualStatusOrder.REGISTERED ||
				respDto.Status == accrualStatusOrder.PROCESSING {
				if !p.enqueueDelayedOrder(ctx, j, 2*time.Second) {
					return
				}

				continue
			}
		}
	}
}

func (p *WorkerOrder) enqueueDelayedOrder(ctx context.Context, order dto.OrderDto, delay time.Duration) bool {
	select {
	case p.chQueueOrder <- delayedOrder{
		order: order,
		delay: delay,
	}:
		return true

	case <-ctx.Done():
		return false
	}
}

func (p *WorkerOrder) ProcessingOrders(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	jobs := make(chan dto.OrderDto, p.numWorkers)
	resultJob := make(chan *accrualStatusOrder.OrderDto, p.numWorkers)
	p.chQueueOrder = make(chan delayedOrder)

	var workersWg sync.WaitGroup

	for n := 0; n < p.numWorkers; n++ {
		workersWg.Add(1)
		p.wg.Add(1)
		go func() {
			defer workersWg.Done()
			defer p.wg.Done()
			p.WorkerOrder(ctx, jobs, resultJob)
		}()
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		workersWg.Wait()
		close(resultJob)
	}()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer cancel()
		defer close(jobs)
		for {
			select {
			case <-ctx.Done():
				p.appLogger.Info("worker order dispatcher stopped")
				return
			case j, ok := <-p.orderManager.NewOrderChan:
				if !ok {
					p.appLogger.Info("new order channel closed")
					close(jobs)
					return
				}
				if j == nil {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case jobs <- *j:
				}

			case res := <-resultJob:
				if res == nil {
					continue
				}
				p.HandleEvaluatingOrders(ctx, res)
			case delayed := <-p.chQueueOrder:
				p.runDelayedJob(ctx, jobs, delayed)
			}
		}
	}()
}

func (p *WorkerOrder) runDelayedJob(ctx context.Context, jobs chan<- dto.OrderDto, delayed delayedOrder) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		timer := time.NewTimer(delayed.delay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			select {
			case jobs <- delayed.order:
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (p *WorkerOrder) HandleEvaluatingOrders(ctx context.Context, order *accrualStatusOrder.OrderDto) {
	convStatus, err := common.ConvertStatus(order.Status)
	if err != nil {
		p.appLogger.Error("Error converting status", zap.Error(err))
		return
	}
	status, err := p.orderManager.UpdateOrder(ctx, &models.Order{
		ID:     order.Order,
		Status: convStatus,
	})
	if err != nil {
		p.appLogger.Error("Error updating orders", zap.Error(err))
		return
	}
	if status == models.UNKNOWN {
		p.appLogger.Error("unknown status order", zap.String("status", string(order.Status)))
		return
	}
	if status == models.PROCESSED {
		his, err := p.balanceManager.UpdateBalance(ctx, order)
		if err != nil {
			p.appLogger.Error("Error updating balance", zap.Error(err))
			return
		}
		p.appLogger.Info("Updated balance", zap.String("Order number", his.OrderID), zap.Float64("Point", *his.AmountTransactionPoint))
	}
}

func (ex *externalRtrClient) getResultFromExternalSystem(
	ctx context.Context,
	order dto.OrderDto,
	appLogger *zap.Logger,
) (*accrualStatusOrder.OrderDto, *time.Duration, error) {
	u := ex.baseURL.JoinPath("api/orders/", order.OrderNumber)

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := ex.client.Do(req)
	if err != nil {
		appLogger.Error("Error getting order", zap.Error(err))
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		str := resp.Header.Get("Retry-After")

		seconds, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			appLogger.Error(
				"Error parsing Retry-After",
				zap.String("retry_after", str),
				zap.Error(err),
			)
			return nil, nil, err
		}

		dur := time.Duration(seconds) * time.Second
		return nil, &dur, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected accrual status code: %d", resp.StatusCode)
	}

	model := &accrualStatusOrder.OrderDto{}
	if err := json.NewDecoder(resp.Body).Decode(model); err != nil {
		appLogger.Error("error decode resp from accrual system order", zap.Error(err))
		return nil, nil, err
	}

	model.UserID = order.UserID

	return model, nil, nil
}

func (p *WorkerOrder) Shutdown(ctx context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}

	done := make(chan struct{})

	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.appLogger.Info("worker order stopped gracefully")
		return nil

	case <-ctx.Done():
		p.appLogger.Error("worker order shutdown timeout", zap.Error(ctx.Err()))
		return ctx.Err()
	}
}
