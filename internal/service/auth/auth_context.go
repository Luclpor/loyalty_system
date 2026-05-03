package auth

import (
	"context"

	"github.com/Luclpor/loyalty_system.git/internal/dto"
	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
)

type contextKey string

const userContextKey contextKey = "user"

func (ua *UserAuth) SetUserOnContext(ctx context.Context, user *dto.UserDto) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func (ua *UserAuth) GetUserFromContext(ctx context.Context) (*dto.UserDto, error) {
	user, ok := ctx.Value(userContextKey).(*dto.UserDto)
	if !ok {
		return nil, appErrors.ErrorNotFoundUser
	}
	return user, nil
}
