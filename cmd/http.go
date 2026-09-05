package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
)

func corsMiddleware(origins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				w.Header().Add("Vary", "Origin")
				for _, allowed := range origins {
					allowed = strings.TrimSpace(allowed)
					if allowed == "*" || allowed == origin {
						w.Header().Set("Access-Control-Allow-Origin", allowed)
						w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
						w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Origin, X-Requested-With")
						break
					}
				}
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
func recoverJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if failure := recover(); failure != nil {
				Log.Error().Str("panic", fmt.Sprint(failure)).Msg("request panic")
				_ = response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
func docsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>CapitalSight API Docs</title><style>body{margin:0}</style></head><body><script id="api-reference" data-url="/openapi.yaml" data-theme="default" data-layout="modern" src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script></body></html>`))
}
