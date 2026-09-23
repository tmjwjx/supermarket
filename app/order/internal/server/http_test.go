package server

import (
	"net/http"
	"testing"

	"github.com/tmjwjx/supermarket/app/order/internal/conf"
	"github.com/tmjwjx/supermarket/pkg/httpauth/httpauthtest"
)

func TestHTTPRejectsForgedIdentity(t *testing.T) {
	srv := NewHTTPServer(&conf.Server{}, &conf.Auth{JWTSecret: "user-secret", AdminJWTSecret: "admin-secret"}, nil, nil)
	httpauthtest.RejectsForged(t, srv,
		httpauthtest.Route{Method: http.MethodPost, Path: "/v1/cart/items"},
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/cart/items"},
		httpauthtest.Route{Method: http.MethodPost, Path: "/v1/cart/items:batchDelete"},
		httpauthtest.Route{Method: http.MethodPost, Path: "/v1/orders"},
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/orders/o1"},
		httpauthtest.Route{Method: http.MethodGet, Path: "/v1/admin/orders"},
		httpauthtest.Route{Method: http.MethodPost, Path: "/v1/admin/orders/o1:ship"},
	)
}
