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
	Register(context.Context, string, string) (repository.User, repository.Portfolio, error)
	Login(context.Context, string, string) (repository.User, error)
}

type AuthService struct {
	store  AuthStore
	jwtCfg auth.Config
}

type LoginResult struct {
	User      repository.User
	Token     string
	ExpiresAt time.Time
}

func NewAuthService(store AuthStore, jwtCfg auth.Config) *AuthService {
	return &AuthService{store: store, jwtCfg: jwtCfg}
}

func (s *AuthService) Register(ctx context.Context, email string, password string) (repository.User, repository.Portfolio, error) {
	user, portfolio, err := s.store.Register(ctx, email, password)
	if err != nil {
		return repository.User{}, repository.Portfolio{}, fmt.Errorf("authService.Register -> AuthStore.Register: %w", err)
	}

	return user, portfolio, nil
}

func (s *AuthService) Login(ctx context.Context, email string, password string, now time.Time) (LoginResult, error) {
	user, err := s.store.Login(ctx, email, password)
	if err != nil {
		return LoginResult{}, fmt.Errorf("authService.Login -> AuthStore.Login: %w", err)
	}

	token, expiresAt, err := auth.GenerateToken(s.jwtCfg, user, now.UTC())
	if err != nil {
		return LoginResult{}, fmt.Errorf("authService.Login -> auth.GenerateToken: %w", err)
	}

	return LoginResult{User: user, Token: token, ExpiresAt: expiresAt}, nil
}
