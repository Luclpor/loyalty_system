package logger

import (
	"errors"

	"github.com/Luclpor/loyalty_system.git/internal/config"
	"go.uber.org/zap"
)

func New(env config.AppEnvironment) (*zap.Logger, error) {
	switch env {
	case config.ProdEnv:
		logger, err := zap.NewProduction()
		if err != nil {
			return nil, err
		}
		return logger, nil
	case config.DevEnv:
		logger, err := zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
		return logger, nil
	default:
		return nil, errors.New("invalid app environment")
	}
}
