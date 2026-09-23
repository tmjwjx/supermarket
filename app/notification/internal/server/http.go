package server

import (
	notificationv1 "github.com/tmjwjx/supermarket/api/notification/v1"
	"github.com/tmjwjx/supermarket/app/notification/internal/conf"
	notesvc "github.com/tmjwjx/supermarket/app/notification/internal/service/notification"
	"github.com/tmjwjx/supermarket/pkg/httpauth"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/http"
)

func NewHTTPServer(c *conf.Server, auth *conf.Auth, note *notesvc.NotificationService) *http.Server {
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
	notificationv1.RegisterNotificationServiceHTTPServer(srv, note)
	return srv
}
