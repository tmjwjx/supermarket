package browse

import (
	"context"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/catalog"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

var ErrInvalid = kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "invalid argument")

// Item 是一条浏览记录
type Item struct {
	ProductID string
	Card      catalog.Card
}

// BrowseRepo 是浏览记录存取接口
type BrowseRepo interface {
	Touch(ctx context.Context, userID, productID string) error
	List(ctx context.Context, userID string) ([]*Item, error)
	Clear(ctx context.Context, userID string) error
}

// BrowseUsecase 持有浏览记录规则
type BrowseUsecase struct {
	repo BrowseRepo
}

func NewBrowseUsecase(repo BrowseRepo) *BrowseUsecase {
	return &BrowseUsecase{repo: repo}
}

// List 按最后浏览时间倒序返回记录
func (uc *BrowseUsecase) List(ctx context.Context, userID string) ([]*Item, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalid
	}
	return uc.repo.List(ctx, userID)
}

// Clear 清空当前用户的浏览记录
func (uc *BrowseUsecase) Clear(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrInvalid
	}
	return uc.repo.Clear(ctx, userID)
}
