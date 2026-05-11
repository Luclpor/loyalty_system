package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Luclpor/loyalty_system.git/internal/service/order"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/dto/accrualStatusOrder"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance/balanceDto"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"go.uber.org/zap"
)

type WorkerOrder struct {
	numWorkers          int
	orderChan           chan order.OrderDto
	resultAccOrders     []accrualStatusOrder.OrderDto
	accrualSystemStatus string
	balanceManager      *userBalance.BalanceManager
	orderManager        *order.OrderManager
	appLogger           *zap.Logger
}

func NewWorkerOrder(balanceManager *userBalance.BalanceManager, orderManager *order.OrderManager, appLogger *zap.Logger) *WorkerOrder {
	return &WorkerOrder{
		numWorkers:     5,
		orderChan:      make(chan order.OrderDto),
		balanceManager: balanceManager,
		appLogger:      appLogger,
		orderManager:   orderManager,
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
		t := time.NewTimer(5 * time.Second)
		for {
			select {
			case <-t.C:
				if len(p.resultAccOrders) == 0 {
					continue
				}
			}
		}
	}()
	for {
		select {
		case j := <-p.orderChan:
			jobs <- j
		case res := <-resultJob:
			p.resultAccOrders = append(p.resultAccOrders, *res)
		}
	}
}

func (p *WorkerOrder) HandleEvaluatingOrders(orders []accrualStatusOrder.OrderDto) {
	balanceUpdOrders := make([]balanceDto.BalanceDto, 0)
	updatedOrders := make([]models.Order, 0, len(orders))
	for _, o := range orders {
		num, _ := strconv.Atoi(o.Order)
		if o.Status == accrualStatusOrder.PROCESSED {
			balanceUpdOrders = append(balanceUpdOrders, balanceDto.BalanceDto{
				OrderID: num,
				Point:   o.Accrual,
				UserID:  o.UserID,
			})
		}

		updatedOrders = append(updatedOrders, models.Order{
			ID:     num,
			Status: convertStatus(o.Status),
		})
	}
	err := p.balanceManager.BulkUpdateBalance(context.TODO(), balanceUpdOrders)
	if err != nil {
		p.appLogger.Error("Error updating balance", zap.Error(err))
	}
	err = p.orderManager.UpdateOrders(context.TODO(), updatedOrders)
	if err != nil {
		p.appLogger.Error("Error updating orders", zap.Error(err))
	}
}

func convertStatus(statusAccr accrualStatusOrder.AccrualSystemStatus) models.OrderStatus {
	switch statusAccr {
	case accrualStatusOrder.REGISTERED, accrualStatusOrder.PROCESSING:
		return models.PROCESSING
	case accrualStatusOrder.PROCESSED:
		return models.PROCESSED
	case accrualStatusOrder.INVALID:
		return models.INVALID
	default:
		return models.INVALID
	}
}

func (p *WorkerOrder) GetResultOrderEvaluating(order order.OrderDto) *accrualStatusOrder.OrderDto {
	u, err := url.JoinPath(p.accrualSystemStatus, strconv.FormatInt(int64(order.OrderNumber), 10))
	if err != nil {
		panic(err)
	}
	resp, err := http.Get(u)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusOK {

	}
	model := &accrualStatusOrder.OrderDto{}
	err = json.NewDecoder(resp.Body).Decode(&model)
	if err != nil {
		panic(err)
	}
	model.UserID = order.UserID
	return model
}
