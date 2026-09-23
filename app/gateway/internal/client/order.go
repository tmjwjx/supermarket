package client

import (
	orderv1 "github.com/tmjwjx/supermarket/api/order/v1"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"
)

// 同一条 order 连接上的购物车和订单客户端
type OrderClients struct {
	Carts  orderv1.CartServiceClient
	Orders orderv1.OrderServiceClient
}

// 拨到 order 的 gRPC 供购物车和订单转发
func NewOrderClient(c *conf.Client) (*OrderClients, func(), error) {
	conn, err := dial(c.Order)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = conn.Close() }
	return &OrderClients{
		Carts:  orderv1.NewCartServiceClient(conn),
		Orders: orderv1.NewOrderServiceClient(conn),
	}, cleanup, nil
}
