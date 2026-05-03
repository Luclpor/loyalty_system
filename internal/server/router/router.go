package router

import (
	"time"

	"github.com/Luclpor/loyalty_system.git/internal/config"
	"github.com/Luclpor/loyalty_system.git/internal/handler"
	"github.com/Luclpor/loyalty_system.git/internal/handler/authHandler"
	customMiddleware "github.com/Luclpor/loyalty_system.git/internal/server/middleware"
	"github.com/Luclpor/loyalty_system.git/internal/service/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func NewRouter(cfg *config.Config, authService *auth.UserAuth, appLogger *zap.Logger) (*chi.Mux, error) {

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(customMiddleware.RequestLogger(appLogger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.Timeout * time.Second))
	r.Post("/api/user/register", authHandler.UserRegistration(authService, appLogger))
	r.Post("/api/user/login", authHandler.UserLogin(authService, appLogger))
	r.Route("/", func(r chi.Router) {
		r.Use(customMiddleware.Auth(authService, appLogger))
		r.Get("/check", handler.Order(authService, appLogger))
	})

	return r, nil
}
