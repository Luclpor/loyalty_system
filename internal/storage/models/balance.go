package models

import "github.com/google/uuid"

type Balance struct {
	ID                      uuid.UUID                `db:"id"`
	UserID                  uuid.UUID                `db:"user_id"`
	Point                   *float64                 `db:"point"`
	HistoryBalanceOperation *HistoryBalanceOperation `db:"history_balance_operation"`
}
