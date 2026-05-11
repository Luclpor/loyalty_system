package errors

import "errors"

var (
	ErrorNotFoundRows     = errors.New("rows not found")
	ErrorNotFoundUser     = errors.New("user not found")
	ErrorAlreadyExistUser = errors.New("user already exists")
	ErrorInvalidOrderNum  = errors.New("invalid order number")
	ErrorNotEnoughBalance = errors.New("not enough balance")
)
