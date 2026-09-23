package brand

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

var (
	ErrNotFound   = kerrors.NotFound(v1.ErrorReason_BRAND_NOT_FOUND.String(), "brand not found")
	ErrNameExists = kerrors.Conflict(v1.ErrorReason_BRAND_NAME_EXISTS.String(), "brand name exists")
	ErrInUse      = kerrors.Conflict(v1.ErrorReason_BRAND_IN_USE.String(), "brand in use")
	ErrInvalid    = kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "invalid argument")
)

// Brand 是品牌领域对象
type Brand struct {
	ID          string
	Name        string
	Initial     string
	LogoURL     string
	Description string
	Visible     bool
	Sort        int32
}

// BrandRepo 是品牌存取接口
type BrandRepo interface {
	Save(ctx context.Context, b *Brand) (*Brand, error)
	Update(ctx context.Context, b *Brand) (*Brand, error)
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (*Brand, error)
	FindByName(ctx context.Context, name string) (*Brand, error)
	List(ctx context.Context, visibleOnly bool) ([]*Brand, error)
	Used(ctx context.Context, id string) (bool, error)
}

// BrandUsecase 持有品牌规则
type BrandUsecase struct {
	repo BrandRepo
}

func NewBrandUsecase(repo BrandRepo) *BrandUsecase {
	return &BrandUsecase{repo: repo}
}

// Create 新建品牌 名称重复返回冲突
func (uc *BrandUsecase) Create(ctx context.Context, b *Brand) (*Brand, error) {
	if b == nil {
		return nil, ErrInvalid
	}
	b.Name = strings.TrimSpace(b.Name)
	b.Initial = strings.TrimSpace(b.Initial)
	if b.Name == "" {
		return nil, ErrInvalid
	}
	existing, err := uc.repo.FindByName(ctx, b.Name)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrNameExists
	}
	return uc.repo.Save(ctx, b)
}

// Patch 是品牌的局部修改 为空的字段保持原值
type Patch struct {
	ID          string
	Name        *string
	Initial     *string
	LogoURL     *string
	Description *string
	Visible     *bool
	Sort        *int32
}

// Update 只改补丁里带上的字段
func (uc *BrandUsecase) Update(ctx context.Context, p Patch) (*Brand, error) {
	p.ID = strings.TrimSpace(p.ID)
	if p.ID == "" {
		return nil, ErrInvalid
	}
	b, err := uc.repo.Find(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if p.Name != nil {
		b.Name = strings.TrimSpace(*p.Name)
	}
	if p.Initial != nil {
		b.Initial = strings.TrimSpace(*p.Initial)
	}
	if p.LogoURL != nil {
		b.LogoURL = *p.LogoURL
	}
	if p.Description != nil {
		b.Description = *p.Description
	}
	if p.Visible != nil {
		b.Visible = *p.Visible
	}
	if p.Sort != nil {
		b.Sort = *p.Sort
	}
	if b.Name == "" {
		return nil, ErrInvalid
	}
	existing, err := uc.repo.FindByName(ctx, b.Name)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil && existing.ID != b.ID {
		return nil, ErrNameExists
	}
	return uc.repo.Update(ctx, b)
}

// Delete 删除品牌 仍有商品时拒绝
func (uc *BrandUsecase) Delete(ctx context.Context, id string) error {
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

// List 按排序和首字母列出品牌 visibleOnly 为真时只留显示中的
func (uc *BrandUsecase) List(ctx context.Context, visibleOnly bool) ([]*Brand, error) {
	return uc.repo.List(ctx, visibleOnly)
}
