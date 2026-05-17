package apiModel

import "time"

type BalanceApi struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type ReadBalanceApi struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type ReadWithdrawApi struct {
	OrderID     string     `json:"order"`
	Sum         float64    `json:"sum"`
	ProcessedAt *time.Time `json:"processed_at"`
}
