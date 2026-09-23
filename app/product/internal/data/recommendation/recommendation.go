package recommendation

import (
	"context"
	"time"

	"github.com/tmjwjx/supermarket/app/product/internal/biz/catalog"
	bizrec "github.com/tmjwjx/supermarket/app/product/internal/biz/recommendation"
	dataproduct "github.com/tmjwjx/supermarket/app/product/internal/data/product"
	"github.com/tmjwjx/supermarket/app/product/internal/data/sqlerr"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Recommendation 是推荐位表 起止秒为 0 表示不限
type Recommendation struct {
	ID          string `gorm:"type:varchar(36);primaryKey"`
	Slot        string `gorm:"size:32;not null;uniqueIndex:uk_rec_slot_product,priority:1"`
	ProductID   string `gorm:"type:varchar(36);not null;uniqueIndex:uk_rec_slot_product,priority:2"`
	Sort        int32  `gorm:"not null;index"`
	StartAtUnix int64  `gorm:"not null;default:0"`
	EndAtUnix   int64  `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func newRecommendation(r *bizrec.Recommendation) *Recommendation {
	return &Recommendation{
		ID: r.ID, Slot: r.Slot, ProductID: r.ProductID, Sort: r.Sort,
		StartAtUnix: r.StartAt, EndAtUnix: r.EndAt,
	}
}

func (p *Recommendation) toBiz() *bizrec.Recommendation {
	return &bizrec.Recommendation{
		ID: p.ID, Slot: p.Slot, ProductID: p.ProductID, Sort: p.Sort,
		StartAt: p.StartAtUnix, EndAt: p.EndAtUnix,
	}
}

type recommendationRepo struct {
	db *gorm.DB
}

func NewRecommendationRepo(db *gorm.DB) bizrec.RecommendationRepo {
	return &recommendationRepo{db: db}
}

func (r *recommendationRepo) Save(ctx context.Context, item *bizrec.Recommendation) (*bizrec.Recommendation, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	item.ID = id.String()
	po := newRecommendation(item)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if sqlerr.IsUnique(err) {
			return r.Find(ctx, item.Slot, item.ProductID)
		}
		return nil, err
	}
	return r.withCard(ctx, po.toBiz())
}

func (r *recommendationRepo) Get(ctx context.Context, id string) (*bizrec.Recommendation, error) {
	var po Recommendation
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if sqlerr.IsNotFound(err) {
			return nil, bizrec.ErrNotFound
		}
		return nil, err
	}
	return r.withCard(ctx, po.toBiz())
}

func (r *recommendationRepo) Find(ctx context.Context, slot, productID string) (*bizrec.Recommendation, error) {
	var po Recommendation
	err := r.db.WithContext(ctx).Where("slot = ? AND product_id = ?", slot, productID).First(&po).Error
	if err != nil {
		if sqlerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return r.withCard(ctx, po.toBiz())
}

func (r *recommendationRepo) Update(ctx context.Context, item *bizrec.Recommendation) (*bizrec.Recommendation, error) {
	res := r.db.WithContext(ctx).Model(&Recommendation{}).Where("id = ?", item.ID).Updates(map[string]any{
		"slot": item.Slot, "product_id": item.ProductID, "sort": item.Sort,
		"start_at_unix": item.StartAt, "end_at_unix": item.EndAt,
	})
	if res.Error != nil {
		if sqlerr.IsUnique(res.Error) {
			return nil, bizrec.ErrInvalid
		}
		return nil, res.Error
	}
	return r.Get(ctx, item.ID)
}

func (r *recommendationRepo) Delete(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Delete(&Recommendation{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return bizrec.ErrNotFound
	}
	return nil
}

func (r *recommendationRepo) List(ctx context.Context, f bizrec.ListFilter) ([]*bizrec.Recommendation, error) {
	db := r.db.WithContext(ctx).Model(&Recommendation{})
	if f.Slot != "" {
		db = db.Where("slot = ?", f.Slot)
	}
	if f.ActiveAt > 0 {
		db = db.Where("(start_at_unix = 0 OR start_at_unix <= ?) AND (end_at_unix = 0 OR end_at_unix > ?)", f.ActiveAt, f.ActiveAt)
	}
	var rows []Recommendation
	if err := db.Order("slot asc").Order("sort asc").Find(&rows).Error; err != nil {
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
	out := make([]*bizrec.Recommendation, 0, len(rows))
	for i := range rows {
		item := rows[i].toBiz()
		snap, ok := snaps[item.ProductID]
		onSale := ok && !snap.Deleted && snap.Card.Status == catalog.StatusOnSale
		if f.ActiveAt > 0 && !onSale {
			continue
		}
		if ok && !snap.Deleted {
			item.Card = snap.Card
		}
		out = append(out, item)
	}
	return out, nil
}

// 后台读单条时带上任意状态的商品卡片 已删除的不带
func (r *recommendationRepo) withCard(ctx context.Context, item *bizrec.Recommendation) (*bizrec.Recommendation, error) {
	snaps, err := dataproduct.LoadCards(r.db, ctx, []string{item.ProductID})
	if err != nil {
		return nil, err
	}
	if snap, ok := snaps[item.ProductID]; ok && !snap.Deleted {
		item.Card = snap.Card
	}
	return item, nil
}
