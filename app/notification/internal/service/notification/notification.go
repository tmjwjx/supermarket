package notification

import (
	"context"

	v1 "github.com/tmjwjx/supermarket/api/notification/v1"
	biznote "github.com/tmjwjx/supermarket/app/notification/internal/biz/notification"

	kmd "github.com/go-kratos/kratos/v3/metadata"
	"github.com/google/uuid"
)

type NotificationService struct {
	v1.UnimplementedNotificationServiceServer
	uc *biznote.NotificationUsecase
}

func NewNotificationService(uc *biznote.NotificationUsecase) *NotificationService {
	return &NotificationService{uc: uc}
}

func (s *NotificationService) Push(ctx context.Context, req *v1.PushRequest) (*v1.PushResponse, error) {
	if req == nil {
		return nil, biznote.ErrNotificationInvalidArgument
	}
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, biznote.ErrNotificationInvalidArgument
	}
	if err := s.uc.Push(ctx, &biznote.Notification{
		UserID:  userID,
		Title:   req.GetTitle(),
		Body:    req.GetBody(),
		OrderID: req.GetOrderId(),
	}); err != nil {
		return nil, err
	}
	return &v1.PushResponse{}, nil
}

func (s *NotificationService) ListNotifications(ctx context.Context, _ *v1.ListNotificationsRequest) (*v1.ListNotificationsResponse, error) {
	userID, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	items, unread, err := s.uc.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := &v1.ListNotificationsResponse{Unread: unread}
	for _, item := range items {
		resp.Notifications = append(resp.Notifications, toProto(item))
	}
	return resp, nil
}

func (s *NotificationService) MarkNotificationsRead(ctx context.Context, req *v1.MarkNotificationsReadRequest) (*v1.MarkNotificationsReadResponse, error) {
	userID, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(req.GetIds()))
	for _, raw := range req.GetIds() {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, biznote.ErrNotificationInvalidArgument
		}
		ids = append(ids, id)
	}
	if err := s.uc.MarkRead(ctx, userID, ids); err != nil {
		return nil, err
	}
	return &v1.MarkNotificationsReadResponse{}, nil
}

func callerID(ctx context.Context) (uuid.UUID, error) {
	md, ok := kmd.FromServerContext(ctx)
	if !ok {
		return uuid.Nil, biznote.ErrNotificationUnauthenticated
	}
	id, err := uuid.Parse(md.Get("x-md-global-user-id"))
	if err != nil {
		return uuid.Nil, biznote.ErrNotificationUnauthenticated
	}
	return id, nil
}

func toProto(n *biznote.Notification) *v1.Notification {
	if n == nil {
		return nil
	}
	return &v1.Notification{
		Id:      n.ID.String(),
		Title:   n.Title,
		Body:    n.Body,
		OrderId: n.OrderID,
		Read:    n.Read,
	}
}
