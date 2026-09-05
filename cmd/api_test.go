package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func testApplication() Application {
	log := zerolog.New(io.Discard)
	Log = &log
	return Application{config: Config{
		jwt:        auth.Config{Secret: "test-secret", Issuer: "test-issuer", TTL: time.Hour},
		goldSymbol: "GC=F", wtiSymbol: "CL=F", brentSymbol: "BZ=F",
		corsOrigins: []string{"https://frontend.test"},
	}}
}

var v2Routes = []struct{ method, path string }{
	{"POST", "/api/v1/public/auth/register"},
	{"POST", "/api/v1/public/auth/login"},
	{"GET", "/api/v1/private/admin/master-data"},
	{"PUT", "/api/v1/private/admin/master-data/{key}"},
	{"GET", "/api/v1/private/commodities/{commodity}/quote"},
	{"GET", "/api/v1/private/commodities/{commodity}/kline"},
	{"GET", "/api/v1/private/stocks/tickers"},
	{"GET", "/api/v1/private/stocks/{ticker}/quote"},
	{"GET", "/api/v1/private/stocks/{ticker}/kline"},
	{"GET", "/api/v1/private/stocks/{ticker}/fundamentals"},
	{"GET", "/api/v1/private/admin/stocks"},
	{"POST", "/api/v1/private/admin/stocks"},
	{"GET", "/api/v1/private/admin/stocks/{ticker}"},
	{"PUT", "/api/v1/private/admin/stocks/{ticker}"},
	{"PUT", "/api/v1/private/admin/stocks/{ticker}/status"},
	{"GET", "/api/v1/private/portfolios"},
	{"POST", "/api/v1/private/portfolios"},
	{"GET", "/api/v1/private/portfolios/{portfolio_id}"},
	{"PUT", "/api/v1/private/portfolios/{portfolio_id}"},
	{"DELETE", "/api/v1/private/portfolios/{portfolio_id}"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/cash"},
	{"POST", "/api/v1/private/portfolio/{portfolio_id}/cash"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/cash/snapshots"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/cash/transactions"},
	{"POST", "/api/v1/private/portfolio/{portfolio_id}/cash/transactions"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/cash/transactions/{transaction_id}"},
	{"PUT", "/api/v1/private/portfolio/{portfolio_id}/cash/transactions/{transaction_id}"},
	{"DELETE", "/api/v1/private/portfolio/{portfolio_id}/cash/transactions/{transaction_id}"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/bonds"},
	{"POST", "/api/v1/private/portfolio/{portfolio_id}/bonds"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/bonds/snapshots"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/bonds/transactions"},
	{"POST", "/api/v1/private/portfolio/{portfolio_id}/bonds/transactions"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/bonds/transactions/{transaction_id}"},
	{"PUT", "/api/v1/private/portfolio/{portfolio_id}/bonds/transactions/{transaction_id}"},
	{"DELETE", "/api/v1/private/portfolio/{portfolio_id}/bonds/transactions/{transaction_id}"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/bonds/{asset_id}"},
	{"PUT", "/api/v1/private/portfolio/{portfolio_id}/bonds/{asset_id}"},
	{"POST", "/api/v1/private/portfolio/{portfolio_id}/bonds/{asset_id}/valuation"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/commodities/gold"},
	{"POST", "/api/v1/private/portfolio/{portfolio_id}/commodities/gold"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/commodities/gold/transactions"},
	{"POST", "/api/v1/private/portfolio/{portfolio_id}/commodities/gold/transactions"},
	{"GET", "/api/v1/private/portfolio/{portfolio_id}/commodities/gold/transactions/{transaction_id}"},
	{"PUT", "/api/v1/private/portfolio/{portfolio_id}/commodities/gold/transactions/{transaction_id}"},
	{"DELETE", "/api/v1/private/portfolio/{portfolio_id}/commodities/gold/transactions/{transaction_id}"},
}

func TestNestedRoutesPreserveV2PathsAndAuthorization(t *testing.T) {
	app := testApplication()
	router := app.routes()
	paths := map[string]bool{}
	if err := chi.Walk(router.(chi.Routes), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		paths[method+" "+strings.TrimSuffix(route, "/")] = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	token, _, err := auth.GenerateToken(app.config.jwt, repository.User{ID: "user-1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range v2Routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			if !paths[route.method+" "+route.path] {
				t.Fatal("missing V2 route")
			}
			if !strings.Contains(route.path, "/private/") {
				return
			}
			path := strings.NewReplacer("{portfolio_id}", "p-1", "{transaction_id}", "tx-1", "{asset_id}", "a-1", "{ticker}", "BBCA", "{commodity}", "gold", "{key}", "bi_rate").Replace(route.path)
			for _, tc := range []struct {
				token  string
				status int
			}{{"", 401}, {token, 403}} {
				req := httptest.NewRequest(route.method, path, nil)
				if tc.token != "" {
					req.Header.Set("Authorization", "Bearer "+tc.token)
				}
				res := httptest.NewRecorder()
				router.ServeHTTP(res, req)
				if res.Code != tc.status {
					t.Fatalf("status=%d want=%d body=%s", res.Code, tc.status, res.Body.String())
				}
				var body map[string]any
				if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["status"] != "nok" || body["error"] == nil {
					t.Fatalf("wrong V2 error envelope: %v", body)
				}
			}
		})
	}
}
func TestNestedCommodityACLUsesCapturedParameter(t *testing.T) {
	app := testApplication()
	router := app.routes()
	token, _, err := auth.GenerateToken(app.config.jwt, repository.User{ID: "u-1", Rules: []string{"market.gold.read"}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/api/v1/private/commodities/gold/kline?range=invalid", 400},
		{"/api/v1/private/commodities/GC=F/kline?range=invalid", 400},
		{"/api/v1/private/commodities/oil-wti/kline?range=invalid", 403},
		{"/api/v1/private/commodities/oil-brent/kline?range=invalid", 403},
	} {
		req := httptest.NewRequest("GET", tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != tc.status {
			t.Fatalf("%s: status=%d body=%s", tc.path, res.Code, res.Body)
		}
	}
}
func TestCORSPreflightBypassesAuthentication(t *testing.T) {
	router := testApplication().routes()
	req := httptest.NewRequest("OPTIONS", "/api/v1/private/portfolios", nil)
	req.Header.Set("Origin", "https://frontend.test")
	req.Header.Set("Access-Control-Request-Method", "POST")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != 204 || res.Header().Get("Access-Control-Allow-Origin") != "https://frontend.test" {
		t.Fatalf("status=%d headers=%v", res.Code, res.Header())
	}
}
func TestPublicAuthRejectsInvalidJSONWithoutDatabase(t *testing.T) {
	router := testApplication().routes()
	for _, path := range []string{"/api/v1/public/auth/login", "/api/v1/public/auth/register", "/auth/login"} {
		req := httptest.NewRequest("POST", path, strings.NewReader(`{"email":"a@example.com","email":"b@example.com"}`))
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != 400 {
			t.Fatalf("%s: status=%d body=%s", path, res.Code, res.Body)
		}
	}
}
