package product

import (
	"context"

	"github.com/tmjwjx/supermarket/app/product/internal/biz/catalog"

	"gorm.io/gorm"
)

// Snap 是给收藏和浏览用的商品卡片 含是否已删除
type Snap struct {
	Card    catalog.Card
	Deleted bool
}

// LoadCards 按 id 取卡片 包含回收站里的商品
func LoadCards(db *gorm.DB, ctx context.Context, ids []string) (map[string]Snap, error) {
	out := map[string]Snap{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []Product
	if err := db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	markets, err := MinMarket(db, ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = Snap{
			Card: catalog.Card{
				ID:          row.ID,
				Name:        row.Name,
				MainImage:   row.MainImage,
				MinPrice:    row.MinPrice,
				MarketPrice: markets[row.ID],
				Sales:       row.SalesCount,
				Status:      row.Status,
			},
			Deleted: row.DeletedAt.Valid,
		}
	}
	return out, nil
}

// MinMarket 取每个商品启用 SKU 里最低售价对应的划线价
func MinMarket(db *gorm.DB, ctx context.Context, ids []string) (map[string]int64, error) {
	out := map[string]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		ProductID   string
		MarketPrice int64
	}
	var rows []row
	err := db.WithContext(ctx).Raw(`
SELECT s.product_id, s.market_price
FROM skus s
INNER JOIN (
    SELECT product_id, MIN(price) AS min_price
    FROM skus
    WHERE deleted_at IS NULL AND enabled = 1 AND product_id IN ?
    GROUP BY product_id
) t ON t.product_id = s.product_id AND t.min_price = s.price
WHERE s.deleted_at IS NULL AND s.enabled = 1
`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, ok := out[row.ProductID]; ok {
			continue
		}
		out[row.ProductID] = row.MarketPrice
	}
	return out, nil
}
