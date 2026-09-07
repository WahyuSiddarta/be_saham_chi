package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

type StockYahooProvider interface {
	GetQuote(context.Context, string) (repository.MarketPrice, error)
	GetKlines(context.Context, string, time.Time, time.Time) ([]repository.StockKline, error)
}
type StockRepository interface {
	CreateStock(context.Context, repository.Stock) (repository.Stock, error)
	ListStocks(context.Context) ([]repository.Stock, error)
	SearchActiveStocks(context.Context, string, int) ([]repository.Stock, error)
	GetStock(context.Context, string) (repository.Stock, error)
	UpdateStockName(context.Context, string, string) (repository.Stock, error)
	UpdateStockStatus(context.Context, string, bool) (repository.Stock, error)
	ListKlines(context.Context, string, repository.Source, string, time.Time, time.Time) ([]repository.StockKline, error)
	UpsertKlines(context.Context, []repository.StockKline) error
	GetFundamentals(context.Context, string) (repository.StockFundamentals, error)
	GetFinancialStatementSummary(context.Context, string, int, repository.StockFinancialStatementFilter) (repository.StockFinancialStatementSummary, error)
	ListFinancialStatementValues(context.Context, string, int, repository.StockFinancialStatementFilter) ([]repository.StockFinancialStatementValue, error)
}
type StockService struct {
	yahooProvider StockYahooProvider
	repository    StockRepository
}

func NewStockService(yahooProvider StockYahooProvider, repo StockRepository) *StockService {
	return &StockService{yahooProvider: yahooProvider, repository: repo}
}

var (
	ErrStockNotFound             = errors.New("stock not found")
	ErrInvalidStock              = errors.New("ticker and name are required")
	ErrInvalidStockName          = errors.New("name is required")
	ErrInactiveStock             = errors.New("stock is inactive")
	ErrInvalidFinancialStatement = errors.New("invalid financial statement")
)

type FinancialStatementType string

const (
	FinancialStatementIncome   FinancialStatementType = "income_statement"
	FinancialStatementBalance  FinancialStatementType = "balance_sheet"
	FinancialStatementCashFlow FinancialStatementType = "cash_flow"
)

type StockFinancialStatement struct {
	Ticker        string
	StatementType FinancialStatementType
	ReportType    string
	ScrapedAt     time.Time
	Periods       []StockFinancialPeriod
	Rows          []StockFinancialRow
	Page          int
	PerPage       int
	TotalRows     int
	TotalPages    int
}

type StockFinancialPeriod struct {
	Key     string
	Label   string
	Year    int
	Quarter *int
	Basis   string
}

type StockFinancialRow struct {
	RowID      string
	RowRole    string
	MetricKey  string
	MetricName string
	Label      string
	LabelEN    *string
	LabelID    *string
	Values     []StockFinancialValue
}

type StockFinancialValue struct {
	PeriodKey    string
	Currency     *string
	Value        *string
	IsMissing    bool
	DisplayValue string
	Percentage   *string
}

func (s *StockService) CreateStock(ctx context.Context, ticker, name string) (repository.Stock, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	name = strings.TrimSpace(name)
	if ticker == "" || name == "" {
		return repository.Stock{}, ErrInvalidStock
	}
	stock, err := s.repository.CreateStock(ctx, repository.Stock{Ticker: ticker, Name: name})
	return serviceResult(stock, err, "stockService.CreateStock")
}
func (s *StockService) ListStocks(ctx context.Context) ([]repository.Stock, error) {
	stocks, err := s.repository.ListStocks(ctx)
	return serviceResult(stocks, err, "stockService.ListStocks")
}
func (s *StockService) SearchTickers(ctx context.Context, query string, limit int) ([]repository.Stock, error) {
	query = strings.ToUpper(strings.TrimSpace(query))
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	stocks, err := s.repository.SearchActiveStocks(ctx, query, limit)
	return serviceResult(stocks, err, "stockService.SearchTickers")
}
func (s *StockService) GetStock(ctx context.Context, ticker string) (repository.Stock, error) {
	stock, err := s.repository.GetStock(ctx, strings.ToUpper(strings.TrimSpace(ticker)))
	if errors.Is(err, repository.ErrStockNotFound) {
		return repository.Stock{}, ErrStockNotFound
	}
	return serviceResult(stock, err, "stockService.GetStock")
}
func (s *StockService) UpdateStockName(ctx context.Context, ticker, name string) (repository.Stock, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	name = strings.TrimSpace(name)
	if name == "" {
		return repository.Stock{}, ErrInvalidStockName
	}
	stock, err := s.repository.UpdateStockName(ctx, ticker, name)
	if errors.Is(err, repository.ErrStockNotFound) {
		return repository.Stock{}, ErrStockNotFound
	}
	return serviceResult(stock, err, "stockService.UpdateStockName")
}
func (s *StockService) UpdateStockStatus(ctx context.Context, ticker string, active bool) (repository.Stock, error) {
	stock, err := s.repository.UpdateStockStatus(ctx, strings.ToUpper(strings.TrimSpace(ticker)), active)
	if errors.Is(err, repository.ErrStockNotFound) {
		return repository.Stock{}, ErrStockNotFound
	}
	return serviceResult(stock, err, "stockService.UpdateStockStatus")
}
func (s *StockService) GetKlines(ctx context.Context, ticker string, from, to time.Time) ([]repository.StockKline, error) {
	stock, err := s.activeStock(ctx, ticker)
	if err != nil {
		return nil, err
	}
	symbol := yahooStockSymbol(stock.Ticker)
	items, err := s.repository.ListKlines(ctx, symbol, repository.SourceYahooFinance, "1d", from, to)
	if err != nil {
		return nil, fmt.Errorf("stockService.GetKlines -> StockRepository.ListKlines: %w", err)
	}
	if len(items) > 0 {
		return items, nil
	}

	fetched, err := s.yahooProvider.GetKlines(ctx, symbol, from, to)
	if err != nil {
		return nil, fmt.Errorf("stockService.GetKlines -> Yahoo.GetKlines: %w", err)
	}
	if err := s.repository.UpsertKlines(ctx, fetched); err != nil {
		return nil, fmt.Errorf("stockService.GetKlines -> StockRepository.UpsertKlines: %w", err)
	}
	items, err = s.repository.ListKlines(ctx, symbol, repository.SourceYahooFinance, "1d", from, to)
	return serviceResult(items, err, "stockService.GetKlines -> StockRepository.ListKlinesAfterFetch")
}

func (s *StockService) GetQuote(ctx context.Context, ticker string) (repository.MarketPrice, error) {
	stock, err := s.activeStock(ctx, ticker)
	if err != nil {
		return repository.MarketPrice{}, err
	}
	quote, err := s.yahooProvider.GetQuote(ctx, yahooStockSymbol(stock.Ticker))
	return serviceResult(quote, err, "stockService.GetQuote -> Yahoo.GetQuote")
}

func (s *StockService) GetFundamentals(ctx context.Context, ticker string) (repository.StockFundamentals, error) {
	stock, err := s.activeStock(ctx, ticker)
	if err != nil {
		return repository.StockFundamentals{}, err
	}
	fundamentals, err := s.repository.GetFundamentals(ctx, stock.Ticker)
	if errors.Is(err, repository.ErrStockNotFound) {
		return repository.StockFundamentals{}, ErrStockNotFound
	}
	return serviceResult(fundamentals, err, "stockService.GetFundamentals -> GetFundamentals")
}

func (s *StockService) GetFinancialStatement(ctx context.Context, ticker string, statementType FinancialStatementType, filter repository.StockFinancialStatementFilter) (StockFinancialStatement, error) {
	stock, err := s.activeStock(ctx, ticker)
	if err != nil {
		return StockFinancialStatement{}, err
	}
	report, ok := financialStatementReport(statementType)
	if !ok {
		return StockFinancialStatement{}, ErrInvalidFinancialStatement
	}

	summary, err := s.repository.GetFinancialStatementSummary(ctx, stock.Ticker, report, filter)
	if errors.Is(err, repository.ErrStockNotFound) {
		return StockFinancialStatement{}, ErrStockNotFound
	}
	if err != nil {
		return StockFinancialStatement{}, fmt.Errorf("stockService.GetFinancialStatement -> GetFinancialStatementSummary: %w", err)
	}
	values, err := s.repository.ListFinancialStatementValues(ctx, stock.Ticker, report, filter)
	if err != nil {
		return StockFinancialStatement{}, fmt.Errorf("stockService.GetFinancialStatement -> ListFinancialStatementValues: %w", err)
	}

	statement := StockFinancialStatement{
		Ticker: stock.Ticker, StatementType: statementType, ReportType: filter.ReportType,
		ScrapedAt: summary.ScrapedAt, Periods: make([]StockFinancialPeriod, 0), Rows: make([]StockFinancialRow, 0),
		Page: filter.Page, PerPage: filter.PerPage, TotalRows: summary.TotalRows,
	}
	if summary.TotalRows > 0 {
		statement.TotalPages = (summary.TotalRows + filter.PerPage - 1) / filter.PerPage
	}

	periodIndexes := make(map[string]struct{})
	rowIndexes := make(map[string]int)
	for _, item := range values {
		if _, exists := periodIndexes[item.PeriodKey]; !exists {
			statement.Periods = append(statement.Periods, StockFinancialPeriod{
				Key: item.PeriodKey, Label: item.PeriodLabel, Year: item.PeriodYear,
				Quarter: nullIntPointer(item.PeriodQuarter), Basis: item.PeriodBasis,
			})
			periodIndexes[item.PeriodKey] = struct{}{}
		}
		rowIndex, exists := rowIndexes[item.RowID]
		if !exists {
			statement.Rows = append(statement.Rows, StockFinancialRow{
				RowID: item.RowID, RowRole: item.RowRole, MetricKey: item.MetricKey,
				MetricName: item.MetricName, Label: item.Label, LabelEN: nullStringPointer(item.LabelEN),
				LabelID: nullStringPointer(item.LabelID), Values: make([]StockFinancialValue, 0),
			})
			rowIndex = len(statement.Rows) - 1
			rowIndexes[item.RowID] = rowIndex
		}
		statement.Rows[rowIndex].Values = append(statement.Rows[rowIndex].Values, StockFinancialValue{
			PeriodKey: item.PeriodKey, Currency: nullStringPointer(item.Currency), Value: nullStringPointer(item.Value),
			IsMissing: item.IsMissing, DisplayValue: item.DisplayValue, Percentage: nullStringPointer(item.Percentage),
		})
	}
	return statement, nil
}

func financialStatementReport(statementType FinancialStatementType) (int, bool) {
	switch statementType {
	case FinancialStatementIncome:
		return 1, true
	case FinancialStatementBalance:
		return 2, true
	case FinancialStatementCashFlow:
		return 3, true
	default:
		return 0, false
	}
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullIntPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}

func (s *StockService) activeStock(ctx context.Context, ticker string) (repository.Stock, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return repository.Stock{}, fmt.Errorf("ticker is required")
	}
	stock, err := s.repository.GetStock(ctx, ticker)
	if errors.Is(err, repository.ErrStockNotFound) {
		return repository.Stock{}, fmt.Errorf("%w: %s", ErrStockNotFound, ticker)
	}
	if err != nil {
		return repository.Stock{}, fmt.Errorf("stockService.activeStock -> GetStock: %w", err)
	}
	if !stock.Active {
		return repository.Stock{}, ErrInactiveStock
	}
	return stock, nil
}

func yahooStockSymbol(ticker string) string {
	return strings.TrimSuffix(strings.ToUpper(strings.TrimSpace(ticker)), ".JK") + ".JK"
}
