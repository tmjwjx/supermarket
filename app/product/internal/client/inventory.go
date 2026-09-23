package client

import (
	"context"
	"time"

	inventoryv1 "github.com/tmjwjx/supermarket/api/inventory/v1"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/product"
	"github.com/tmjwjx/supermarket/app/product/internal/conf"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/transport/grpc"
)

type Inventory struct {
	cli inventoryv1.StockServiceClient
}

func NewInventory(c *conf.Client) (*Inventory, func(), error) {
	addr := "127.0.0.1:9002"
	var timeout time.Duration
	if c != nil {
		if c.Inventory.Addr != "" {
			addr = c.Inventory.Addr
		}
		timeout = c.Inventory.Timeout()
	}
	opts := []grpc.ClientOption{
		grpc.WithEndpoint(addr),
		grpc.WithMiddleware(tracing.Client(), metadata.Client()),
	}
	if timeout != 0 {
		opts = append(opts, grpc.WithTimeout(timeout))
	}
	conn, err := grpc.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, nil, err
	}
	return &Inventory{cli: inventoryv1.NewStockServiceClient(conn)}, func() { _ = conn.Close() }, nil
}

func NewStockGate(c *conf.Client) (product.StockGate, func(), error) {
	return NewInventory(c)
}

func (i *Inventory) Ensure(ctx context.Context, skuID string) error {
	_, err := i.cli.CreateStock(ctx, &inventoryv1.CreateStockRequest{SkuId: skuID})
	return err
}
