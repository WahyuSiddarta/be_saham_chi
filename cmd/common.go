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

// registerRoutes mounts a resource group and applies its permission prefix.
// Empty route rules rely on the supplied or inherited middleware.
func registerRoutes(router chi.Router, handlers handler.Handler, path, permissionPrefix string, routes []apiRoute, middlewares ...func(http.Handler) http.Handler) {
	router.Route(path, func(group chi.Router) {
		group.Use(middlewares...)
		for _, route := range routes {
			target := group
			if route.rule != "" {
				rule := route.rule
				if permissionPrefix != "" {
					rule = permissionPrefix + "." + rule
				}
				target = group.With(middleware.RequireRule(rule))
			}
			target.MethodFunc(route.method, route.path, handlers.Handle(route.handle))
		}
	})
}
