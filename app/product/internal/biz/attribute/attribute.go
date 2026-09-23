package attribute

import (
	"context"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

const (
	KindSpec  int32 = 1
	KindParam int32 = 2
)

var (
	ErrInvalid          = kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "invalid argument")
	ErrNotFound         = kerrors.NotFound(v1.ErrorReason_PRODUCT_NOT_FOUND.String(), "attribute template not found")
	ErrAttributeMissing = kerrors.NotFound(v1.ErrorReason_ATTRIBUTE_NOT_FOUND.String(), "attribute not found")
	ErrInUse            = kerrors.Conflict(v1.ErrorReason_ATTRIBUTE_TEMPLATE_IN_USE.String(), "attribute template in use")
)

// Template 是属性模板
type Template struct {
	ID   string
	Name string
}

// Attribute 是模板下的规格或参数
type Attribute struct {
	ID          string
	TemplateID  string
	Name        string
	Kind        int32
	Options     []string
	AllowCustom bool
	Sort        int32
}

// Patch 是属性的局部修改 为空的字段保持原值
type Patch struct {
	ID           string
	Name         *string
	Kind         *int32
	Options      []string
	ClearOptions bool
	AllowCustom  *bool
	Sort         *int32
}

// TemplateRepo 存取模板和属性
type TemplateRepo interface {
	SaveTemplate(ctx context.Context, name string) (*Template, error)
	DeleteTemplate(ctx context.Context, id string) error
	FindTemplate(ctx context.Context, id string) (*Template, error)
	ListTemplates(ctx context.Context) ([]*Template, error)
	SaveAttribute(ctx context.Context, attr *Attribute) (*Attribute, error)
	FindAttribute(ctx context.Context, id string) (*Attribute, error)
	UpdateAttribute(ctx context.Context, attr *Attribute) (*Attribute, error)
	DeleteAttribute(ctx context.Context, id string) error
	ListAttributes(ctx context.Context, templateID string) ([]*Attribute, error)
}

// CategoryBind 判断模板是否已被分类占用
type CategoryBind interface {
	UsesTemplate(ctx context.Context, templateID string) (bool, error)
}

// AttributeUsecase 管理模板以及分类占用时的删除限制
type AttributeUsecase struct {
	repo       TemplateRepo
	categories CategoryBind
}

func NewAttributeUsecase(repo TemplateRepo, categories CategoryBind) *AttributeUsecase {
	return &AttributeUsecase{repo: repo, categories: categories}
}

func (uc *AttributeUsecase) CreateTemplate(ctx context.Context, name string) (*Template, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalid
	}
	return uc.repo.SaveTemplate(ctx, name)
}

func (uc *AttributeUsecase) DeleteTemplate(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalid
	}
	if _, err := uc.repo.FindTemplate(ctx, id); err != nil {
		return err
	}
	used, err := uc.categories.UsesTemplate(ctx, id)
	if err != nil {
		return err
	}
	if used {
		return ErrInUse
	}
	return uc.repo.DeleteTemplate(ctx, id)
}

func (uc *AttributeUsecase) ListTemplates(ctx context.Context) ([]*Template, error) {
	return uc.repo.ListTemplates(ctx)
}

func (uc *AttributeUsecase) CreateAttribute(ctx context.Context, attr *Attribute) (*Attribute, error) {
	if attr == nil {
		return nil, ErrInvalid
	}
	attr.TemplateID = strings.TrimSpace(attr.TemplateID)
	attr.Name = strings.TrimSpace(attr.Name)
	if attr.TemplateID == "" || attr.Name == "" {
		return nil, ErrInvalid
	}
	if attr.Kind != KindSpec && attr.Kind != KindParam {
		return nil, ErrInvalid
	}
	if _, err := uc.repo.FindTemplate(ctx, attr.TemplateID); err != nil {
		return nil, err
	}
	attr.Options = cleanOptions(attr.Options)
	return uc.repo.SaveAttribute(ctx, attr)
}

// UpdateAttribute 按补丁改属性 可选值非空时整体替换 ClearOptions 为真时清空
func (uc *AttributeUsecase) UpdateAttribute(ctx context.Context, p Patch) (*Attribute, error) {
	p.ID = strings.TrimSpace(p.ID)
	if p.ID == "" {
		return nil, ErrInvalid
	}
	current, err := uc.repo.FindAttribute(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if p.Name != nil {
		current.Name = strings.TrimSpace(*p.Name)
	}
	if p.Kind != nil {
		current.Kind = *p.Kind
	}
	if p.ClearOptions {
		current.Options = []string{}
	} else if len(p.Options) > 0 {
		current.Options = cleanOptions(p.Options)
	}
	if p.AllowCustom != nil {
		current.AllowCustom = *p.AllowCustom
	}
	if p.Sort != nil {
		current.Sort = *p.Sort
	}
	if current.Name == "" || (current.Kind != KindSpec && current.Kind != KindParam) {
		return nil, ErrInvalid
	}
	return uc.repo.UpdateAttribute(ctx, current)
}

func cleanOptions(in []string) []string {
	clean := make([]string, 0, len(in))
	for _, option := range in {
		option = strings.TrimSpace(option)
		if option != "" {
			clean = append(clean, option)
		}
	}
	return clean
}

func (uc *AttributeUsecase) DeleteAttribute(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalid
	}
	return uc.repo.DeleteAttribute(ctx, id)
}

func (uc *AttributeUsecase) ListAttributes(ctx context.Context, templateID string) ([]*Attribute, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, ErrInvalid
	}
	if _, err := uc.repo.FindTemplate(ctx, templateID); err != nil {
		return nil, err
	}
	return uc.repo.ListAttributes(ctx, templateID)
}
