package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/handler/authHandler"
	"github.com/Luclpor/loyalty_system.git/internal/service/order"
	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
	"go.uber.org/zap"
)

func CreateNewOrder(authService authHandler.UserAuthenticationService, orderManager *order.OrderManager, appLogger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := authService.GetUserFromContext(r.Context())
		if err != nil {
			appLogger.Error("failed to get user from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m := &apiModel.OrderApiModel{
			UserID:   u.ID,
			OrderNum: string(body),
		}
		err = orderManager.SaveNewOrder(r.Context(), m)
		if err != nil && errors.Is(err, appErrors.ErrorInvalidOrderNum) {
			appLogger.Error("failed save order", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err != nil {
			appLogger.Error("failed to save new order", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
	}
}
