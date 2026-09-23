package server

import (
	notificationv1 "github.com/tmjwjx/supermarket/api/notification/v1"

	"github.com/go-kratos/kratos/v3/transport/http"
)

type notificationProxy struct {
	gate
	notifications notificationv1.NotificationServiceClient
}

func (p *notificationProxy) routes(r *http.Router) {
	// 登录
	r.GET("/v1/notifications", p.requireLogin(p.listNotifications))
	// 登录
	r.POST("/v1/notifications:read", p.requireLogin(p.markNotificationsRead))
}

func (p *notificationProxy) listNotifications(ctx http.Context) error {
	var in notificationv1.ListNotificationsRequest
	return call(ctx, notificationv1.OperationNotificationServiceListNotifications, &in, rpc(p.notifications.ListNotifications))
}

func (p *notificationProxy) markNotificationsRead(ctx http.Context) error {
	var in notificationv1.MarkNotificationsReadRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, notificationv1.OperationNotificationServiceMarkNotificationsRead, &in, rpc(p.notifications.MarkNotificationsRead))
}
