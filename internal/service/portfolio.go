package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidPortfolioID         = errors.New("invalid portfolio id")
	ErrInvalidPortfolioName       = errors.New("invalid portfolio name")
	ErrInvalidPortfolioMove       = errors.New("invalid portfolio move")
	ErrDuplicatePortfolioName     = errors.New("duplicate portfolio name")
	ErrInvalidCashAmount          = errors.New("invalid cash amount")
	ErrInvalidCashAccount         = errors.New("invalid cash account")
	ErrInvalidCashTransaction     = errors.New("invalid cash transaction")
	ErrInvalidBondAsset           = errors.New("invalid bond asset")
	ErrInvalidBondAccount         = errors.New("invalid bond account")
	ErrInvalidBondAmount          = errors.New("invalid bond amount")
	ErrInvalidBondTerm            = errors.New("invalid bond term")
	ErrInvalidBondCouponFrequency = errors.New("invalid bond coupon frequency")
	ErrInvalidBondValuation       = errors.New("invalid bond valuation")
	ErrInvalidSnapshotRange       = errors.New("invalid snapshot range")
	ErrInvalidTransactionType     = errors.New("invalid transaction type")
	ErrInvalidTransactionID       = errors.New("invalid transaction id")
	ErrInvalidGoldAccount         = errors.New("invalid gold account")
	ErrInvalidGoldTransaction     = errors.New("invalid gold transaction")
	ErrInsufficientGoldQuantity   = errors.New("insufficient gold quantity")
)

type PortfolioInput struct {
	Name string
}

type PortfolioRepository interface {
	ListPortfolio(context.Context, string) ([]repository.Portfolio, error)
	GetByID(context.Context, string, string) (repository.Portfolio, error)
	CreatePortfolio(context.Context, string, repository.PortfolioCommand) (repository.Portfolio, error)
	UpdatePortfolio(context.Context, string, string, repository.PortfolioCommand) (repository.Portfolio, error)
	DeleteAndMove(context.Context, string, string, string) error
}

type PortfolioService struct {
	repository PortfolioRepository
}

func NewPortfolioService(repo PortfolioRepository) *PortfolioService {
	return &PortfolioService{
		repository: repo,
	}
}

func (s *PortfolioService) ListPortfolio(ctx context.Context, userID string) ([]repository.Portfolio, error) {
	portfolios, err := s.repository.ListPortfolio(ctx, userID)
	return serviceResult(portfolios, err, "portfolioService.List -> PortfolioRepository.List")
}

func (s *PortfolioService) Get(ctx context.Context, userID string, portfolioID string) (repository.Portfolio, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.Portfolio{}, ErrInvalidPortfolioID
	}

	portfolio, err := s.repository.GetByID(ctx, userID, portfolioID)
	return serviceResult(portfolio, err, "portfolioService.Get -> PortfolioRepository.GetByID")
}

func (s *PortfolioService) CreatePortfolio(ctx context.Context, userID string, input PortfolioInput) (repository.Portfolio, error) {
	repoInput, err := validatePortfolioInput(input)
	if err != nil {
		return repository.Portfolio{}, err
	}

	portfolio, err := s.repository.CreatePortfolio(ctx, userID, repoInput)
	if err != nil {
		if isUniqueViolation(err) {
			return repository.Portfolio{}, fmt.Errorf("portfolioService.Create -> PortfolioRepository.Create: %w", ErrDuplicatePortfolioName)
		}
		return repository.Portfolio{}, fmt.Errorf("portfolioService.Create -> PortfolioRepository.Create: %w", err)
	}
	return portfolio, nil
}

func (s *PortfolioService) UpdatePortfolio(ctx context.Context, userID string, portfolioID string, input PortfolioInput) (repository.Portfolio, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.Portfolio{}, ErrInvalidPortfolioID
	}
	repoInput, err := validatePortfolioInput(input)
	if err != nil {
		return repository.Portfolio{}, err
	}

	portfolio, err := s.repository.UpdatePortfolio(ctx, userID, portfolioID, repoInput)
	if err != nil {
		if isUniqueViolation(err) {
			return repository.Portfolio{}, fmt.Errorf("portfolioService.Update -> PortfolioRepository.Update: %w", ErrDuplicatePortfolioName)
		}
		return repository.Portfolio{}, wrapError("portfolioService.Update -> PortfolioRepository.Update", err)
	}
	return portfolio, nil
}

func (s *PortfolioService) Delete(ctx context.Context, userID string, portfolioID string, targetPortfolioID string) error {
	portfolioID = strings.TrimSpace(portfolioID)
	targetPortfolioID = strings.TrimSpace(targetPortfolioID)
	if portfolioID == "" || targetPortfolioID == "" {
		return ErrInvalidPortfolioID
	}
	if portfolioID == targetPortfolioID {
		return ErrInvalidPortfolioMove
	}

	err := s.repository.DeleteAndMove(ctx, userID, portfolioID, targetPortfolioID)
	return wrapError("portfolioService.Delete -> PortfolioRepository.DeleteAndMove", err)
}
func validatePortfolioInput(input PortfolioInput) (repository.PortfolioCommand, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return repository.PortfolioCommand{}, ErrInvalidPortfolioName
	}
	return repository.PortfolioCommand{Name: name}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
