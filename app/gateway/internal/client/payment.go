package client

import (
	paymentv1 "github.com/tmjwjx/supermarket/api/payment/v1"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"
)

// 拨到 payment 的 gRPC 供支付转发
func NewPaymentClient(c *conf.Client) (paymentv1.PaymentServiceClient, func(), error) {
	conn, err := dial(c.Payment)
	if err != nil {
		return nil, nil, err
	}
	return paymentv1.NewPaymentServiceClient(conn), func() { _ = conn.Close() }, nil
}
