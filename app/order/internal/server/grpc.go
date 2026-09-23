package server

import (
	orderv1 "github.com/tmjwjx/supermarket/api/order/v1"
	"github.com/tmjwjx/supermarket/app/order/internal/conf"
	cartsvc "github.com/tmjwjx/supermarket/app/order/internal/service/cart"
	ordersvc "github.com/tmjwjx/supermarket/app/order/internal/service/order"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/grpc"
)

func NewGRPCServer(c *conf.Server, cart *cartsvc.CartService, order *ordersvc.OrderService) *grpc.Server {
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
	orderv1.RegisterCartServiceServer(srv, cart)
	orderv1.RegisterOrderServiceServer(srv, order)
	return srv
}
