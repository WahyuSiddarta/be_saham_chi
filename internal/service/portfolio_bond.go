package service

import (
	"context"
	"strings"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

type BondAssetInput struct {
	Symbol          string
	Name            string
	IssueDate       time.Time
	MaturityDate    time.Time
	AnnualRate      float64
	CouponFrequency string
	PrincipalValue  float64
}

type BondInput struct {
	BondAssetInput
	AccountID           string
	AccountName         string
	PrincipalAmount     float64
	CostAmount          float64
	AccruedCouponAmount float64
	FeeAmount           float64
	MarketValue         float64
	TransactionDate     time.Time
	Notes               string
}

type BondValuationInput struct {
	AccountID     string
	ValuationDate time.Time
	Price         float64
	MarketValue   float64
	Notes         string
}

type BondTransactionInput struct {
	AccountID           string
	AccountName         string
	AssetID             string
	TransactionType     string
	PrincipalAmount     float64
	Price               float64
	GrossAmount         float64
	CostAmount          float64
	AccruedCouponAmount float64
	FeeAmount           float64
	TaxAmount           float64
	NetAmount           float64
	TransactionDate     time.Time
	Notes               string
}

type BondRepository interface {
	CreateBond(context.Context, string, string, repository.BondCommand, time.Time) (repository.PortfolioBond, error)
	ListBonds(context.Context, string, string) ([]repository.PortfolioBond, error)
	GetBond(context.Context, string, string, string) (repository.PortfolioBond, error)
	UpdateBond(context.Context, string, string, string, repository.BondAssetCommand) (repository.PortfolioBond, error)
	AdjustBondValuation(context.Context, string, string, string, repository.BondValuationCommand, time.Time) (repository.PortfolioBond, error)
	CreateBondTransaction(context.Context, string, string, repository.BondTransactionCommand, time.Time) (repository.PortfolioBondTransaction, error)
	ListBondTransactions(context.Context, string, string) ([]repository.PortfolioBondTransaction, error)
	GetBondTransaction(context.Context, string, string, string) (repository.PortfolioBondTransaction, error)
	UpdateBondTransaction(context.Context, string, string, string, repository.BondTransactionCommand, time.Time) (repository.PortfolioBondTransaction, error)
	DeleteBondTransaction(context.Context, string, string, string) error
	ListBondSnapshots(context.Context, string, string, time.Time, time.Time) ([]repository.PortfolioBondSnapshot, error)
}

type BondService struct {
	repository BondRepository
}

func NewBondService(repo BondRepository) *BondService {
	return &BondService{repository: repo}
}

func (s *BondService) CreateBond(ctx context.Context, userID string, portfolioID string, input BondInput) (repository.PortfolioBond, error) {
	repoInput, err := validateBondInput(input)
	if err != nil {
		return repository.PortfolioBond{}, err
	}

	bond, err := s.repository.CreateBond(ctx, userID, portfolioID, repoInput, time.Now().UTC())
	if err != nil {
		return repository.PortfolioBond{}, bondRepositoryError(err, "portfolioService.CreateBond -> PortfolioRepository.CreateBond")
	}
	return bond, nil
}

func (s *BondService) ListBonds(ctx context.Context, userID string, portfolioID string) ([]repository.PortfolioBond, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return nil, ErrInvalidPortfolioID
	}

	bonds, err := s.repository.ListBonds(ctx, userID, portfolioID)
	if err != nil {
		return nil, bondRepositoryError(err, "portfolioService.ListBonds -> PortfolioRepository.ListBonds")
	}
	return bonds, nil
}

func (s *BondService) GetBond(ctx context.Context, userID string, portfolioID string, assetID string) (repository.PortfolioBond, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.PortfolioBond{}, ErrInvalidPortfolioID
	}
	if strings.TrimSpace(assetID) == "" {
		return repository.PortfolioBond{}, ErrInvalidBondAsset
	}

	bond, err := s.repository.GetBond(ctx, userID, portfolioID, assetID)
	if err != nil {
		return repository.PortfolioBond{}, bondRepositoryError(err, "portfolioService.GetBond -> PortfolioRepository.GetBond")
	}
	return bond, nil
}

func (s *BondService) UpdateBond(ctx context.Context, userID string, portfolioID string, assetID string, input BondAssetInput) (repository.PortfolioBond, error) {
	if strings.TrimSpace(assetID) == "" {
		return repository.PortfolioBond{}, ErrInvalidBondAsset
	}
	repoInput, err := validateBondAssetInput(input)
	if err != nil {
		return repository.PortfolioBond{}, err
	}

	bond, err := s.repository.UpdateBond(ctx, userID, portfolioID, assetID, repoInput)
	if err != nil {
		return repository.PortfolioBond{}, bondRepositoryError(err, "portfolioService.UpdateBond -> PortfolioRepository.UpdateBond")
	}
	return bond, nil
}

func (s *BondService) AdjustBondValuation(ctx context.Context, userID string, portfolioID string, assetID string, input BondValuationInput) (repository.PortfolioBond, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.PortfolioBond{}, ErrInvalidPortfolioID
	}
	if strings.TrimSpace(assetID) == "" {
		return repository.PortfolioBond{}, ErrInvalidBondAsset
	}
	repoInput, err := validateBondValuationInput(input)
	if err != nil {
		return repository.PortfolioBond{}, err
	}

	bond, err := s.repository.AdjustBondValuation(ctx, userID, portfolioID, assetID, repoInput, time.Now().UTC())
	if err != nil {
		return repository.PortfolioBond{}, bondRepositoryError(err, "portfolioService.AdjustBondValuation -> PortfolioRepository.AdjustBondValuation")
	}
	return bond, nil
}

func (s *BondService) CreateBondTransaction(ctx context.Context, userID string, portfolioID string, input BondTransactionInput) (repository.PortfolioBondTransaction, error) {
	repoInput, err := validateBondTransactionInput(portfolioID, input)
	if err != nil {
		return repository.PortfolioBondTransaction{}, err
	}

	bondTx, err := s.repository.CreateBondTransaction(ctx, userID, portfolioID, repoInput, time.Now().UTC())
	if err != nil {
		return repository.PortfolioBondTransaction{}, bondRepositoryErrorTx(err, "portfolioService.CreateBondTransaction -> PortfolioRepository.CreateBondTransaction")
	}
	return bondTx, nil
}

func (s *BondService) ListBondTransactions(ctx context.Context, userID string, portfolioID string) ([]repository.PortfolioBondTransaction, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return nil, ErrInvalidPortfolioID
	}

	transactions, err := s.repository.ListBondTransactions(ctx, userID, portfolioID)
	if err != nil {
		return nil, bondRepositoryErrorTx(err, "portfolioService.ListBondTransactions -> PortfolioRepository.ListBondTransactions")
	}
	return transactions, nil
}

func (s *BondService) GetBondTransaction(ctx context.Context, userID string, portfolioID string, transactionID string) (repository.PortfolioBondTransaction, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.PortfolioBondTransaction{}, ErrInvalidPortfolioID
	}
	if strings.TrimSpace(transactionID) == "" {
		return repository.PortfolioBondTransaction{}, ErrInvalidTransactionID
	}

	bondTx, err := s.repository.GetBondTransaction(ctx, userID, portfolioID, transactionID)
	if err != nil {
		return repository.PortfolioBondTransaction{}, bondRepositoryErrorTx(err, "portfolioService.GetBondTransaction -> PortfolioRepository.GetBondTransaction")
	}
	return bondTx, nil
}

func (s *BondService) UpdateBondTransaction(ctx context.Context, userID string, portfolioID string, transactionID string, input BondTransactionInput) (repository.PortfolioBondTransaction, error) {
	if strings.TrimSpace(transactionID) == "" {
		return repository.PortfolioBondTransaction{}, ErrInvalidTransactionID
	}
	repoInput, err := validateBondTransactionInput(portfolioID, input)
	if err != nil {
		return repository.PortfolioBondTransaction{}, err
	}

	bondTx, err := s.repository.UpdateBondTransaction(ctx, userID, portfolioID, transactionID, repoInput, time.Now().UTC())
	if err != nil {
		return repository.PortfolioBondTransaction{}, bondRepositoryErrorTx(err, "portfolioService.UpdateBondTransaction -> PortfolioRepository.UpdateBondTransaction")
	}
	return bondTx, nil
}

func (s *BondService) DeleteBondTransaction(ctx context.Context, userID string, portfolioID string, transactionID string) error {
	if strings.TrimSpace(portfolioID) == "" {
		return ErrInvalidPortfolioID
	}
	if strings.TrimSpace(transactionID) == "" {
		return ErrInvalidTransactionID
	}

	err := s.repository.DeleteBondTransaction(ctx, userID, portfolioID, transactionID)
	if err != nil {
		return bondRepositoryErrorTx(err, "portfolioService.DeleteBondTransaction -> PortfolioRepository.DeleteBondTransaction")
	}
	return nil
}

func (s *BondService) ListBondSnapshots(ctx context.Context, userID string, portfolioID string, from time.Time, to time.Time) ([]repository.PortfolioBondSnapshot, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return nil, ErrInvalidPortfolioID
	}
	from, to = normalizeSnapshotRange(from, to, time.Now().UTC())
	if from.After(to) {
		return nil, ErrInvalidSnapshotRange
	}

	snapshots, err := s.repository.ListBondSnapshots(ctx, userID, portfolioID, from, to)
	if err != nil {
		return nil, bondRepositoryError(err, "portfolioService.ListBondSnapshots -> PortfolioRepository.ListBondSnapshots")
	}
	return snapshots, nil
}
func validateBondInput(input BondInput) (repository.BondCommand, error) {
	assetInput, err := validateBondAssetInput(input.BondAssetInput)
	if err != nil {
		return repository.BondCommand{}, err
	}
	accountID := strings.TrimSpace(input.AccountID)
	accountName := strings.TrimSpace(input.AccountName)
	if accountID == "" && accountName == "" {
		accountName = "Bonds"
	}
	if input.PrincipalAmount <= 0 {
		return repository.BondCommand{}, ErrInvalidBondAmount
	}
	if input.CostAmount < 0 || input.AccruedCouponAmount < 0 || input.FeeAmount < 0 || input.MarketValue < 0 {
		return repository.BondCommand{}, ErrInvalidBondAmount
	}
	costAmount := input.CostAmount
	if costAmount == 0 {
		costAmount = input.PrincipalAmount
	}

	return repository.BondCommand{
		BondAssetCommand:    assetInput,
		AccountID:           accountID,
		AccountName:         accountName,
		PrincipalAmount:     input.PrincipalAmount,
		CostAmount:          costAmount,
		AccruedCouponAmount: input.AccruedCouponAmount,
		FeeAmount:           input.FeeAmount,
		MarketValue:         input.MarketValue,
		TransactionDate:     input.TransactionDate,
		Notes:               strings.TrimSpace(input.Notes),
	}, nil
}

func validateBondAssetInput(input BondAssetInput) (repository.BondAssetCommand, error) {
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	name := strings.TrimSpace(input.Name)
	if symbol == "" || name == "" {
		return repository.BondAssetCommand{}, ErrInvalidBondAsset
	}
	if input.AnnualRate < 0 || input.PrincipalValue < 0 {
		return repository.BondAssetCommand{}, ErrInvalidBondTerm
	}
	if !input.IssueDate.IsZero() && !input.MaturityDate.IsZero() && truncateDate(input.MaturityDate).Before(truncateDate(input.IssueDate)) {
		return repository.BondAssetCommand{}, ErrInvalidBondTerm
	}

	couponFrequency := strings.ToLower(strings.TrimSpace(input.CouponFrequency))
	if couponFrequency != "" && !repository.IsBondCouponFrequency(couponFrequency) {
		return repository.BondAssetCommand{}, ErrInvalidBondCouponFrequency
	}

	return repository.BondAssetCommand{
		Symbol:          symbol,
		Name:            name,
		IssueDate:       truncateDate(input.IssueDate),
		MaturityDate:    truncateDate(input.MaturityDate),
		AnnualRate:      input.AnnualRate,
		CouponFrequency: couponFrequency,
		PrincipalValue:  input.PrincipalValue,
	}, nil
}

func validateBondValuationInput(input BondValuationInput) (repository.BondValuationCommand, error) {
	accountID := strings.TrimSpace(input.AccountID)
	if accountID == "" {
		return repository.BondValuationCommand{}, ErrInvalidBondAccount
	}
	if input.Price < 0 || input.MarketValue < 0 {
		return repository.BondValuationCommand{}, ErrInvalidBondValuation
	}
	if input.Price == 0 && input.MarketValue == 0 {
		return repository.BondValuationCommand{}, ErrInvalidBondValuation
	}

	return repository.BondValuationCommand{
		AccountID:     accountID,
		ValuationDate: truncateDate(input.ValuationDate),
		Price:         input.Price,
		MarketValue:   input.MarketValue,
		Notes:         strings.TrimSpace(input.Notes),
	}, nil
}

func validateBondTransactionInput(portfolioID string, input BondTransactionInput) (repository.BondTransactionCommand, error) {
	if strings.TrimSpace(portfolioID) == "" {
		return repository.BondTransactionCommand{}, ErrInvalidPortfolioID
	}
	accountID := strings.TrimSpace(input.AccountID)
	accountName := strings.TrimSpace(input.AccountName)
	if accountID == "" && accountName == "" {
		return repository.BondTransactionCommand{}, ErrInvalidBondAccount
	}
	assetID := strings.TrimSpace(input.AssetID)
	if assetID == "" {
		return repository.BondTransactionCommand{}, ErrInvalidBondAsset
	}
	transactionType := strings.ToLower(strings.TrimSpace(input.TransactionType))
	if transactionType == "" {
		transactionType = repository.BondTransactionBuy
	}
	if !repository.IsBondTransactionType(transactionType) {
		return repository.BondTransactionCommand{}, ErrInvalidTransactionType
	}
	if bondTransactionNeedsPrincipal(transactionType) && input.PrincipalAmount <= 0 {
		return repository.BondTransactionCommand{}, ErrInvalidBondAmount
	}
	if input.PrincipalAmount < 0 ||
		input.Price < 0 ||
		input.GrossAmount < 0 ||
		input.CostAmount < 0 ||
		input.AccruedCouponAmount < 0 ||
		input.FeeAmount < 0 ||
		input.TaxAmount < 0 ||
		input.NetAmount < 0 {
		return repository.BondTransactionCommand{}, ErrInvalidBondAmount
	}

	principal := input.PrincipalAmount
	price := input.Price
	if price == 0 && principal > 0 {
		price = 1
	}
	grossAmount := input.GrossAmount
	if grossAmount == 0 && principal > 0 {
		grossAmount = principal * price
	}
	costAmount := input.CostAmount
	if costAmount == 0 && transactionType == repository.BondTransactionBuy {
		costAmount = grossAmount
	}
	accruedCouponAmount := input.AccruedCouponAmount
	if transactionType != repository.BondTransactionBuy && transactionType != repository.BondTransactionSell {
		accruedCouponAmount = 0
	}
	netAmount := input.NetAmount
	if netAmount == 0 {
		netAmount = grossAmount + accruedCouponAmount + input.FeeAmount + input.TaxAmount
	}

	return repository.BondTransactionCommand{
		AccountID:           accountID,
		AccountName:         accountName,
		AssetID:             assetID,
		TransactionType:     transactionType,
		PrincipalAmount:     principal,
		Price:               price,
		GrossAmount:         grossAmount,
		CostAmount:          costAmount,
		AccruedCouponAmount: accruedCouponAmount,
		FeeAmount:           input.FeeAmount,
		TaxAmount:           input.TaxAmount,
		NetAmount:           netAmount,
		TransactionDate:     input.TransactionDate,
		Notes:               strings.TrimSpace(input.Notes),
	}, nil
}

func bondTransactionNeedsPrincipal(transactionType string) bool {
	switch transactionType {
	case repository.BondTransactionBuy, repository.BondTransactionSell, repository.BondTransactionMaturity:
		return true
	default:
		return false
	}
}

func bondRepositoryError(err error, action string) error {
	return wrapPortfolioServiceError(action, err)
}

func bondRepositoryErrorTx(err error, action string) error {
	return wrapPortfolioServiceError(action, err)
}
