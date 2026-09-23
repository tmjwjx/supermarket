package client

import (
	"context"
	"errors"
	"time"

	orderv1 "github.com/tmjwjx/supermarket/api/order/v1"
	bizpayment "github.com/tmjwjx/supermarket/app/payment/internal/biz/payment"
	"github.com/tmjwjx/supermarket/app/payment/internal/conf"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	kerrors "github.com/go-kratos/kratos/v3/errors"
	kmd "github.com/go-kratos/kratos/v3/metadata"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/google/uuid"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewOrderClient)

type orderClient struct {
	api orderv1.OrderServiceClient
}

func NewOrderClient(c *conf.Client) (bizpayment.OrderClient, func(), error) {
	if c == nil || c.Order.Addr == "" {
		return nil, nil, errors.New("order client addr is empty")
	}
	opts := []grpc.ClientOption{
		grpc.WithEndpoint(c.Order.Addr),
		grpc.WithMiddleware(tracing.Client(), metadata.Client()),
	}
	if d := c.Order.Timeout(); d != 0 {
		opts = append(opts, grpc.WithTimeout(d))
	}
	conn, err := grpc.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, nil, err
	}
	return &orderClient{api: orderv1.NewOrderServiceClient(conn)}, func() { _ = conn.Close() }, nil
}

func (c *orderClient) GetOrder(ctx context.Context, userID uuid.UUID, orderID string) (*bizpayment.OrderView, error) {
	ctx = kmd.AppendToClientContext(ctx, "x-md-global-user-id", userID.String())
	resp, err := c.api.GetOrder(ctx, &orderv1.GetOrderRequest{Id: orderID})
	if err != nil {
		e := kerrors.FromError(err)
		if kerrors.IsNotFound(err) || e.Reason == orderv1.ErrorReason_ORDER_NOT_FOUND.String() {
			return nil, bizpayment.ErrPaymentOrderNotPayable
		}
		return nil, bizpayment.ErrPaymentUpstream
	}
	order := resp.GetOrder()
	if order == nil || order.GetId() == "" {
		return nil, bizpayment.ErrPaymentOrderNotPayable
	}
	return &bizpayment.OrderView{
		ID:        order.GetId(),
		UserID:    userID,
		Status:    order.GetStatus(),
		PayAmount: order.GetPayAmount(),
		ExpiresAt: time.Unix(order.GetExpiresAtUnix(), 0),
	}, nil
}

func (c *orderClient) MarkOrderPaid(ctx context.Context, orderID string, paymentID uuid.UUID) error {
	_, err := c.api.MarkOrderPaid(ctx, &orderv1.MarkOrderPaidRequest{
		OrderId:   orderID,
		PaymentId: paymentID.String(),
	})
	if err == nil {
		return nil
	}
	e := kerrors.FromError(err)
	if e.Reason == orderv1.ErrorReason_ORDER_STATUS_CONFLICT.String() {
		return bizpayment.ErrOrderStatusConflict
	}
	return bizpayment.ErrPaymentUpstream
}

func (c *orderClient) ListPaidOrders(ctx context.Context, from, to time.Time) ([]bizpayment.PaidOrder, error) {
	resp, err := c.api.ListPaidOrders(ctx, &orderv1.ListPaidOrdersRequest{
		PaidFromUnix: from.Unix(),
		PaidToUnix:   to.Unix(),
	})
	if err != nil {
		return nil, bizpayment.ErrPaymentUpstream
	}
	out := make([]bizpayment.PaidOrder, 0, len(resp.GetOrders()))
	for _, row := range resp.GetOrders() {
		if row == nil {
			continue
		}
		out = append(out, bizpayment.PaidOrder{ID: row.GetId(), PaymentID: row.GetPaymentId(), Amount: row.GetPayAmount()})
	}
	return out, nil
}
