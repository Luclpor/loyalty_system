package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Luclpor/loyalty_system.git/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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
	err = runMigrations(cfg.DataBaseDSN)
	if err != nil {
		appLogger.Error("failed to run migrations", zap.Error(err))
		return nil, err
	}
	return pool, nil
}

func runMigrations(databaseURL string) error {
	m, err := migrate.New(
		"file://migrations",
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			log.Printf("close migrate source: %v", sourceErr)
		}
		if dbErr != nil {
			log.Printf("close migrate db: %v", dbErr)
		}
	}()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
