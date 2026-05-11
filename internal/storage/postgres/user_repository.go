package postgres

import (
	"context"
	"errors"

	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool       *pgxpool.Pool
	balanceRep *BalanceRepository
}

func NewUserRepository(pool *pgxpool.Pool, br *BalanceRepository) *UserRepository {
	return &UserRepository{pool: pool, balanceRep: br}
}

func (ur *UserRepository) CreateUserAndBalance(ctx context.Context, login string, hashPassword string) (*models.User, error) {
	tr, err := ur.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tr.Rollback(ctx)
	const query = `
		INSERT INTO loyalty_system.user (login, password)
		VALUES ($1, $2)
		RETURNING id, login, password
	`
	var u models.User
	err = tr.QueryRow(ctx, query, login, hashPassword).Scan(&u.ID, &u.Login, &u.Password)
	if err != nil {
		return nil, err
	}
	err = ur.balanceRep.CreateNewBalance(ctx, tr, u.ID)
	if err != nil {
		return nil, err
	}
	err = tr.Commit(ctx)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (ur *UserRepository) CheckExistLogin(ctx context.Context, login string) (*bool, error) {
	const query = `
		select EXISTS (SELECT 1 FROM loyalty_system."user" u WHERE u.login= $1)
	`

	var exist bool
	err := ur.pool.QueryRow(ctx, query, login).Scan(&exist)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.ErrorNotFoundUser
		}
		return nil, err
	}
	return &exist, nil
}

func (ur *UserRepository) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	const query = `
		SELECT u.id, u.login, u.password FROM loyalty_system."user" u WHERE u.login= $1
	`

	var u models.User
	err := ur.pool.QueryRow(ctx, query, login).Scan(&u.ID, &u.Login, &u.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.ErrorNotFoundUser
		}
		return nil, err
	}
	return &u, nil
}
