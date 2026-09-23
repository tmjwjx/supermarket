package server

import (
	orderv1 "github.com/tmjwjx/supermarket/api/order/v1"
	"github.com/tmjwjx/supermarket/app/order/internal/conf"
	cartsvc "github.com/tmjwjx/supermarket/app/order/internal/service/cart"
	ordersvc "github.com/tmjwjx/supermarket/app/order/internal/service/order"
	"github.com/tmjwjx/supermarket/pkg/httpauth"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/http"
)

func NewHTTPServer(c *conf.Server, auth *conf.Auth, cart *cartsvc.CartService, order *ordersvc.OrderService) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			metadata.Server(),
			httpauth.Server(httpauth.Options{UserSecret: auth.JWTSecret, AdminSecret: auth.AdminJWTSecret}),
		),
	}
	if c.HTTP.Network != "" {
		opts = append(opts, http.Network(c.HTTP.Network))
	}
	if c.HTTP.Addr != "" {
		opts = append(opts, http.Address(c.HTTP.Addr))
	}
	if d := c.HTTP.Timeout(); d != 0 {
		opts = append(opts, http.Timeout(d))
	}
	srv := http.NewServer(opts...)
	orderv1.RegisterCartServiceHTTPServer(srv, cart)
	orderv1.RegisterOrderServiceHTTPServer(srv, order)
	return srv
}
