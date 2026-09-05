package middleware

import (
	"bytes"
	"encoding/json"
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

func TestAuthenticationAndPermissions(t *testing.T) {
	config := auth.Config{Secret: "test-secret", Issuer: "test", TTL: time.Hour}
	token, _, err := auth.GenerateToken(config, repository.User{ID: "user-1", Rules: []string{"portfolio.read"}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, token, rule string
		status            int
		message           string
	}{
		{"missing token", "", "portfolio.read", 401, "missing bearer token"},
		{"invalid token", "invalid", "portfolio.read", 401, "invalid bearer token"},
		{"missing rule", token, "portfolio.delete", 403, "forbidden"},
		{"allowed", token, "portfolio.read", 204, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				claims, ok := ClaimsFromContext(r.Context())
				id, hasID := UserIDFromContext(r.Context())
				if !ok || !hasID || id != "user-1" || claims.UserID != id {
					t.Fatalf("verified claims missing: %+v", claims)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			res := httptest.NewRecorder()
			Authenticate(config)(RequireRule(tc.rule)(next)).ServeHTTP(res, req)
			if res.Code != tc.status || called != (tc.status == 204) {
				t.Fatalf("status=%d called=%t body=%s", res.Code, called, res.Body)
			}
			if tc.message != "" {
				var body map[string]any
				if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if len(body) != 2 || body["status"] != "nok" || body["data"] != tc.message {
					t.Fatalf("wrong failure envelope: %v", body)
				}
			}
		})
	}
}

func TestRecoveryAndRequestLogging(t *testing.T) {
	var logs bytes.Buffer
	log := zerolog.New(&logs)
	router := chi.NewRouter()
	router.Use(RequestID)
	router.Use(RequestLogger(&log))
	router.Use(Recover(&log))
	router.Get("/panic", func(http.ResponseWriter, *http.Request) { panic("internal diagnostic") })
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != 500 || res.Body.String() != `{"status":"nok","data":"Internal Server Error"}` {
		t.Fatalf("status=%d body=%s", res.Code, res.Body)
	}
	if !strings.Contains(logs.String(), "internal diagnostic") {
		t.Fatalf("missing panic diagnostic: %s", &logs)
	}
	foundRequest := false
	for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatal(err)
		}
		if entry["message"] != "HTTP request" {
			continue
		}
		foundRequest = true
		requestID, _ := entry["request_id"].(string)
		if entry["status"] != float64(500) || entry["level"] != "error" || entry["path"] != "/panic" || entry["remote_ip"] != req.RemoteAddr || requestID == "" {
			t.Fatalf("wrong request log: %v", entry)
		}
	}
	if !foundRequest {
		t.Fatal("missing HTTP request log")
	}
}
