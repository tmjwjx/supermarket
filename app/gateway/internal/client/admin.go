package client

import (
	adminv1 "github.com/tmjwjx/supermarket/api/admin/v1"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"
)

// 拨到 admin 的 gRPC 供后台登录和运营账号转发
func NewAdminClient(c *conf.Client) (adminv1.AdminUserServiceClient, func(), error) {
	conn, err := dial(c.Admin)
	if err != nil {
		return nil, nil, err
	}
	return adminv1.NewAdminUserServiceClient(conn), func() { _ = conn.Close() }, nil
}
