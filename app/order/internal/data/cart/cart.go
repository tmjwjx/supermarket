package cart

import (
	"context"
	"errors"
	"time"

	bizcart "github.com/tmjwjx/supermarket/app/order/internal/biz/cart"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartItem struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	UserID    string `gorm:"size:36;uniqueIndex:uk_user_sku"`
	SkuID     string `gorm:"size:64;uniqueIndex:uk_user_sku"`
	Quantity  int64
	Checked   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (CartItem) TableName() string { return "cart_items" }

func (p *CartItem) toBiz() *bizcart.CartItem {
	if p == nil {
		return nil
	}
	id, _ := uuid.Parse(p.ID)
	userID, _ := uuid.Parse(p.UserID)
	return &bizcart.CartItem{
		ID: id, UserID: userID, SkuID: p.SkuID, Quantity: p.Quantity, Checked: p.Checked, CreatedAt: p.CreatedAt,
	}
}

type cartRepo struct{ db *gorm.DB }

func NewCartRepo(db *gorm.DB) bizcart.CartRepo { return &cartRepo{db: db} }

func (r *cartRepo) Find(ctx context.Context, userID uuid.UUID, skuID string) (*bizcart.CartItem, error) {
	var po CartItem
	err := r.db.WithContext(ctx).Where("user_id = ? AND sku_id = ?", userID.String(), skuID).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizcart.ErrCartItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *cartRepo) Create(ctx context.Context, item *bizcart.CartItem) (*bizcart.CartItem, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	po := &CartItem{
		ID: id.String(), UserID: item.UserID.String(), SkuID: item.SkuID,
		Quantity: item.Quantity, Checked: item.Checked,
	}
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if isDup(err) {
			return nil, bizcart.ErrCartDuplicate
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *cartRepo) Save(ctx context.Context, item *bizcart.CartItem) (*bizcart.CartItem, error) {
	res := r.db.WithContext(ctx).Model(&CartItem{}).
		Where("user_id = ? AND sku_id = ?", item.UserID.String(), item.SkuID).
		Updates(map[string]any{"quantity": item.Quantity, "checked": item.Checked})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return r.Find(ctx, item.UserID, item.SkuID)
	}
	return r.Find(ctx, item.UserID, item.SkuID)
}

func (r *cartRepo) Delete(ctx context.Context, userID uuid.UUID, skuID string) error {
	res := r.db.WithContext(ctx).Where("user_id = ? AND sku_id = ?", userID.String(), skuID).Delete(&CartItem{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return bizcart.ErrCartItemNotFound
	}
	return nil
}

func (r *cartRepo) List(ctx context.Context, userID uuid.UUID) ([]*bizcart.CartItem, error) {
	var rows []CartItem
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID.String()).Order("created_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*bizcart.CartItem, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toBiz())
	}
	return out, nil
}

func (r *cartRepo) Count(ctx context.Context, userID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&CartItem{}).Where("user_id = ?", userID.String()).Count(&n).Error
	return n, err
}

func (r *cartRepo) RemoveSKUs(ctx context.Context, userID uuid.UUID, skuIDs []string) error {
	if len(skuIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("user_id = ? AND sku_id IN ?", userID.String(), skuIDs).Delete(&CartItem{}).Error
}

func isDup(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
