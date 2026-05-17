package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/service/order"
	"github.com/go-chi/render"
	"go.uber.org/zap"
)

func CreateNewOrder(authService userContextGetter, orderManager orderManagerService, appLogger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := authService.GetUserFromContext(r.Context())
		if err != nil {
			appLogger.Error("failed to get user from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
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
			if errors.Is(err, order.ErrorOrderAlreadyUploadSomeUser) {
				w.WriteHeader(http.StatusConflict)
				return
			}
			if errors.Is(err, order.ErrorOrderAlreadyUploadThisUser) {
				w.WriteHeader(http.StatusOK)
				return
			}
			if errors.Is(err, order.ErrorInvalidOrderNum) {
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

func GetUserOrdersHandler(authService userContextGetter, orderManager orderManagerService, appLogger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := authService.GetUserFromContext(r.Context())
		if err != nil {
			appLogger.Error("failed to get user from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ords, err := orderManager.GetUserOrders(r.Context(), u.ID)
		if err != nil {
			appLogger.Error("failed to get user's orders", zap.Error(err))
			if errors.Is(err, order.ErrorInvalidOrderNum) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		readApi := make([]apiModel.ReadOrderApiModel, len(ords))
		for i, o := range ords {
			readApi[i] = apiModel.ReadOrderApiModel{
				OrderNum:   o.OrderNumber,
				Accrual:    o.Point,
				Status:     o.Status,
				UploadedAt: o.UploadedAt,
			}
		}
		render.JSON(w, r, readApi)
	}
}
