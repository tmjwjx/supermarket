package product

import (
	"context"
	"encoding/json"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/category"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/paging"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
)

// UpdateCheck 由仓库在事务里锁住商品行后调用 current 是锁住读到的库内商品
// 返回是否替换规格结构和要写的操作记录 返回错误则整个事务回滚
type UpdateCheck func(ctx context.Context, current *Product) (replaceSpecs bool, trail *Trail, err error)

// Update 在售只改价格和文字 下架或非在售才允许改规格结构 在售与否以锁住的库内状态为准
func (uc *ProductUsecase) Update(ctx context.Context, p *Product, operator string) (*Product, error) {
	if p == nil || strings.TrimSpace(p.ID) == "" {
		return nil, ErrInvalid
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return nil, ErrInvalid
	}
	if len(p.Images) > 9 {
		return nil, ErrInvalid
	}
	if err := normalizeSkus(p); err != nil {
		return nil, err
	}
	if len(p.Images) > 0 {
		p.MainImage = p.Images[0]
	}
	updated, err := uc.repo.Update(ctx, p, func(ctx context.Context, current *Product) (bool, *Trail, error) {
		replaceSpecs, err := uc.planUpdate(ctx, current, p)
		if err != nil {
			return false, nil, err
		}
		before, after := priceTrail(current.Skus, p.Skus)
		return replaceSpecs, &Trail{Operator: operator, Action: "update", Before: before, After: after}, nil
	})
	if err != nil {
		return nil, err
	}
	// 事务已提交 补库存失败只记日志 上架前 ready 还会再补
	if err := uc.ensureStocks(ctx, updated); err != nil {
		log.Warn("ensure stocks after update", "product", updated.ID, "err", err)
	}
	return updated, nil
}

// planUpdate 按当前状态校验改动 在售时把分类品牌钉回当前值
func (uc *ProductUsecase) planUpdate(ctx context.Context, current *Product, p *Product) (bool, error) {
	if current.Status == StatusOnSale {
		if p.CategoryID != "" && p.CategoryID != current.CategoryID {
			return false, kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "on sale product cannot change category")
		}
		if p.BrandID != "" && p.BrandID != current.BrandID {
			return false, kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "on sale product cannot change brand")
		}
		if !sameSpecStructure(current.Skus, p.Skus) {
			return false, kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "on sale product cannot change sku structure")
		}
		p.CategoryID = current.CategoryID
		p.BrandID = current.BrandID
		return false, nil
	}
	p.CategoryID = strings.TrimSpace(p.CategoryID)
	p.BrandID = strings.TrimSpace(p.BrandID)
	if p.CategoryID == "" || p.BrandID == "" {
		return false, ErrInvalid
	}
	cat, err := uc.categories.Find(ctx, p.CategoryID)
	if err != nil {
		return false, err
	}
	if cat.ParentID == "" {
		return false, category.ErrNotLeaf
	}
	if err := uc.checkTemplate(ctx, cat, p); err != nil {
		return false, err
	}
	if _, err := uc.brands.Find(ctx, p.BrandID); err != nil {
		return false, err
	}
	return true, nil
}

type skuPrice struct {
	SkuID string `json:"sku_id"`
	Specs string `json:"specs"`
	Price int64  `json:"price"`
}

// priceTrail 按 id 其次按规格哈希对上旧 SKU 只列出售价变了的 没有变化返回空串
func priceTrail(oldRows, next []Sku) (string, string) {
	byID := make(map[string]Sku, len(oldRows))
	byHash := make(map[string]Sku, len(oldRows))
	for _, sku := range oldRows {
		byID[sku.ID] = sku
		byHash[sku.SpecHash] = sku
	}
	var before, after []skuPrice
	for _, sku := range next {
		prev, ok := byID[sku.ID]
		if sku.ID == "" || !ok {
			prev, ok = byHash[sku.SpecHash]
		}
		if !ok || prev.Price == sku.Price {
			continue
		}
		before = append(before, skuPrice{SkuID: prev.ID, Specs: prev.SpecsJSON, Price: prev.Price})
		after = append(after, skuPrice{SkuID: prev.ID, Specs: sku.SpecsJSON, Price: sku.Price})
	}
	if len(before) == 0 {
		return "", ""
	}
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	return string(b), string(a)
}

func sameSpecStructure(oldRows, next []Sku) bool {
	if len(oldRows) != len(next) {
		return false
	}
	seen := make(map[string]Sku, len(oldRows))
	for _, sku := range oldRows {
		seen[sku.ID] = sku
	}
	for _, sku := range next {
		prev, ok := seen[sku.ID]
		if !ok || sku.ID == "" {
			return false
		}
		if prev.SpecHash != sku.SpecHash || prev.Enabled != sku.Enabled {
			return false
		}
	}
	return true
}

func (uc *ProductUsecase) ListAudits(ctx context.Context, productID, token string, size int32) ([]Audit, string, error) {
	off, err := paging.Offset(token)
	if err != nil {
		return nil, "", ErrInvalid
	}
	n := paging.Size(size)
	rows, hasMore, err := uc.repo.ListAudits(ctx, strings.TrimSpace(productID), n, off)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if hasMore {
		next = paging.Token(off + n)
	}
	return rows, next, nil
}

func (uc *ProductUsecase) ListLogs(ctx context.Context, productID, token string, size int32) ([]OpLog, string, error) {
	off, err := paging.Offset(token)
	if err != nil {
		return nil, "", ErrInvalid
	}
	n := paging.Size(size)
	rows, hasMore, err := uc.repo.ListLogs(ctx, strings.TrimSpace(productID), n, off)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if hasMore {
		next = paging.Token(off + n)
	}
	return rows, next, nil
}
