package handler

import "net/http"

func (h Handler) Docs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>CapitalSight API Docs</title><style>body{margin:0}</style></head><body><script id="api-reference" data-url="/api/v1/public/openapi.yaml" data-theme="default" data-layout="modern" src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script></body></html>`))
}
