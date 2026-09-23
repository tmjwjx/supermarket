package notification

import (
	"context"
	"strings"
	"time"

	biznote "github.com/tmjwjx/supermarket/app/notification/internal/biz/notification"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notification struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	UserID    string `gorm:"type:char(36);index:idx_user_read,priority:1"`
	Title     string `gorm:"size:128"`
	Body      string `gorm:"type:text"`
	OrderID   string `gorm:"type:char(36);index"`
	Read      bool   `gorm:"column:is_read;index:idx_user_read,priority:2"`
	CreatedAt time.Time
}

func (Notification) TableName() string { return "notifications" }

// 发件箱表结构 本轮不连接 Kafka
type OutboxEvent struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	Topic     string `gorm:"size:128;index"`
	BizKey    string `gorm:"size:128"`
	Payload   string `gorm:"type:text"`
	Delivered bool   `gorm:"index"`
	CreatedAt time.Time
}

func (OutboxEvent) TableName() string { return "outbox_events" }

func (p *Notification) toBiz() *biznote.Notification {
	if p == nil {
		return nil
	}
	id, _ := uuid.Parse(p.ID)
	userID, _ := uuid.Parse(p.UserID)
	return &biznote.Notification{
		ID:        id,
		UserID:    userID,
		Title:     p.Title,
		Body:      p.Body,
		OrderID:   p.OrderID,
		Read:      p.Read,
		CreatedAt: p.CreatedAt,
	}
}

type notificationRepo struct {
	db *gorm.DB
}

func NewNotificationRepo(db *gorm.DB) biznote.NotificationRepo {
	return &notificationRepo{db: db}
}

func (r *notificationRepo) Create(ctx context.Context, n *biznote.Notification) (*biznote.Notification, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	po := &Notification{
		ID:      id.String(),
		UserID:  n.UserID.String(),
		Title:   n.Title,
		Body:    n.Body,
		OrderID: n.OrderID,
		Read:    false,
	}
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *notificationRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*biznote.Notification, error) {
	var rows []Notification
	err := r.db.WithContext(ctx).Where("user_id = ?", userID.String()).Order("created_at desc").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*biznote.Notification, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toBiz())
	}
	return out, nil
}

func (r *notificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID.String(), false).
		Count(&n).Error
	return n, err
}

func (r *notificationRepo) MarkRead(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		keys = append(keys, id.String())
	}
	if len(keys) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND id IN ?", userID.String(), keys).
		Update("is_read", true).Error
}

// ConsumedEvent 记录已经写过站内信的事件
type ConsumedEvent struct {
	ID        string `gorm:"type:varchar(80);primaryKey"`
	CreatedAt time.Time
}

func (ConsumedEvent) TableName() string { return "consumed_events" }

func (r *notificationRepo) SaveOnce(ctx context.Context, eventID string, n *biznote.Notification) error {
	if eventID == "" || n == nil {
		return biznote.ErrNotificationInvalidArgument
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ConsumedEvent{ID: eventID}).Error; err != nil {
			if isUnique(err) {
				return nil
			}
			return err
		}
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		return tx.Create(&Notification{
			ID: id.String(), UserID: n.UserID.String(), Title: n.Title, Body: n.Body, OrderID: n.OrderID,
		}).Error
	})
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Duplicate")
}
