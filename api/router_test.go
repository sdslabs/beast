package api

import (
	"testing"
)

func TestRouterRoutes(t *testing.T) {
	r := InitGinRouter()
	routes := r.Routes()
	if len(routes) != 99 {
		t.Fatalf("expected 99 routes, got %d", len(routes))
	}
}
