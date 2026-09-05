package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"
	"github.com/WahyuSiddarta/be_saham_chi/internal/middleware"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func assertFailure(t *testing.T, res *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON response: %s: %v", res.Body, err)
	}
	if res.Code != status || len(body) != 2 || body["status"] != "nok" || body["data"] != message {
		t.Fatalf("status=%d body=%s; want status=%d message=%q", res.Code, res.Body, status, message)
	}
}

func TestHandlerValidationWritesFailure(t *testing.T) {
	log := zerolog.New(io.Discard)
	h := New("test", &log, nil, Domains{})
	for _, tc := range []struct {
		name      string
		handler   func(http.ResponseWriter, *http.Request) error
		url, body string
		status    int
		message   string
	}{
		{"register body", h.Register, "/", "{", 400, "invalid request body"},
		{"login email", h.Login, "/", `{}`, 400, "email is required"},
		{"stock body", h.CreateStock, "/", "{", 400, "invalid request body"},
		{"master data body", h.UpdateMasterData, "/", "{", 400, "invalid request body"},
		{"stock from", h.GetStockKlines, "/?from=bad", "", 400, "invalid from date"},
		{"stock to", h.GetStockKlines, "/?to=bad", "", 400, "invalid to date"},
		{"stock range", h.GetStockKlines, "/?from=2026-02-02&to=2026-01-01", "", 400, "from must not be after to"},
		{"portfolio user", h.ListPortfolio, "/", "", 401, "missing user_id"},
		{"cash user", h.GetCash, "/", "", 401, "missing user_id"},
		{"bond user", h.ListBonds, "/", "", 401, "missing user_id"},
		{"gold user", h.GetGold, "/", "", 401, "missing user_id"},
		{"commodity user", h.GetCommodityQuote, "/", "", 401, "missing user_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			// Call directly to ensure errors are written without relying on Handle.
			if err := tc.handler(res, httptest.NewRequest(http.MethodPost, tc.url, strings.NewReader(tc.body))); err != nil {
				t.Fatal(err)
			}
			assertFailure(t, res, tc.status, tc.message)
		})
	}
}

func TestAuthenticatedHandlerErrorsPreserveMessages(t *testing.T) {
	config := auth.Config{Secret: "test-secret", Issuer: "test", TTL: time.Hour}
	token, _, err := auth.GenerateToken(config, repository.User{ID: "u-1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, body string
		storeError error
		gold       bool
		status     int
		message    string
	}{
		{"missing target", `{}`, nil, false, 400, "portfolio_id and target_portfolio_id are required"},
		{"same target", `{"target_portfolio_id":"p-1"}`, nil, false, 400, "target_portfolio_id must be different from portfolio_id"},
		{"not found", `{"target_portfolio_id":"p-2"}`, repository.ErrPortfolioNotFound, false, 404, "portfolio not found"},
		{"main protected", `{"target_portfolio_id":"p-2"}`, repository.ErrMainPortfolioProtected, false, 409, "main portfolio cannot be deleted"},
		{"internal cause", `{"target_portfolio_id":"p-2"}`, errors.New("private database diagnostic"), false, 500, "failed to delete portfolio"},
		{"gold body", "{", nil, true, 400, "invalid request body"},
		{"gold date", `{"transaction_date":"bad"}`, nil, true, 400, "invalid transaction_date"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			log := zerolog.New(&logs)
			store := &deletePortfolioStore{err: tc.storeError}
			h := New("test", &log, nil, Domains{Portfolio: service.NewPortfolioService(store)})
			next := h.DeletePortfolio
			if tc.gold {
				next = h.CreateGold
			}
			router := chi.NewRouter()
			router.Use(middleware.Authenticate(config))
			router.Post("/portfolios/{portfolio_id}", h.Handle(next))
			req := httptest.NewRequest(http.MethodPost, "/portfolios/p-1", strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			assertFailure(t, res, tc.status, tc.message)
			if tc.storeError != nil && !strings.Contains(logs.String(), tc.storeError.Error()) {
				t.Fatalf("internal cause missing from logs: %s", &logs)
			}
		})
	}
}

type failedResponseWriter struct {
	*httptest.ResponseRecorder
	writes int
}

func (w *failedResponseWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, errors.New("connection closed")
}

func TestHandleDoesNotRetryStartedResponse(t *testing.T) {
	log := zerolog.New(io.Discard)
	h := New("test", &log, nil, Domains{})
	writer := &failedResponseWriter{ResponseRecorder: httptest.NewRecorder()}
	h.Handle(func(w http.ResponseWriter, r *http.Request) error {
		return response.Fail(w, http.StatusBadRequest, "invalid request body")
	})(writer, httptest.NewRequest(http.MethodPost, "/", nil))
	if writer.Code != http.StatusBadRequest || writer.writes != 1 {
		t.Fatalf("status=%d writes=%d; want 400 and one write", writer.Code, writer.writes)
	}
}

func TestHandleUnexpectedErrorUsesGenericMessage(t *testing.T) {
	log := zerolog.New(io.Discard)
	h := New("test", &log, nil, Domains{})
	res := httptest.NewRecorder()
	h.Handle(func(http.ResponseWriter, *http.Request) error {
		return errors.New("private internal details")
	})(res, httptest.NewRequest(http.MethodGet, "/", nil))
	assertFailure(t, res, 500, "Internal Server Error")
}
