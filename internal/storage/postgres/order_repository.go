package postgres

import (
	"context"

	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
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
