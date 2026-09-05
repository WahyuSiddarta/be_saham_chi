package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"

	"github.com/go-chi/chi/v5"
)

// Authenticate verifies a bearer token and stores its claims in the request context.
func Authenticate(config auth.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenText, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || strings.TrimSpace(tokenText) == "" {
				_ = response.Fail(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			claims, err := auth.ParseToken(config, strings.TrimSpace(tokenText))
			if err != nil {
				_ = response.Fail(w, http.StatusUnauthorized, "invalid bearer token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRule(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				_ = response.Fail(w, http.StatusUnauthorized, "missing authentication claims")
				return
			}
			for _, rule := range claims.Rules {
				if rule == required {
					next.ServeHTTP(w, r)
					return
				}
			}
			_ = response.Fail(w, http.StatusForbidden, "forbidden")
		})
	}
}
func CommodityRule(next http.Handler) http.Handler {
	rules := map[string]string{"gold": "market.gold.read", "gc=f": "market.gold.read", "oil-wti": "market.wti.read", "cl=f": "market.wti.read", "oil-brent": "market.crude.read", "bz=f": "market.crude.read"}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rule, ok := rules[strings.ToLower(chi.URLParam(r, "commodity"))]
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		RequireRule(rule)(next).ServeHTTP(w, r)
	})
}
