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
	workerOrd  *worker.WorkerOrder
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
	closers = append(closers, func() error {
		pool.Close()
		return nil
	})
	br := postgres.NewBalanceRepository(pool)
	ur := postgres.NewUserRepository(pool, br)
	or := postgres.NewOrderRepository(pool)
	authService, err = auth.NewAuthService([]byte(cfg.SecretKey), ur)
	if err != nil {
		appLogger.Error("Failed to create auth service", zap.Error(err))
		return nil, err
	}
	balanceManager := userBalance.NewBalanceManager(br)
	orderManager := order.NewOrderManager(cfg.AccrualSystemAddress, or)
	workerOrder, err := worker.NewWorkerOrder(cfg.AccrualSystemAddress, balanceManager, orderManager, appLogger)
	if err != nil {
		appLogger.Error("Failed to create worker order", zap.Error(err))
		return nil, err
	}
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
		workerOrder,
		closers,
		appLogger,
	}

	return server, nil
}

func (s *Server) Start() error {
	appCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	s.workerOrd.ProcessingOrders(appCtx)
	serverErr := make(chan error, 1)
	go func() {
		s.appLogger.Info("Server listening on",
			zap.String("server_address", s.httpServer.Addr),
		)

		if err := s.httpServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()
	select {
	case <-appCtx.Done():
		s.appLogger.Info("Server shutting down...")

	case err := <-serverErr:
		if err != nil {
			s.appLogger.Error("server failed", zap.Error(err))
			return err
		}
		return nil
	}
	httpCtx, httpCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer httpCancel()
	if err := s.httpServer.Shutdown(httpCtx); err != nil {
		s.appLogger.Error("server forced to shutdown", zap.Error(err))
		return err
	}
	workerCtx, workerCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer workerCancel()

	if err := s.workerOrd.Shutdown(workerCtx); err != nil {
		s.appLogger.Error("worker order forced to shutdown", zap.Error(err))
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
