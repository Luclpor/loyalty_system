package auth

import (
	"errors"
	"time"

	"github.com/Luclpor/loyalty_system.git/internal/dto"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type JWTClaim struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	jwt.StandardClaims
}

func (ua *UserAuth) GenerateJWT(ID uuid.UUID, username string) (tokenString string, err error) {
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &JWTClaim{
		Username: username,
		ID:       ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString(ua.jwtKey)
	return
}

func (ua *UserAuth) ValidateToken(signedToken string) (*dto.UserDto, error) {
	token, err := jwt.ParseWithClaims(
		signedToken,
		&JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return ua.jwtKey, nil
		},
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaim)
	if !ok {
		return nil, errors.New("couldn't parse claims")
	}

	if claims.ExpiresAt < time.Now().Unix() {
		return nil, errors.New("token expired")
	}
	return &dto.UserDto{ID: claims.ID, Login: claims.Username}, nil
}
