package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Luclpor/loyalty_system.git/internal/config"
	"github.com/Luclpor/loyalty_system.git/internal/config/db"
	"github.com/Luclpor/loyalty_system.git/internal/logger"
	"github.com/Luclpor/loyalty_system.git/internal/server/router"
	"github.com/Luclpor/loyalty_system.git/internal/service/auth"
	"github.com/Luclpor/loyalty_system.git/internal/service/order"
	"github.com/Luclpor/loyalty_system.git/internal/service/order/worker"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance"
	"github.com/Luclpor/loyalty_system.git/internal/storage/postgres"
	"go.uber.org/zap"
)

const (
	shutdownTimeout = 10 * time.Second
)

type Server struct {
	httpServer *http.Server
	closers    []func() error
	appLogger  *zap.Logger
}

func NewServer() (*Server, error) {
	cfg, err := config.InitConfig()
	if err != nil {
		return nil, err
	}
	appLogger, err := logger.New(cfg.AppEnv)
	if err != nil {
		return nil, err
	}
	var authService *auth.UserAuth
	var closers []func() error
	pool, err := postgres.NewPool(context.Background(), cfg.Postgres, appLogger)
	if err != nil {
		appLogger.Error("Failed to connect to database", zap.Error(err))
		return nil, err
	}
	br := postgres.NewBalanceRepository(pool)
	ur := postgres.NewUserRepository(pool, br)
	or := postgres.NewOrderRepository(pool)
	err = db.RunMigrations(cfg.Postgres.DataBaseDSN)
	if err != nil {
		appLogger.Error("Could not run migrations", zap.Error(err))
		//return nil, err
	}
	authService, err = auth.NewAuthService([]byte(cfg.SecretKey), ur)
	if err != nil {
		appLogger.Error("Failed to create auth service", zap.Error(err))
		return nil, err
	}
	balanceManager := userBalance.NewBalanceManager(br)
	orderManager := order.NewOrderManager(cfg.AccrualSystemAddress, or)
	workerOrder := worker.NewWorkerOrder(cfg.AccrualSystemAddress, balanceManager, orderManager, appLogger)
	workerOrder.ProcessingOrders()
	chiRouter, err := router.NewRouter(cfg, authService, orderManager, balanceManager, appLogger)
	if err != nil {
		appLogger.Error("Could not initialize router", zap.Error(err))
		return nil, err
	}

	server := &Server{
		&http.Server{
			Addr:         cfg.ServerAddress,
			Handler:      chiRouter,
			ReadTimeout:  cfg.Timeout,
			WriteTimeout: cfg.Timeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		closers,
		appLogger,
	}

	return server, nil
}

func (s *Server) Start() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)

	go func() {
		s.appLogger.Info("Server listening on",
			zap.String("server_address", s.httpServer.Addr),
		)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.appLogger.Fatal("Error starting server", zap.Error(err))
		}
	}()

	<-quit
	s.appLogger.Info("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.appLogger.Error("server forced to shutdown", zap.Error(err))
		return err
	}
	for _, closeFn := range s.closers {
		if err := closeFn(); err != nil {
			s.appLogger.Error("close function failed", zap.Error(err))
			return err
		}
	}
	s.appLogger.Info("Server exited properly")
	return nil
}
