package browse

import (
	"context"
	"time"

	bizbrowse "github.com/tmjwjx/supermarket/app/product/internal/biz/browse"
	dataproduct "github.com/tmjwjx/supermarket/app/product/internal/data/product"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BrowseHistory 是浏览记录 同一用户和商品只留最新一次
type BrowseHistory struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	UserID    string    `gorm:"type:varchar(36);not null;uniqueIndex:uk_browse_user_product,priority:1;index"`
	ProductID string    `gorm:"type:varchar(36);not null;uniqueIndex:uk_browse_user_product,priority:2"`
	ViewedAt  time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (BrowseHistory) TableName() string { return "browse_histories" }

type browseRepo struct {
	db *gorm.DB
}

func NewBrowseRepo(db *gorm.DB) bizbrowse.BrowseRepo {
	return &browseRepo{db: db}
}

func (r *browseRepo) Touch(ctx context.Context, userID, productID string) error {
	now := time.Now()
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	row := &BrowseHistory{
		ID:        id.String(),
		UserID:    userID,
		ProductID: productID,
		ViewedAt:  now,
	}
	err = r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "product_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"viewed_at", "updated_at"}),
	}).Create(row).Error
	if err != nil {
		return err
	}
	return r.trim(ctx, userID)
}

func (r *browseRepo) trim(ctx context.Context, userID string) error {
	var ids []string
	err := r.db.WithContext(ctx).Raw(`
SELECT id FROM browse_histories
WHERE user_id = ?
ORDER BY viewed_at DESC
LIMIT 1000 OFFSET 100
`, userID).Scan(&ids).Error
	if err != nil || len(ids) == 0 {
		return err
	}
	return r.db.WithContext(ctx).Delete(&BrowseHistory{}, "id IN ?", ids).Error
}

func (r *browseRepo) List(ctx context.Context, userID string) ([]*bizbrowse.Item, error) {
	var rows []BrowseHistory
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("viewed_at desc").Limit(100).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ProductID)
	}
	snaps, err := dataproduct.LoadCards(r.db, ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]*bizbrowse.Item, 0, len(rows))
	for _, row := range rows {
		item := &bizbrowse.Item{ProductID: row.ProductID}
		if snap, ok := snaps[row.ProductID]; ok {
			item.Card = snap.Card
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *browseRepo) Clear(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&BrowseHistory{}).Error
}
