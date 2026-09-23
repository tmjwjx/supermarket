package client

import (
	"context"

	userv1 "github.com/tmjwjx/supermarket/api/user/v1"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"

	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewUserClient)

// 拨到 user 的 gRPC 供 Register Login GetUser 转发
func NewUserClient(c *conf.Client) (userv1.UserServiceClient, func(), error) {
	opts := []grpc.ClientOption{
		grpc.WithEndpoint(c.User.Addr),
	}
	if d := c.User.Timeout(); d != 0 {
		opts = append(opts, grpc.WithTimeout(d))
	}
	conn, err := grpc.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, nil, err
	}
	return userv1.NewUserServiceClient(conn), func() { _ = conn.Close() }, nil
}
