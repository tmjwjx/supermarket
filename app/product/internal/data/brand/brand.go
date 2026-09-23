package brand

import (
	"context"
	"time"

	bizbrand "github.com/tmjwjx/supermarket/app/product/internal/biz/brand"
	"github.com/tmjwjx/supermarket/app/product/internal/data/sqlerr"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Brand 是品牌表
type Brand struct {
	ID          string `gorm:"type:varchar(36);primaryKey"`
	Name        string `gorm:"size:64;not null;uniqueIndex"`
	Initial     string `gorm:"size:8;index"`
	LogoURL     string `gorm:"size:512"`
	Description string `gorm:"size:512"`
	Visible     bool   `gorm:"not null;index"`
	Sort        int32  `gorm:"not null;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func newBrand(b *bizbrand.Brand) *Brand {
	return &Brand{
		ID:          b.ID,
		Name:        b.Name,
		Initial:     b.Initial,
		LogoURL:     b.LogoURL,
		Description: b.Description,
		Visible:     b.Visible,
		Sort:        b.Sort,
	}
}

func (p *Brand) toBiz() *bizbrand.Brand {
	if p == nil {
		return nil
	}
	return &bizbrand.Brand{
		ID:          p.ID,
		Name:        p.Name,
		Initial:     p.Initial,
		LogoURL:     p.LogoURL,
		Description: p.Description,
		Visible:     p.Visible,
		Sort:        p.Sort,
	}
}

type brandRepo struct {
	db *gorm.DB
}

func NewBrandRepo(db *gorm.DB) bizbrand.BrandRepo {
	return &brandRepo{db: db}
}

func (r *brandRepo) Save(ctx context.Context, b *bizbrand.Brand) (*bizbrand.Brand, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	b.ID = id.String()
	po := newBrand(b)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if sqlerr.IsUnique(err) {
			return nil, bizbrand.ErrNameExists
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *brandRepo) Update(ctx context.Context, b *bizbrand.Brand) (*bizbrand.Brand, error) {
	po := newBrand(b)
	res := r.db.WithContext(ctx).Model(&Brand{}).Where("id = ?", b.ID).Updates(map[string]any{
		"name":        po.Name,
		"initial":     po.Initial,
		"logo_url":    po.LogoURL,
		"description": po.Description,
		"visible":     po.Visible,
		"sort":        po.Sort,
	})
	if res.Error != nil {
		if sqlerr.IsUnique(res.Error) {
			return nil, bizbrand.ErrNameExists
		}
		return nil, res.Error
	}
	return r.Find(ctx, b.ID)
}

func (r *brandRepo) Delete(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Delete(&Brand{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return bizbrand.ErrNotFound
	}
	return nil
}

func (r *brandRepo) Find(ctx context.Context, id string) (*bizbrand.Brand, error) {
	var po Brand
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if sqlerr.IsNotFound(err) {
			return nil, bizbrand.ErrNotFound
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *brandRepo) FindByName(ctx context.Context, name string) (*bizbrand.Brand, error) {
	var po Brand
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&po).Error; err != nil {
		if sqlerr.IsNotFound(err) {
			return nil, bizbrand.ErrNotFound
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *brandRepo) List(ctx context.Context, visibleOnly bool) ([]*bizbrand.Brand, error) {
	q := r.db.WithContext(ctx).Order("sort asc").Order("initial asc").Order("name asc")
	if visibleOnly {
		q = q.Where("visible = ?", true)
	}
	var rows []Brand
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*bizbrand.Brand, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toBiz())
	}
	return out, nil
}

func (r *brandRepo) Used(ctx context.Context, id string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Unscoped().Table("products").Where("brand_id = ?", id).Count(&n).Error
	return n > 0, err
}
