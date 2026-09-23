package client

import (
	notificationv1 "github.com/tmjwjx/supermarket/api/notification/v1"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"
)

// 拨到 notification 的 gRPC 供站内信转发
func NewNotificationClient(c *conf.Client) (notificationv1.NotificationServiceClient, func(), error) {
	conn, err := dial(c.Notification)
	if err != nil {
		return nil, nil, err
	}
	return notificationv1.NewNotificationServiceClient(conn), func() { _ = conn.Close() }, nil
}
