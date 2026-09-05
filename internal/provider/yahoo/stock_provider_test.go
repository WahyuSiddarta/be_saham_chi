package yahoo

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestStockProviderGetKlines(t *testing.T) {
	provider := NewStockProvider()
	provider.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v8/finance/chart/BBCA.JK" || r.URL.Query().Get("interval") != "1d" {
			t.Fatalf("unexpected Yahoo request: %s", r.URL.String())
		}
		return response(http.StatusOK, `{"chart":{"result":[{"timestamp":[1716163200,1716249600],"indicators":{"quote":[{"open":[9250,null],"high":[9350,null],"low":[9200,null],"close":[9300,null],"volume":[123456,null]}]}}]}}`), nil
	})}

	items, err := provider.GetKlines(context.Background(), "BBCA.JK", time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC), time.Date(2024, 5, 21, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetKlines returned error: %v", err)
	}
	if len(items) != 1 || items[0].Symbol != "BBCA.JK" || items[0].Interval != "1d" || items[0].Close != 9300 || items[0].Volume != 123456 {
		t.Fatalf("unexpected klines: %#v", items)
	}
}
