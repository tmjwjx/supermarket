package category

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

var (
	ErrNotFound   = kerrors.NotFound(v1.ErrorReason_CATEGORY_NOT_FOUND.String(), "category not found")
	ErrNameExists = kerrors.Conflict(v1.ErrorReason_CATEGORY_NAME_EXISTS.String(), "category name exists")
	ErrInUse      = kerrors.Conflict(v1.ErrorReason_CATEGORY_IN_USE.String(), "category in use")
	ErrNotLeaf    = kerrors.BadRequest(v1.ErrorReason_CATEGORY_NOT_LEAF.String(), "category is not a leaf")
	ErrInvalid    = kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "invalid argument")
)

// Category 是分类领域对象 最多两级
type Category struct {
	ID         string
	ParentID   string
	Name       string
	IconURL    string
	Sort       int32
	Visible    bool
	TemplateID string
	Children   []*Category
}

// Patch 是分类的局部修改 为空的字段保持原值
type Patch struct {
	ID         string
	Name       *string
	IconURL    *string
	Sort       *int32
	Visible    *bool
	TemplateID *string
}

// CategoryRepo 是分类存取接口
type CategoryRepo interface {
	Save(ctx context.Context, c *Category) (*Category, error)
	Update(ctx context.Context, c *Category) (*Category, error)
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (*Category, error)
	FindByParentAndName(ctx context.Context, parentID, name string) (*Category, error)
	ListVisible(ctx context.Context) ([]*Category, error)
	ListAll(ctx context.Context) ([]*Category, error)
	ChildIDs(ctx context.Context, parentID string) ([]string, error)
	Used(ctx context.Context, id string) (bool, error)
	UsesTemplate(ctx context.Context, templateID string) (bool, error)
}

// CategoryUsecase 持有分类层级规则
type CategoryUsecase struct {
	repo CategoryRepo
}

func NewCategoryUsecase(repo CategoryRepo) *CategoryUsecase {
	return &CategoryUsecase{repo: repo}
}

// Create 新建分类 父级必须是一级或不填
func (uc *CategoryUsecase) Create(ctx context.Context, c *Category) (*Category, error) {
	if c == nil {
		return nil, ErrInvalid
	}
	c.Name = strings.TrimSpace(c.Name)
	c.ParentID = strings.TrimSpace(c.ParentID)
	if c.Name == "" {
		return nil, ErrInvalid
	}
	if c.ParentID != "" {
		parent, err := uc.repo.Find(ctx, c.ParentID)
		if err != nil {
			return nil, err
		}
		if parent.ParentID != "" {
			return nil, ErrInvalid
		}
	}
	existing, err := uc.repo.FindByParentAndName(ctx, c.ParentID, c.Name)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrNameExists
	}
	return uc.repo.Save(ctx, c)
}

// Delete 删除分类 有子级或商品时拒绝
func (uc *CategoryUsecase) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalid
	}
	if _, err := uc.repo.Find(ctx, id); err != nil {
		return err
	}
	used, err := uc.repo.Used(ctx, id)
	if err != nil {
		return err
	}
	if used {
		return ErrInUse
	}
	return uc.repo.Delete(ctx, id)
}

// Update 按补丁改名称 图标 排序 显示和模板 父级不可改
func (uc *CategoryUsecase) Update(ctx context.Context, p Patch) (*Category, error) {
	p.ID = strings.TrimSpace(p.ID)
	if p.ID == "" {
		return nil, ErrInvalid
	}
	current, err := uc.repo.Find(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if p.Name != nil {
		current.Name = strings.TrimSpace(*p.Name)
	}
	if p.IconURL != nil {
		current.IconURL = strings.TrimSpace(*p.IconURL)
	}
	if p.Sort != nil {
		current.Sort = *p.Sort
	}
	if p.Visible != nil {
		current.Visible = *p.Visible
	}
	if p.TemplateID != nil {
		current.TemplateID = strings.TrimSpace(*p.TemplateID)
	}
	if current.Name == "" {
		return nil, ErrInvalid
	}
	existing, err := uc.repo.FindByParentAndName(ctx, current.ParentID, current.Name)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil && existing.ID != current.ID {
		return nil, ErrNameExists
	}
	return uc.repo.Update(ctx, current)
}

// Tree 把显示中的分类拼成两层 隐藏的一级下不再带出二级
func (uc *CategoryUsecase) Tree(ctx context.Context) ([]*Category, error) {
	rows, err := uc.repo.ListVisible(ctx)
	if err != nil {
		return nil, err
	}
	return buildTree(rows), nil
}

// AdminTree 给后台用 含隐藏分类
func (uc *CategoryUsecase) AdminTree(ctx context.Context) ([]*Category, error) {
	rows, err := uc.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return buildTree(rows), nil
}

// 找不到父级的二级分类会被丢掉
func buildTree(rows []*Category) []*Category {
	index := make(map[string]*Category, len(rows))
	for _, row := range rows {
		row.Children = nil
		index[row.ID] = row
	}
	roots := make([]*Category, 0)
	for _, row := range rows {
		if row.ParentID == "" {
			roots = append(roots, row)
			continue
		}
		parent, ok := index[row.ParentID]
		if !ok {
			continue
		}
		parent.Children = append(parent.Children, row)
	}
	return roots
}
