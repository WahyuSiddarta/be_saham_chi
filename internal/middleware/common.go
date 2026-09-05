package middleware

import (
	"context"
	"net/http"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type claimsContextKey struct{}

// RequestID attaches Chi's request ID to the request context.
func RequestID(next http.Handler) http.Handler {
	return chimiddleware.RequestID(next)
}

func ClaimsFromContext(ctx context.Context) (auth.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(auth.Claims)
	return claims, ok
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	return claims.UserID, ok && claims.UserID != ""
}
