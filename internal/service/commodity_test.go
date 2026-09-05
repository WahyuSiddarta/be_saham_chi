package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

var errStubQuote = errors.New("stub quote failure")

func TestShouldRefreshDailyKlinesWhenCacheDoesNotCoverRequestedStartDate(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	startDate := time.Date(2025, 6, 17, 0, 0, 0, 0, time.UTC)
	klines := []repository.MarketKline{
		{OpenTime: time.Date(2026, 5, 17, 0, 0, 0, 0, time.UTC)},
		{OpenTime: time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)},
	}

	if !shouldRefreshDailyKlines(klines, startDate, now) {
		t.Fatal("expected refresh when cached klines are newer than the requested start date")
	}
}

func TestShouldRefreshDailyKlinesWhenCacheCoversRequestedDateAndIsCurrent(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	startDate := time.Date(2025, 6, 17, 0, 0, 0, 0, time.UTC)
	klines := []repository.MarketKline{
		{OpenTime: time.Date(2025, 6, 17, 0, 0, 0, 0, time.UTC)},
		{OpenTime: time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)},
	}

	if shouldRefreshDailyKlines(klines, startDate, now) {
		t.Fatal("expected no refresh when cached klines cover the requested range and are current")
	}
}

func TestGetQuoteConvertsGoldUSDPerOunceToIDRPerGram(t *testing.T) {
	provider := &stubCommodityProvider{
		quotes: map[string]repository.MarketPrice{
			"GC=F": {
				Symbol: "GC=F",
				Open:   3100,
				High:   3200,
				Low:    3000,
				Close:  3110.34768,
				Source: repository.SourceYahooFinance,
			},
			defaultUSDIDRSymbol: {
				Symbol: defaultUSDIDRSymbol,
				Close:  16000,
				Source: repository.SourceYahooFinance,
			},
		},
	}
	fxRepository := &stubFXRateRepository{item: repository.MasterData{Key: usdIDRMasterDataKey}}
	service := NewCommodityService(map[string]string{"gold": "GC=F"}, provider, &stubMarketKlineRepository{}, fxRepository)

	quote, err := service.GetQuote(context.Background(), "gold")
	if err != nil {
		t.Fatalf("GetQuote returned error: %v", err)
	}

	wantClose := 1600000.0
	if quote.Close != wantClose {
		t.Fatalf("expected close %v, got %v", wantClose, quote.Close)
	}
	if provider.quoteCalls[defaultUSDIDRSymbol] != 1 {
		t.Fatalf("expected one usd idr quote fetch, got %d", provider.quoteCalls[defaultUSDIDRSymbol])
	}
}

func TestGetQuoteUsesFreshDatabaseUSDIDRForGold(t *testing.T) {
	provider := &stubCommodityProvider{
		quotes: map[string]repository.MarketPrice{
			"GC=F": {
				Symbol: "GC=F",
				Close:  troyOunceGrams,
				Source: repository.SourceYahooFinance,
			},
			defaultUSDIDRSymbol: {
				Symbol: defaultUSDIDRSymbol,
				Close:  16000,
				Source: repository.SourceYahooFinance,
			},
		},
	}
	fxRepository := &stubFXRateRepository{item: repository.MasterData{
		Key:       usdIDRMasterDataKey,
		Value:     16000,
		UpdatedAt: time.Now().UTC().Add(-time.Hour),
	}}
	service := NewCommodityService(map[string]string{"gold": "GC=F"}, provider, &stubMarketKlineRepository{}, fxRepository)

	if _, err := service.GetQuote(context.Background(), "gold"); err != nil {
		t.Fatalf("first GetQuote returned error: %v", err)
	}
	if _, err := service.GetQuote(context.Background(), "gold"); err != nil {
		t.Fatalf("second GetQuote returned error: %v", err)
	}

	if provider.quoteCalls[defaultUSDIDRSymbol] != 0 {
		t.Fatalf("expected database usd idr rate, got %d provider fetches", provider.quoteCalls[defaultUSDIDRSymbol])
	}
}

func TestGetQuoteRefreshesStaleDatabaseUSDIDRForGold(t *testing.T) {
	provider := &stubCommodityProvider{quotes: map[string]repository.MarketPrice{
		"GC=F":              {Symbol: "GC=F", Close: troyOunceGrams},
		defaultUSDIDRSymbol: {Symbol: defaultUSDIDRSymbol, Close: 16500},
	}}
	fxRepository := &stubFXRateRepository{item: repository.MasterData{
		Key:       usdIDRMasterDataKey,
		Value:     16000,
		UpdatedAt: time.Now().UTC().Add(-usdIDRCacheTTL),
	}}
	service := NewCommodityService(map[string]string{"gold": "GC=F"}, provider, &stubMarketKlineRepository{}, fxRepository)

	quote, err := service.GetQuote(context.Background(), "gold")
	if err != nil {
		t.Fatalf("GetQuote returned error: %v", err)
	}
	if quote.Close != 16500 {
		t.Fatalf("close = %v, want 16500", quote.Close)
	}
	if provider.quoteCalls[defaultUSDIDRSymbol] != 1 || fxRepository.updateCalls != 1 {
		t.Fatalf("provider calls = %d, update calls = %d", provider.quoteCalls[defaultUSDIDRSymbol], fxRepository.updateCalls)
	}
}

func TestGetQuoteIdentifiesGoldYahooFailure(t *testing.T) {
	provider := &stubCommodityProvider{quoteErrors: map[string]error{"GC=F": errStubQuote}}
	service := NewCommodityService(map[string]string{"gold": "GC=F"}, provider, &stubMarketKlineRepository{}, &stubFXRateRepository{})

	_, err := service.GetQuote(context.Background(), "gold")
	if !errors.Is(err, errStubQuote) {
		t.Fatalf("expected wrapped provider error, got %v", err)
	}
	if !strings.Contains(err.Error(), "commodityService.getCachedQuote -> CommodityProvider.GetQuote symbol=GC=F") {
		t.Fatalf("expected gold symbol in error, got %v", err)
	}
}

func TestGetQuoteIdentifiesUSDIDRYahooFailure(t *testing.T) {
	provider := &stubCommodityProvider{
		quotes:      map[string]repository.MarketPrice{"GC=F": {Symbol: "GC=F", Close: 3000}},
		quoteErrors: map[string]error{defaultUSDIDRSymbol: errStubQuote},
	}
	service := NewCommodityService(map[string]string{"gold": "GC=F"}, provider, &stubMarketKlineRepository{}, &stubFXRateRepository{item: repository.MasterData{Key: usdIDRMasterDataKey}})

	_, err := service.GetQuote(context.Background(), "gold")
	if !errors.Is(err, errStubQuote) {
		t.Fatalf("expected wrapped provider error, got %v", err)
	}
	if !strings.Contains(err.Error(), "commodityService.getUSDIDRRate -> CommodityProvider.GetQuote symbol=IDR=X") {
		t.Fatalf("expected USD/IDR symbol in error, got %v", err)
	}
}

func TestGetQuoteIncludesInvalidFXValues(t *testing.T) {
	provider := &stubCommodityProvider{quotes: map[string]repository.MarketPrice{
		"GC=F":              {Symbol: "GC=F", Close: 3000},
		defaultUSDIDRSymbol: {Symbol: defaultUSDIDRSymbol, Open: 1, High: 2, Low: 0.5},
	}}
	service := NewCommodityService(map[string]string{"gold": "GC=F"}, provider, &stubMarketKlineRepository{}, &stubFXRateRepository{item: repository.MasterData{Key: usdIDRMasterDataKey}})

	_, err := service.GetQuote(context.Background(), "gold")
	if !errors.Is(err, ErrInvalidFXQuote) {
		t.Fatalf("expected ErrInvalidFXQuote, got %v", err)
	}
	want := "invalid fx quote: symbol=IDR=X open=1 high=2 low=0.5 close=0"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

func TestGetKlinesConvertsGoldUSDPerOunceToIDRPerGram(t *testing.T) {
	provider := &stubCommodityProvider{
		quotes: map[string]repository.MarketPrice{
			defaultUSDIDRSymbol: {
				Symbol: defaultUSDIDRSymbol,
				Close:  16000,
				Source: repository.SourceYahooFinance,
			},
		},
	}
	repoStub := &stubMarketKlineRepository{
		klines: []repository.MarketKline{
			{
				Symbol:   "GC=F",
				Interval: marketKlineInterval,
				OpenTime: time.Date(time.Now().UTC().Year(), 1, 1, 0, 0, 0, 0, time.UTC),
				Open:     troyOunceGrams,
				High:     troyOunceGrams * 2,
				Low:      troyOunceGrams / 2,
				Close:    troyOunceGrams * 3,
				Source:   repository.SourceYahooFinance,
			},
			{
				Symbol:   "GC=F",
				Interval: marketKlineInterval,
				OpenTime: truncateUTCDate(time.Now().UTC()),
				Open:     troyOunceGrams,
				High:     troyOunceGrams,
				Low:      troyOunceGrams,
				Close:    troyOunceGrams,
				Source:   repository.SourceYahooFinance,
			},
		},
	}
	service := NewCommodityService(map[string]string{"gold": "GC=F"}, provider, repoStub, &stubFXRateRepository{item: repository.MasterData{Key: usdIDRMasterDataKey}})

	klines, err := service.GetKlines(context.Background(), "gold", "ytd")
	if err != nil {
		t.Fatalf("GetKlines returned error: %v", err)
	}

	if len(klines) != 2 {
		t.Fatalf("expected 2 klines, got %d", len(klines))
	}
	if klines[0].Close != 48000 {
		t.Fatalf("expected converted close 48000, got %v", klines[0].Close)
	}
	if repoStub.klines[0].Close != troyOunceGrams*3 {
		t.Fatalf("expected stored kline to remain raw, got %v", repoStub.klines[0].Close)
	}
}

type stubCommodityProvider struct {
	quotes      map[string]repository.MarketPrice
	quoteErrors map[string]error
	klines      map[string][]repository.MarketKline
	quoteCalls  map[string]int
}

func (p *stubCommodityProvider) GetQuote(ctx context.Context, symbol string) (repository.MarketPrice, error) {
	if p.quoteCalls == nil {
		p.quoteCalls = make(map[string]int)
	}
	p.quoteCalls[symbol]++
	if err := p.quoteErrors[symbol]; err != nil {
		return repository.MarketPrice{}, err
	}
	return p.quotes[symbol], nil
}

func (p *stubCommodityProvider) GetKlines(ctx context.Context, symbol string, dataRange string) ([]repository.MarketKline, error) {
	return p.klines[symbol], nil
}

type stubMarketKlineRepository struct {
	klines []repository.MarketKline
}

type stubFXRateRepository struct {
	item        repository.MasterData
	getErr      error
	updateErr   error
	updateCalls int
}

func (r *stubFXRateRepository) GetMasterData(context.Context, string) (repository.MasterData, error) {
	return r.item, r.getErr
}

func (r *stubFXRateRepository) UpdateMasterData(_ context.Context, key string, value float64) (repository.MasterData, error) {
	r.updateCalls++
	if r.updateErr != nil {
		return repository.MasterData{}, r.updateErr
	}
	r.item = repository.MasterData{Key: key, Value: value, UpdatedAt: time.Now().UTC()}
	return r.item, nil
}

func (r *stubMarketKlineRepository) ListMarketKlines(ctx context.Context, symbol string, source repository.Source, interval string, startDate time.Time) ([]repository.MarketKline, error) {
	return r.klines, nil
}

func (r *stubMarketKlineRepository) UpsertMany(ctx context.Context, klines []repository.MarketKline) error {
	r.klines = klines
	return nil
}
