package postgres

import (
	"context"

	"github.com/Luclpor/loyalty_system.git/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func NewPool(ctx context.Context, cfg *config.PostgresConfig, appLogger *zap.Logger) (*pgxpool.Pool, error) {
	pgxConfig, err := pgxpool.ParseConfig(cfg.DataBaseDSN)
	if err != nil {
		appLogger.Error("failed to parse database DSN", zap.Error(err))
		return nil, err
	}
	pgxConfig.MaxConns = cfg.MaxConns
	pgxConfig.MinConns = cfg.MinConns
	pgxConfig.MaxConnLifetime = cfg.MaxConnLifetime
	pgxConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	pgxConfig.HealthCheckPeriod = cfg.HealthCheckPeriod
	pool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		appLogger.Error("failed to connect to database", zap.Error(err))
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		appLogger.Error("failed to ping database", zap.Error(err))
		return nil, err
	}
	return pool, nil
}
