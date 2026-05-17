package validator

import (
	"context"
	"errors"

	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
)

type UserValidator interface {
	CheckExistLogin(ctx context.Context, login string) (*bool, error)
}

func ValidationUserLogin(ctx context.Context, login string, userValidator UserValidator) error {
	exist, err := userValidator.CheckExistLogin(ctx, login)
	if err != nil {
		if errors.Is(err, appErrors.ErrorNotFoundUser) {
			return nil
		}
		return err
	}
	if *exist {
		return appErrors.ErrorAlreadyExistUser
	}
	return nil
}
