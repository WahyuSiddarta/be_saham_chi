package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

type CashTransactionInput struct {
	AccountID       string
	AccountName     string
	TransactionType string
	Amount          float64
	CostAmount      float64
	TransactionDate time.Time
	Notes           string
}

type CashRepository interface {
	GetCash(context.Context, string, string) (repository.PortfolioCash, error)
	CreateCashTransaction(context.Context, string, string, repository.CashTransactionCommand, time.Time) (repository.PortfolioCashTransaction, error)
	ListCashTransactions(context.Context, string, string) ([]repository.PortfolioCashTransaction, error)
	GetCashTransaction(context.Context, string, string, string) (repository.PortfolioCashTransaction, error)
	UpdateCashTransaction(context.Context, string, string, string, repository.CashTransactionCommand, time.Time) (repository.PortfolioCashTransaction, error)
	DeleteCashTransaction(context.Context, string, string, string) error
	ListCashSnapshots(context.Context, string, string, time.Time, time.Time) ([]repository.PortfolioCashSnapshot, error)
}

type CashService struct {
	repository CashRepository
}

func NewCashService(repo CashRepository) *CashService {
	return &CashService{repository: repo}
}

func (s *CashService) GetCash(ctx context.Context, userID string, portfolioID string) (repository.PortfolioCash, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.PortfolioCash{}, ErrInvalidPortfolioID
	}

	cash, err := s.repository.GetCash(ctx, userID, portfolioID)
	return serviceResult(cash, err, "portfolioService.GetCash -> PortfolioRepository.GetCash")
}

func (s *CashService) CreateCashTransaction(ctx context.Context, userID string, portfolioID string, input CashTransactionInput) (repository.PortfolioCashTransaction, error) {
	repoInput, err := validateCashTransactionInput(portfolioID, input)
	if err != nil {
		return repository.PortfolioCashTransaction{}, err
	}

	cashTx, err := s.repository.CreateCashTransaction(ctx, userID, portfolioID, repoInput, time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrCashAccountNotFound) {
			return repository.PortfolioCashTransaction{}, fmt.Errorf("portfolioService.CreateCashTransaction -> PortfolioRepository.CreateCashTransaction: %w", ErrInvalidCashAccount)
		}
		return repository.PortfolioCashTransaction{}, wrapError("portfolioService.CreateCashTransaction -> PortfolioRepository.CreateCashTransaction", err)
	}

	return cashTx, nil
}

func (s *CashService) ListCashTransactions(ctx context.Context, userID string, portfolioID string) ([]repository.PortfolioCashTransaction, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return nil, ErrInvalidPortfolioID
	}

	transactions, err := s.repository.ListCashTransactions(ctx, userID, portfolioID)
	return serviceResult(transactions, err, "portfolioService.ListCashTransactions -> PortfolioRepository.ListCashTransactions")
}

func (s *CashService) GetCashTransaction(ctx context.Context, userID string, portfolioID string, transactionID string) (repository.PortfolioCashTransaction, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.PortfolioCashTransaction{}, ErrInvalidPortfolioID
	}
	if strings.TrimSpace(transactionID) == "" {
		return repository.PortfolioCashTransaction{}, ErrInvalidTransactionID
	}

	cashTx, err := s.repository.GetCashTransaction(ctx, userID, portfolioID, transactionID)
	return serviceResult(cashTx, err, "portfolioService.GetCashTransaction -> PortfolioRepository.GetCashTransaction")
}

func (s *CashService) UpdateCashTransaction(ctx context.Context, userID string, portfolioID string, transactionID string, input CashTransactionInput) (repository.PortfolioCashTransaction, error) {
	if strings.TrimSpace(transactionID) == "" {
		return repository.PortfolioCashTransaction{}, ErrInvalidTransactionID
	}
	repoInput, err := validateCashTransactionInput(portfolioID, input)
	if err != nil {
		return repository.PortfolioCashTransaction{}, err
	}

	cashTx, err := s.repository.UpdateCashTransaction(ctx, userID, portfolioID, transactionID, repoInput, time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrCashAccountNotFound) {
			return repository.PortfolioCashTransaction{}, fmt.Errorf("portfolioService.UpdateCashTransaction -> PortfolioRepository.UpdateCashTransaction: %w", ErrInvalidCashAccount)
		}
		return repository.PortfolioCashTransaction{}, wrapError("portfolioService.UpdateCashTransaction -> PortfolioRepository.UpdateCashTransaction", err)
	}
	return cashTx, nil
}

func (s *CashService) DeleteCashTransaction(ctx context.Context, userID string, portfolioID string, transactionID string) error {
	if strings.TrimSpace(portfolioID) == "" {
		return ErrInvalidPortfolioID
	}
	if strings.TrimSpace(transactionID) == "" {
		return ErrInvalidTransactionID
	}

	err := s.repository.DeleteCashTransaction(ctx, userID, portfolioID, transactionID)
	return wrapError("portfolioService.DeleteCashTransaction -> PortfolioRepository.DeleteCashTransaction", err)
}

func (s *CashService) ListCashSnapshots(ctx context.Context, userID string, portfolioID string, from time.Time, to time.Time) ([]repository.PortfolioCashSnapshot, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return nil, ErrInvalidPortfolioID
	}
	from, to = normalizeSnapshotRange(from, to, time.Now().UTC())
	if from.After(to) {
		return nil, ErrInvalidSnapshotRange
	}

	snapshots, err := s.repository.ListCashSnapshots(ctx, userID, portfolioID, from, to)
	return serviceResult(snapshots, err, "portfolioService.ListCashSnapshots -> PortfolioRepository.ListCashSnapshots")
}
func validateCashTransactionInput(portfolioID string, input CashTransactionInput) (repository.CashTransactionCommand, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.CashTransactionCommand{}, ErrInvalidPortfolioID
	}
	accountID := strings.TrimSpace(input.AccountID)
	accountName := strings.TrimSpace(input.AccountName)
	if accountID == "" && accountName == "" {
		return repository.CashTransactionCommand{}, ErrInvalidCashAccount
	}
	if input.Amount <= 0 {
		return repository.CashTransactionCommand{}, ErrInvalidCashAmount
	}
	if input.CostAmount < 0 {
		return repository.CashTransactionCommand{}, ErrInvalidCashAmount
	}
	transactionType := strings.ToLower(strings.TrimSpace(input.TransactionType))
	if transactionType == "" {
		transactionType = "deposit"
	}
	if !validCashTransactionType(transactionType) {
		return repository.CashTransactionCommand{}, ErrInvalidTransactionType
	}

	return repository.CashTransactionCommand{
		AccountID:       accountID,
		AccountName:     accountName,
		TransactionType: transactionType,
		Amount:          input.Amount,
		CostAmount:      normalizeCashCostAmount(transactionType, input.Amount, input.CostAmount),
		TransactionDate: input.TransactionDate,
		Notes:           strings.TrimSpace(input.Notes),
	}, nil
}
func validCashTransactionType(transactionType string) bool {
	return repository.IsCashTransactionType(transactionType)
}

func normalizeCashCostAmount(transactionType string, amount float64, costAmount float64) float64 {
	if costAmount > 0 {
		return costAmount
	}

	if repository.IsNoCostCashTransactionType(transactionType) {
		return 0
	}
	return amount
}

func normalizeSnapshotRange(from time.Time, to time.Time, now time.Time) (time.Time, time.Time) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if from.IsZero() {
		from = time.Date(today.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if to.IsZero() {
		to = today
	}
	return truncateDate(from), truncateDate(to)
}

func truncateDate(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
