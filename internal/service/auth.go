package service

import (
	"context"
	"fmt"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

var (
	ErrInvalidCredentials = repository.ErrInvalidCredentials
	ErrUserBanned         = repository.ErrUserBanned
)

type AuthStore interface {
	Login(context.Context, string, string) (repository.User, error)
}

type AuthService struct {
	store  AuthStore
	config auth.Config
}

type LoginResult struct {
	User      repository.User
	Token     string
	ExpiresAt time.Time
}

func NewAuthService(store AuthStore, config auth.Config) *AuthService {
	return &AuthService{store: store, config: config}
}

func (s *AuthService) Login(ctx context.Context, email, password string, now time.Time) (LoginResult, error) {
	user, err := s.store.Login(ctx, email, password)
	if err != nil {
		return LoginResult{}, fmt.Errorf("auth service login: %w", err)
	}

	token, expiresAt, err := auth.GenerateToken(s.config, user, now.UTC())
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate token: %w", err)
	}

	return LoginResult{User: user, Token: token, ExpiresAt: expiresAt}, nil
}
