package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Luclpor/loyalty_system.git/internal/dto"
	"go.uber.org/zap"
)

type UserAuthenticationMiddleware interface {
	ValidateToken(signedToken string) (*dto.UserDto, error)
	SetUserOnContext(ctx context.Context, user *dto.UserDto) context.Context
}

func Auth(authService UserAuthenticationMiddleware, appLogger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := r.Header.Get("Authorization")
			if tokenString == "" {
				appLogger.Error("Authorization header missing", zap.String("token", r.Header.Get("Authorization")))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			signedToken, err := extractBearerToken(tokenString)
			if err != nil {
				appLogger.Error("Authorization header invalid", zap.String("token", r.Header.Get("Authorization")))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			userDto, err := authService.ValidateToken(signedToken)
			if err != nil {
				appLogger.Error("Invalid token", zap.String("token", tokenString), zap.String("error", err.Error()))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx := authService.SetUserOnContext(r.Context(), userDto)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(header string) (string, error) {
	fields := strings.Fields(header)

	if len(fields) != 2 {
		return "", errors.New("invalid authorization header")
	}

	if !strings.EqualFold(fields[0], "Bearer") {
		return "", errors.New("authorization header must start with Bearer")
	}

	return fields[1], nil
}
