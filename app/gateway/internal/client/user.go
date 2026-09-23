package client

import (
	userv1 "github.com/tmjwjx/supermarket/api/user/v1"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"
)

// 同一条 user 连接上的用户和地址客户端
type UserClients struct {
	Users     userv1.UserServiceClient
	Addresses userv1.AddressServiceClient
}

// 拨到 user 的 gRPC 供用户和地址转发
func NewUserClient(c *conf.Client) (*UserClients, func(), error) {
	conn, err := dial(c.User)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = conn.Close() }
	return &UserClients{
		Users:     userv1.NewUserServiceClient(conn),
		Addresses: userv1.NewAddressServiceClient(conn),
	}, cleanup, nil
}
