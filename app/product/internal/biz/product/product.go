package product

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/attribute"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/brand"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/catalog"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/category"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/paging"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
)

var (
	ErrNotFound = kerrors.NotFound(v1.ErrorReason_PRODUCT_NOT_FOUND.String(), "product not found")
	ErrInvalid  = kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "invalid argument")
	ErrStatus   = kerrors.Conflict(v1.ErrorReason_PRODUCT_STATUS_CONFLICT.String(), "status conflict")
	ErrSpec     = kerrors.Conflict(v1.ErrorReason_PRODUCT_SPEC_DUPLICATED.String(), "spec duplicated")
)

const (
	StatusDraft    = catalog.StatusDraft
	StatusPending  = catalog.StatusPending
	StatusRejected = catalog.StatusRejected
	StatusApproved = catalog.StatusApproved
	StatusOnSale   = catalog.StatusOnSale
	StatusOff      = catalog.StatusOff
)

// Card 是商品卡片
type Card = catalog.Card

// Sku 是规格领域对象 价格单位是分
type Sku struct {
	ID          string
	ProductID   string
	SpecsJSON   string
	SpecHash    string
	Price       int64
	MarketPrice int64
	Image       string
	Barcode     string
	Enabled     bool
}

// Param 是商品填写的参数值
type Param struct {
	Name  string
	Value string
}

// Trail 是同一次状态变更要落下的审核或操作记录
type Trail struct {
	Operator string
	Action   string
	Reason   string
	Before   string
	After    string
	Audit    bool
}

// Fail 是批量上下架里没成功的一条
type Fail struct {
	ID     string
	Reason string
}

// Product 是商品领域对象
type Product struct {
	ID         string
	CategoryID string
	BrandID    string
	Name       string
	Subtitle   string
	Keywords   string
	MainImage  string
	Unit       string
	DetailHTML string
	WeightGram int32
	Status     int32
	MinPrice   int64
	MaxPrice   int64
	Sales      int64
	Images     []string
	Skus       []Sku
	Params     []Param
}

// SkuView 是给订单用的 SKU 快照
type SkuView struct {
	SkuID     string
	ProductID string
	Name      string
	SpecsJSON string
	Price     int64
	Image     string
	Sellable  bool
}

// ListInput 是买家列表的查询条件
type ListInput struct {
	CategoryID string
	BrandID    string
	MinPrice   int64
	MaxPrice   int64
	OrderBy    string
	PageSize   int32
	PageToken  string
	OnSaleOnly bool
	Deleted    bool
}

// ListFilter 是交给仓库的列表条件 分页已收成偏移
type ListFilter struct {
	CategoryIDs []string
	BrandID     string
	MinPrice    int64
	MaxPrice    int64
	OrderBy     string
	PageSize    int
	Offset      int
	OnSaleOnly  bool
	Deleted     bool
}

// ProductRepo 是商品存取接口
type ProductRepo interface {
	Save(ctx context.Context, p *Product) (*Product, error)
	Get(ctx context.Context, id string) (*Product, error)
	GetFresh(ctx context.Context, id string) (*Product, error)
	GetAny(ctx context.Context, id string) (*Product, error)
	FindByName(ctx context.Context, name string) (*Product, error)
	SetStatus(ctx context.Context, id string, from []int32, to int32, trail *Trail) error
	SoftDelete(ctx context.Context, id string, trail *Trail) error
	Restore(ctx context.Context, id string, trail *Trail) error
	Purge(ctx context.Context, id string) error
	ChangePrice(ctx context.Context, productID, skuID string, price int64, trail *Trail) (*Product, error)
	List(ctx context.Context, q ListFilter) ([]Card, bool, error)
	Search(ctx context.Context, keyword, orderBy string, size, offset int) ([]Card, bool, error)
	Skus(ctx context.Context, ids []string) ([]SkuView, error)
	AddSales(ctx context.Context, eventID, orderID, productID string, n int64) error
	Update(ctx context.Context, p *Product, check UpdateCheck) (*Product, error)
	ListAudits(ctx context.Context, productID string, size, offset int) ([]Audit, bool, error)
	ListLogs(ctx context.Context, productID string, size, offset int) ([]OpLog, bool, error)
}

// Audit 是审核记录
type Audit struct {
	ID        string
	ProductID string
	Action    string
	Operator  string
	Reason    string
	CreatedAt int64
}

// OpLog 是操作日志
type OpLog struct {
	ID        string
	ProductID string
	Operator  string
	Action    string
	Before    string
	After     string
	CreatedAt int64
}

// StockGate 在创建或上架时补齐 SKU 库存记录
type StockGate interface {
	Ensure(ctx context.Context, skuID string) error
}

// TemplateReader 读取分类绑定的属性模板
type TemplateReader interface {
	List(ctx context.Context, templateID string) ([]attribute.Attribute, error)
}

// BrandReader 读取品牌供创建和上架检查
type BrandReader interface {
	Find(ctx context.Context, id string) (*brand.Brand, error)
}

// CategoryLookup 取分类并展开一级分类的子级
type CategoryLookup interface {
	Find(ctx context.Context, id string) (*category.Category, error)
	ChildIDs(ctx context.Context, parentID string) ([]string, error)
}

// BrowseTouch 在打开详情时记一条浏览
type BrowseTouch interface {
	Touch(ctx context.Context, userID, productID string) error
}

// ProductUsecase 持有商品规则
type ProductUsecase struct {
	repo       ProductRepo
	brands     BrandReader
	categories CategoryLookup
	browse     BrowseTouch
	templates  TemplateReader
	stocks     StockGate
}

func NewProductUsecase(repo ProductRepo, brands BrandReader, categories CategoryLookup, browse BrowseTouch, templates TemplateReader) *ProductUsecase {
	return &ProductUsecase{repo: repo, brands: brands, categories: categories, browse: browse, templates: templates}
}

func (uc *ProductUsecase) UseStocks(stocks StockGate) {
	uc.stocks = stocks
}

// Create 在一个事务里写入商品 SKU 图片和详情 状态为草稿
func (uc *ProductUsecase) Create(ctx context.Context, p *Product) (*Product, error) {
	if p == nil {
		return nil, ErrInvalid
	}
	p.Name = strings.TrimSpace(p.Name)
	p.CategoryID = strings.TrimSpace(p.CategoryID)
	p.BrandID = strings.TrimSpace(p.BrandID)
	if p.Name == "" || p.CategoryID == "" || p.BrandID == "" {
		return nil, ErrInvalid
	}
	if len(p.Images) > 9 {
		return nil, ErrInvalid
	}
	if err := normalizeSkus(p); err != nil {
		return nil, err
	}
	cat, err := uc.categories.Find(ctx, p.CategoryID)
	if err != nil {
		return nil, err
	}
	if cat.ParentID == "" {
		return nil, category.ErrNotLeaf
	}
	if err := uc.checkTemplate(ctx, cat, p); err != nil {
		return nil, err
	}
	if _, err := uc.brands.Find(ctx, p.BrandID); err != nil {
		return nil, err
	}
	applyPrice(p)
	if len(p.Images) > 0 {
		p.MainImage = p.Images[0]
	}
	p.Status = StatusDraft
	p.Sales = 0
	saved, err := uc.repo.Save(ctx, p)
	if err != nil {
		return nil, err
	}
	// 事务已提交 补库存失败只记日志 上架前 ready 还会再补
	if err := uc.ensureStocks(ctx, saved); err != nil {
		log.Warn("ensure stocks after create", "product", saved.ID, "err", err)
	}
	return saved, nil
}

func normalizeSkus(p *Product) error {
	seen := make(map[string]struct{}, len(p.Skus))
	for i := range p.Skus {
		sku := &p.Skus[i]
		if sku.Price < 0 || sku.MarketPrice < 0 {
			return ErrInvalid
		}
		canonical, hash, err := CanonicalSpecs(sku.SpecsJSON)
		if err != nil {
			return ErrInvalid
		}
		if _, ok := seen[hash]; ok {
			return ErrSpec
		}
		seen[hash] = struct{}{}
		sku.SpecsJSON = canonical
		sku.SpecHash = hash
	}
	return nil
}

func applyPrice(p *Product) {
	var min, max int64
	has := false
	for _, sku := range p.Skus {
		if !sku.Enabled {
			continue
		}
		if !has || sku.Price < min {
			min = sku.Price
		}
		if sku.Price > max {
			max = sku.Price
		}
		has = true
	}
	if !has {
		p.MinPrice, p.MaxPrice = 0, 0
		return
	}
	p.MinPrice, p.MaxPrice = min, max
}

// Publish 仅允许已通过或下架的商品上架为在售
func (uc *ProductUsecase) Publish(ctx context.Context, id, operator string) (*Product, error) {
	if id == "" {
		return nil, ErrInvalid
	}
	p, err := uc.repo.GetFresh(ctx, id)
	if err != nil {
		return nil, err
	}
	from := []int32{StatusApproved, StatusOff}
	if !containsStatus(from, p.Status) {
		return nil, ErrStatus
	}
	if err := uc.ready(ctx, p); err != nil {
		return nil, err
	}
	if err := uc.repo.SetStatus(ctx, id, from, StatusOnSale, &Trail{Operator: operator, Action: "publish", Before: statusName(p.Status), After: statusName(StatusOnSale)}); err != nil {
		return nil, err
	}
	p.Status = StatusOnSale
	return p, nil
}

func containsStatus(from []int32, status int32) bool {
	for _, s := range from {
		if s == status {
			return true
		}
	}
	return false
}

func (uc *ProductUsecase) ready(ctx context.Context, p *Product) error {
	var miss []string
	cat, err := uc.categories.Find(ctx, p.CategoryID)
	if err != nil && !errors.Is(err, category.ErrNotFound) {
		return err
	}
	if err != nil || cat == nil || cat.ParentID == "" || !cat.Visible {
		miss = append(miss, "category")
	}
	br, err := uc.brands.Find(ctx, p.BrandID)
	if err != nil && !errors.Is(err, brand.ErrNotFound) {
		return err
	}
	if err != nil || br == nil || !br.Visible {
		miss = append(miss, "brand")
	}
	if len(p.Images) == 0 {
		miss = append(miss, "image")
	}
	enabled := 0
	var zero []string
	for _, sku := range p.Skus {
		if !sku.Enabled {
			continue
		}
		enabled++
		if sku.Price <= 0 {
			label := sku.ID
			if label == "" {
				label = sku.SpecsJSON
			}
			if label == "" {
				label = "sku"
			}
			zero = append(zero, label)
		}
	}
	if enabled == 0 {
		miss = append(miss, "sku")
	}
	if len(zero) > 0 {
		return kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "enabled sku price must be greater than 0: "+strings.Join(zero, " "))
	}
	if len(miss) > 0 {
		return kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "missing "+strings.Join(miss, " "))
	}
	return uc.ensureStocks(ctx, p)
}

func (uc *ProductUsecase) ensureStocks(ctx context.Context, p *Product) error {
	if uc == nil || uc.stocks == nil || p == nil {
		return nil
	}
	for _, sku := range p.Skus {
		if !sku.Enabled || sku.ID == "" {
			continue
		}
		if err := uc.stocks.Ensure(ctx, sku.ID); err != nil {
			log.Error("ensure stock", "sku", sku.ID, "err", err)
			return kerrors.ServiceUnavailable("PRODUCT_UNAVAILABLE", "stock create failed")
		}
	}
	return nil
}

// Unpublish 把在售改为下架
func (uc *ProductUsecase) Unpublish(ctx context.Context, id, operator string) (*Product, error) {
	if id == "" {
		return nil, ErrInvalid
	}
	if err := uc.repo.SetStatus(ctx, id, []int32{StatusOnSale}, StatusOff, &Trail{Operator: operator, Action: "unpublish", Before: statusName(StatusOnSale), After: statusName(StatusOff)}); err != nil {
		return nil, err
	}
	return uc.repo.Get(ctx, id)
}

// Get 返回买家看的商品 只放行在售和下架 只带启用的 SKU 带用户 id 时记浏览
func (uc *ProductUsecase) Get(ctx context.Context, id, userID string) (*Product, error) {
	if id == "" {
		return nil, ErrInvalid
	}
	p, err := uc.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Status != StatusOnSale && p.Status != StatusOff {
		return nil, ErrNotFound
	}
	enabled := make([]Sku, 0, len(p.Skus))
	for _, sku := range p.Skus {
		if sku.Enabled {
			enabled = append(enabled, sku)
		}
	}
	p.Skus = enabled
	if userID != "" && uc.browse != nil {
		if err := uc.browse.Touch(ctx, userID, id); err != nil {
			log.Error("touch browse", "user", userID, "product", id, "err", err)
		}
	}
	return p, nil
}

// AdminGet 给后台编辑用 不论状态 回收站里的也返回
func (uc *ProductUsecase) AdminGet(ctx context.Context, id string) (*Product, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalid
	}
	return uc.repo.GetAny(ctx, id)
}

// Detail 返回图文详情
func (uc *ProductUsecase) Detail(ctx context.Context, id string) (string, error) {
	p, err := uc.Get(ctx, id, "")
	if err != nil {
		return "", err
	}
	return p.DetailHTML, nil
}

// List 只列在售商品 一级分类会展开到其子分类
func (uc *ProductUsecase) List(ctx context.Context, in ListInput) ([]Card, string, error) {
	order, err := normalizeOrder(in.OrderBy)
	if err != nil {
		return nil, "", err
	}
	in.OrderBy = order
	off, err := paging.Offset(in.PageToken)
	if err != nil {
		return nil, "", ErrInvalid
	}
	size := paging.Size(in.PageSize)
	filter := ListFilter{
		BrandID:    strings.TrimSpace(in.BrandID),
		MinPrice:   in.MinPrice,
		MaxPrice:   in.MaxPrice,
		OrderBy:    in.OrderBy,
		PageSize:   size,
		Offset:     off,
		OnSaleOnly: in.OnSaleOnly,
		Deleted:    in.Deleted,
	}
	if id := strings.TrimSpace(in.CategoryID); id != "" {
		ids, err := uc.expandCategory(ctx, id)
		if err != nil {
			return nil, "", err
		}
		if len(ids) == 0 {
			return []Card{}, "", nil
		}
		filter.CategoryIDs = ids
	}
	cards, hasMore, err := uc.repo.List(ctx, filter)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if hasMore {
		next = paging.Token(off + size)
	}
	return cards, next, nil
}

func (uc *ProductUsecase) expandCategory(ctx context.Context, id string) ([]string, error) {
	cat, err := uc.categories.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if cat.ParentID != "" {
		return []string{cat.ID}, nil
	}
	return uc.categories.ChildIDs(ctx, cat.ID)
}

// normalizeOrder 只认 latest price price_desc sales 空串按 latest
func normalizeOrder(orderBy string) (string, error) {
	switch orderBy = strings.TrimSpace(orderBy); orderBy {
	case "":
		return "latest", nil
	case "latest", "price", "price_desc", "sales":
		return orderBy, nil
	default:
		return "", ErrInvalid
	}
}

// Search 在名称和关键词里模糊匹配在售商品 排序规则和列表一致
func (uc *ProductUsecase) Search(ctx context.Context, keyword, orderBy, token string, size int32) ([]Card, string, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, "", ErrInvalid
	}
	order, err := normalizeOrder(orderBy)
	if err != nil {
		return nil, "", err
	}
	off, err := paging.Offset(token)
	if err != nil {
		return nil, "", ErrInvalid
	}
	n := paging.Size(size)
	cards, hasMore, err := uc.repo.Search(ctx, keyword, order, n, off)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if hasMore {
		next = paging.Token(off + n)
	}
	return cards, next, nil
}

// BatchSkus 一次最多 50 个 SKU 缺失的标为不可售
func (uc *ProductUsecase) BatchSkus(ctx context.Context, ids []string) ([]SkuView, error) {
	return uc.skuViews(ctx, ids)
}

// CheckSellable 用和快照相同的可售判断
func (uc *ProductUsecase) CheckSellable(ctx context.Context, ids []string) ([]SkuView, error) {
	return uc.skuViews(ctx, ids)
}

func (uc *ProductUsecase) skuViews(ctx context.Context, ids []string) ([]SkuView, error) {
	if len(ids) > 50 {
		return nil, ErrInvalid
	}
	found, err := uc.repo.Skus(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]SkuView, len(found))
	for _, item := range found {
		byID[item.SkuID] = item
	}
	out := make([]SkuView, 0, len(ids))
	for _, id := range ids {
		item, ok := byID[id]
		if !ok {
			out = append(out, SkuView{SkuID: id, Sellable: false})
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

// AddSales 给商品增加销量 同一订单或同一事件只计一次
func (uc *ProductUsecase) AddSales(ctx context.Context, eventID, orderID, productID string, n int64) error {
	if productID == "" || n <= 0 {
		return ErrInvalid
	}
	return uc.repo.AddSales(ctx, eventID, orderID, productID, n)
}
