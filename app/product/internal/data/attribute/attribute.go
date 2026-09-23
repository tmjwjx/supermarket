package attribute

import (
	"context"
	"encoding/json"
	"time"

	bizattr "github.com/tmjwjx/supermarket/app/product/internal/biz/attribute"
	"github.com/tmjwjx/supermarket/app/product/internal/data/sqlerr"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Template struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	Name      string `gorm:"size:64;not null;uniqueIndex"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Template) TableName() string { return "attribute_templates" }

type Attribute struct {
	ID          string `gorm:"type:varchar(36);primaryKey"`
	TemplateID  string `gorm:"type:varchar(36);not null;uniqueIndex:uk_attr_tpl_name,priority:1;index"`
	Name        string `gorm:"size:64;not null;uniqueIndex:uk_attr_tpl_name,priority:2"`
	Kind        int32  `gorm:"not null"`
	Options     string `gorm:"type:text"`
	AllowCustom bool   `gorm:"not null"`
	Sort        int32  `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Attribute) TableName() string { return "attributes" }

type templateRepo struct{ db *gorm.DB }

func NewTemplateRepo(db *gorm.DB) bizattr.TemplateRepo {
	return &templateRepo{db: db}
}

func (r *templateRepo) SaveTemplate(ctx context.Context, name string) (*bizattr.Template, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	po := &Template{ID: id.String(), Name: name}
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if sqlerr.IsUnique(err) {
			return nil, bizattr.ErrInvalid
		}
		return nil, err
	}
	return &bizattr.Template{ID: po.ID, Name: po.Name}, nil
}

func (r *templateRepo) DeleteTemplate(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("template_id = ?", id).Delete(&Attribute{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&Template{}, "id = ?", id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return bizattr.ErrNotFound
		}
		return nil
	})
}

func (r *templateRepo) FindTemplate(ctx context.Context, id string) (*bizattr.Template, error) {
	var po Template
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if sqlerr.IsNotFound(err) {
			return nil, bizattr.ErrNotFound
		}
		return nil, err
	}
	return &bizattr.Template{ID: po.ID, Name: po.Name}, nil
}

func (r *templateRepo) ListTemplates(ctx context.Context) ([]*bizattr.Template, error) {
	var rows []Template
	if err := r.db.WithContext(ctx).Order("name asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*bizattr.Template, 0, len(rows))
	for _, row := range rows {
		out = append(out, &bizattr.Template{ID: row.ID, Name: row.Name})
	}
	return out, nil
}

func (r *templateRepo) SaveAttribute(ctx context.Context, attr *bizattr.Attribute) (*bizattr.Attribute, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(attr.Options)
	if err != nil {
		return nil, bizattr.ErrInvalid
	}
	po := &Attribute{
		ID: id.String(), TemplateID: attr.TemplateID, Name: attr.Name, Kind: attr.Kind,
		Options: string(raw), AllowCustom: attr.AllowCustom, Sort: attr.Sort,
	}
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if sqlerr.IsUnique(err) {
			return nil, bizattr.ErrInvalid
		}
		return nil, err
	}
	attr.ID = po.ID
	return attr, nil
}

func (r *templateRepo) FindAttribute(ctx context.Context, id string) (*bizattr.Attribute, error) {
	var po Attribute
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if sqlerr.IsNotFound(err) {
			return nil, bizattr.ErrAttributeMissing
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *templateRepo) UpdateAttribute(ctx context.Context, attr *bizattr.Attribute) (*bizattr.Attribute, error) {
	raw, err := json.Marshal(attr.Options)
	if err != nil {
		return nil, bizattr.ErrInvalid
	}
	res := r.db.WithContext(ctx).Model(&Attribute{}).Where("id = ?", attr.ID).Updates(map[string]any{
		"name": attr.Name, "kind": attr.Kind, "options": string(raw), "allow_custom": attr.AllowCustom, "sort": attr.Sort,
	})
	if res.Error != nil {
		if sqlerr.IsUnique(res.Error) {
			return nil, bizattr.ErrInvalid
		}
		return nil, res.Error
	}
	return r.FindAttribute(ctx, attr.ID)
}

func (r *templateRepo) DeleteAttribute(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Delete(&Attribute{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return bizattr.ErrNotFound
	}
	return nil
}

func (r *templateRepo) ListAttributes(ctx context.Context, templateID string) ([]*bizattr.Attribute, error) {
	var rows []Attribute
	err := r.db.WithContext(ctx).Where("template_id = ?", templateID).Order("sort asc").Order("name asc").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*bizattr.Attribute, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toBiz())
	}
	return out, nil
}

func (p *Attribute) toBiz() *bizattr.Attribute {
	item := &bizattr.Attribute{
		ID: p.ID, TemplateID: p.TemplateID, Name: p.Name, Kind: p.Kind,
		AllowCustom: p.AllowCustom, Sort: p.Sort,
	}
	if p.Options != "" {
		_ = json.Unmarshal([]byte(p.Options), &item.Options)
	}
	return item
}
