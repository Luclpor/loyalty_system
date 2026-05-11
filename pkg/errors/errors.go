package errors

import "errors"

var (
	ErrorOrderAlreadyUploadThisUser = errors.New("order already upload this user")
	ErrorOrderAlreadyUploadSomeUser = errors.New("order already upload some user")

	ErrorNotFoundRows     = errors.New("rows not found")
	ErrorNotFoundUser     = errors.New("user not found")
	ErrorAlreadyExistUser = errors.New("user already exists")
	ErrorInvalidOrderNum  = errors.New("invalid order number")
	ErrorNotEnoughBalance = errors.New("not enough balance")
)
