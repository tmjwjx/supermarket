package client

import (
	inventoryv1 "github.com/tmjwjx/supermarket/api/inventory/v1"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"
)

// 拨到 inventory 的 gRPC 供库存查询转发
func NewInventoryClient(c *conf.Client) (inventoryv1.StockServiceClient, func(), error) {
	conn, err := dial(c.Inventory)
	if err != nil {
		return nil, nil, err
	}
	return inventoryv1.NewStockServiceClient(conn), func() { _ = conn.Close() }, nil
}
