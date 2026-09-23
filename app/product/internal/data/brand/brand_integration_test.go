//go:build integration

package brand_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	bizbrand "github.com/tmjwjx/supermarket/app/product/internal/biz/brand"
	databrand "github.com/tmjwjx/supermarket/app/product/internal/data/brand"
	"github.com/tmjwjx/supermarket/pkg/testdb"
)

// 唯一索引冲突要翻译成品牌名已存在 并发同名创建也只留一条
func TestUniqueConflictTranslated(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t, "product_it_brand", &databrand.Brand{})
	repo := databrand.NewBrandRepo(db)
	if _, err := repo.Save(ctx, &bizbrand.Brand{Name: "蒙牛", Visible: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Save(ctx, &bizbrand.Brand{Name: "蒙牛"}); !errors.Is(err, bizbrand.ErrNameExists) {
		t.Fatalf("duplicate save got %v", err)
	}
	other, err := repo.Save(ctx, &bizbrand.Brand{Name: "伊利"})
	if err != nil {
		t.Fatal(err)
	}
	other.Name = "蒙牛"
	if _, err := repo.Update(ctx, other); !errors.Is(err, bizbrand.ErrNameExists) {
		t.Fatalf("duplicate update got %v", err)
	}

	var ok, dup int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.Save(ctx, &bizbrand.Brand{Name: "光明"})
			switch {
			case err == nil:
				atomic.AddInt32(&ok, 1)
			case errors.Is(err, bizbrand.ErrNameExists):
				atomic.AddInt32(&dup, 1)
			default:
				t.Errorf("save: %v", err)
			}
		}()
	}
	wg.Wait()
	t.Logf("concurrent same name success=%d name_exists=%d", ok, dup)
	if ok != 1 || dup != 9 {
		t.Fatalf("success %d exists %d", ok, dup)
	}
}
