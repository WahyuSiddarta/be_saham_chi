package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/WahyuSiddarta/be_saham_chi/internal/response"

	"github.com/go-chi/chi/v5"
)

type claimsContextKey struct{}

func Middleware(config Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenText, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || strings.TrimSpace(tokenText) == "" {
				writeUnauthorizedForRequest(w, r, "missing bearer token")
				return
			}

			claims, err := ParseToken(config, strings.TrimSpace(tokenText))
			if err != nil {
				writeUnauthorizedForRequest(w, r, "invalid bearer token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	_ = response.Fail(w, http.StatusUnauthorized, message)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(Claims)
	return claims, ok
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	return claims.UserID, ok && claims.UserID != ""
}

func RequireRule(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				_ = response.Error(w, http.StatusUnauthorized, "missing authentication claims")
				return
			}
			for _, rule := range claims.Rules {
				if rule == required {
					next.ServeHTTP(w, r)
					return
				}
			}
			_ = response.Error(w, http.StatusForbidden, "forbidden")
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

func writeUnauthorizedForRequest(w http.ResponseWriter, r *http.Request, message string) {
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		_ = response.Error(w, http.StatusUnauthorized, message)
		return
	}
	writeUnauthorized(w, message)
}
