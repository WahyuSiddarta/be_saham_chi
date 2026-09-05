package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

type GoldTransactionInput struct {
	AccountID, TransactionType, Notes          string
	QuantityGrams, Price, FeeAmount, TaxAmount float64
	TransactionDate                            time.Time
}

type GoldInput struct {
	AccountID, AccountName, Notes              string
	QuantityGrams, Price, FeeAmount, TaxAmount float64
	TransactionDate                            time.Time
}

type GoldRepository interface {
	CreateGold(context.Context, string, string, repository.GoldCommand, time.Time) (repository.PortfolioGold, error)
	GetGold(context.Context, string, string) (repository.PortfolioGold, error)
	CreateGoldTransaction(context.Context, string, string, repository.GoldTransactionCommand, time.Time) (repository.PortfolioGoldTransaction, error)
	ListGoldTransactions(context.Context, string, string) ([]repository.PortfolioGoldTransaction, error)
	GetGoldTransaction(context.Context, string, string, string) (repository.PortfolioGoldTransaction, error)
	UpdateGoldTransaction(context.Context, string, string, string, repository.GoldTransactionCommand, time.Time) (repository.PortfolioGoldTransaction, error)
	DeleteGoldTransaction(context.Context, string, string, string) error
}

type GoldService struct {
	repository GoldRepository
}

func NewGoldService(repo GoldRepository) *GoldService {
	return &GoldService{repository: repo}
}

func (s *GoldService) CreateGold(ctx context.Context, userID, portfolioID string, input GoldInput) (repository.PortfolioGold, error) {
	v, e := validateInitialGoldInput(portfolioID, input)
	if e != nil {
		return repository.PortfolioGold{}, e
	}
	out, e := s.repository.CreateGold(ctx, userID, portfolioID, v, time.Now().UTC())
	return out, mapGoldError(e, "portfolioService.CreateGold -> PortfolioRepository.CreateGold")
}

func (s *GoldService) GetGold(ctx context.Context, userID, portfolioID string) (repository.PortfolioGold, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.PortfolioGold{}, ErrInvalidPortfolioID
	}
	v, e := s.repository.GetGold(ctx, userID, portfolioID)
	if e != nil {
		return v, fmt.Errorf("portfolioService.GetGold -> PortfolioRepository.GetGold: %w", e)
	}
	return v, nil
}
func (s *GoldService) CreateGoldTransaction(ctx context.Context, userID, portfolioID string, input GoldTransactionInput) (repository.PortfolioGoldTransaction, error) {
	v, e := validateGoldInput(portfolioID, input)
	if e != nil {
		return repository.PortfolioGoldTransaction{}, e
	}
	out, e := s.repository.CreateGoldTransaction(ctx, userID, portfolioID, v, time.Now().UTC())
	return out, mapGoldError(e, "portfolioService.CreateGoldTransaction -> PortfolioRepository.CreateGoldTransaction")
}
func (s *GoldService) ListGoldTransactions(ctx context.Context, userID, portfolioID string) ([]repository.PortfolioGoldTransaction, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return nil, ErrInvalidPortfolioID
	}
	v, e := s.repository.ListGoldTransactions(ctx, userID, portfolioID)
	if e != nil {
		return nil, fmt.Errorf("portfolioService.ListGoldTransactions -> PortfolioRepository.ListGoldTransactions: %w", e)
	}
	return v, nil
}
func (s *GoldService) GetGoldTransaction(ctx context.Context, userID, portfolioID, id string) (repository.PortfolioGoldTransaction, error) {
	if strings.TrimSpace(id) == "" {
		return repository.PortfolioGoldTransaction{}, ErrInvalidTransactionID
	}
	v, e := s.repository.GetGoldTransaction(ctx, userID, portfolioID, id)
	return v, mapGoldError(e, "portfolioService.GetGoldTransaction -> PortfolioRepository.GetGoldTransaction")
}
func (s *GoldService) UpdateGoldTransaction(ctx context.Context, userID, portfolioID, id string, input GoldTransactionInput) (repository.PortfolioGoldTransaction, error) {
	if strings.TrimSpace(id) == "" {
		return repository.PortfolioGoldTransaction{}, ErrInvalidTransactionID
	}
	v, e := validateGoldInput(portfolioID, input)
	if e != nil {
		return repository.PortfolioGoldTransaction{}, e
	}
	out, e := s.repository.UpdateGoldTransaction(ctx, userID, portfolioID, id, v, time.Now().UTC())
	return out, mapGoldError(e, "portfolioService.UpdateGoldTransaction -> PortfolioRepository.UpdateGoldTransaction")
}
func (s *GoldService) DeleteGoldTransaction(ctx context.Context, userID, portfolioID, id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidTransactionID
	}
	return mapGoldError(s.repository.DeleteGoldTransaction(ctx, userID, portfolioID, id), "portfolioService.DeleteGoldTransaction -> PortfolioRepository.DeleteGoldTransaction")
}

func validateGoldInput(portfolioID string, input GoldTransactionInput) (repository.GoldTransactionCommand, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.GoldTransactionCommand{}, ErrInvalidPortfolioID
	}
	input.AccountID = strings.TrimSpace(input.AccountID)
	if input.AccountID == "" {
		return repository.GoldTransactionCommand{}, ErrInvalidGoldAccount
	}
	input.TransactionType = strings.ToLower(strings.TrimSpace(input.TransactionType))
	if input.TransactionType != repository.GoldTransactionBuy && input.TransactionType != repository.GoldTransactionSell {
		return repository.GoldTransactionCommand{}, ErrInvalidTransactionType
	}
	if input.QuantityGrams <= 0 || input.Price <= 0 || input.FeeAmount < 0 || input.TaxAmount < 0 || input.QuantityGrams*input.Price-input.FeeAmount-input.TaxAmount < 0 {
		return repository.GoldTransactionCommand{}, ErrInvalidGoldTransaction
	}
	return repository.GoldTransactionCommand{AccountID: input.AccountID, TransactionType: input.TransactionType, QuantityGrams: input.QuantityGrams, Price: input.Price, FeeAmount: input.FeeAmount, TaxAmount: input.TaxAmount, TransactionDate: input.TransactionDate, Notes: strings.TrimSpace(input.Notes)}, nil
}

func validateInitialGoldInput(portfolioID string, input GoldInput) (repository.GoldCommand, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.GoldCommand{}, ErrInvalidPortfolioID
	}
	input.AccountID = strings.TrimSpace(input.AccountID)
	input.AccountName = strings.TrimSpace(input.AccountName)
	if input.AccountID == "" && input.AccountName == "" {
		input.AccountName = "Gold"
	}
	if input.QuantityGrams <= 0 || input.Price <= 0 || input.FeeAmount < 0 || input.TaxAmount < 0 {
		return repository.GoldCommand{}, ErrInvalidGoldTransaction
	}
	return repository.GoldCommand{
		AccountID: input.AccountID, AccountName: input.AccountName, QuantityGrams: input.QuantityGrams,
		Price: input.Price, FeeAmount: input.FeeAmount, TaxAmount: input.TaxAmount,
		TransactionDate: input.TransactionDate, Notes: strings.TrimSpace(input.Notes),
	}, nil
}
func mapGoldError(err error, action string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrGoldAccountNotFound) {
		return fmt.Errorf("%s: %w", action, ErrInvalidGoldAccount)
	}
	if errors.Is(err, repository.ErrGoldHoldingQuantity) {
		return fmt.Errorf("%s: %w", action, ErrInsufficientGoldQuantity)
	}
	return wrapError(action, err)
}
