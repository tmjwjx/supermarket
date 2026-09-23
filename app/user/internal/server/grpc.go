package server

import (
	userv1 "github.com/tmjwjx/supermarket/api/user/v1"
	"github.com/tmjwjx/supermarket/app/user/internal/conf"
	usersvc "github.com/tmjwjx/supermarket/app/user/internal/service/user"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, user *usersvc.UserService, address *usersvc.AddressService) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			metadata.Server(),
		),
	}
	if c.GRPC.Network != "" {
		opts = append(opts, grpc.Network(c.GRPC.Network))
	}
	if c.GRPC.Addr != "" {
		opts = append(opts, grpc.Address(c.GRPC.Addr))
	}
	if d := c.GRPC.Timeout(); d != 0 {
		opts = append(opts, grpc.Timeout(d))
	}
	srv := grpc.NewServer(opts...)
	userv1.RegisterUserServiceServer(srv, user)
	userv1.RegisterAddressServiceServer(srv, address)
	return srv
}
