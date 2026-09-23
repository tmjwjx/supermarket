package server

import (
	"net/http"
	"testing"

	"github.com/tmjwjx/supermarket/app/user/internal/conf"
	"github.com/tmjwjx/supermarket/pkg/httpauth/httpauthtest"
)

func TestHTTPRejectsForgedIdentity(t *testing.T) {
	srv := NewHTTPServer(&conf.Server{}, &conf.Auth{JWTSecret: "user-secret"}, nil, nil)
	httpauthtest.RejectsForged(t, srv,
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/users/me"},
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/users/u1"},
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/addresses"},
		httpauthtest.Route{Method: http.MethodPost, Path: "/v1/addresses"},
	)
}
