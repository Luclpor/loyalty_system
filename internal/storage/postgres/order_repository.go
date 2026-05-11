package postgres

import (
	"context"
	"errors"

	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (or *OrderRepository) SaveOrder(ctx context.Context, order *models.Order) error {
	const query = `
		INSERT INTO loyalty_system.order (id, user_id, status)
		VALUES ($1, $2, $3)
	`
	_, err := or.pool.Exec(ctx, query, order.ID, order.UserID, order.Status)
	if err != nil {
		return err
	}
	return nil
}

func (or *OrderRepository) GetOrderByID(ctx context.Context, orderID int64) (*models.Order, error) {
	const query = `SELECT id, user_id, status, created_at FROM loyalty_system.order WHERE id = $1`
	order := &models.Order{}
	err := or.pool.QueryRow(ctx, query, orderID).Scan(&order.ID, &order.UserID, &order.Status, &order.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.ErrorNotFoundRows
		}
		return nil, err
	}
	return order, nil
}

func (or *OrderRepository) GetOrderByStatus(ctx context.Context, status string) ([]models.Order, error) {
	const query = `
       SELECT id, user_id, status FROM loyalty_system.order WHERE status = $1
       `
	rows, err := or.pool.Query(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ords := make([]models.Order, 0)
	for rows.Next() {
		order := models.Order{}
		err = rows.Scan(&order.ID, &order.UserID, &order.Status)
		if err != nil {
			return nil, err
		}
		ords = append(ords, order)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return ords, nil
}

func (or *OrderRepository) UpdateOrdersStatus(ctx context.Context, entToUpd []models.Order) error {
	tr, err := or.pool.Begin(ctx)
	if err != nil {
		return err
	}
	const query = `UPDATE loyalty_system.order SET status = $1 WHERE id = $2;`
	for _, ent := range entToUpd {
		_, err = tr.Exec(ctx, query, ent.Status, ent.ID)
		if err != nil {
			return err
		}
	}
	err = tr.Commit(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (or *OrderRepository) GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]models.Order, error) {
	const query = `SELECT o.id, o.user_id, o.status, o.created_at, hbo.id, hbo.amount_transaction_point, hbo.is_positive_transaction FROM loyalty_system.order o
				   LEFT JOIN loyalty_system.history_balance_operation hbo on hbo.order_id = o.id  
                                       WHERE o.user_id = $1`
	orders := make([]models.Order, 0)
	rows, err := or.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		order := models.Order{}
		var hisId pgtype.Int4
		var amount *float64
		var isPositive pgtype.Bool
		err = rows.Scan(&order.ID, &order.UserID, &order.Status, &order.CreatedAt, &hisId, &amount, &isPositive)
		if err != nil {
			return nil, err
		}
		if hisId.Valid {
			order.Transaction = &models.HistoryBalanceOperation{
				AmountTransactionPoint: amount,
				IsPositiveTransaction:  isPositive.Bool,
			}
		}
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.ErrorNotFoundRows
		}
		return nil, err
	}
	return orders, nil
}
