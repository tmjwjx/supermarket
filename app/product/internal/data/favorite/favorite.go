package favorite

import (
	"context"
	"time"

	"github.com/tmjwjx/supermarket/app/product/internal/biz/catalog"
	bizfav "github.com/tmjwjx/supermarket/app/product/internal/biz/favorite"
	dataproduct "github.com/tmjwjx/supermarket/app/product/internal/data/product"
	"github.com/tmjwjx/supermarket/app/product/internal/data/sqlerr"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Favorite 是收藏表 用户加商品唯一
type Favorite struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	UserID    string    `gorm:"type:varchar(36);not null;uniqueIndex:uk_fav_user_product,priority:1;index"`
	ProductID string    `gorm:"type:varchar(36);not null;uniqueIndex:uk_fav_user_product,priority:2"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
}

type favoriteRepo struct {
	db *gorm.DB
}

func NewFavoriteRepo(db *gorm.DB) bizfav.FavoriteRepo {
	return &favoriteRepo{db: db}
}

func (r *favoriteRepo) Add(ctx context.Context, userID, productID string) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	err = r.db.WithContext(ctx).Create(&Favorite{ID: id.String(), UserID: userID, ProductID: productID}).Error
	if sqlerr.IsUnique(err) {
		return nil
	}
	return err
}

func (r *favoriteRepo) Remove(ctx context.Context, userID, productID string) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND product_id = ?", userID, productID).Delete(&Favorite{}).Error
}

func (r *favoriteRepo) List(ctx context.Context, userID string) ([]*bizfav.Favorite, error) {
	var rows []Favorite
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&rows).Error; err != nil {
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
	out := make([]*bizfav.Favorite, 0, len(rows))
	for _, row := range rows {
		item := &bizfav.Favorite{ProductID: row.ProductID, Invalid: true}
		if snap, ok := snaps[row.ProductID]; ok {
			item.Card = snap.Card
			item.Invalid = snap.Deleted || snap.Card.Status != catalog.StatusOnSale
		}
		out = append(out, item)
	}
	return out, nil
}
