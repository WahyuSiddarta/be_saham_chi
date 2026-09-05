package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

const (
	marketKlineInterval = "1d"
	tickerCacheTTL      = time.Minute
	usdIDRCacheTTL      = 6 * time.Hour
	usdIDRMasterDataKey = "usd_idr_rate"
	defaultUSDIDRSymbol = "IDR=X"
	troyOunceGrams      = 31.1034768
)

var (
	ErrInvalidCommodity = errors.New("invalid commodity")
	ErrInvalidRange     = errors.New("invalid range")
	ErrInvalidFXQuote   = errors.New("invalid fx quote")
)

type CommodityProvider interface {
	GetQuote(ctx context.Context, symbol string) (repository.MarketPrice, error)
	GetKlines(ctx context.Context, symbol string, dataRange string) ([]repository.MarketKline, error)
}

type MarketKlineRepository interface {
	ListMarketKlines(ctx context.Context, symbol string, source repository.Source, interval string, startDate time.Time) ([]repository.MarketKline, error)
	UpsertMany(ctx context.Context, klines []repository.MarketKline) error
}

type FXRateRepository interface {
	GetMasterData(ctx context.Context, key string) (repository.MasterData, error)
	UpdateMasterData(ctx context.Context, key string, value float64) (repository.MasterData, error)
}

type CommodityService struct {
	symbols          map[string]string
	provider         CommodityProvider
	klineRepository  MarketKlineRepository
	fxRateRepository FXRateRepository
	quoteCache       *quoteCache
}

func NewCommodityService(symbols map[string]string, provider CommodityProvider, klineRepository MarketKlineRepository, fxRateRepository FXRateRepository) *CommodityService {
	return &CommodityService{
		symbols:          symbols,
		provider:         provider,
		klineRepository:  klineRepository,
		fxRateRepository: fxRateRepository,
		quoteCache:       newQuoteCache(),
	}
}

func (s *CommodityService) GetQuote(ctx context.Context, commodity string) (repository.MarketPrice, error) {
	symbol, err := s.symbolFor(commodity)
	if err != nil {
		return repository.MarketPrice{}, err
	}

	now := time.Now().UTC()
	quote, err := s.getCachedQuote(ctx, symbol, now)
	if err != nil {
		return repository.MarketPrice{}, err
	}

	if s.isGoldSymbol(symbol) {
		return s.convertGoldQuoteToIDRGram(ctx, quote, now)
	}

	return quote, nil
}

func (s *CommodityService) GetKlines(ctx context.Context, commodity string, dataRange string) ([]repository.MarketKline, error) {
	symbol, err := s.symbolFor(commodity)
	if err != nil {
		return nil, err
	}

	startDate, err := rangeStartDate(dataRange, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	klines, err := s.klineRepository.ListMarketKlines(ctx, symbol, repository.SourceYahooFinance, marketKlineInterval, startDate)
	if err != nil {
		return nil, fmt.Errorf("commodityService.GetKlines -> MarketKlineRepository.List: %w", err)
	}

	if shouldRefreshDailyKlines(klines, startDate, time.Now().UTC()) {
		fetchedKlines, err := s.provider.GetKlines(ctx, symbol, dataRange)
		if err != nil {
			return nil, fmt.Errorf("commodityService.GetKlines -> CommodityProvider.GetKlines: %w", err)
		}
		if err := s.klineRepository.UpsertMany(ctx, fetchedKlines); err != nil {
			return nil, fmt.Errorf("commodityService.GetKlines -> MarketKlineRepository.UpsertMany: %w", err)
		}

		klines, err = s.klineRepository.ListMarketKlines(ctx, symbol, repository.SourceYahooFinance, marketKlineInterval, startDate)
		if err != nil {
			return nil, fmt.Errorf("commodityService.GetKlines -> MarketKlineRepository.List: %w", err)
		}
	}

	if s.isGoldSymbol(symbol) {
		return s.convertGoldKlinesToIDRGram(ctx, klines, time.Now().UTC())
	}

	return klines, nil
}

func (s *CommodityService) getCachedQuote(ctx context.Context, symbol string, now time.Time) (repository.MarketPrice, error) {
	if quote, ok := s.quoteCache.Get(symbol, now); ok {
		return quote, nil
	}

	quote, err := s.provider.GetQuote(ctx, symbol)
	if err != nil {
		return repository.MarketPrice{}, fmt.Errorf("commodityService.getCachedQuote -> CommodityProvider.GetQuote symbol=%s: %w", symbol, err)
	}
	s.quoteCache.Set(symbol, quote, now.Add(tickerCacheTTL))

	return quote, nil
}

func (s *CommodityService) convertGoldQuoteToIDRGram(ctx context.Context, quote repository.MarketPrice, now time.Time) (repository.MarketPrice, error) {
	usdIDR, err := s.getUSDIDRRate(ctx, now)
	if err != nil {
		return repository.MarketPrice{}, err
	}

	return convertMarketPriceToIDRGram(quote, usdIDR), nil
}

func (s *CommodityService) convertGoldKlinesToIDRGram(ctx context.Context, klines []repository.MarketKline, now time.Time) ([]repository.MarketKline, error) {
	usdIDR, err := s.getUSDIDRRate(ctx, now)
	if err != nil {
		return nil, err
	}

	converted := make([]repository.MarketKline, 0, len(klines))
	for _, kline := range klines {
		converted = append(converted, convertMarketKlineToIDRGram(kline, usdIDR))
	}

	return converted, nil
}

func (s *CommodityService) getUSDIDRRate(ctx context.Context, now time.Time) (float64, error) {
	cached, err := s.fxRateRepository.GetMasterData(ctx, usdIDRMasterDataKey)
	if err != nil {
		return 0, fmt.Errorf("commodityService.getUSDIDRRate -> FXRateRepository.Get: %w", err)
	}
	if cached.Value > 0 && cached.UpdatedAt.Add(usdIDRCacheTTL).After(now) {
		return cached.Value, nil
	}

	quote, err := s.provider.GetQuote(ctx, defaultUSDIDRSymbol)
	if err != nil {
		return 0, fmt.Errorf("commodityService.getUSDIDRRate -> CommodityProvider.GetQuote symbol=%s: %w", defaultUSDIDRSymbol, err)
	}
	if quote.Close <= 0 {
		return 0, invalidFXQuoteError(quote)
	}

	updated, err := s.fxRateRepository.UpdateMasterData(ctx, usdIDRMasterDataKey, quote.Close)
	if err != nil {
		return 0, fmt.Errorf("commodityService.getUSDIDRRate -> FXRateRepository.Update: %w", err)
	}
	return updated.Value, nil
}

func invalidFXQuoteError(quote repository.MarketPrice) error {
	return fmt.Errorf("%w: symbol=%s open=%g high=%g low=%g close=%g", ErrInvalidFXQuote, quote.Symbol, quote.Open, quote.High, quote.Low, quote.Close)
}

func (s *CommodityService) isGoldSymbol(symbol string) bool {
	goldSymbol, ok := s.symbols["gold"]
	return ok && strings.EqualFold(symbol, goldSymbol)
}

func convertMarketPriceToIDRGram(quote repository.MarketPrice, usdIDR float64) repository.MarketPrice {
	conversionRate := usdIDR / troyOunceGrams
	quote.Open *= conversionRate
	quote.High *= conversionRate
	quote.Low *= conversionRate
	quote.Close *= conversionRate
	return quote
}

func convertMarketKlineToIDRGram(kline repository.MarketKline, usdIDR float64) repository.MarketKline {
	conversionRate := usdIDR / troyOunceGrams
	kline.Open *= conversionRate
	kline.High *= conversionRate
	kline.Low *= conversionRate
	kline.Close *= conversionRate
	return kline
}

func (s *CommodityService) symbolFor(commodity string) (string, error) {
	normalizedCommodity := strings.ToLower(commodity)
	symbol, ok := s.symbols[normalizedCommodity]
	if !ok || symbol == "" {
		for _, configuredSymbol := range s.symbols {
			if strings.EqualFold(configuredSymbol, commodity) {
				return configuredSymbol, nil
			}
		}
		return "", fmt.Errorf("%w: %s", ErrInvalidCommodity, commodity)
	}
	return symbol, nil
}

func shouldRefreshDailyKlines(klines []repository.MarketKline, startDate time.Time, now time.Time) bool {
	if len(klines) == 0 {
		return true
	}

	oldestOpenTime := truncateUTCDate(klines[0].OpenTime)
	if oldestOpenTime.After(truncateUTCDate(startDate)) {
		return true
	}

	today := truncateUTCDate(now)
	latestOpenTime := truncateUTCDate(klines[len(klines)-1].OpenTime)
	return latestOpenTime.Before(today)
}

func rangeStartDate(dataRange string, now time.Time) (time.Time, error) {
	now = truncateUTCDate(now)
	if dataRange == "" {
		dataRange = "1mo"
	}
	if dataRange == "ytd" {
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC), nil
	}

	unit := ""
	valueText := ""
	for _, suffix := range []string{"mo", "wk", "d", "y"} {
		if strings.HasSuffix(dataRange, suffix) {
			unit = suffix
			valueText = strings.TrimSuffix(dataRange, suffix)
			break
		}
	}
	if unit == "" {
		return time.Time{}, fmt.Errorf("%w: %s", ErrInvalidRange, dataRange)
	}

	value, err := strconv.Atoi(valueText)
	if err != nil || value <= 0 {
		return time.Time{}, fmt.Errorf("%w: %s", ErrInvalidRange, dataRange)
	}

	switch unit {
	case "d":
		return now.AddDate(0, 0, -value), nil
	case "wk":
		return now.AddDate(0, 0, -value*7), nil
	case "mo":
		return now.AddDate(0, -value, 0), nil
	case "y":
		return now.AddDate(-value, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("%w: %s", ErrInvalidRange, dataRange)
	}
}

func truncateUTCDate(t time.Time) time.Time {
	year, month, day := t.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
