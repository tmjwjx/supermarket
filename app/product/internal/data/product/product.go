package product

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	bizproduct "github.com/tmjwjx/supermarket/app/product/internal/biz/product"
	"github.com/tmjwjx/supermarket/app/product/internal/data/sqlerr"
	"github.com/tmjwjx/supermarket/pkg/kafkaout"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const detailTTL = 10 * time.Minute

// Product 是商品表
type Product struct {
	ID         string    `gorm:"type:varchar(36);primaryKey"`
	CategoryID string    `gorm:"type:varchar(36);not null;index:idx_status_category_created,priority:2"`
	BrandID    string    `gorm:"type:varchar(36);not null;index"`
	Name       string    `gorm:"size:128;not null;index"`
	Subtitle   string    `gorm:"size:256"`
	Keywords   string    `gorm:"size:256"`
	MainImage  string    `gorm:"size:512"`
	Unit       string    `gorm:"size:32"`
	WeightGram int32     `gorm:"not null"`
	Status     int32     `gorm:"not null;index:idx_status_category_created,priority:1;index:idx_status_min_price,priority:1"`
	MinPrice   int64     `gorm:"not null;index:idx_status_min_price,priority:2"`
	MaxPrice   int64     `gorm:"not null"`
	SalesCount int64     `gorm:"not null"`
	CreatedAt  time.Time `gorm:"index:idx_status_category_created,priority:3"`
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// Sku 是规格表 商品加规格哈希唯一
type Sku struct {
	ID          string `gorm:"type:varchar(36);primaryKey"`
	ProductID   string `gorm:"type:varchar(36);not null;uniqueIndex:uk_product_spec,priority:1;index"`
	Specs       string `gorm:"type:text"`
	SpecHash    string `gorm:"size:64;not null;uniqueIndex:uk_product_spec,priority:2"`
	Price       int64  `gorm:"not null"`
	MarketPrice int64  `gorm:"not null"`
	Image       string `gorm:"size:512"`
	Barcode     string `gorm:"size:64"`
	Enabled     bool   `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// ProductImage 是商品图片 每个商品最多 9 张
type ProductImage struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	ProductID string `gorm:"type:varchar(36);not null;index"`
	URL       string `gorm:"size:512;not null"`
	Sort      int32  `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ProductImage) TableName() string { return "product_images" }

// ProductDetail 是图文详情 一个商品一条
type ProductDetail struct {
	ProductID string `gorm:"type:varchar(36);primaryKey"`
	HTML      string `gorm:"type:mediumtext"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ProductDetail) TableName() string { return "product_details" }

// ProductParam 是商品参数值 同一商品下名称唯一
type ProductParam struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	ProductID string `gorm:"type:varchar(36);not null;uniqueIndex:uk_product_param,priority:1;index"`
	Name      string `gorm:"size:64;not null;uniqueIndex:uk_product_param,priority:2"`
	Value     string `gorm:"size:512"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ProductParam) TableName() string { return "product_params" }

// ProductAudit 是只增不改的审核记录
type ProductAudit struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	ProductID string `gorm:"type:varchar(36);not null;index"`
	Action    string `gorm:"size:32;not null"`
	Operator  string `gorm:"size:64;not null"`
	Reason    string `gorm:"size:512"`
	CreatedAt time.Time
}

func (ProductAudit) TableName() string { return "product_audits" }

// ProductLog 是只增不改的操作日志
type ProductLog struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	ProductID string `gorm:"type:varchar(36);not null;index"`
	Operator  string `gorm:"size:64;not null"`
	Action    string `gorm:"size:32;not null"`
	Before    string `gorm:"type:text"`
	After     string `gorm:"type:text"`
	CreatedAt time.Time
}

func (ProductLog) TableName() string { return "product_logs" }

// ConsumedEvent 记下已经处理过的订单完成事件
type ConsumedEvent struct {
	ID        string `gorm:"type:varchar(80);primaryKey"`
	CreatedAt time.Time
}

func (ConsumedEvent) TableName() string { return "consumed_events" }

// SalesApply 保证同一订单对同一商品只加一次销量
type SalesApply struct {
	OrderID   string `gorm:"type:varchar(64);primaryKey"`
	ProductID string `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time
}

func (SalesApply) TableName() string { return "sales_applies" }

type productRepo struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewProductRepo(db *gorm.DB, rdb *redis.Client) bizproduct.ProductRepo {
	return &productRepo{db: db, rdb: rdb}
}

func detailKey(id string) string { return "product:detail:" + id }

func (r *productRepo) Save(ctx context.Context, p *bizproduct.Product) (*bizproduct.Product, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	p.ID = id.String()
	po := &Product{
		ID:         p.ID,
		CategoryID: p.CategoryID,
		BrandID:    p.BrandID,
		Name:       p.Name,
		Subtitle:   p.Subtitle,
		Keywords:   p.Keywords,
		MainImage:  p.MainImage,
		Unit:       p.Unit,
		WeightGram: p.WeightGram,
		Status:     p.Status,
		MinPrice:   p.MinPrice,
		MaxPrice:   p.MaxPrice,
		SalesCount: p.Sales,
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(po).Error; err != nil {
			return err
		}
		for i := range p.Skus {
			sku := &p.Skus[i]
			sid, err := uuid.NewV7()
			if err != nil {
				return err
			}
			canonical, hash, err := bizproduct.CanonicalSpecs(sku.SpecsJSON)
			if err != nil {
				return bizproduct.ErrInvalid
			}
			row := &Sku{
				ID:          sid.String(),
				ProductID:   p.ID,
				Specs:       canonical,
				SpecHash:    hash,
				Price:       sku.Price,
				MarketPrice: sku.MarketPrice,
				Image:       sku.Image,
				Barcode:     sku.Barcode,
				Enabled:     sku.Enabled,
			}
			if err := tx.Create(row).Error; err != nil {
				if sqlerr.IsUnique(err) {
					return bizproduct.ErrSpec
				}
				return err
			}
			sku.ID = row.ID
			sku.ProductID = p.ID
			sku.SpecsJSON = canonical
			sku.SpecHash = hash
			if err := kafkaout.Insert(tx, "product.sku.created", sku.ID, `{"sku_id":"`+sku.ID+`","product_id":"`+p.ID+`"}`); err != nil {
				return err
			}
		}
		for i, url := range p.Images {
			iid, err := uuid.NewV7()
			if err != nil {
				return err
			}
			if err := tx.Create(&ProductImage{
				ID:        iid.String(),
				ProductID: p.ID,
				URL:       url,
				Sort:      int32(i),
			}).Error; err != nil {
				return err
			}
		}
		for _, param := range p.Params {
			pid, err := uuid.NewV7()
			if err != nil {
				return err
			}
			if err := tx.Create(&ProductParam{
				ID: pid.String(), ProductID: p.ID, Name: param.Name, Value: param.Value,
			}).Error; err != nil {
				return err
			}
		}
		return tx.Create(&ProductDetail{ProductID: p.ID, HTML: p.DetailHTML}).Error
	})
	if err != nil {
		return nil, err
	}
	r.storeCache(ctx, p)
	return p, nil
}

func (r *productRepo) Get(ctx context.Context, id string) (*bizproduct.Product, error) {
	if id == "" {
		return nil, bizproduct.ErrInvalid
	}
	if p := r.loadCache(ctx, id); p != nil {
		return p, nil
	}
	p, err := r.load(ctx, id)
	if err != nil {
		return nil, err
	}
	r.storeCache(ctx, p)
	return p, nil
}

func (r *productRepo) load(ctx context.Context, id string) (*bizproduct.Product, error) {
	return r.loadFrom(r.db.WithContext(ctx), id)
}

// GetFresh 直接读库 不读也不写缓存
func (r *productRepo) GetFresh(ctx context.Context, id string) (*bizproduct.Product, error) {
	if id == "" {
		return nil, bizproduct.ErrInvalid
	}
	return r.load(ctx, id)
}

// lockProduct 在事务里 SELECT ... FOR UPDATE 锁住商品行 再读出规格图片参数
func lockProduct(tx *gorm.DB, id string) (*bizproduct.Product, error) {
	var po Product
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, "id = ?", id).Error; err != nil {
		if sqlerr.IsNotFound(err) {
			return nil, bizproduct.ErrNotFound
		}
		return nil, err
	}
	return assemble(tx, &po)
}

// refreshPrice 按库里启用的 SKU 重新聚合商品的最低价和最高价
func refreshPrice(tx *gorm.DB, productID string) error {
	var agg struct {
		MinPrice int64
		MaxPrice int64
	}
	if err := tx.Model(&Sku{}).
		Select("COALESCE(MIN(price), 0) AS min_price, COALESCE(MAX(price), 0) AS max_price").
		Where("product_id = ? AND enabled = ?", productID, true).
		Scan(&agg).Error; err != nil {
		return err
	}
	return tx.Model(&Product{}).Where("id = ?", productID).Updates(map[string]any{"min_price": agg.MinPrice, "max_price": agg.MaxPrice}).Error
}

// GetAny 不走缓存 回收站里的商品也能取到
func (r *productRepo) GetAny(ctx context.Context, id string) (*bizproduct.Product, error) {
	if id == "" {
		return nil, bizproduct.ErrInvalid
	}
	return r.loadFrom(r.db.WithContext(ctx).Unscoped().Session(&gorm.Session{}), id)
}

func (r *productRepo) loadFrom(db *gorm.DB, id string) (*bizproduct.Product, error) {
	var po Product
	if err := db.First(&po, "id = ?", id).Error; err != nil {
		if sqlerr.IsNotFound(err) {
			return nil, bizproduct.ErrNotFound
		}
		return nil, err
	}
	return assemble(db, &po)
}

func assemble(db *gorm.DB, po *Product) (*bizproduct.Product, error) {
	id := po.ID
	var skus []Sku
	if err := db.Where("product_id = ?", id).Order("created_at asc").Find(&skus).Error; err != nil {
		return nil, err
	}
	var images []ProductImage
	if err := db.Where("product_id = ?", id).Order("sort asc").Find(&images).Error; err != nil {
		return nil, err
	}
	var detail ProductDetail
	err := db.First(&detail, "product_id = ?", id).Error
	if err != nil && !sqlerr.IsNotFound(err) {
		return nil, err
	}
	var params []ProductParam
	if err := db.Where("product_id = ?", id).Order("name asc").Find(&params).Error; err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(images))
	for _, image := range images {
		urls = append(urls, image.URL)
	}
	out := &bizproduct.Product{
		ID:         po.ID,
		CategoryID: po.CategoryID,
		BrandID:    po.BrandID,
		Name:       po.Name,
		Subtitle:   po.Subtitle,
		Keywords:   po.Keywords,
		MainImage:  po.MainImage,
		Unit:       po.Unit,
		DetailHTML: detail.HTML,
		WeightGram: po.WeightGram,
		Status:     po.Status,
		MinPrice:   po.MinPrice,
		MaxPrice:   po.MaxPrice,
		Sales:      po.SalesCount,
		Images:     urls,
	}
	out.Skus = make([]bizproduct.Sku, 0, len(skus))
	for _, sku := range skus {
		out.Skus = append(out.Skus, bizproduct.Sku{
			ID:          sku.ID,
			ProductID:   sku.ProductID,
			SpecsJSON:   sku.Specs,
			SpecHash:    sku.SpecHash,
			Price:       sku.Price,
			MarketPrice: sku.MarketPrice,
			Image:       sku.Image,
			Barcode:     sku.Barcode,
			Enabled:     sku.Enabled,
		})
	}
	out.Params = make([]bizproduct.Param, 0, len(params))
	for _, param := range params {
		out.Params = append(out.Params, bizproduct.Param{Name: param.Name, Value: param.Value})
	}
	return out, nil
}

func (r *productRepo) FindByName(ctx context.Context, name string) (*bizproduct.Product, error) {
	var po Product
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&po).Error; err != nil {
		if sqlerr.IsNotFound(err) {
			return nil, bizproduct.ErrNotFound
		}
		return nil, err
	}
	return r.load(ctx, po.ID)
}

func (r *productRepo) SetStatus(ctx context.Context, id string, from []int32, to int32, trail *bizproduct.Trail) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Product{}).Where("id = ? AND status IN ?", id, from).Update("status", to)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			var n int64
			if err := tx.Model(&Product{}).Where("id = ?", id).Count(&n).Error; err != nil {
				return err
			}
			if n == 0 {
				return bizproduct.ErrNotFound
			}
			return bizproduct.ErrStatus
		}
		return writeTrail(tx, id, trail)
	})
	if err != nil {
		return err
	}
	r.dropCache(ctx, id)
	return nil
}

func (r *productRepo) SoftDelete(ctx context.Context, id string, trail *bizproduct.Trail) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ?", id).Delete(&Product{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return bizproduct.ErrNotFound
		}
		if err := tx.Where("product_id = ?", id).Delete(&Sku{}).Error; err != nil {
			return err
		}
		return writeTrail(tx, id, trail)
	})
	if err != nil {
		return err
	}
	r.dropCache(ctx, id)
	return nil
}

func (r *productRepo) Restore(ctx context.Context, id string, trail *bizproduct.Trail) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Unscoped().Model(&Product{}).Where("id = ? AND deleted_at IS NOT NULL", id).Updates(map[string]any{
			"deleted_at": gorm.Expr("NULL"),
			"status":     bizproduct.StatusOff,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			var n int64
			if err := tx.Unscoped().Model(&Product{}).Where("id = ?", id).Count(&n).Error; err != nil {
				return err
			}
			if n == 0 {
				return bizproduct.ErrNotFound
			}
			return bizproduct.ErrStatus
		}
		if err := tx.Unscoped().Model(&Sku{}).Where("product_id = ?", id).Update("deleted_at", gorm.Expr("NULL")).Error; err != nil {
			return err
		}
		return writeTrail(tx, id, trail)
	})
	if err != nil {
		return err
	}
	r.dropCache(ctx, id)
	return nil
}

func (r *productRepo) Purge(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Unscoped().Model(&Product{}).Where("id = ? AND deleted_at IS NOT NULL", id).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			var alive int64
			if err := tx.Unscoped().Model(&Product{}).Where("id = ?", id).Count(&alive).Error; err != nil {
				return err
			}
			if alive == 0 {
				return bizproduct.ErrNotFound
			}
			return bizproduct.ErrStatus
		}
		if err := tx.Unscoped().Where("product_id = ?", id).Delete(&Sku{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&ProductImage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&ProductDetail{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&ProductParam{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Where("id = ?", id).Delete(&Product{}).Error
	})
	if err != nil {
		return err
	}
	r.dropCache(ctx, id)
	return nil
}

func (r *productRepo) ChangePrice(ctx context.Context, productID, skuID string, price int64, trail *bizproduct.Trail) (*bizproduct.Product, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, "id = ?", productID).Error; err != nil {
			if sqlerr.IsNotFound(err) {
				return bizproduct.ErrNotFound
			}
			return err
		}
		var sku Sku
		if err := tx.Where("id = ? AND product_id = ?", skuID, productID).First(&sku).Error; err != nil {
			if sqlerr.IsNotFound(err) {
				return bizproduct.ErrNotFound
			}
			return err
		}
		before := sku.Price
		if err := tx.Model(&Sku{}).Where("id = ?", skuID).Update("price", price).Error; err != nil {
			return err
		}
		if err := refreshPrice(tx, productID); err != nil {
			return err
		}
		if trail != nil {
			trail.Before = strconv.FormatInt(before, 10)
			trail.After = strconv.FormatInt(price, 10)
		}
		return writeTrail(tx, productID, trail)
	})
	if err != nil {
		return nil, err
	}
	r.dropCache(ctx, productID)
	return r.load(ctx, productID)
}

func writeTrail(tx *gorm.DB, productID string, trail *bizproduct.Trail) error {
	if trail == nil || trail.Action == "" {
		return nil
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	op := trail.Operator
	if op == "" {
		op = "system"
	}
	if trail.Audit {
		return tx.Create(&ProductAudit{
			ID: id.String(), ProductID: productID, Action: trail.Action, Operator: op, Reason: trail.Reason,
		}).Error
	}
	return tx.Create(&ProductLog{
		ID: id.String(), ProductID: productID, Operator: op, Action: trail.Action, Before: trail.Before, After: trail.After,
	}).Error
}

func (r *productRepo) List(ctx context.Context, q bizproduct.ListFilter) ([]bizproduct.Card, bool, error) {
	db := r.db.WithContext(ctx).Model(&Product{})
	if q.Deleted {
		db = r.db.WithContext(ctx).Unscoped().Model(&Product{}).Where("deleted_at IS NOT NULL")
	} else if q.OnSaleOnly {
		db = db.Where("status = ?", bizproduct.StatusOnSale)
	}
	if len(q.CategoryIDs) > 0 {
		db = db.Where("category_id IN ?", q.CategoryIDs)
	}
	if q.BrandID != "" {
		db = db.Where("brand_id = ?", q.BrandID)
	}
	if q.MinPrice > 0 {
		db = db.Where("min_price >= ?", q.MinPrice)
	}
	if q.MaxPrice > 0 {
		db = db.Where("min_price <= ?", q.MaxPrice)
	}
	db, err := orderCards(db, q.OrderBy)
	if err != nil {
		return nil, false, err
	}
	return r.cards(ctx, db, q.PageSize, q.Offset)
}

// orderCards 把列表排序名译成 SQL 同价同销量时新的在前
func orderCards(db *gorm.DB, orderBy string) (*gorm.DB, error) {
	switch orderBy {
	case "", "latest":
		return db.Order("created_at desc"), nil
	case "price":
		return db.Order("min_price asc").Order("created_at desc"), nil
	case "price_desc":
		return db.Order("min_price desc").Order("created_at desc"), nil
	case "sales":
		return db.Order("sales_count desc").Order("created_at desc"), nil
	default:
		return nil, bizproduct.ErrInvalid
	}
}

func (r *productRepo) Search(ctx context.Context, keyword, orderBy string, size, offset int) ([]bizproduct.Card, bool, error) {
	like := likePattern(keyword)
	db := r.db.WithContext(ctx).Model(&Product{}).
		Where("status = ?", bizproduct.StatusOnSale).
		Where("name LIKE ? OR keywords LIKE ?", like, like)
	db, err := orderCards(db, orderBy)
	if err != nil {
		return nil, false, err
	}
	return r.cards(ctx, db, size, offset)
}

func (r *productRepo) cards(ctx context.Context, db *gorm.DB, size, offset int) ([]bizproduct.Card, bool, error) {
	if size <= 0 {
		size = 20
	}
	var rows []Product
	if err := db.Offset(offset).Limit(size + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > size
	if hasMore {
		rows = rows[:size]
	}
	ids := make([]string, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	markets, err := MinMarket(r.db, ctx, ids)
	if err != nil {
		return nil, false, err
	}
	out := make([]bizproduct.Card, 0, len(rows))
	for _, row := range rows {
		out = append(out, bizproduct.Card{
			ID:          row.ID,
			Name:        row.Name,
			MainImage:   row.MainImage,
			MinPrice:    row.MinPrice,
			MarketPrice: markets[row.ID],
			Sales:       row.SalesCount,
			Status:      row.Status,
		})
	}
	return out, hasMore, nil
}

func (r *productRepo) Skus(ctx context.Context, ids []string) ([]bizproduct.SkuView, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	type row struct {
		ID        string
		ProductID string
		Name      string
		Specs     string
		Price     int64
		Image     string
		MainImage string
		Enabled   bool
		Status    int32
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("skus").
		Select("skus.id, skus.product_id, products.name, skus.specs, skus.price, skus.image, products.main_image, skus.enabled, products.status").
		Joins("JOIN products ON products.id = skus.product_id AND products.deleted_at IS NULL").
		Where("skus.id IN ? AND skus.deleted_at IS NULL", ids).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]bizproduct.SkuView, 0, len(rows))
	for _, item := range rows {
		image := item.Image
		if image == "" {
			image = item.MainImage
		}
		out = append(out, bizproduct.SkuView{
			SkuID:     item.ID,
			ProductID: item.ProductID,
			Name:      item.Name,
			SpecsJSON: item.Specs,
			Price:     item.Price,
			Image:     image,
			Sellable:  item.Enabled && item.Status == bizproduct.StatusOnSale,
		})
	}
	return out, nil
}

func (r *productRepo) AddSales(ctx context.Context, eventID, orderID, productID string, n int64) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if eventID != "" {
			var seen int64
			if err := tx.Model(&ConsumedEvent{}).Where("id = ?", eventID).Count(&seen).Error; err != nil {
				return err
			}
			if seen > 0 {
				return nil
			}
		}
		if orderID != "" {
			var seen int64
			if err := tx.Model(&SalesApply{}).Where("order_id = ? AND product_id = ?", orderID, productID).Count(&seen).Error; err != nil {
				return err
			}
			if seen > 0 {
				return rememberEvent(tx, eventID)
			}
			if err := tx.Create(&SalesApply{OrderID: orderID, ProductID: productID}).Error; err != nil {
				if sqlerr.IsUnique(err) {
					return rememberEvent(tx, eventID)
				}
				return err
			}
		}
		res := tx.Model(&Product{}).Where("id = ?", productID).Update("sales_count", gorm.Expr("sales_count + ?", n))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return bizproduct.ErrNotFound
		}
		return rememberEvent(tx, eventID)
	})
	if err != nil {
		return err
	}
	r.dropCache(ctx, productID)
	return nil
}

func rememberEvent(tx *gorm.DB, eventID string) error {
	if eventID == "" {
		return nil
	}
	err := tx.Create(&ConsumedEvent{ID: eventID}).Error
	if sqlerr.IsUnique(err) {
		return nil
	}
	return err
}

func (r *productRepo) loadCache(ctx context.Context, id string) *bizproduct.Product {
	if r.rdb == nil {
		return nil
	}
	raw, err := r.rdb.Get(ctx, detailKey(id)).Bytes()
	if err != nil {
		return nil
	}
	var p bizproduct.Product
	if err := json.Unmarshal(raw, &p); err != nil || p.ID == "" {
		return nil
	}
	return &p
}

func (r *productRepo) storeCache(ctx context.Context, p *bizproduct.Product) {
	if r.rdb == nil || p == nil || p.ID == "" {
		return
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, detailKey(p.ID), raw, detailTTL).Err()
}

func (r *productRepo) dropCache(ctx context.Context, id string) {
	if r.rdb == nil || id == "" {
		return
	}
	if err := r.rdb.Del(ctx, detailKey(id)).Err(); err != nil {
		log.Error("drop product cache", "id", id, "err", err)
	}
}

func likePattern(keyword string) string {
	keyword = strings.ReplaceAll(keyword, `\`, `\\`)
	keyword = strings.ReplaceAll(keyword, `%`, `\%`)
	keyword = strings.ReplaceAll(keyword, `_`, `\_`)
	return "%" + keyword + "%"
}
