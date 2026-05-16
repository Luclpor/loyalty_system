package validator

import (
	"context"
	"errors"

	"github.com/Luclpor/loyalty_system.git/internal/storage/postgres"
)

type UserValidator interface {
	CheckExistLogin(ctx context.Context, login string) (*bool, error)
}

var (
	ErrorAlreadyExistUser = errors.New("user already exist")
)

func ValidationUserLogin(ctx context.Context, login string, userValidator UserValidator) error {
	exist, err := userValidator.CheckExistLogin(ctx, login)
	if err != nil {
		if errors.Is(err, postgres.ErrorNotFoundUser) {
			return nil
		}
		return err
	}
	if *exist {
		return ErrorAlreadyExistUser
	}
	return nil
}
