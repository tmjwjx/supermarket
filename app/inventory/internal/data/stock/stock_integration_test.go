//go:build integration

package stock_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bizstock "github.com/tmjwjx/supermarket/app/inventory/internal/biz/stock"
	datastock "github.com/tmjwjx/supermarket/app/inventory/internal/data/stock"
	"github.com/tmjwjx/supermarket/pkg/testdb"

	"github.com/go-kratos/kratos/v3/errors"
)

func newRepo(t *testing.T) bizstock.StockRepo {
	db := testdb.Open(t, "inventory_it", &datastock.Stock{}, &datastock.StockReservation{}, &datastock.StockLog{})
	return datastock.NewStockRepo(db)
}

// 50 个订单同时各抢 1 件 库存 10 件 只能成功 10 个且不超卖
func TestReserveConcurrentNoOversell(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)
	if _, err := repo.Create(ctx, "sku-hot"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Adjust(ctx, "sku-hot", 10, bizstock.ReasonInbound); err != nil {
		t.Fatal(err)
	}
	var ok, short int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			err := repo.Reserve(ctx, fmt.Sprintf("order-%02d", i), []bizstock.Item{{SkuID: "sku-hot", Quantity: 1}}, time.Now().Add(time.Hour))
			switch {
			case err == nil:
				atomic.AddInt32(&ok, 1)
			case errors.Reason(err) == bizstock.ErrStockInsufficient.Reason:
				atomic.AddInt32(&short, 1)
			default:
				t.Errorf("order-%02d: %v", i, err)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	rows, err := repo.Get(ctx, []string{"sku-hot"})
	if err != nil || len(rows) != 1 {
		t.Fatalf("get %v %v", rows, err)
	}
	got := rows[0]
	t.Logf("success=%d insufficient=%d on_hand=%d reserved=%d available=%d", ok, short, got.OnHand, got.Reserved, got.Available)
	if ok != 10 || short != 40 {
		t.Fatalf("success %d insufficient %d", ok, short)
	}
	if got.OnHand != 10 || got.Reserved != 10 || got.Available != 0 {
		t.Fatalf("stock %+v", got)
	}
}

// 同一订单并发确认 实际数量只扣一次
func TestConfirmConcurrentAppliesOnce(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)
	if _, err := repo.Create(ctx, "sku-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Adjust(ctx, "sku-a", 5, bizstock.ReasonInbound); err != nil {
		t.Fatal(err)
	}
	if err := repo.Reserve(ctx, "order-x", []bizstock.Item{{SkuID: "sku-a", Quantity: 2}}, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.Confirm(ctx, "order-x")
			if err != nil && errors.Reason(err) != bizstock.ErrReservationStatusConflict.Reason {
				t.Errorf("confirm: %v", err)
			}
		}()
	}
	wg.Wait()
	rows, err := repo.Get(ctx, []string{"sku-a"})
	if err != nil || len(rows) != 1 {
		t.Fatalf("get %v %v", rows, err)
	}
	t.Logf("on_hand=%d reserved=%d", rows[0].OnHand, rows[0].Reserved)
	if rows[0].OnHand != 3 || rows[0].Reserved != 0 {
		t.Fatalf("stock %+v", rows[0])
	}
}
