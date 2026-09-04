package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
)

type claimsContextKey struct{}

func Middleware(config Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenText, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || strings.TrimSpace(tokenText) == "" {
				writeUnauthorized(w, "missing bearer token")
				return
			}

			claims, err := ParseToken(config, strings.TrimSpace(tokenText))
			if err != nil {
				writeUnauthorized(w, "invalid bearer token")
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
