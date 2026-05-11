package authHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Luclpor/loyalty_system.git/internal/dto"
	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/storage/models"
	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UserAuthenticationService interface {
	RegisterUser(ctx context.Context, user *apiModel.User, appLogger *zap.Logger) (*models.User, error)
	ValidateToken(signedToken string) (*dto.UserDto, error)
	GenerateJWT(ID uuid.UUID, username string) (tokenString string, err error)
	LoginUser(ctx context.Context, user *apiModel.User, appLogger *zap.Logger) (*models.User, error)
	GetUserFromContext(ctx context.Context) (*dto.UserDto, error)
}

func UserRegistration(authService UserAuthenticationService, appLogger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var model apiModel.User
		err := json.NewDecoder(r.Body).Decode(&model)
		if err != nil {
			appLogger.Error("failed decode request api model", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		user, err := authService.RegisterUser(r.Context(), &model, appLogger)
		if err != nil {
			if errors.Is(err, appErrors.ErrorAlreadyExistUser) {
				w.WriteHeader(http.StatusConflict)
				return
			}
			appLogger.Error("failed register user", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		if user != nil {
			appLogger.Info("register user", zap.Any("user", user))
			token, err := authService.GenerateJWT(user.ID, user.Login)
			if err != nil {
				appLogger.Error("failed generate token", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Authorization", "Bearer "+token)
			w.WriteHeader(http.StatusOK)
			return
		}
		appLogger.Warn("user registration failed")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func UserLogin(authService UserAuthenticationService, appLogger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var model apiModel.User
		err := json.NewDecoder(r.Body).Decode(&model)
		if err != nil {
			appLogger.Error("failed decode request api model", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		u, err := authService.LoginUser(r.Context(), &model, appLogger)
		if err != nil {
			if errors.Is(err, appErrors.ErrorNotFoundUser) {
				appLogger.Warn("user not found")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			appLogger.Error("failed login user", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		token, err := authService.GenerateJWT(u.ID, u.Login)
		if err != nil {
			appLogger.Error("failed generate token", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Authorization", "Bearer "+token)
		w.WriteHeader(http.StatusOK)
	}
}
