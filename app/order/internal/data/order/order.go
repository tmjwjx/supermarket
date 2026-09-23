package order

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	bizorder "github.com/tmjwjx/supermarket/app/order/internal/biz/order"
	"github.com/tmjwjx/supermarket/pkg/kafkaout"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Order struct {
	ID             string `gorm:"type:char(36);primaryKey"`
	OrderNo        string `gorm:"size:32;uniqueIndex:uk_order_no"`
	UserID         string `gorm:"size:36;uniqueIndex:uk_user_request;index:idx_user_created,priority:1"`
	RequestID      string `gorm:"size:64;uniqueIndex:uk_user_request"`
	Status         int32  `gorm:"index:idx_status_expires,priority:1"`
	ItemsAmount    int64
	FreightAmount  int64
	PayAmount      int64
	Receiver       string    `gorm:"size:64"`
	Phone          string    `gorm:"size:20"`
	Province       string    `gorm:"size:32"`
	City           string    `gorm:"size:32"`
	District       string    `gorm:"size:32"`
	Detail         string    `gorm:"size:255"`
	Remark         string    `gorm:"size:255"`
	ExpiresAt      time.Time `gorm:"index:idx_status_expires,priority:2"`
	StockReleased  bool
	StockConfirmed bool
	// 存量行加列时要落成 false 否则 NULL 会漏出未确认扫描的条件
	StockConfirmFailed bool `gorm:"not null;default:false"`
	PaidAt             *time.Time
	ShippedAt          *time.Time
	CompletedAt        *time.Time
	CancelledAt        *time.Time
	CancelReason       string    `gorm:"size:32"`
	PaymentID          string    `gorm:"size:64"`
	CreatedAt          time.Time `gorm:"index:idx_user_created,priority:2"`
	UpdatedAt          time.Time
}

func (Order) TableName() string { return "orders" }

type OrderItem struct {
	ID          string `gorm:"type:char(36);primaryKey"`
	OrderID     string `gorm:"size:36;index"`
	SkuID       string `gorm:"size:64"`
	ProductID   string `gorm:"size:36"`
	ProductName string `gorm:"size:128"`
	SpecsJSON   string `gorm:"size:1024"`
	Image       string `gorm:"size:512"`
	Price       int64
	Quantity    int64
	Amount      int64
	Reviewed    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (OrderItem) TableName() string { return "order_items" }

func (p *Order) toBiz() *bizorder.Order {
	id, _ := uuid.Parse(p.ID)
	userID, _ := uuid.Parse(p.UserID)
	return &bizorder.Order{
		ID: id, OrderNo: p.OrderNo, UserID: userID, RequestID: p.RequestID, Status: p.Status,
		ItemsAmount: p.ItemsAmount, FreightAmount: p.FreightAmount, PayAmount: p.PayAmount,
		Receiver: p.Receiver, Phone: p.Phone, Province: p.Province, City: p.City,
		District: p.District, Detail: p.Detail, Remark: p.Remark, ExpiresAt: p.ExpiresAt,
		StockReleased: p.StockReleased, StockConfirmed: p.StockConfirmed, StockConfirmFailed: p.StockConfirmFailed,
		PaidAt: p.PaidAt, ShippedAt: p.ShippedAt, CompletedAt: p.CompletedAt, CancelledAt: p.CancelledAt,
		CancelReason: p.CancelReason, PaymentID: p.PaymentID, CreatedAt: p.CreatedAt,
	}
}

func (p *OrderItem) toBiz() *bizorder.OrderItem {
	id, _ := uuid.Parse(p.ID)
	orderID, _ := uuid.Parse(p.OrderID)
	return &bizorder.OrderItem{
		ID: id, OrderID: orderID, SkuID: p.SkuID, ProductID: p.ProductID, ProductName: p.ProductName,
		SpecsJSON: p.SpecsJSON, Image: p.Image, Price: p.Price, Quantity: p.Quantity, Amount: p.Amount,
		Reviewed: p.Reviewed,
	}
}

type orderRepo struct{ db *gorm.DB }

func NewOrderRepo(db *gorm.DB) bizorder.OrderRepo { return &orderRepo{db: db} }

func (r *orderRepo) FindByUserRequest(ctx context.Context, userID uuid.UUID, requestID string) (*bizorder.Order, error) {
	return r.one(ctx, "user_id = ? AND request_id = ?", userID.String(), requestID)
}

func (r *orderRepo) FindByUser(ctx context.Context, userID, id uuid.UUID) (*bizorder.Order, error) {
	return r.one(ctx, "id = ? AND user_id = ?", id.String(), userID.String())
}

func (r *orderRepo) Find(ctx context.Context, id uuid.UUID) (*bizorder.Order, error) {
	return r.one(ctx, "id = ?", id.String())
}

func (r *orderRepo) Create(ctx context.Context, in *bizorder.Order) (*bizorder.Order, error) {
	var created *bizorder.Order
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := 0; i < 5; i++ {
			no, err := newOrderNo(time.Now())
			if err != nil {
				return err
			}
			po := &Order{
				ID: in.ID.String(), OrderNo: no, UserID: in.UserID.String(), RequestID: in.RequestID,
				Status: in.Status, ItemsAmount: in.ItemsAmount, FreightAmount: in.FreightAmount, PayAmount: in.PayAmount,
				Receiver: in.Receiver, Phone: in.Phone, Province: in.Province, City: in.City, District: in.District,
				Detail: in.Detail, Remark: in.Remark, ExpiresAt: in.ExpiresAt,
			}
			if err := tx.Create(po).Error; err != nil {
				msg := dupMessage(err)
				if msg == "" {
					return err
				}
				if strings.Contains(msg, "uk_user_request") {
					return bizorder.ErrDuplicateRequest
				}
				if strings.Contains(msg, "uk_order_no") {
					continue
				}
				return err
			}
			items := make([]OrderItem, 0, len(in.Items))
			for _, it := range in.Items {
				id, err := uuid.NewV7()
				if err != nil {
					return err
				}
				items = append(items, OrderItem{
					ID: id.String(), OrderID: po.ID, SkuID: it.SkuID, ProductID: it.ProductID,
					ProductName: it.ProductName, SpecsJSON: it.SpecsJSON, Image: it.Image,
					Price: it.Price, Quantity: it.Quantity, Amount: it.Amount,
				})
			}
			if len(items) > 0 {
				if err := tx.Create(&items).Error; err != nil {
					return err
				}
			}
			created = po.toBiz()
			created.Items = make([]*bizorder.OrderItem, len(items))
			for i := range items {
				created.Items[i] = items[i].toBiz()
			}
			return enqueue(tx, "order.created", po.ID, map[string]string{"order_id": po.ID, "user_id": po.UserID})
		}
		return bizorder.ErrOrderInvalidArgument
	})
	return created, err
}

func (r *orderRepo) List(ctx context.Context, userID uuid.UUID, status int32, offset, limit int) ([]*bizorder.Order, error) {
	q := r.db.WithContext(ctx).Where("user_id = ?", userID.String())
	if status != 0 {
		q = q.Where("status = ?", status)
	}
	var rows []Order
	if err := q.Order("created_at desc").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.attach(ctx, rows)
}

func (r *orderRepo) ListAdmin(ctx context.Context, status int32, offset, limit int) ([]*bizorder.Order, error) {
	q := r.db.WithContext(ctx).Model(&Order{})
	if status != 0 {
		q = q.Where("status = ?", status)
	}
	var rows []Order
	if err := q.Order("created_at desc").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.attach(ctx, rows)
}

func (r *orderRepo) CancelIfPending(ctx context.Context, id uuid.UUID, reason string, at time.Time) (bool, error) {
	var ok bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Order{}).
			Where("id = ? AND status = ?", id.String(), bizorder.StatusPending).
			Updates(map[string]any{
				"status": bizorder.StatusCancelled, "cancel_reason": reason, "cancelled_at": at, "stock_released": false,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		ok = true
		var po Order
		if err := tx.First(&po, "id = ?", id.String()).Error; err != nil {
			return err
		}
		return enqueue(tx, "order.cancelled", po.ID, map[string]string{"order_id": po.ID, "user_id": po.UserID})
	})
	return ok, err
}

func (r *orderRepo) MarkReleased(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Order{}).Where("id = ?", id.String()).Update("stock_released", true).Error
}

func (r *orderRepo) MarkPaid(ctx context.Context, id uuid.UUID, paymentID string, at time.Time) (bool, error) {
	var ok bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Order{}).
			Where("id = ? AND status = ?", id.String(), bizorder.StatusPending).
			Updates(map[string]any{"status": bizorder.StatusPaid, "payment_id": paymentID, "paid_at": at})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		ok = true
		var po Order
		if err := tx.First(&po, "id = ?", id.String()).Error; err != nil {
			return err
		}
		return enqueue(tx, "order.paid", po.ID, map[string]string{"order_id": po.ID, "user_id": po.UserID})
	})
	return ok, err
}

func (r *orderRepo) MarkStockConfirmed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Order{}).Where("id = ?", id.String()).Update("stock_confirmed", true).Error
}

func (r *orderRepo) MarkStockConfirmFailed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Order{}).Where("id = ? AND stock_confirmed = ?", id.String(), false).Update("stock_confirm_failed", true).Error
}

func (r *orderRepo) MarkShipped(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	var ok bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Order{}).
			Where("id = ? AND status = ?", id.String(), bizorder.StatusPaid).
			Updates(map[string]any{"status": bizorder.StatusShipped, "shipped_at": at})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		ok = true
		var po Order
		if err := tx.First(&po, "id = ?", id.String()).Error; err != nil {
			return err
		}
		return enqueue(tx, "order.shipped", po.ID, map[string]string{"order_id": po.ID, "user_id": po.UserID})
	})
	return ok, err
}

func (r *orderRepo) MarkCompleted(ctx context.Context, userID, id uuid.UUID, at time.Time) (bool, error) {
	var ok bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Order{}).
			Where("id = ? AND user_id = ? AND status = ?", id.String(), userID.String(), bizorder.StatusShipped).
			Updates(map[string]any{"status": bizorder.StatusCompleted, "completed_at": at})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		ok = true
		var items []OrderItem
		if err := tx.Where("order_id = ?", id.String()).Find(&items).Error; err != nil {
			return err
		}
		type line struct {
			ProductID string `json:"product_id"`
			Count     int64  `json:"count"`
		}
		body := struct {
			OrderID string `json:"order_id"`
			UserID  string `json:"user_id"`
			Items   []line `json:"items"`
		}{OrderID: id.String(), UserID: userID.String()}
		for _, item := range items {
			body.Items = append(body.Items, line{ProductID: item.ProductID, Count: item.Quantity})
		}
		return enqueue(tx, "order.completed", id.String(), body)
	})
	return ok, err
}

func (r *orderRepo) ListPaidBetween(ctx context.Context, from, to time.Time) ([]bizorder.PaidRef, error) {
	var rows []Order
	err := r.db.WithContext(ctx).
		Where("paid_at >= ? AND paid_at < ? AND status IN ?", from, to, []int32{bizorder.StatusPaid, bizorder.StatusShipped, bizorder.StatusCompleted}).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]bizorder.PaidRef, 0, len(rows))
	for _, row := range rows {
		out = append(out, bizorder.PaidRef{ID: row.ID, PaymentID: row.PaymentID, Amount: row.PayAmount})
	}
	return out, nil
}

func enqueue(tx *gorm.DB, topic, key string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return kafkaout.Insert(tx, topic, key, string(raw))
}

func (r *orderRepo) ListExpiredPending(ctx context.Context, now time.Time, limit int) ([]*bizorder.Order, error) {
	var rows []Order
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at <= ?", bizorder.StatusPending, now).
		Order("expires_at").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return r.attach(ctx, rows)
}

func (r *orderRepo) ListUnreleased(ctx context.Context, limit int) ([]*bizorder.Order, error) {
	var rows []Order
	err := r.db.WithContext(ctx).
		Where("status = ? AND stock_released = ?", bizorder.StatusCancelled, false).
		Order("cancelled_at").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return r.attach(ctx, rows)
}

func (r *orderRepo) ListUnconfirmed(ctx context.Context, limit int) ([]*bizorder.Order, error) {
	var rows []Order
	err := r.db.WithContext(ctx).
		Where("status IN ? AND stock_confirmed = ? AND stock_confirm_failed = ?", []int32{bizorder.StatusPaid, bizorder.StatusShipped, bizorder.StatusCompleted}, false, false).
		Order("paid_at").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return r.attach(ctx, rows)
}

func (r *orderRepo) ListShippedBefore(ctx context.Context, before time.Time, limit int) ([]*bizorder.Order, error) {
	var rows []Order
	err := r.db.WithContext(ctx).
		Where("status = ? AND shipped_at IS NOT NULL AND shipped_at <= ?", bizorder.StatusShipped, before).
		Order("shipped_at").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return r.attach(ctx, rows)
}

func (r *orderRepo) GetItem(ctx context.Context, itemID, userID uuid.UUID) (*bizorder.OrderItemView, error) {
	var item OrderItem
	err := r.db.WithContext(ctx).Where("id = ?", itemID.String()).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizorder.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	var po Order
	err = r.db.WithContext(ctx).Where("id = ? AND user_id = ?", item.OrderID, userID.String()).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizorder.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	id, _ := uuid.Parse(item.ID)
	return &bizorder.OrderItemView{
		ID: id, ProductID: item.ProductID, UserID: userID, SpecsJSON: item.SpecsJSON,
		Completed: po.Status == bizorder.StatusCompleted, Reviewed: item.Reviewed,
	}, nil
}

func (r *orderRepo) MarkItemReviewed(ctx context.Context, itemID uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&OrderItem{}).Where("id = ? AND reviewed = ?", itemID.String(), false).Update("reviewed", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		return nil
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&OrderItem{}).Where("id = ?", itemID.String()).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return bizorder.ErrOrderNotFound
	}
	return nil
}

func (r *orderRepo) one(ctx context.Context, query string, args ...any) (*bizorder.Order, error) {
	var po Order
	err := r.db.WithContext(ctx).Where(query, args...).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizorder.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.attach(ctx, []Order{po})
	if err != nil {
		return nil, err
	}
	return rows[0], nil
}

func (r *orderRepo) attach(ctx context.Context, rows []Order) ([]*bizorder.Order, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	ids := make([]string, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	var items []OrderItem
	if err := r.db.WithContext(ctx).Where("order_id IN ?", ids).Order("id").Find(&items).Error; err != nil {
		return nil, err
	}
	grouped := make(map[string][]*bizorder.OrderItem, len(rows))
	for i := range items {
		grouped[items[i].OrderID] = append(grouped[items[i].OrderID], items[i].toBiz())
	}
	out := make([]*bizorder.Order, 0, len(rows))
	for i := range rows {
		bo := rows[i].toBiz()
		bo.Items = grouped[rows[i].ID]
		out = append(out, bo)
	}
	return out, nil
}

func newOrderNo(now time.Time) (string, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint32(buf[:]) % 1000000
	return now.Format("20060102150405") + fmt.Sprintf("%06d", n), nil
}

func dupMessage(err error) string {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return mysqlErr.Message
	}
	return ""
}
