package notification

import (
	"context"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrNotificationInvalidArgument = kerrors.BadRequest("NOTIFICATION_INVALID_ARGUMENT", "invalid argument")
	ErrNotificationUnauthenticated = kerrors.Unauthorized("NOTIFICATION_UNAUTHENTICATED", "unauthenticated")
)

type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Title     string
	Body      string
	OrderID   string
	Read      bool
	CreatedAt time.Time
}

type NotificationRepo interface {
	Create(ctx context.Context, n *Notification) (*Notification, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Notification, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int64, error)
	MarkRead(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error
	SaveOnce(ctx context.Context, eventID string, n *Notification) error
}

type NotificationUsecase struct {
	repo NotificationRepo
}

func NewNotificationUsecase(repo NotificationRepo) *NotificationUsecase {
	return &NotificationUsecase{repo: repo}
}

func (uc *NotificationUsecase) Push(ctx context.Context, n *Notification) error {
	if n == nil || n.UserID == uuid.Nil {
		return ErrNotificationInvalidArgument
	}
	n.Title = strings.TrimSpace(n.Title)
	n.Body = strings.TrimSpace(n.Body)
	n.OrderID = strings.TrimSpace(n.OrderID)
	if n.Title == "" {
		return ErrNotificationInvalidArgument
	}
	_, err := uc.repo.Create(ctx, n)
	return err
}

// 按当前用户列出站内信并带上未读数
func (uc *NotificationUsecase) List(ctx context.Context, userID uuid.UUID) ([]*Notification, int64, error) {
	if userID == uuid.Nil {
		return nil, 0, ErrNotificationInvalidArgument
	}
	items, err := uc.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	unread, err := uc.repo.CountUnread(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return items, unread, nil
}

func (uc *NotificationUsecase) MarkRead(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error {
	if userID == uuid.Nil {
		return ErrNotificationInvalidArgument
	}
	if len(ids) == 0 {
		return nil
	}
	return uc.repo.MarkRead(ctx, userID, ids)
}
