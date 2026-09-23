package stock

import (
	"context"
	"sort"
	"strings"
	"time"

	v1 "github.com/tmjwjx/supermarket/api/inventory/v1"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

const (
	ReservationHeld      int32 = 1
	ReservationConfirmed int32 = 2
	ReservationReleased  int32 = 3
)

const (
	ReasonInbound  = "inbound"
	ReasonOutbound = "outbound"
	ReasonReserve  = "reserve"
	ReasonConfirm  = "confirm"
	ReasonRelease  = "release"
)

var (
	ErrStockNotFound             = kerrors.NotFound(v1.ErrorReason_STOCK_NOT_FOUND.String(), "stock not found")
	ErrStockInsufficient         = kerrors.Conflict(v1.ErrorReason_STOCK_INSUFFICIENT.String(), "stock insufficient")
	ErrStockInvalidArgument      = kerrors.BadRequest(v1.ErrorReason_STOCK_INVALID_ARGUMENT.String(), "invalid stock argument")
	ErrReservationNotFound       = kerrors.NotFound(v1.ErrorReason_RESERVATION_NOT_FOUND.String(), "reservation not found")
	ErrReservationStatusConflict = kerrors.Conflict(v1.ErrorReason_RESERVATION_STATUS_CONFLICT.String(), "reservation status conflict")
)

// Insufficient 带上不够的 SKU 列表
func Insufficient(skuIDs []string) error {
	return kerrors.Conflict(v1.ErrorReason_STOCK_INSUFFICIENT.String(), "stock insufficient: "+strings.Join(skuIDs, ","))
}

type Stock struct {
	SkuID     string
	OnHand    int64
	Reserved  int64
	Available int64
	Version   int64
}

type Item struct {
	SkuID    string
	Quantity int64
}

type StockRepo interface {
	Create(ctx context.Context, skuID string) (*Stock, error)
	Adjust(ctx context.Context, skuID string, delta int64, reason string) (*Stock, error)
	Get(ctx context.Context, skuIDs []string) ([]*Stock, error)
	Reserve(ctx context.Context, orderID string, items []Item, expiresAt time.Time) error
	Confirm(ctx context.Context, orderID string) error
	Release(ctx context.Context, orderID string) error
	ListExpiredOrderIDs(ctx context.Context, now time.Time, limit int) ([]string, error)
}

type StockUsecase struct {
	repo StockRepo
}

func NewStockUsecase(repo StockRepo) *StockUsecase {
	return &StockUsecase{repo: repo}
}

// Create 给 SKU 建一条数量为 0 的库存 已存在则直接返回
func (uc *StockUsecase) Create(ctx context.Context, skuID string) (*Stock, error) {
	skuID = strings.TrimSpace(skuID)
	if skuID == "" {
		return nil, ErrStockInvalidArgument
	}
	return uc.repo.Create(ctx, skuID)
}

// Adjust 入库或出库 变化后可卖数量不能小于 0
func (uc *StockUsecase) Adjust(ctx context.Context, skuID string, delta int64, reason string) (*Stock, error) {
	skuID = strings.TrimSpace(skuID)
	if skuID == "" || delta == 0 {
		return nil, ErrStockInvalidArgument
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		if delta > 0 {
			reason = ReasonInbound
		} else {
			reason = ReasonOutbound
		}
	}
	return uc.repo.Adjust(ctx, skuID, delta, reason)
}

// Get 批量取库存 可卖数量由实际数量减已占数量得到
func (uc *StockUsecase) Get(ctx context.Context, skuIDs []string) ([]*Stock, error) {
	ids := compactIDs(skuIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	return uc.repo.Get(ctx, ids)
}

// Reserve 按订单占库存 同一订单已有预占则直接成功
func (uc *StockUsecase) Reserve(ctx context.Context, orderID string, items []Item, expireUnix int64) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" || expireUnix <= 0 {
		return ErrStockInvalidArgument
	}
	merged, err := mergeItems(items)
	if err != nil {
		return err
	}
	return uc.repo.Reserve(ctx, orderID, merged, time.Unix(expireUnix, 0))
}

// Confirm 把占住的预占改成确认并扣减实际数量和已占数量
func (uc *StockUsecase) Confirm(ctx context.Context, orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return ErrStockInvalidArgument
	}
	return uc.repo.Confirm(ctx, orderID)
}

// Release 把占住的预占改成释放并减回已占数量
func (uc *StockUsecase) Release(ctx context.Context, orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return ErrStockInvalidArgument
	}
	return uc.repo.Release(ctx, orderID)
}

// ReleaseExpired 释放已过期且仍占住的预占
func (uc *StockUsecase) ReleaseExpired(ctx context.Context) error {
	ids, err := uc.repo.ListExpiredOrderIDs(ctx, time.Now(), 200)
	if err != nil {
		return err
	}
	var first error
	for _, id := range ids {
		if err := uc.repo.Release(ctx, id); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func mergeItems(items []Item) ([]Item, error) {
	if len(items) == 0 {
		return nil, ErrStockInvalidArgument
	}
	qty := make(map[string]int64, len(items))
	for _, it := range items {
		id := strings.TrimSpace(it.SkuID)
		if id == "" || it.Quantity <= 0 {
			return nil, ErrStockInvalidArgument
		}
		qty[id] += it.Quantity
	}
	out := make([]Item, 0, len(qty))
	for id, q := range qty {
		out = append(out, Item{SkuID: id, Quantity: q})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SkuID < out[j].SkuID })
	return out, nil
}

func compactIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
