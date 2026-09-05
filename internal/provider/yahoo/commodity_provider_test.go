package yahoo

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetQuoteUsesChartRegularMarketMetadata(t *testing.T) {
	provider := NewCommodityProvider("GC=F")
	provider.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v8/finance/chart/IDR=X" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("range") != "1d" || r.URL.Query().Get("interval") != "1m" {
			t.Fatalf("unexpected query %s", r.URL.RawQuery)
		}
		return response(http.StatusOK, `{"chart":{"result":[{"meta":{"regularMarketPrice":16350,"regularMarketOpen":16300,"regularMarketDayHigh":16400,"regularMarketDayLow":16250,"regularMarketVolume":7}}]}}`), nil
	})}

	quote, err := provider.GetQuote(context.Background(), "IDR=X")
	if err != nil {
		t.Fatalf("GetQuote returned error: %v", err)
	}
	if quote.Close != 16350 || quote.Open != 16300 || quote.High != 16400 || quote.Low != 16250 || quote.Volume != 7 {
		t.Fatalf("unexpected quote: %+v", quote)
	}
}

func TestGetQuoteReportsYahooHTTPStatus(t *testing.T) {
	provider := NewCommodityProvider("GC=F")
	provider.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusTooManyRequests, "rate limited"), nil
	})}
	_, err := provider.GetQuote(context.Background(), "IDR=X")
	if err == nil || !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("expected HTTP 429 error, got %v", err)
	}
}

func TestGetQuoteFallsBackToIntradayCandlesForMissingMetadataOHLC(t *testing.T) {
	provider := NewCommodityProvider("GC=F")
	provider.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK, `{"chart":{"result":[{"meta":{"regularMarketPrice":16350},"indicators":{"quote":[{"open":[null,16300,16320],"high":[null,16340,16400],"low":[null,16290,16250]}]}}]}}`), nil
	})}

	quote, err := provider.GetQuote(context.Background(), "IDR=X")
	if err != nil {
		t.Fatalf("GetQuote returned error: %v", err)
	}
	if quote.Open != 16300 || quote.High != 16400 || quote.Low != 16250 || quote.Close != 16350 {
		t.Fatalf("unexpected quote: %+v", quote)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
