package auth

import (
	"context"

	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/service/auth/validator"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	"go.uber.org/zap"
)

func NewAuthService(jwtKey []byte, ur UserRepository) (*UserAuth, error) {
	return &UserAuth{jwtKey, ur}, nil
}

type UserRepository interface {
	CreateUserAndBalance(ctx context.Context, login string, hashPassword string) (*models.User, error)
	GetUserByLogin(ctx context.Context, login string) (*models.User, error)
	CheckExistLogin(ctx context.Context, login string) (*bool, error)
}

//go:generate mockgen -source=user_registration.go -destination=../../storage/mock/mock_user_repository.go -package=mock

type UserAuth struct {
	jwtKey   []byte
	userRepo UserRepository
}

func (ua *UserAuth) RegisterUser(ctx context.Context, user *apiModel.User, appLogger *zap.Logger) (*models.User, error) {
	err := validator.ValidationUserLogin(ctx, user.Login, ua.userRepo)
	if err != nil {
		appLogger.Error("User validation failed", zap.Error(err))
		return nil, err
	}

	hashedPass, err := ua.hashPassword(user.Password, appLogger)
	if err != nil {
		appLogger.Error("Failed to hash user password", zap.Error(err))
		return nil, err
	}

	u, err := ua.userRepo.CreateUserAndBalance(ctx, user.Login, hashedPass)
	if err != nil {
		appLogger.Error("Failed to create user", zap.Error(err))
		return nil, err
	}
	return u, nil
}

func (ua *UserAuth) LoginUser(ctx context.Context, user *apiModel.User, appLogger *zap.Logger) (*models.User, error) {
	u, err := ua.userRepo.GetUserByLogin(ctx, user.Login)
	if err != nil {
		appLogger.Error("Failed to get user by login", zap.Error(err))
		return nil, err
	}
	err = ua.checkPassword(user.Password, u.Password, appLogger)
	if err != nil {
		appLogger.Error("Invalid password", zap.Error(err))
		return nil, err
	}
	return u, nil
}
