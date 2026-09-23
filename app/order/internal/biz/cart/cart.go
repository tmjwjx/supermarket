package cart

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	v1 "github.com/tmjwjx/supermarket/api/order/v1"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

const (
	maxLineQty = 99
	maxLines   = 100
)

var (
	ErrCartInvalid        = kerrors.BadRequest(v1.ErrorReason_ORDER_INVALID_ARGUMENT.String(), "invalid cart argument")
	ErrCartFull           = kerrors.BadRequest(v1.ErrorReason_ORDER_INVALID_ARGUMENT.String(), "cart line limit")
	ErrCartItemNotFound   = kerrors.NotFound(v1.ErrorReason_ORDER_NOT_FOUND.String(), "cart item not found")
	ErrProductNotSellable = kerrors.BadRequest("PRODUCT_NOT_SELLABLE", "product not sellable")
	ErrUnauthenticated    = kerrors.Unauthorized(v1.ErrorReason_ORDER_INVALID_ARGUMENT.String(), "unauthenticated")
	ErrUpstream           = kerrors.ServiceUnavailable("ORDER_UPSTREAM", "upstream unavailable")
	ErrCartDuplicate      = errors.New("cart duplicate")
)

type CartItem struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	SkuID       string
	Quantity    int64
	Checked     bool
	ProductName string
	SpecsJSON   string
	Price       int64
	Image       string
	Invalid     bool
	ShortStock  bool
	CreatedAt   time.Time
}

type CartList struct {
	Items         []*CartItem
	CheckedAmount int64
}

type SkuView struct {
	SkuID       string
	ProductName string
	SpecsJSON   string
	Price       int64
	Image       string
	Sellable    bool
}

type ProductGateway interface {
	CheckSellable(ctx context.Context, skuIDs []string) ([]SkuView, error)
	BatchGetSkus(ctx context.Context, skuIDs []string) ([]SkuView, error)
}

type StockGateway interface {
	GetAvailable(ctx context.Context, skuIDs []string) (map[string]int64, error)
}

type CartRepo interface {
	Find(ctx context.Context, userID uuid.UUID, skuID string) (*CartItem, error)
	Create(ctx context.Context, item *CartItem) (*CartItem, error)
	Save(ctx context.Context, item *CartItem) (*CartItem, error)
	Delete(ctx context.Context, userID uuid.UUID, skuID string) error
	List(ctx context.Context, userID uuid.UUID) ([]*CartItem, error)
	Count(ctx context.Context, userID uuid.UUID) (int64, error)
	RemoveSKUs(ctx context.Context, userID uuid.UUID, skuIDs []string) error
}

type CartUsecase struct {
	repo     CartRepo
	products ProductGateway
	stocks   StockGateway
}

func NewCartUsecase(repo CartRepo, products ProductGateway, stocks StockGateway) *CartUsecase {
	return &CartUsecase{repo: repo, products: products, stocks: stocks}
}

// Add 确认可卖后累加数量 已存在则相加并封顶 99
func (uc *CartUsecase) Add(ctx context.Context, userID uuid.UUID, skuID string, quantity int64) (*CartItem, error) {
	skuID = strings.TrimSpace(skuID)
	if userID == uuid.Nil || skuID == "" || quantity <= 0 {
		return nil, ErrCartInvalid
	}
	if err := uc.ensureSellable(ctx, skuID); err != nil {
		return nil, err
	}
	existing, err := uc.repo.Find(ctx, userID, skuID)
	if err != nil && !errors.Is(err, ErrCartItemNotFound) {
		return nil, err
	}
	if existing != nil {
		next := existing.Quantity + quantity
		if next > maxLineQty {
			next = maxLineQty
		}
		existing.Quantity = next
		saved, err := uc.repo.Save(ctx, existing)
		if err != nil {
			return nil, err
		}
		return uc.enrichOne(ctx, saved), nil
	}
	n, err := uc.repo.Count(ctx, userID)
	if err != nil {
		return nil, err
	}
	if n >= maxLines {
		return nil, ErrCartFull
	}
	q := quantity
	if q > maxLineQty {
		q = maxLineQty
	}
	created, err := uc.repo.Create(ctx, &CartItem{UserID: userID, SkuID: skuID, Quantity: q, Checked: true})
	if errors.Is(err, ErrCartDuplicate) {
		again, ferr := uc.repo.Find(ctx, userID, skuID)
		if ferr != nil {
			return nil, ferr
		}
		next := again.Quantity + quantity
		if next > maxLineQty {
			next = maxLineQty
		}
		again.Quantity = next
		saved, serr := uc.repo.Save(ctx, again)
		if serr != nil {
			return nil, serr
		}
		return uc.enrichOne(ctx, saved), nil
	}
	if err != nil {
		return nil, err
	}
	return uc.enrichOne(ctx, created), nil
}

// Update 改数量和勾选 数量为 0 时删除该行
func (uc *CartUsecase) Update(ctx context.Context, userID uuid.UUID, skuID string, quantity int64, checked bool) (*CartItem, error) {
	skuID = strings.TrimSpace(skuID)
	if userID == uuid.Nil || skuID == "" || quantity < 0 || quantity > maxLineQty {
		return nil, ErrCartInvalid
	}
	if quantity == 0 {
		if err := uc.repo.Delete(ctx, userID, skuID); err != nil {
			return nil, err
		}
		return &CartItem{UserID: userID, SkuID: skuID}, nil
	}
	existing, err := uc.repo.Find(ctx, userID, skuID)
	if err != nil {
		return nil, err
	}
	existing.Quantity = quantity
	existing.Checked = checked
	saved, err := uc.repo.Save(ctx, existing)
	if err != nil {
		return nil, err
	}
	return uc.enrichOne(ctx, saved), nil
}

// Remove 按 SKU 批量删掉当前用户的购物车行 本来就不在车里的忽略
func (uc *CartUsecase) Remove(ctx context.Context, userID uuid.UUID, skuIDs []string) error {
	if userID == uuid.Nil || len(skuIDs) == 0 || len(skuIDs) > maxLines {
		return ErrCartInvalid
	}
	ids := make([]string, 0, len(skuIDs))
	for _, id := range skuIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			return ErrCartInvalid
		}
		ids = append(ids, id)
	}
	return uc.repo.RemoveSKUs(ctx, userID, ids)
}

// List 拼上当前商品信息和可卖数量 失效行排在最后
func (uc *CartUsecase) List(ctx context.Context, userID uuid.UUID) (*CartList, error) {
	if userID == uuid.Nil {
		return nil, ErrCartInvalid
	}
	rows, err := uc.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &CartList{}, nil
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.SkuID
	}
	views, err := uc.products.BatchGetSkus(ctx, ids)
	if err != nil {
		return nil, err
	}
	available, err := uc.stocks.GetAvailable(ctx, ids)
	if err != nil {
		return nil, err
	}
	bySKU := make(map[string]SkuView, len(views))
	for _, view := range views {
		bySKU[view.SkuID] = view
	}
	out := make([]*CartItem, 0, len(rows))
	var amount int64
	for _, row := range rows {
		view := *row
		snap, ok := bySKU[row.SkuID]
		if !ok || !snap.Sellable {
			view.Invalid = true
			view.Checked = false
		}
		view.ProductName = snap.ProductName
		view.SpecsJSON = snap.SpecsJSON
		view.Price = snap.Price
		view.Image = snap.Image
		have := available[row.SkuID]
		view.ShortStock = view.Quantity > have
		if view.Checked && !view.Invalid {
			amount += view.Price * view.Quantity
		}
		copied := view
		out = append(out, &copied)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Invalid != out[j].Invalid {
			return !out[i].Invalid
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return &CartList{Items: out, CheckedAmount: amount}, nil
}

func (uc *CartUsecase) ensureSellable(ctx context.Context, skuID string) error {
	snaps, err := uc.products.CheckSellable(ctx, []string{skuID})
	if err != nil {
		return err
	}
	for _, snap := range snaps {
		if snap.SkuID == skuID && snap.Sellable {
			return nil
		}
	}
	return ErrProductNotSellable
}

func (uc *CartUsecase) enrichOne(ctx context.Context, item *CartItem) *CartItem {
	listed, err := uc.List(ctx, item.UserID)
	if err != nil || listed == nil {
		return item
	}
	for _, row := range listed.Items {
		if row.SkuID == item.SkuID {
			return row
		}
	}
	return item
}
