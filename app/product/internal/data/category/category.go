package category

import (
	"context"
	"time"

	bizcategory "github.com/tmjwjx/supermarket/app/product/internal/biz/category"
	"github.com/tmjwjx/supermarket/app/product/internal/data/sqlerr"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category 是分类表 父级为空表示一级
type Category struct {
	ID         string `gorm:"type:varchar(36);primaryKey"`
	ParentID   string `gorm:"type:varchar(36);not null;default:'';uniqueIndex:uk_category_parent_name,priority:1;index"`
	Name       string `gorm:"size:64;not null;uniqueIndex:uk_category_parent_name,priority:2"`
	IconURL    string `gorm:"size:512"`
	Sort       int32  `gorm:"not null;index"`
	Visible    bool   `gorm:"not null;index"`
	TemplateID string `gorm:"type:varchar(36)"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func newCategory(c *bizcategory.Category) *Category {
	return &Category{
		ID:         c.ID,
		ParentID:   c.ParentID,
		Name:       c.Name,
		IconURL:    c.IconURL,
		Sort:       c.Sort,
		Visible:    c.Visible,
		TemplateID: c.TemplateID,
	}
}

func (p *Category) toBiz() *bizcategory.Category {
	if p == nil {
		return nil
	}
	return &bizcategory.Category{
		ID:         p.ID,
		ParentID:   p.ParentID,
		Name:       p.Name,
		IconURL:    p.IconURL,
		Sort:       p.Sort,
		Visible:    p.Visible,
		TemplateID: p.TemplateID,
	}
}

type categoryRepo struct {
	db *gorm.DB
}

func NewCategoryRepo(db *gorm.DB) bizcategory.CategoryRepo {
	return &categoryRepo{db: db}
}

func (r *categoryRepo) Save(ctx context.Context, c *bizcategory.Category) (*bizcategory.Category, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	c.ID = id.String()
	po := newCategory(c)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if sqlerr.IsUnique(err) {
			return nil, bizcategory.ErrNameExists
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *categoryRepo) Update(ctx context.Context, c *bizcategory.Category) (*bizcategory.Category, error) {
	res := r.db.WithContext(ctx).Model(&Category{}).Where("id = ?", c.ID).Updates(map[string]any{
		"name": c.Name, "icon_url": c.IconURL, "sort": c.Sort, "visible": c.Visible, "template_id": c.TemplateID,
	})
	if res.Error != nil {
		if sqlerr.IsUnique(res.Error) {
			return nil, bizcategory.ErrNameExists
		}
		return nil, res.Error
	}
	return r.Find(ctx, c.ID)
}

func (r *categoryRepo) ListAll(ctx context.Context) ([]*bizcategory.Category, error) {
	var rows []Category
	if err := r.db.WithContext(ctx).Order("sort asc").Order("name asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*bizcategory.Category, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toBiz())
	}
	return out, nil
}

func (r *categoryRepo) Delete(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Delete(&Category{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return bizcategory.ErrNotFound
	}
	return nil
}

func (r *categoryRepo) Find(ctx context.Context, id string) (*bizcategory.Category, error) {
	var po Category
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if sqlerr.IsNotFound(err) {
			return nil, bizcategory.ErrNotFound
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *categoryRepo) FindByParentAndName(ctx context.Context, parentID, name string) (*bizcategory.Category, error) {
	var po Category
	err := r.db.WithContext(ctx).Where("parent_id = ? AND name = ?", parentID, name).First(&po).Error
	if err != nil {
		if sqlerr.IsNotFound(err) {
			return nil, bizcategory.ErrNotFound
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *categoryRepo) ListVisible(ctx context.Context) ([]*bizcategory.Category, error) {
	var rows []Category
	err := r.db.WithContext(ctx).Where("visible = ?", true).Order("sort asc").Order("name asc").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*bizcategory.Category, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toBiz())
	}
	return out, nil
}

func (r *categoryRepo) ChildIDs(ctx context.Context, parentID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&Category{}).Where("parent_id = ?", parentID).Pluck("id", &ids).Error
	return ids, err
}

func (r *categoryRepo) UsesTemplate(ctx context.Context, templateID string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Category{}).Where("template_id = ?", templateID).Count(&n).Error
	return n > 0, err
}

func (r *categoryRepo) Used(ctx context.Context, id string) (bool, error) {
	var children int64
	if err := r.db.WithContext(ctx).Model(&Category{}).Where("parent_id = ?", id).Count(&children).Error; err != nil {
		return false, err
	}
	if children > 0 {
		return true, nil
	}
	var products int64
	err := r.db.WithContext(ctx).Unscoped().Table("products").Where("category_id = ?", id).Count(&products).Error
	return products > 0, err
}
