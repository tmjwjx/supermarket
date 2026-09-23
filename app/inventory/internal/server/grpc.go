package server

import (
	inventoryv1 "github.com/tmjwjx/supermarket/api/inventory/v1"
	"github.com/tmjwjx/supermarket/app/inventory/internal/conf"
	stocksvc "github.com/tmjwjx/supermarket/app/inventory/internal/service/stock"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/grpc"
)

func NewGRPCServer(c *conf.Server, stock *stocksvc.StockService) *grpc.Server {
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
	inventoryv1.RegisterStockServiceServer(srv, stock)
	return srv
}
