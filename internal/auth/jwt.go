package auth

import (
	"errors"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Config struct {
	Secret string
	Issuer string
	TTL    time.Duration
}

type Claims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	RoleID int      `json:"role_id"`
	Rules  []string `json:"rules"`
	jwt.RegisteredClaims
}

func GenerateToken(config Config, user repository.User, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(config.TTL)
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		RoleID: user.RoleID,
		Rules:  user.Rules,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    config.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(config.Secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return signedToken, expiresAt, nil
}

func ParseToken(config Config, tokenText string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(
		tokenText,
		&claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return []byte(config.Secret), nil
		},
		jwt.WithIssuer(config.Issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || token == nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}
