package server

import (
	adminv1 "github.com/tmjwjx/supermarket/api/admin/v1"
	"github.com/tmjwjx/supermarket/app/admin/internal/conf"
	adminsvc "github.com/tmjwjx/supermarket/app/admin/internal/service/adminuser"
	"github.com/tmjwjx/supermarket/pkg/httpauth"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/http"
)

// 签发和校验运营令牌共用 auth.jwt_secret
func NewHTTPServer(c *conf.Server, auth *conf.Auth, admin *adminsvc.AdminUserService) *http.Server {
	var secret string
	if auth != nil {
		secret = auth.JWTSecret
	}
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			metadata.Server(),
			httpauth.Server(httpauth.Options{
				AdminSecret: secret,
				Public:      []string{adminv1.OperationAdminUserServiceLogin},
			}),
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
	adminv1.RegisterAdminUserServiceHTTPServer(srv, admin)
	return srv
}
