package main

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

var routes = []string{
	"/",
	"/login",
	"/logout",
	"/register",
	"/activate-account",
	"/members/plans",
	"/members/subscribe",
}

func Test_Routes_Exists(t *testing.T) {
	testRoutes := testApp.routes()

	chiRoutes := testRoutes.(chi.Router)

	for _, route := range routes {
		routeExist(t, chiRoutes, route)
	}
}

func routeExist(t *testing.T, router chi.Router, route string) {
	found := false

	_ = chi.Walk(router, func(method string, foundRoute string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if foundRoute == route {
			found = true
		}

		return nil
	})

	if !found {
		t.Errorf("did not find %s in registred root", route)
	}
}
