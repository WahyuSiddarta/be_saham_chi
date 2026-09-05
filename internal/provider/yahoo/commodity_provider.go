package yahoo

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"

	yfa "github.com/oscarli916/yahoo-finance-api"
)

type CommodityProvider struct {
	defaultSymbol string
	client        *http.Client
	baseURL       string
}

const yahooChartBaseURL = "https://query2.finance.yahoo.com"

func NewCommodityProvider(defaultSymbol string) *CommodityProvider {
	return &CommodityProvider{
		defaultSymbol: defaultSymbol,
		client:        &http.Client{Timeout: 10 * time.Second},
		baseURL:       yahooChartBaseURL,
	}
}

func (p *CommodityProvider) GetQuote(ctx context.Context, symbol string) (repository.MarketPrice, error) {
	endpoint := fmt.Sprintf("%s/v8/finance/chart/%s?range=1d&interval=1m", p.baseURL, url.PathEscape(symbol))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return repository.MarketPrice{}, fmt.Errorf("create Yahoo chart request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return repository.MarketPrice{}, fmt.Errorf("request Yahoo chart metadata for %s: %w", symbol, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return repository.MarketPrice{}, fmt.Errorf("Yahoo chart metadata for %s returned HTTP %d", symbol, resp.StatusCode)
	}

	var payload yahooChartResponse
	if err := json.UnmarshalRead(resp.Body, &payload); err != nil {
		return repository.MarketPrice{}, fmt.Errorf("decode Yahoo chart metadata for %s: %w", symbol, err)
	}
	if len(payload.Chart.Result) == 0 {
		return repository.MarketPrice{}, fmt.Errorf("Yahoo chart metadata for %s returned no result", symbol)
	}
	meta := payload.Chart.Result[0].Meta
	if meta.RegularMarketPrice <= 0 {
		return repository.MarketPrice{}, fmt.Errorf("Yahoo chart metadata for %s returned invalid regularMarketPrice=%g", symbol, meta.RegularMarketPrice)
	}
	open, high, low := chartOHLC(payload.Chart.Result[0].Indicators.Quote)
	if meta.RegularMarketOpen > 0 {
		open = meta.RegularMarketOpen
	}
	if meta.RegularMarketDayHigh > 0 {
		high = meta.RegularMarketDayHigh
	}
	if meta.RegularMarketDayLow > 0 {
		low = meta.RegularMarketDayLow
	}

	return repository.MarketPrice{
		Symbol:    symbol,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     meta.RegularMarketPrice,
		Volume:    meta.RegularMarketVolume,
		Source:    repository.SourceYahooFinance,
		FetchedAt: time.Now().UTC(),
	}, nil
}

type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice   float64 `json:"regularMarketPrice"`
				RegularMarketOpen    float64 `json:"regularMarketOpen"`
				RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
				RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
				RegularMarketVolume  int64   `json:"regularMarketVolume"`
			} `json:"meta"`
			Indicators struct {
				Quote []yahooChartQuote `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

type yahooChartQuote struct {
	Open []*float64 `json:"open"`
	High []*float64 `json:"high"`
	Low  []*float64 `json:"low"`
}

func chartOHLC(quotes []yahooChartQuote) (open, high, low float64) {
	if len(quotes) == 0 {
		return 0, 0, 0
	}
	for _, value := range quotes[0].Open {
		if value != nil && *value > 0 {
			open = *value
			break
		}
	}
	for _, value := range quotes[0].High {
		if value != nil && *value > high {
			high = *value
		}
	}
	for _, value := range quotes[0].Low {
		if value != nil && *value > 0 && (low == 0 || *value < low) {
			low = *value
		}
	}
	return open, high, low
}

func (p *CommodityProvider) GetKlines(ctx context.Context, symbol string, dataRange string) ([]repository.MarketKline, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	ticker := yfa.NewTicker(symbol)
	history, err := ticker.History(yfa.HistoryQuery{
		Range:    dataRange,
		Interval: "1d",
	})
	if err != nil {
		return nil, err
	}

	dates := make([]string, 0, len(history))
	for date := range history {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	fetchedAt := time.Now().UTC()
	klines := make([]repository.MarketKline, 0, len(dates))
	for _, date := range dates {
		openTime, err := time.Parse("2006-01-02", date)
		if err != nil {
			return nil, err
		}

		price := history[date]
		klines = append(klines, repository.MarketKline{
			Symbol:    symbol,
			Interval:  "1d",
			OpenTime:  openTime,
			Open:      price.Open,
			High:      price.High,
			Low:       price.Low,
			Close:     price.Close,
			Volume:    price.Volume,
			Source:    repository.SourceYahooFinance,
			FetchedAt: fetchedAt,
		})
	}

	return klines, nil
}
