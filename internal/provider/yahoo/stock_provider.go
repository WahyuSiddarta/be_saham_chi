package yahoo

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

type StockProvider struct {
	client  *http.Client
	baseURL string
}

func NewStockProvider() *StockProvider {
	return &StockProvider{client: &http.Client{Timeout: 10 * time.Second}, baseURL: yahooChartBaseURL}
}

func (p *StockProvider) GetQuote(ctx context.Context, symbol string) (repository.MarketPrice, error) {
	provider := &CommodityProvider{client: p.client, baseURL: p.baseURL}
	return provider.GetQuote(ctx, symbol)
}

func (p *StockProvider) GetKlines(ctx context.Context, symbol string, from, to time.Time) ([]repository.StockKline, error) {
	query := url.Values{}
	query.Set("period1", fmt.Sprintf("%d", from.UTC().Unix()))
	query.Set("period2", fmt.Sprintf("%d", to.UTC().Add(24*time.Hour).Unix()))
	query.Set("interval", "1d")
	query.Set("events", "history")
	endpoint := fmt.Sprintf("%s/v8/finance/chart/%s?%s", p.baseURL, url.PathEscape(symbol), query.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("yahoo.StockProvider.GetKlines -> create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo.StockProvider.GetKlines -> request %s: %w", symbol, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("yahoo.StockProvider.GetKlines -> %s returned HTTP %d", symbol, resp.StatusCode)
	}

	var payload stockChartResponse
	if err := json.UnmarshalRead(resp.Body, &payload); err != nil {
		return nil, fmt.Errorf("yahoo.StockProvider.GetKlines -> decode %s: %w", symbol, err)
	}
	if len(payload.Chart.Result) == 0 {
		return nil, fmt.Errorf("yahoo.StockProvider.GetKlines -> %s returned no result", symbol)
	}
	result := payload.Chart.Result[0]
	if len(result.Indicators.Quote) == 0 {
		return []repository.StockKline{}, nil
	}
	quote := result.Indicators.Quote[0]
	fetchedAt := time.Now().UTC()
	items := make([]repository.StockKline, 0, len(result.Timestamp))
	for i, timestamp := range result.Timestamp {
		if i >= len(quote.Open) || i >= len(quote.High) || i >= len(quote.Low) || i >= len(quote.Close) || i >= len(quote.Volume) || quote.Open[i] == nil || quote.High[i] == nil || quote.Low[i] == nil || quote.Close[i] == nil || quote.Volume[i] == nil {
			continue
		}
		items = append(items, repository.StockKline{Symbol: symbol, Interval: "1d", OpenTime: time.Unix(timestamp, 0).UTC(), Open: *quote.Open[i], High: *quote.High[i], Low: *quote.Low[i], Close: *quote.Close[i], Volume: *quote.Volume[i], Source: repository.SourceYahooFinance, FetchedAt: fetchedAt})
	}
	return items, nil
}

type stockChartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []*float64 `json:"open"`
					High   []*float64 `json:"high"`
					Low    []*float64 `json:"low"`
					Close  []*float64 `json:"close"`
					Volume []*int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}
