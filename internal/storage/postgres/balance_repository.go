package postgres

import (
	"context"
	"errors"

	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BalanceRepository struct {
	pool *pgxpool.Pool
	tx   pgx.Tx
}

func NewBalanceRepository(pool *pgxpool.Pool) *BalanceRepository {
	return &BalanceRepository{pool: pool}
}

func (r *BalanceRepository) CreateNewBalance(ctx context.Context, tr pgx.Tx, userID uuid.UUID) error {
	const query = `INSERT INTO loyalty_system.balance (user_id) VALUES ($1)`
	var err error
	if tr != nil {
		_, err = tr.Exec(ctx, query, userID)
	} else {
		_, err = r.pool.Exec(ctx, query, userID)
	}
	if err != nil {
		return err
	}
	return nil
}

func (r *BalanceRepository) UpdateBalance(ctx context.Context, updEntity *models.Balance) (*models.Balance, error) {
	tr, err := r.pool.Begin(ctx)
	r.tx = tr
	if err != nil {
		return nil, err
	}
	const query = `UPDATE TABLE loyalty_system.balance SET point = $1 WHERE user_id = $2 RETURNING user_id, point;`
	resultEntity := models.Balance{}
	b := models.Balance{}
	err = tr.QueryRow(ctx, query, updEntity.Point, updEntity.UserID).Scan(&b.UserID, &b.Point)
	if err != nil {
		return nil, err
	}
	hisResEntity, err := r.CreateNewBalanceHistoryTransaction(ctx, updEntity.HistoryBalanceOperation)
	if err != nil {
		return nil, err
	}
	resultEntity.HistoryBalanceOperation = hisResEntity
	err = tr.Commit(ctx)
	if err != nil {
		return nil, err
	}
	return &resultEntity, nil
}

func (r *BalanceRepository) BulkUpdateBalances(ctx context.Context, entToUpd []models.Balance) ([]models.Balance, error) {
	tr, err := r.pool.Begin(ctx)
	r.tx = tr
	if err != nil {
		return nil, err
	}
	const query = `UPDATE loyalty_system.balance SET point = $1 WHERE user_id = $2 RETURNING user_id, point;`
	b := []models.Balance{}
	for _, entry := range entToUpd {
		entity := models.Balance{}
		err = tr.QueryRow(ctx, query, entry.Point, entry.UserID).Scan(&entity.UserID, &entity.Point)
		if err != nil {
			return nil, err
		}
		hisResEntity, err := r.CreateNewBalanceHistoryTransaction(ctx, entry.HistoryBalanceOperation)
		if err != nil {
			return nil, err
		}
		entity.HistoryBalanceOperation = hisResEntity
		b = append(b, entity)
	}
	err = tr.Commit(ctx)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *BalanceRepository) CreateNewBalanceHistoryTransaction(ctx context.Context, hisBalanceEntity *models.HistoryBalanceOperation) (*models.HistoryBalanceOperation, error) {
	const query = `INSERT INTO loyalty_system.history_balance_operation (user_id, order_id, is_positive_transaction, amount_transaction_point, balance_points) 
			VALUES ($1, $2, $3, $4, $5) returning user_id, order_id, is_positive_transaction, amount_transaction_point, balance_points;`
	entity := models.HistoryBalanceOperation{}
	var err error
	if r.tx != nil {
		err = r.tx.
			QueryRow(ctx, query, hisBalanceEntity.UserID, hisBalanceEntity.OrderID, hisBalanceEntity.IsPositiveTransaction, hisBalanceEntity.AmountTransactionPoint, hisBalanceEntity.BalancePoint).
			Scan(&entity.UserID, &entity.OrderID, &entity.IsPositiveTransaction, &entity.AmountTransactionPoint, &entity.BalancePoint)
	} else {
		err = r.pool.
			QueryRow(ctx, query, hisBalanceEntity.UserID, hisBalanceEntity.OrderID, hisBalanceEntity.IsPositiveTransaction, hisBalanceEntity.AmountTransactionPoint, hisBalanceEntity.BalancePoint).
			Scan(&entity.UserID, &entity.OrderID, &entity.IsPositiveTransaction, &entity.AmountTransactionPoint, &entity.BalancePoint)
	}
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *BalanceRepository) GetWithDraws(ctx context.Context, userID uuid.UUID) ([]models.HistoryBalanceOperation, error) {
	const query = `SELECT user_id, amount_transaction_point, created_at FROM loyalty_system.history_balance_operation WHERE user_id = $1 and is_positive_transaction = false
		  ORDER BY created_at DESC;`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var operations []models.HistoryBalanceOperation
	for rows.Next() {
		hisOpe := models.HistoryBalanceOperation{}
		err = rows.Scan(&hisOpe.UserID, &hisOpe.AmountTransactionPoint, &hisOpe.CreatedAt)
		if err != nil {
			return nil, err
		}
		operations = append(operations, hisOpe)
	}
	err = rows.Err()
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.ErrorNotFoundRows
		}
		return nil, err
	}
	return operations, nil
}

func (r *BalanceRepository) GetUserBalance(ctx context.Context, userID uuid.UUID) (*models.Balance, error) {
	const query = `SELECT user_id, point FROM loyalty_system.balance WHERE user_id = $1;`
	ent := models.Balance{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(&ent.UserID, &ent.Point)
	if err != nil {
		return nil, err
	}
	return &ent, nil
}

func (r *BalanceRepository) GetUsersBalances(ctx context.Context, userIDs []uuid.UUID) ([]models.Balance, error) {
	const query = `SELECT user_id, point FROM loyalty_system.balance WHERE user_id = ANY($1);`
	rows, err := r.pool.Query(ctx, query, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var balances []models.Balance
	for rows.Next() {
		bal := models.Balance{}
		err = rows.Scan(&bal.UserID, &bal.Point)
		if err != nil {
			return nil, err
		}
		balances = append(balances, bal)
	}
	err = rows.Err()
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.ErrorNotFoundRows
		}
		return nil, err
	}
	return balances, nil
}
