package main

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

func TestStockPortfolioRoutesEnforceActionRules(t *testing.T) {
	app := testApplication()
	router := app.routes()
	for _, route := range []struct{ method, path, rule string }{
		{"GET", "/stocks", "read"},
		{"GET", "/stocks/transactions", "read"},
		{"POST", "/stocks/transactions", "create"},
		{"PUT", "/stocks/transactions/invalid-id", "update"},
		{"DELETE", "/stocks/transactions/invalid-id", "delete"},
	} {
		t.Run(route.method+route.path, func(t *testing.T) {
			for _, tc := range []struct {
				name   string
				rules  []string
				status int
			}{
				{"unauthenticated", nil, 401},
				{"wrong domain", []string{"portfolio.bond." + route.rule}, 403},
				{"correct action reaches validation", []string{"portfolio.stock." + route.rule}, 400},
			} {
				t.Run(tc.name, func(t *testing.T) {
					req := httptest.NewRequest(route.method, "/api/v1/private/portfolio/invalid-id"+route.path, strings.NewReader(`{}`))
					if tc.rules != nil {
						token, _, err := auth.GenerateToken(app.config.jwt, repository.User{ID: "user", Rules: tc.rules}, time.Now())
						if err != nil {
							t.Fatal(err)
						}
						req.Header.Set("Authorization", "Bearer "+token)
					}
					res := httptest.NewRecorder()
					router.ServeHTTP(res, req)
					if res.Code != tc.status {
						t.Fatalf("status=%d want=%d body=%s", res.Code, tc.status, res.Body.String())
					}
					if !strings.Contains(res.Body.String(), `"status":"nok"`) {
						t.Fatalf("missing error envelope: %s", res.Body.String())
					}
				})
			}
		})
	}
}
