package stock

import (
	"context"
	"errors"
	"sort"
	"time"

	bizstock "github.com/tmjwjx/supermarket/app/inventory/internal/biz/stock"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	errNoRow      = errors.New("stock condition missed")
	errIdempotent = errors.New("reservation already exists")
)

type Stock struct {
	SkuID     string `gorm:"primaryKey;size:64"`
	OnHand    int64
	Reserved  int64
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Stock) TableName() string { return "stocks" }

type StockReservation struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	OrderID   string `gorm:"size:36;uniqueIndex:uk_order_sku"`
	SkuID     string `gorm:"size:64;uniqueIndex:uk_order_sku"`
	Quantity  int64
	Status    int32     `gorm:"index:idx_status_expires,priority:1"`
	ExpiresAt time.Time `gorm:"index:idx_status_expires,priority:2"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (StockReservation) TableName() string { return "stock_reservations" }

type StockLog struct {
	ID             string `gorm:"type:char(36);primaryKey"`
	SkuID          string `gorm:"size:64;index"`
	OrderID        string `gorm:"size:36;index"`
	Reason         string `gorm:"size:64"`
	OnHandBefore   int64
	OnHandAfter    int64
	ReservedBefore int64
	ReservedAfter  int64
	CreatedAt      time.Time
}

func (StockLog) TableName() string { return "stock_logs" }

func toBiz(po *Stock) *bizstock.Stock {
	if po == nil {
		return nil
	}
	return &bizstock.Stock{
		SkuID:     po.SkuID,
		OnHand:    po.OnHand,
		Reserved:  po.Reserved,
		Available: po.OnHand - po.Reserved,
		Version:   po.Version,
	}
}

type stockRepo struct {
	db *gorm.DB
}

func NewStockRepo(db *gorm.DB) bizstock.StockRepo {
	return &stockRepo{db: db}
}

func (r *stockRepo) Create(ctx context.Context, skuID string) (*bizstock.Stock, error) {
	po := &Stock{SkuID: skuID}
	err := r.db.WithContext(ctx).Create(po).Error
	if isDup(err) {
		return r.get(ctx, skuID)
	}
	if err != nil {
		return nil, err
	}
	return toBiz(po), nil
}

func (r *stockRepo) Adjust(ctx context.Context, skuID string, delta int64, reason string) (*bizstock.Stock, error) {
	var out *bizstock.Stock
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := applyStock(tx, skuID, delta, 0, reason, "", "sku_id = ? AND on_hand + ? - reserved >= 0", skuID, delta)
		if errors.Is(err, errNoRow) {
			var n int64
			if err := tx.Model(&Stock{}).Where("sku_id = ?", skuID).Count(&n).Error; err != nil {
				return err
			}
			if n == 0 {
				return bizstock.ErrStockNotFound
			}
			return bizstock.ErrStockInsufficient
		}
		if err != nil {
			return err
		}
		var po Stock
		if err := tx.Where("sku_id = ?", skuID).First(&po).Error; err != nil {
			return err
		}
		out = toBiz(&po)
		return nil
	})
	return out, err
}

func (r *stockRepo) Get(ctx context.Context, skuIDs []string) ([]*bizstock.Stock, error) {
	var rows []Stock
	if err := r.db.WithContext(ctx).Where("sku_id IN ?", skuIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	byID := make(map[string]*bizstock.Stock, len(rows))
	for i := range rows {
		byID[rows[i].SkuID] = toBiz(&rows[i])
	}
	out := make([]*bizstock.Stock, 0, len(skuIDs))
	for _, id := range skuIDs {
		if row, ok := byID[id]; ok {
			out = append(out, row)
		}
	}
	return out, nil
}

// Reserve 按 SKU 排序后条件更新 不够则整单回滚
func (r *stockRepo) Reserve(ctx context.Context, orderID string, items []bizstock.Item, expiresAt time.Time) error {
	items = append([]bizstock.Item(nil), items...)
	sort.Slice(items, func(i, j int) bool { return items[i].SkuID < items[j].SkuID })
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&StockReservation{}).Where("order_id = ?", orderID).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
		var bad []string
		for _, it := range items {
			err := applyStock(tx, it.SkuID, 0, it.Quantity, bizstock.ReasonReserve, orderID,
				"sku_id = ? AND on_hand - reserved >= ?", it.SkuID, it.Quantity)
			if errors.Is(err, errNoRow) {
				bad = append(bad, it.SkuID)
				continue
			}
			if err != nil {
				return err
			}
		}
		if len(bad) > 0 {
			var again int64
			if err := tx.Model(&StockReservation{}).Where("order_id = ?", orderID).Count(&again).Error; err != nil {
				return err
			}
			if again > 0 {
				return errIdempotent
			}
			return bizstock.Insufficient(bad)
		}
		for _, it := range items {
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			row := &StockReservation{
				ID: id.String(), OrderID: orderID, SkuID: it.SkuID, Quantity: it.Quantity,
				Status: bizstock.ReservationHeld, ExpiresAt: expiresAt,
			}
			if err := tx.Create(row).Error; err != nil {
				if isDup(err) {
					return errIdempotent
				}
				return err
			}
		}
		return nil
	})
	if errors.Is(err, errIdempotent) {
		return nil
	}
	return err
}

func (r *stockRepo) Confirm(ctx context.Context, orderID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, err := listReservations(tx, orderID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return bizstock.ErrReservationNotFound
		}
		held := make([]StockReservation, 0, len(rows))
		for _, row := range rows {
			switch row.Status {
			case bizstock.ReservationHeld:
				held = append(held, row)
			case bizstock.ReservationConfirmed:
			case bizstock.ReservationReleased:
				return bizstock.ErrReservationStatusConflict
			default:
				return bizstock.ErrReservationStatusConflict
			}
		}
		if len(held) == 0 {
			return nil
		}
		for _, row := range held {
			err := applyStock(tx, row.SkuID, -row.Quantity, -row.Quantity, bizstock.ReasonConfirm, orderID,
				"sku_id = ? AND on_hand >= ? AND reserved >= ?", row.SkuID, row.Quantity, row.Quantity)
			if errors.Is(err, errNoRow) {
				return bizstock.ErrReservationStatusConflict
			}
			if err != nil {
				return err
			}
			res := tx.Model(&StockReservation{}).Where("id = ? AND status = ?", row.ID, bizstock.ReservationHeld).
				Update("status", bizstock.ReservationConfirmed)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return bizstock.ErrReservationStatusConflict
			}
		}
		return nil
	})
}

func (r *stockRepo) Release(ctx context.Context, orderID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, err := listReservations(tx, orderID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return bizstock.ErrReservationNotFound
		}
		held := make([]StockReservation, 0, len(rows))
		for _, row := range rows {
			switch row.Status {
			case bizstock.ReservationHeld:
				held = append(held, row)
			case bizstock.ReservationReleased:
			case bizstock.ReservationConfirmed:
				return bizstock.ErrReservationStatusConflict
			default:
				return bizstock.ErrReservationStatusConflict
			}
		}
		if len(held) == 0 {
			return nil
		}
		for _, row := range held {
			err := applyStock(tx, row.SkuID, 0, -row.Quantity, bizstock.ReasonRelease, orderID,
				"sku_id = ? AND reserved >= ?", row.SkuID, row.Quantity)
			if errors.Is(err, errNoRow) {
				return bizstock.ErrReservationStatusConflict
			}
			if err != nil {
				return err
			}
			res := tx.Model(&StockReservation{}).Where("id = ? AND status = ?", row.ID, bizstock.ReservationHeld).
				Update("status", bizstock.ReservationReleased)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return bizstock.ErrReservationStatusConflict
			}
		}
		return nil
	})
}

func (r *stockRepo) ListExpiredOrderIDs(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 200
	}
	var rows []struct {
		OrderID string `gorm:"column:order_id"`
	}
	err := r.db.WithContext(ctx).Raw(
		"SELECT order_id FROM stock_reservations WHERE status = ? AND expires_at <= ? GROUP BY order_id ORDER BY MIN(expires_at) LIMIT ?",
		bizstock.ReservationHeld, now, limit,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.OrderID)
	}
	return ids, nil
}

func (r *stockRepo) get(ctx context.Context, skuID string) (*bizstock.Stock, error) {
	var po Stock
	if err := r.db.WithContext(ctx).Where("sku_id = ?", skuID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizstock.ErrStockNotFound
		}
		return nil, err
	}
	return toBiz(&po), nil
}

// listReservations 加行锁读最新状态 并发确认或释放同一订单时后到的会看到已提交的结果
func listReservations(tx *gorm.DB, orderID string) ([]StockReservation, error) {
	var rows []StockReservation
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_id = ?", orderID).Order("sku_id").Find(&rows).Error
	return rows, err
}

func applyStock(tx *gorm.DB, skuID string, deltaOn, deltaReserved int64, reason, orderID, where string, args ...any) error {
	res := tx.Model(&Stock{}).Where(where, args...).Updates(map[string]any{
		"on_hand":  gorm.Expr("on_hand + ?", deltaOn),
		"reserved": gorm.Expr("reserved + ?", deltaReserved),
		"version":  gorm.Expr("version + 1"),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errNoRow
	}
	var po Stock
	if err := tx.Where("sku_id = ?", skuID).First(&po).Error; err != nil {
		return err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return tx.Create(&StockLog{
		ID: id.String(), SkuID: skuID, OrderID: orderID, Reason: reason,
		OnHandBefore: po.OnHand - deltaOn, OnHandAfter: po.OnHand,
		ReservedBefore: po.Reserved - deltaReserved, ReservedAfter: po.Reserved,
	}).Error
}

func isDup(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
