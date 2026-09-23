package server

import (
	"net/http"
	"testing"

	"github.com/tmjwjx/supermarket/app/product/internal/conf"
	"github.com/tmjwjx/supermarket/pkg/httpauth/httpauthtest"
)

func TestHTTPRejectsForgedIdentity(t *testing.T) {
	srv := NewHTTPServer(&conf.Server{}, &conf.Auth{JWTSecret: "user-secret", AdminJWTSecret: "admin-secret"}, nil, nil, nil, nil, nil, nil, nil, nil)
	httpauthtest.RejectsForged(t, srv,
		httpauthtest.Route{Method: http.MethodPost, Path: "/v1/favorites"},
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/favorites"},
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/browse-histories"},
		httpauthtest.Route{Method: http.MethodPost, Path: "/v1/reviews"},
		httpauthtest.Route{Method: http.MethodPost, Path: "/v1/admin/products"},
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/admin/products/p1"},
		httpauthtest.Route{Method: http.MethodPatch, Path: "/v1/admin/categories/c1"},
		httpauthtest.Route{Method: http.MethodDelete, Path: "/v1/admin/recommendations/r1"},
	)
}
