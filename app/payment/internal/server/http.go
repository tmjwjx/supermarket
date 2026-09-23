package server

import (
	"github.com/tmjwjx/supermarket/app/payment/internal/conf"

	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/http"
)

func NewHTTPServer(c *conf.Server) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
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
	return http.NewServer(opts...)
}
