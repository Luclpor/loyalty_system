package errors

import "errors"

var (
	ErrorNotFoundUser     = errors.New("user not found")
	ErrorAlreadyExistUser = errors.New("user already exists")
)
