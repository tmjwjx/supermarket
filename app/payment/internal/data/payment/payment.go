package payment

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	bizpayment "github.com/tmjwjx/supermarket/app/payment/internal/biz/payment"
	"github.com/tmjwjx/supermarket/pkg/kafkaout"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Payment struct {
	ID             string `gorm:"type:char(36);primaryKey"`
	OrderID        string `gorm:"type:char(36);uniqueIndex"`
	UserID         string `gorm:"type:char(36);index"`
	Amount         int64
	Status         int32  `gorm:"index:idx_status_notified,priority:1"`
	Channel        string `gorm:"size:16"`
	ChannelTradeNo string `gorm:"size:64"`
	Notified       bool   `gorm:"index:idx_status_notified,priority:2"`
	PaidAt         *time.Time
	RefundedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Payment) TableName() string { return "payments" }

type ReconcileDiff struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	Day       string `gorm:"size:10;index"`
	Kind      string `gorm:"size:64"`
	OrderID   string `gorm:"size:64"`
	PaymentID string `gorm:"size:64"`
	Detail    string `gorm:"size:512"`
	CreatedAt time.Time
}

func (ReconcileDiff) TableName() string { return "reconcile_diffs" }

func newPayment(p *bizpayment.Payment) *Payment {
	po := &Payment{
		OrderID:        p.OrderID,
		UserID:         p.UserID.String(),
		Amount:         p.Amount,
		Status:         p.Status,
		Channel:        p.Channel,
		ChannelTradeNo: p.ChannelTradeNo,
		Notified:       p.Notified,
		PaidAt:         p.PaidAt,
		RefundedAt:     p.RefundedAt,
	}
	if p.ID != uuid.Nil {
		po.ID = p.ID.String()
	}
	if po.Channel == "" {
		po.Channel = bizpayment.ChannelMock
	}
	if po.Status == 0 {
		po.Status = bizpayment.StatusPending
	}
	return po
}

func (p *Payment) toBiz() *bizpayment.Payment {
	if p == nil {
		return nil
	}
	id, _ := uuid.Parse(p.ID)
	userID, _ := uuid.Parse(p.UserID)
	return &bizpayment.Payment{
		ID:             id,
		OrderID:        p.OrderID,
		UserID:         userID,
		Amount:         p.Amount,
		Status:         p.Status,
		Channel:        p.Channel,
		ChannelTradeNo: p.ChannelTradeNo,
		Notified:       p.Notified,
		PaidAt:         p.PaidAt,
		RefundedAt:     p.RefundedAt,
	}
}

type paymentRepo struct {
	db *gorm.DB
}

func NewPaymentRepo(db *gorm.DB) bizpayment.PaymentRepo {
	return &paymentRepo{db: db}
}

func (r *paymentRepo) FindByID(ctx context.Context, id uuid.UUID) (*bizpayment.Payment, error) {
	var po Payment
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizpayment.ErrPaymentNotFound
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *paymentRepo) FindByOrderID(ctx context.Context, orderID string) (*bizpayment.Payment, error) {
	var po Payment
	if err := r.db.WithContext(ctx).First(&po, "order_id = ?", orderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizpayment.ErrPaymentNotFound
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *paymentRepo) Create(ctx context.Context, p *bizpayment.Payment) (*bizpayment.Payment, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	p.ID = id
	po := newPayment(p)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, bizpayment.ErrPaymentDuplicate
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *paymentRepo) ResetFailed(ctx context.Context, id uuid.UUID, amount int64) (*bizpayment.Payment, error) {
	res := r.db.WithContext(ctx).Model(&Payment{}).Where("id = ? AND status = ?", id.String(), bizpayment.StatusFailed).Updates(map[string]any{
		"status":           bizpayment.StatusPending,
		"amount":           amount,
		"notified":         false,
		"channel_trade_no": "",
		"paid_at":          gorm.Expr("NULL"),
		"refunded_at":      gorm.Expr("NULL"),
	})
	if res.Error != nil {
		return nil, res.Error
	}
	current, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if res.RowsAffected == 0 && current.Status != bizpayment.StatusPending {
		return nil, bizpayment.ErrPaymentStatusConflict
	}
	return current, nil
}

func (r *paymentRepo) MarkResult(ctx context.Context, id uuid.UUID, success bool) (*bizpayment.Payment, bool, error) {
	updates := map[string]any{"status": bizpayment.StatusFailed}
	if success {
		now := time.Now()
		updates["status"] = bizpayment.StatusSuccess
		updates["paid_at"] = now
		updates["channel_trade_no"] = uuid.NewString()
	}
	res := r.db.WithContext(ctx).Model(&Payment{}).Where("id = ? AND status = ?", id.String(), bizpayment.StatusPending).Updates(updates)
	if res.Error != nil {
		return nil, false, res.Error
	}
	current, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, false, err
	}
	return current, res.RowsAffected > 0, nil
}

func (r *paymentRepo) ListUnnotified(ctx context.Context, limit int) ([]*bizpayment.Payment, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []Payment
	err := r.db.WithContext(ctx).
		Where("status = ? AND notified = ?", bizpayment.StatusSuccess, false).
		Order("paid_at").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*bizpayment.Payment, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toBiz())
	}
	return out, nil
}

func (r *paymentRepo) MarkNotified(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&Payment{}).
		Where("id = ? AND status = ? AND notified = ?", id.String(), bizpayment.StatusSuccess, false).
		Update("notified", true)
	return res.Error
}

func (r *paymentRepo) MarkRefunded(ctx context.Context, id uuid.UUID) (*bizpayment.Payment, error) {
	now := time.Now()
	var current *bizpayment.Payment
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Payment{}).
			Where("id = ? AND status = ?", id.String(), bizpayment.StatusSuccess).
			Updates(map[string]any{
				"status":      bizpayment.StatusRefunded,
				"notified":    true,
				"refunded_at": now,
			})
		if res.Error != nil {
			return res.Error
		}
		var po Payment
		if err := tx.First(&po, "id = ?", id.String()).Error; err != nil {
			return err
		}
		if res.RowsAffected > 0 {
			raw, err := json.Marshal(map[string]string{
				"payment_id": po.ID, "order_id": po.OrderID, "user_id": po.UserID,
			})
			if err != nil {
				return err
			}
			if err := kafkaout.Insert(tx, "payment.refunded", po.ID, string(raw)); err != nil {
				return err
			}
		}
		current = po.toBiz()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return current, nil
}

func (r *paymentRepo) ListSuccessBetween(ctx context.Context, from, to time.Time) ([]*bizpayment.Payment, error) {
	var rows []Payment
	err := r.db.WithContext(ctx).
		Where("status = ? AND paid_at >= ? AND paid_at < ?", bizpayment.StatusSuccess, from, to).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*bizpayment.Payment, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toBiz())
	}
	return out, nil
}

func (r *paymentRepo) SaveDiffs(ctx context.Context, day string, rows []bizpayment.Diff) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("day = ?", day).Delete(&ReconcileDiff{}).Error; err != nil {
			return err
		}
		for _, row := range rows {
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			if err := tx.Create(&ReconcileDiff{
				ID: id.String(), Day: day, Kind: row.Kind, OrderID: row.OrderID, PaymentID: row.PaymentID, Detail: row.Detail,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *paymentRepo) ListDiffs(ctx context.Context, day string, size, offset int) ([]bizpayment.Diff, bool, error) {
	if size <= 0 {
		size = 20
	}
	db := r.db.WithContext(ctx).Model(&ReconcileDiff{})
	if day != "" {
		db = db.Where("day = ?", day)
	}
	var rows []ReconcileDiff
	if err := db.Order("created_at desc").Offset(offset).Limit(size + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > size
	if hasMore {
		rows = rows[:size]
	}
	out := make([]bizpayment.Diff, 0, len(rows))
	for _, row := range rows {
		out = append(out, bizpayment.Diff{ID: row.ID, Day: row.Day, Kind: row.Kind, OrderID: row.OrderID, PaymentID: row.PaymentID, Detail: row.Detail})
	}
	return out, hasMore, nil
}

func isUniqueViolation(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
