package handler

import (
	"net/http"

	"github.com/Luclpor/loyalty_system.git/internal/handler/authHandler"
	"go.uber.org/zap"
)

func Order(authService authHandler.UserAuthenticationService, appLogger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := authService.GetUserFromContext(r.Context())
		if err != nil {
			appLogger.Error("failed to get user from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(u.Login + ";" + u.ID.String()))
	}
}
