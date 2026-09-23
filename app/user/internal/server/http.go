package server

import (
	userv1 "github.com/tmjwjx/supermarket/api/user/v1"
	"github.com/tmjwjx/supermarket/app/user/internal/conf"
	usersvc "github.com/tmjwjx/supermarket/app/user/internal/service/user"
	"github.com/tmjwjx/supermarket/pkg/httpauth"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/middleware/validate"
	"github.com/go-kratos/kratos/v3/transport/http"

	"go.einride.tech/aip/fieldbehavior"
	"google.golang.org/protobuf/proto"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, auth *conf.Auth, user *usersvc.UserService, address *usersvc.AddressService) *http.Server {
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
				UserSecret: secret,
				Public:     []string{userv1.OperationUserServiceRegister, userv1.OperationUserServiceLogin},
			}),
			validate.Validator(func(req any) error {
				if msg, ok := req.(proto.Message); ok {
					if err := fieldbehavior.ValidateRequiredFields(msg); err != nil {
						return err
					}
				}
				return nil
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
	userv1.RegisterUserServiceHTTPServer(srv, user)
	userv1.RegisterAddressServiceHTTPServer(srv, address)
	return srv
}
