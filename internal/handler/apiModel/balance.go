package apiModel

type BalanceApi struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type ReadBalanceApi struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
