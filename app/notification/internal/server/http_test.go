package server

import (
	"net/http"
	"testing"

	"github.com/tmjwjx/supermarket/app/notification/internal/conf"
	"github.com/tmjwjx/supermarket/pkg/httpauth/httpauthtest"
)

func TestHTTPRejectsForgedIdentity(t *testing.T) {
	srv := NewHTTPServer(&conf.Server{}, &conf.Auth{JWTSecret: "user-secret", AdminJWTSecret: "admin-secret"}, nil)
	httpauthtest.RejectsForged(t, srv,
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/notifications"},
		httpauthtest.Route{Method: http.MethodPost, Path: "/v1/notifications:read"},
	)
}
