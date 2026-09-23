package client

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/tmjwjx/supermarket/api/order/v1"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/review"
	"github.com/tmjwjx/supermarket/app/product/internal/conf"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/google/wire"
)

// ProviderSet is client providers.
var ProviderSet = wire.NewSet(NewOrderChecker, NewStockGate, NewProfileReader)

type orderChecker struct {
	cli orderv1.OrderServiceClient
}

// NewOrderChecker 拨到 order 供评价确认订单项
func NewOrderChecker(c *conf.Client) (review.OrderChecker, func(), error) {
	addr := "127.0.0.1:9003"
	var timeout time.Duration
	if c != nil {
		if c.Order.Addr != "" {
			addr = c.Order.Addr
		}
		timeout = c.Order.Timeout()
	}
	opts := []grpc.ClientOption{
		grpc.WithEndpoint(addr),
		grpc.WithMiddleware(tracing.Client()),
	}
	if timeout != 0 {
		opts = append(opts, grpc.WithTimeout(timeout))
	}
	conn, err := grpc.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, nil, err
	}
	return &orderChecker{cli: orderv1.NewOrderServiceClient(conn)}, func() { _ = conn.Close() }, nil
}

func (c *orderChecker) Check(ctx context.Context, userID, orderItemID, productID string) (string, error) {
	resp, err := c.cli.GetOrderItem(ctx, &orderv1.GetOrderItemRequest{
		OrderItemId: orderItemID,
		UserId:      userID,
	})
	if err != nil {
		if transportErr(err) {
			return "", err
		}
		return "", review.ErrDenied
	}
	if resp.GetReviewed() {
		return "", review.ErrAlreadyReviewed
	}
	if resp.GetUserId() != userID || resp.GetProductId() != productID || !resp.GetCompleted() {
		return "", review.ErrDenied
	}
	return resp.GetSpecsJson(), nil
}

func transportErr(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	st, ok := status.FromError(err)
	if !ok {
		return true
	}
	switch st.Code() {
	case codes.DeadlineExceeded, codes.Unavailable, codes.Internal, codes.Unknown, codes.ResourceExhausted, codes.Aborted:
		return true
	default:
		return false
	}
}

func (c *orderChecker) MarkReviewed(ctx context.Context, orderItemID string) error {
	_, err := c.cli.MarkOrderItemReviewed(ctx, &orderv1.MarkOrderItemReviewedRequest{OrderItemId: orderItemID})
	return err
}
