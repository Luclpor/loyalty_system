package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/handler/authHandler"
	"github.com/Luclpor/loyalty_system.git/internal/service/order"
	appErrors "github.com/Luclpor/loyalty_system.git/pkg/errors"
	"github.com/go-chi/render"
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
		if err != nil {
			if errors.Is(err, appErrors.ErrorOrderAlreadyUploadSomeUser) {
				w.WriteHeader(http.StatusConflict)
				return
			}
			if errors.Is(err, appErrors.ErrorOrderAlreadyUploadThisUser) {
				w.WriteHeader(http.StatusOK)
				return
			}
			if errors.Is(err, appErrors.ErrorInvalidOrderNum) {
				w.WriteHeader(http.StatusUnprocessableEntity)
				return
			}
			appLogger.Error("failed to save new order", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func GetUserOrdersHandler(authService authHandler.UserAuthenticationService, orderManager *order.OrderManager, appLogger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := authService.GetUserFromContext(r.Context())
		if err != nil {
			appLogger.Error("failed to get user from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		ords, err := orderManager.GetUserOrders(r.Context(), u.ID)
		if err != nil {
			appLogger.Error("failed to get user's orders", zap.Error(err))
			if errors.Is(err, appErrors.ErrorInvalidOrderNum) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		readApi := make([]apiModel.ReadOrderApiModel, len(ords))
		for i, o := range ords {
			readApi[i] = apiModel.ReadOrderApiModel{
				OrderNum:   strconv.Itoa(o.OrderNumber),
				Accrual:    o.Point,
				Status:     o.Status,
				UploadedAt: o.UploadedAt,
			}
		}
		render.JSON(w, r, readApi)
	}
}
