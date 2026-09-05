package main

import (
	"net/http"

	"github.com/WahyuSiddarta/be_saham_chi/internal/handler"
	"github.com/WahyuSiddarta/be_saham_chi/internal/middleware"

	"github.com/go-chi/chi/v5"
)

type apiRoute struct {
	method string
	path   string
	rule   string
	handle func(http.ResponseWriter, *http.Request) error
}

// registerRoutes applies each route's permission and the shared response adapter.
// An empty rule adds no permission check; parent middleware still applies.
func registerRoutes(router chi.Router, handlers handler.Handler, routes []apiRoute) {
	for _, route := range routes {
		target := router
		if route.rule != "" {
			target = router.With(middleware.RequireRule(route.rule))
		}
		target.MethodFunc(route.method, route.path, handlers.Handle(route.handle))
	}
}
