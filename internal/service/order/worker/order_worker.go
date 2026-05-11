package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/Luclpor/loyalty_system.git/internal/service/order"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/common"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto/accrualStatusOrder"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance/balanceDto"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"go.uber.org/zap"
)

type WorkerOrder struct {
	mu                   sync.Mutex
	numWorkers           int
	resultAccOrders      []accrualStatusOrder.OrderDto
	accrualSystemAddress string
	balanceManager       *userBalance.BalanceManager
	orderManager         *order.OrderManager
	appLogger            *zap.Logger
}

func NewWorkerOrder(accrualSysAddress string, balanceManager *userBalance.BalanceManager, orderManager *order.OrderManager, appLogger *zap.Logger) *WorkerOrder {
	return &WorkerOrder{
		accrualSystemAddress: accrualSysAddress,
		numWorkers:           5,
		balanceManager:       balanceManager,
		appLogger:            appLogger,
		orderManager:         orderManager,
	}
}

func (p *WorkerOrder) WorkerOrder(job chan order.OrderDto, resultJob chan *accrualStatusOrder.OrderDto) {
	for j := range job {
		respDto := p.GetResultOrderEvaluating(j)
		resultJob <- respDto
	}
}

func (p *WorkerOrder) ProcessingOrders() {
	jobs := make(chan order.OrderDto, p.numWorkers)
	resultJob := make(chan *accrualStatusOrder.OrderDto, p.numWorkers)
	for n := 0; n < p.numWorkers; n++ {
		go p.WorkerOrder(jobs, resultJob)
	}
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for range t.C {
			p.mu.Lock()
			if len(p.resultAccOrders) == 0 {
				p.appLogger.Info("empty new orders")
				p.mu.Unlock()
				continue
			}
			orders := append([]accrualStatusOrder.OrderDto(nil), p.resultAccOrders...)
			p.resultAccOrders = p.resultAccOrders[:0]
			p.mu.Unlock()
			p.HandleEvaluatingOrders(orders)
		}
	}()
	go func() {
		for {
			select {
			case j := <-p.orderManager.NewOrderChan:
				jobs <- *j
			case res := <-resultJob:
				if res == nil {
					continue
				}
				p.mu.Lock()
				p.resultAccOrders = append(p.resultAccOrders, *res)
				p.mu.Unlock()
			}
		}
	}()
}

func (p *WorkerOrder) HandleEvaluatingOrders(orders []accrualStatusOrder.OrderDto) {
	balanceUpdOrders := make([]balanceDto.BalanceDto, 0)
	updatedOrders := make([]models.Order, 0, len(orders))
	for _, o := range orders {
		num, _ := strconv.Atoi(o.Order)
		if o.Status == accrualStatusOrder.PROCESSED {
			balanceUpdOrders = append(balanceUpdOrders, balanceDto.BalanceDto{
				OrderID: int64(num),
				Point:   o.Accrual,
				UserID:  o.UserID,
			})
		}

		updatedOrders = append(updatedOrders, models.Order{
			ID:     num,
			Status: common.ConvertStatus(o.Status),
		})
	}
	err := p.balanceManager.BulkUpdateBalance(context.TODO(), balanceUpdOrders)
	if err != nil {
		p.appLogger.Error("Error updating balance", zap.Error(err))
		return
	}
	err = p.orderManager.UpdateOrders(context.TODO(), updatedOrders)
	if err != nil {
		p.appLogger.Error("Error updating orders", zap.Error(err))
		return
	}
}

func (p *WorkerOrder) GetResultOrderEvaluating(order order.OrderDto) *accrualStatusOrder.OrderDto {
	ordNum := strconv.FormatInt(int64(order.OrderNumber), 10)
	u, err := url.JoinPath(p.accrualSystemAddress, "api/orders", ordNum)
	if err != nil {
		panic(err)
	}
	client := &http.Client{
		Timeout: time.Second * 15,
	}
	resp, err := client.Get(u)
	if err != nil {
		p.appLogger.Error("Error getting order", zap.Error(err))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {

	}
	model := &accrualStatusOrder.OrderDto{}
	err = json.NewDecoder(resp.Body).Decode(&model)
	if err != nil {
		p.appLogger.Error("error decode resp from accrual system order", zap.Error(err))
		return nil
	}
	model.UserID = order.UserID
	return model
}
