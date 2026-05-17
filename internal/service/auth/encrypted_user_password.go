package auth

import (
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func (ua *UserAuth) hashPassword(password string, appLogger *zap.Logger) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		appLogger.Error("Failed to hash password", zap.Error(err))
		return "", err
	}
	return string(bytes), nil
}

func (ua *UserAuth) checkPassword(providedPassword string, hashPassword string, appLogger *zap.Logger) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(providedPassword))
	if err != nil {
		appLogger.Error("Failed to compare password", zap.Error(err))
		return err
	}
	return nil
}
