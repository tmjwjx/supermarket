package product

import (
	"context"

	bizproduct "github.com/tmjwjx/supermarket/app/product/internal/biz/product"
	"github.com/tmjwjx/supermarket/app/product/internal/data/sqlerr"
	"github.com/tmjwjx/supermarket/pkg/kafkaout"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Update 先锁住商品行 用锁住读到的库内商品交给 check 决定能否改规格 改完 SKU 后重新聚合价格
func (r *productRepo) Update(ctx context.Context, p *bizproduct.Product, check bizproduct.UpdateCheck) (*bizproduct.Product, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := lockProduct(tx, p.ID)
		if err != nil {
			return err
		}
		replaceSpecs, trail, err := check(ctx, current)
		if err != nil {
			return err
		}
		updates := map[string]any{
			"name": p.Name, "subtitle": p.Subtitle, "keywords": p.Keywords, "unit": p.Unit,
			"weight_gram": p.WeightGram, "main_image": p.MainImage,
		}
		if replaceSpecs {
			updates["category_id"] = p.CategoryID
			updates["brand_id"] = p.BrandID
		}
		if err := tx.Model(&Product{}).Where("id = ?", p.ID).Updates(updates).Error; err != nil {
			return err
		}
		if replaceSpecs {
			if err := replaceSkus(tx, p); err != nil {
				return err
			}
		} else {
			for _, sku := range p.Skus {
				if err := tx.Model(&Sku{}).Where("id = ? AND product_id = ?", sku.ID, p.ID).Updates(map[string]any{
					"price": sku.Price, "market_price": sku.MarketPrice, "image": sku.Image,
				}).Error; err != nil {
					return err
				}
			}
		}
		if err := refreshPrice(tx, p.ID); err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", p.ID).Delete(&ProductImage{}).Error; err != nil {
			return err
		}
		for i, url := range p.Images {
			iid, err := uuid.NewV7()
			if err != nil {
				return err
			}
			if err := tx.Create(&ProductImage{ID: iid.String(), ProductID: p.ID, URL: url, Sort: int32(i)}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("product_id = ?", p.ID).Delete(&ProductParam{}).Error; err != nil {
			return err
		}
		for _, param := range p.Params {
			pid, err := uuid.NewV7()
			if err != nil {
				return err
			}
			if err := tx.Create(&ProductParam{ID: pid.String(), ProductID: p.ID, Name: param.Name, Value: param.Value}).Error; err != nil {
				if sqlerr.IsUnique(err) {
					return bizproduct.ErrInvalid
				}
				return err
			}
		}
		// Save 会把零值 created_at 写回 严格模式下 MySQL 拒绝 所以用 upsert 只改正文
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "product_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"html", "updated_at"}),
		}).Create(&ProductDetail{ProductID: p.ID, HTML: p.DetailHTML}).Error; err != nil {
			return err
		}
		return writeTrail(tx, p.ID, trail)
	})
	if err != nil {
		return nil, err
	}
	r.dropCache(ctx, p.ID)
	return r.load(ctx, p.ID)
}

func replaceSkus(tx *gorm.DB, p *bizproduct.Product) error {
	var existing []Sku
	if err := tx.Where("product_id = ?", p.ID).Find(&existing).Error; err != nil {
		return err
	}
	byID := map[string]*Sku{}
	byHash := map[string]*Sku{}
	for i := range existing {
		byID[existing[i].ID] = &existing[i]
		byHash[existing[i].SpecHash] = &existing[i]
	}
	keep := map[string]bool{}
	for i := range p.Skus {
		sku := &p.Skus[i]
		var match *Sku
		if sku.ID != "" {
			match = byID[sku.ID]
		}
		if match == nil {
			if row := byHash[sku.SpecHash]; row != nil && !keep[row.ID] {
				match = row
			}
		}
		if match != nil {
			keep[match.ID] = true
			if err := tx.Model(&Sku{}).Where("id = ?", match.ID).Updates(map[string]any{
				"specs": sku.SpecsJSON, "spec_hash": sku.SpecHash, "price": sku.Price,
				"market_price": sku.MarketPrice, "image": sku.Image, "barcode": sku.Barcode, "enabled": sku.Enabled,
			}).Error; err != nil {
				if sqlerr.IsUnique(err) {
					return bizproduct.ErrSpec
				}
				return err
			}
			sku.ID = match.ID
			continue
		}
		sid, err := uuid.NewV7()
		if err != nil {
			return err
		}
		row := &Sku{
			ID: sid.String(), ProductID: p.ID, Specs: sku.SpecsJSON, SpecHash: sku.SpecHash,
			Price: sku.Price, MarketPrice: sku.MarketPrice, Image: sku.Image, Barcode: sku.Barcode, Enabled: sku.Enabled,
		}
		if err := tx.Create(row).Error; err != nil {
			if sqlerr.IsUnique(err) {
				return bizproduct.ErrSpec
			}
			return err
		}
		sku.ID = row.ID
		keep[row.ID] = true
		if err := kafkaout.Insert(tx, "product.sku.created", sku.ID, `{"sku_id":"`+sku.ID+`","product_id":"`+p.ID+`"}`); err != nil {
			return err
		}
	}
	for _, row := range existing {
		if keep[row.ID] {
			continue
		}
		if err := tx.Unscoped().Delete(&Sku{}, "id = ?", row.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *productRepo) ListAudits(ctx context.Context, productID string, size, offset int) ([]bizproduct.Audit, bool, error) {
	db := r.db.WithContext(ctx).Model(&ProductAudit{})
	if productID != "" {
		db = db.Where("product_id = ?", productID)
	}
	var rows []ProductAudit
	if err := db.Order("created_at desc").Offset(offset).Limit(size + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > size
	if hasMore {
		rows = rows[:size]
	}
	out := make([]bizproduct.Audit, 0, len(rows))
	for _, row := range rows {
		out = append(out, bizproduct.Audit{
			ID: row.ID, ProductID: row.ProductID, Action: row.Action, Operator: row.Operator,
			Reason: row.Reason, CreatedAt: row.CreatedAt.Unix(),
		})
	}
	return out, hasMore, nil
}

func (r *productRepo) ListLogs(ctx context.Context, productID string, size, offset int) ([]bizproduct.OpLog, bool, error) {
	db := r.db.WithContext(ctx).Model(&ProductLog{})
	if productID != "" {
		db = db.Where("product_id = ?", productID)
	}
	var rows []ProductLog
	if err := db.Order("created_at desc").Offset(offset).Limit(size + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > size
	if hasMore {
		rows = rows[:size]
	}
	out := make([]bizproduct.OpLog, 0, len(rows))
	for _, row := range rows {
		out = append(out, bizproduct.OpLog{
			ID: row.ID, ProductID: row.ProductID, Operator: row.Operator, Action: row.Action,
			Before: row.Before, After: row.After, CreatedAt: row.CreatedAt.Unix(),
		})
	}
	return out, hasMore, nil
}
