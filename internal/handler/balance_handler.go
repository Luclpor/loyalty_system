package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/service/auth"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance"
	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
	"github.com/go-chi/render"
	"go.uber.org/zap"
)

func WithdrawBalanceHandler(authService *auth.UserAuth, balanceManager *userBalance.BalanceManager, appLogger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := authService.GetUserFromContext(r.Context())
		if err != nil {
			appLogger.Error("failed to get user from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}

		balanceApi := apiModel.BalanceApi{}
		err = json.NewDecoder(r.Body).Decode(&balanceApi)
		if err != nil {
			appLogger.Error("error decoding json", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		wd, err := balanceManager.WithDrawUserBalance(r.Context(), &balanceApi, u, appLogger)
		if err != nil {
			if errors.Is(err, appErrors.ErrorNotEnoughBalance) {
				appLogger.Warn("not enough balance on user")
				w.WriteHeader(http.StatusPaymentRequired)
				return
			}
			if errors.Is(err, appErrors.ErrorInvalidOrderNum) {
				appLogger.Warn("invalid order num", zap.Error(err))
				w.WriteHeader(http.StatusUnprocessableEntity)
				return
			}
			appLogger.Error("error while withdraw", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		appLogger.Info("success withdraw balance", zap.String("order num", balanceApi.Order),
			zap.String("user", wd.UserID.String()))
	}
}

func GetWithdrawBalanceHandler(authService *auth.UserAuth, balanceManager *userBalance.BalanceManager, appLogger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := authService.GetUserFromContext(r.Context())
		if err != nil {
			appLogger.Error("failed to get user from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		withdraws, err := balanceManager.GetUserWithDraws(r.Context(), u.ID)
		if err != nil {
			appLogger.Error("error getting withdraws", zap.Error(err))
			if errors.Is(err, appErrors.ErrorNotFoundRows) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		render.JSON(w, r, withdraws)
	}
}

func GetUserBalanceHandler(authService *auth.UserAuth, balanceManager *userBalance.BalanceManager, appLogger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := authService.GetUserFromContext(r.Context())
		if err != nil {
			appLogger.Error("failed to get user from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		balance, err := balanceManager.GetUserBalance(r.Context(), u.ID)
		if err != nil {
			appLogger.Error("error getting balance", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		render.JSON(w, r, balance)
	}
}
