package favorite

import (
	"context"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/catalog"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

var ErrInvalid = kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "invalid argument")

// Favorite 是一条收藏 下架或已删除时 Invalid 为真
type Favorite struct {
	ProductID string
	Card      catalog.Card
	Invalid   bool
}

// FavoriteRepo 是收藏存取接口
type FavoriteRepo interface {
	Add(ctx context.Context, userID, productID string) error
	Remove(ctx context.Context, userID, productID string) error
	List(ctx context.Context, userID string) ([]*Favorite, error)
}

// FavoriteUsecase 持有收藏规则
type FavoriteUsecase struct {
	repo FavoriteRepo
}

func NewFavoriteUsecase(repo FavoriteRepo) *FavoriteUsecase {
	return &FavoriteUsecase{repo: repo}
}

// Add 收藏商品 重复收藏视为成功
func (uc *FavoriteUsecase) Add(ctx context.Context, userID, productID string) error {
	userID = strings.TrimSpace(userID)
	productID = strings.TrimSpace(productID)
	if userID == "" || productID == "" {
		return ErrInvalid
	}
	return uc.repo.Add(ctx, userID, productID)
}

// Remove 取消收藏 本来就没有也视为成功
func (uc *FavoriteUsecase) Remove(ctx context.Context, userID, productID string) error {
	userID = strings.TrimSpace(userID)
	productID = strings.TrimSpace(productID)
	if userID == "" || productID == "" {
		return ErrInvalid
	}
	return uc.repo.Remove(ctx, userID, productID)
}

// List 按收藏时间倒序返回卡片
func (uc *FavoriteUsecase) List(ctx context.Context, userID string) ([]*Favorite, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalid
	}
	return uc.repo.List(ctx, userID)
}
