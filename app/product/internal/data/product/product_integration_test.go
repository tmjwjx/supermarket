//go:build integration

package product_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	bizproduct "github.com/tmjwjx/supermarket/app/product/internal/biz/product"
	dataproduct "github.com/tmjwjx/supermarket/app/product/internal/data/product"
	"github.com/tmjwjx/supermarket/pkg/kafkaout"
	"github.com/tmjwjx/supermarket/pkg/testdb"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// 整体更新要能覆盖已有的图文详情 并在同一事务里记下改价前后
func TestUpdateRewritesDetailAndLogsPrice(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t, "product_it_update",
		&dataproduct.Product{}, &dataproduct.Sku{}, &dataproduct.ProductImage{}, &dataproduct.ProductDetail{},
		&dataproduct.ProductParam{}, &dataproduct.ProductAudit{}, &dataproduct.ProductLog{}, &kafkaout.Row{})
	repo := dataproduct.NewProductRepo(db, nil)
	saved, err := repo.Save(ctx, &bizproduct.Product{
		CategoryID: "c-1", BrandID: "b-1", Name: "牛奶", Status: bizproduct.StatusDraft, DetailHTML: "<p>旧</p>",
		Images: []string{"https://img/1.jpg"},
		Skus:   []bizproduct.Sku{{SpecsJSON: `{"容量":"1L"}`, Price: 1000, Enabled: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	next := *saved
	next.DetailHTML = "<p>新</p>"
	next.Skus = []bizproduct.Sku{saved.Skus[0]}
	next.Skus[0].Price = 1200
	got, err := repo.Update(ctx, &next, func(context.Context, *bizproduct.Product) (bool, *bizproduct.Trail, error) {
		return true, &bizproduct.Trail{Operator: "admin", Action: "update", Before: `[1000]`, After: `[1200]`}, nil
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.DetailHTML != "<p>新</p>" || got.MinPrice != 1200 || len(got.Skus) != 1 || got.Skus[0].Price != 1200 {
		t.Fatalf("got %+v", got)
	}
	var log dataproduct.ProductLog
	if err := db.Where("product_id = ? AND action = ?", saved.ID, "update").First(&log).Error; err != nil {
		t.Fatal(err)
	}
	if log.Before != `[1000]` || log.After != `[1200]` {
		t.Fatalf("log %+v", log)
	}
}

// 20 个请求同时把已通过改成在售 条件更新只能成功一次且只留一条操作日志
func TestSetStatusConcurrentOnlyOnce(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t, "product_it_status", &dataproduct.Product{}, &dataproduct.ProductAudit{}, &dataproduct.ProductLog{})
	if err := db.Create(&dataproduct.Product{ID: "p-1", CategoryID: "c-1", BrandID: "b-1", Name: "牛奶", Status: bizproduct.StatusApproved}).Error; err != nil {
		t.Fatal(err)
	}
	repo := dataproduct.NewProductRepo(db, nil)
	var ok, conflict int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := repo.SetStatus(ctx, "p-1", []int32{bizproduct.StatusApproved}, bizproduct.StatusOnSale, &bizproduct.Trail{Operator: "admin", Action: "publish"})
			switch {
			case err == nil:
				atomic.AddInt32(&ok, 1)
			case errors.Is(err, bizproduct.ErrStatus):
				atomic.AddInt32(&conflict, 1)
			default:
				t.Errorf("set status: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	var logs int64
	if err := db.Model(&dataproduct.ProductLog{}).Where("product_id = ?", "p-1").Count(&logs).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("success=%d conflict=%d logs=%d", ok, conflict, logs)
	if ok != 1 || conflict != 19 || logs != 1 {
		t.Fatalf("success %d conflict %d logs %d", ok, conflict, logs)
	}
}

func openCatalog(t *testing.T, name string) *gorm.DB {
	return testdb.Open(t, name,
		&dataproduct.Product{}, &dataproduct.Sku{}, &dataproduct.ProductImage{}, &dataproduct.ProductDetail{},
		&dataproduct.ProductParam{}, &dataproduct.ProductAudit{}, &dataproduct.ProductLog{}, &kafkaout.Row{})
}

// 缓存里是旧的下架快照 库里已经在售 check 拿到的必须是锁住读到的库内状态
func TestUpdateCheckSeesLockedRowNotCache(t *testing.T) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unavailable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	db := openCatalog(t, "product_it_update_lock")
	repo := dataproduct.NewProductRepo(db, rdb)
	saved, err := repo.Save(ctx, &bizproduct.Product{
		CategoryID: "c-1", BrandID: "b-1", Name: "牛奶", Status: bizproduct.StatusOff,
		Skus: []bizproduct.Sku{{SpecsJSON: `{"容量":"1L"}`, Price: 1000, Enabled: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rdb.Del(context.Background(), "product:detail:"+saved.ID).Err() })
	if cached, err := repo.Get(ctx, saved.ID); err != nil || cached.Status != bizproduct.StatusOff {
		t.Fatalf("cached %+v %v", cached, err)
	}
	if err := db.Model(&dataproduct.Product{}).Where("id = ?", saved.ID).Update("status", bizproduct.StatusOnSale).Error; err != nil {
		t.Fatal(err)
	}
	var seen int32
	next := *saved
	_, err = repo.Update(ctx, &next, func(_ context.Context, current *bizproduct.Product) (bool, *bizproduct.Trail, error) {
		seen = current.Status
		return false, nil, bizproduct.ErrStatus
	})
	if !errors.Is(err, bizproduct.ErrStatus) {
		t.Fatalf("got %v", err)
	}
	if seen != bizproduct.StatusOnSale {
		t.Fatalf("check saw status %d", seen)
	}
}

// 调用方给的最低最高价不可信 改完 SKU 后按库里启用的规格重新聚合
func TestUpdateRecomputesPriceFromSkus(t *testing.T) {
	ctx := context.Background()
	db := openCatalog(t, "product_it_update_price")
	repo := dataproduct.NewProductRepo(db, nil)
	saved, err := repo.Save(ctx, &bizproduct.Product{
		CategoryID: "c-1", BrandID: "b-1", Name: "牛奶", Status: bizproduct.StatusDraft,
		Skus: []bizproduct.Sku{
			{SpecsJSON: `{"容量":"1L"}`, Price: 1000, Enabled: true},
			{SpecsJSON: `{"容量":"2L"}`, Price: 1800, Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	next := *saved
	next.Skus = []bizproduct.Sku{saved.Skus[0], saved.Skus[1], {SpecsJSON: `{"容量":"3L"}`, SpecHash: "h3", Price: 100, Enabled: false}}
	next.Skus[0].Price = 1500
	next.Skus[1].Price = 2500
	next.MinPrice, next.MaxPrice = 1, 1
	got, err := repo.Update(ctx, &next, func(context.Context, *bizproduct.Product) (bool, *bizproduct.Trail, error) {
		return true, nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.MinPrice != 1500 || got.MaxPrice != 2500 {
		t.Fatalf("price %d %d", got.MinPrice, got.MaxPrice)
	}
}

// 并发改同一商品不同 SKU 的价格 锁住商品行后聚合出的价格区间必须和最终 SKU 一致
func TestChangePriceConcurrentKeepsRange(t *testing.T) {
	ctx := context.Background()
	db := openCatalog(t, "product_it_change_price")
	repo := dataproduct.NewProductRepo(db, nil)
	skus := make([]bizproduct.Sku, 0, 10)
	for i := 0; i < 10; i++ {
		skus = append(skus, bizproduct.Sku{SpecsJSON: fmt.Sprintf(`{"编号":"%d"}`, i), Price: 1000, Enabled: true})
	}
	saved, err := repo.Save(ctx, &bizproduct.Product{CategoryID: "c-1", BrandID: "b-1", Name: "牛奶", Status: bizproduct.StatusOnSale, Skus: skus})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i, sku := range saved.Skus {
		wg.Add(1)
		go func(price int64, id string) {
			defer wg.Done()
			<-start
			if _, err := repo.ChangePrice(ctx, saved.ID, id, price, &bizproduct.Trail{Operator: "admin", Action: "change_price"}); err != nil {
				t.Errorf("change price: %v", err)
			}
		}(int64(2000+i*100), sku.ID)
	}
	close(start)
	wg.Wait()
	var po dataproduct.Product
	if err := db.First(&po, "id = ?", saved.ID).Error; err != nil {
		t.Fatal(err)
	}
	if po.MinPrice != 2000 || po.MaxPrice != 2900 {
		t.Fatalf("range %d %d", po.MinPrice, po.MaxPrice)
	}
}
