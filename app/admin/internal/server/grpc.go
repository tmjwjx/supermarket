package server

import (
	adminv1 "github.com/tmjwjx/supermarket/api/admin/v1"
	"github.com/tmjwjx/supermarket/app/admin/internal/conf"
	adminsvc "github.com/tmjwjx/supermarket/app/admin/internal/service/adminuser"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/grpc"
)

func NewGRPCServer(c *conf.Server, admin *adminsvc.AdminUserService) *grpc.Server {
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
	adminv1.RegisterAdminUserServiceServer(srv, admin)
	return srv
}
