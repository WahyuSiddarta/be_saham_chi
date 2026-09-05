package middleware

import (
	"fmt"
	"net/http"

	"github.com/WahyuSiddarta/be_saham_chi/internal/response"

	"github.com/rs/zerolog"
)

// Recover returns the shared JSON error envelope when a request panics.
func Recover(log *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if failure := recover(); failure != nil {
					log.Error().Str("panic", fmt.Sprint(failure)).Msg("request panic")
					_ = response.Fail(w, http.StatusInternalServerError, "Internal Server Error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
