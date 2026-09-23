package client

import (
	"context"
	"time"

	userv1 "github.com/tmjwjx/supermarket/api/user/v1"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/review"
	"github.com/tmjwjx/supermarket/app/product/internal/conf"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/transport/grpc"
)

type profileReader struct {
	cli userv1.UserServiceClient
}

func NewProfileReader(c *conf.Client) (review.ProfileReader, func(), error) {
	addr := "127.0.0.1:9000"
	var timeout time.Duration
	if c != nil {
		if c.User.Addr != "" {
			addr = c.User.Addr
		}
		timeout = c.User.Timeout()
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
	return &profileReader{cli: userv1.NewUserServiceClient(conn)}, func() { _ = conn.Close() }, nil
}

func (p *profileReader) Profile(ctx context.Context, userID string) (review.Profile, error) {
	resp, err := p.cli.GetUser(ctx, &userv1.GetUserRequest{Id: userID})
	if err != nil {
		return review.Profile{}, err
	}
	user := resp.GetUser()
	if user == nil {
		return review.Profile{}, review.ErrUnavailable
	}
	return review.Profile{Nickname: user.GetNickname(), Phone: user.GetPhone()}, nil
}
