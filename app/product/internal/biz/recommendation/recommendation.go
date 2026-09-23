package recommendation

import (
	"context"
	"strings"
	"time"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/catalog"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

var (
	ErrInvalid  = kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "invalid argument")
	ErrNotFound = kerrors.NotFound(v1.ErrorReason_RECOMMENDATION_NOT_FOUND.String(), "recommendation not found")
)

// Recommendation 是首页推荐位 起止为 0 表示不限
type Recommendation struct {
	ID        string
	Slot      string
	ProductID string
	Sort      int32
	StartAt   int64
	EndAt     int64
	Card      catalog.Card
}

// Patch 是推荐位的局部修改 为空的字段保持原值
type Patch struct {
	ID        string
	Slot      *string
	ProductID *string
	Sort      *int32
	StartAt   *int64
	EndAt     *int64
}

// ListFilter 是推荐位列表条件 ActiveAt 非 0 时只留生效中且商品在售的
type ListFilter struct {
	Slot     string
	ActiveAt int64
}

// RecommendationRepo 是推荐位存取接口
type RecommendationRepo interface {
	Save(ctx context.Context, r *Recommendation) (*Recommendation, error)
	Get(ctx context.Context, id string) (*Recommendation, error)
	Find(ctx context.Context, slot, productID string) (*Recommendation, error)
	Update(ctx context.Context, r *Recommendation) (*Recommendation, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f ListFilter) ([]*Recommendation, error)
}

// RecommendationUsecase 持有推荐位规则 公开列表只留生效期内的在售商品
type RecommendationUsecase struct {
	repo RecommendationRepo
	now  func() time.Time
}

func NewRecommendationUsecase(repo RecommendationRepo) *RecommendationUsecase {
	return &RecommendationUsecase{repo: repo, now: time.Now}
}

// Create 写入推荐位 同一位置和商品重复时返回已有记录
func (uc *RecommendationUsecase) Create(ctx context.Context, r *Recommendation) (*Recommendation, error) {
	if r == nil {
		return nil, ErrInvalid
	}
	r.Slot = strings.TrimSpace(r.Slot)
	r.ProductID = strings.TrimSpace(r.ProductID)
	if err := validate(r); err != nil {
		return nil, err
	}
	existing, err := uc.repo.Find(ctx, r.Slot, r.ProductID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	return uc.repo.Save(ctx, r)
}

// Update 按补丁改位置 商品 排序和起止时间
func (uc *RecommendationUsecase) Update(ctx context.Context, p Patch) (*Recommendation, error) {
	p.ID = strings.TrimSpace(p.ID)
	if p.ID == "" {
		return nil, ErrInvalid
	}
	current, err := uc.repo.Get(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if p.Slot != nil {
		current.Slot = strings.TrimSpace(*p.Slot)
	}
	if p.ProductID != nil {
		current.ProductID = strings.TrimSpace(*p.ProductID)
	}
	if p.Sort != nil {
		current.Sort = *p.Sort
	}
	if p.StartAt != nil {
		current.StartAt = *p.StartAt
	}
	if p.EndAt != nil {
		current.EndAt = *p.EndAt
	}
	if err := validate(current); err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, current)
}

// Delete 删除推荐位
func (uc *RecommendationUsecase) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalid
	}
	return uc.repo.Delete(ctx, id)
}

// List 按位置返回当前生效且商品在售的推荐
func (uc *RecommendationUsecase) List(ctx context.Context, slot string) ([]*Recommendation, error) {
	slot = strings.TrimSpace(slot)
	if slot == "" {
		return nil, ErrInvalid
	}
	return uc.repo.List(ctx, ListFilter{Slot: slot, ActiveAt: uc.now().Unix()})
}

// AdminList 返回全部推荐位 含过期和未在售 slot 为空时不按位置过滤
func (uc *RecommendationUsecase) AdminList(ctx context.Context, slot string) ([]*Recommendation, error) {
	return uc.repo.List(ctx, ListFilter{Slot: strings.TrimSpace(slot)})
}

func validate(r *Recommendation) error {
	if r.Slot == "" || r.ProductID == "" || r.StartAt < 0 || r.EndAt < 0 {
		return ErrInvalid
	}
	if r.StartAt > 0 && r.EndAt > 0 && r.EndAt <= r.StartAt {
		return ErrInvalid
	}
	return nil
}
