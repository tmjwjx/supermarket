package server

import (
	"net/http"
	"testing"

	"github.com/tmjwjx/supermarket/app/admin/internal/conf"
	"github.com/tmjwjx/supermarket/pkg/httpauth/httpauthtest"
)

func TestHTTPRejectsForgedIdentity(t *testing.T) {
	srv := NewHTTPServer(&conf.Server{}, &conf.Auth{JWTSecret: "admin-secret"}, nil)
	httpauthtest.RejectsForged(t, srv,
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/admin/me"},
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/admin/users"},
		httpauthtest.Route{Method: http.MethodPost, Path: "/v1/admin/users"},
		httpauthtest.Route{Method: http.MethodPatch, Path: "/v1/admin/users/a1"},
	)
}
