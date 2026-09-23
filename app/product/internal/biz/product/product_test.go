package product

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tmjwjx/supermarket/app/product/internal/biz/attribute"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/brand"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/category"
)

func TestCreateProductDuplicateSpec(t *testing.T) {
	uc := NewProductUsecase(newFakeProductRepo(), fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	_, err := uc.Create(context.Background(), &Product{
		Name:       "牛奶",
		CategoryID: "c2",
		BrandID:    "b1",
		Images:     []string{"https://img.local/a.jpg"},
		Skus: []Sku{
			{SpecsJSON: `{"容量":"500ml","包装":"瓶"}`, Price: 1999, Enabled: true},
			{SpecsJSON: `{"包装":"瓶","容量":"500ml"}`, Price: 2999, Enabled: true},
		},
	})
	if !errors.Is(err, ErrSpec) {
		t.Fatalf("got %v", err)
	}
}

func TestPublishMissingSKU(t *testing.T) {
	repo := newFakeProductRepo()
	repo.products["p1"] = &Product{
		ID:         "p1",
		CategoryID: "c2",
		BrandID:    "b1",
		Name:       "牛奶",
		Status:     StatusApproved,
		Images:     []string{"https://img.local/a.jpg"},
	}
	uc := NewProductUsecase(repo, fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	_, err := uc.Publish(context.Background(), "p1", "admin")
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "sku") {
		t.Fatalf("message %v", err)
	}
}

func leaf() *category.Category {
	return &category.Category{ID: "c2", ParentID: "c1", Name: "乳品", Visible: true}
}

// fakeProductRepo 的 cached 模拟 Redis 里的旧快照 只有 Get 会读它
type fakeProductRepo struct {
	products    map[string]*Product
	cached      map[string]*Product
	gone        map[string]bool
	lastTrail   *Trail
	lastOrder   string
	lastReplace bool
}

func newFakeProductRepo() *fakeProductRepo {
	return &fakeProductRepo{products: map[string]*Product{}, cached: map[string]*Product{}}
}

func (r *fakeProductRepo) GetFresh(_ context.Context, id string) (*Product, error) {
	p, ok := r.products[id]
	if !ok || r.gone[id] {
		return nil, ErrNotFound
	}
	cp := *p
	cp.Skus = append([]Sku(nil), p.Skus...)
	cp.Images = append([]string(nil), p.Images...)
	return &cp, nil
}

func (r *fakeProductRepo) Save(_ context.Context, p *Product) (*Product, error) {
	cp := *p
	cp.ID = "p-new"
	cp.Skus = append([]Sku(nil), p.Skus...)
	cp.Images = append([]string(nil), p.Images...)
	r.products[cp.ID] = &cp
	return &cp, nil
}

func (r *fakeProductRepo) Get(_ context.Context, id string) (*Product, error) {
	if c, ok := r.cached[id]; ok {
		cp := *c
		return &cp, nil
	}
	p, ok := r.products[id]
	if !ok || r.gone[id] {
		return nil, ErrNotFound
	}
	cp := *p
	cp.Skus = append([]Sku(nil), p.Skus...)
	cp.Images = append([]string(nil), p.Images...)
	return &cp, nil
}

func (r *fakeProductRepo) GetAny(_ context.Context, id string) (*Product, error) {
	p, ok := r.products[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *fakeProductRepo) FindByName(context.Context, string) (*Product, error) {
	return nil, ErrNotFound
}

func (r *fakeProductRepo) SetStatus(_ context.Context, id string, from []int32, to int32, _ *Trail) error {
	p, ok := r.products[id]
	if !ok || r.gone[id] {
		return ErrNotFound
	}
	for _, status := range from {
		if p.Status == status {
			p.Status = to
			return nil
		}
	}
	return ErrStatus
}

func (r *fakeProductRepo) SoftDelete(_ context.Context, id string, _ *Trail) error {
	if _, ok := r.products[id]; !ok || r.gone[id] {
		return ErrNotFound
	}
	if r.gone == nil {
		r.gone = map[string]bool{}
	}
	r.gone[id] = true
	return nil
}

func (r *fakeProductRepo) Restore(_ context.Context, id string, _ *Trail) error {
	if _, ok := r.products[id]; !ok {
		return ErrNotFound
	}
	if !r.gone[id] {
		return ErrStatus
	}
	r.gone[id] = false
	r.products[id].Status = StatusOff
	return nil
}

func (r *fakeProductRepo) Purge(_ context.Context, id string) error {
	if _, ok := r.products[id]; !ok {
		return ErrNotFound
	}
	if !r.gone[id] {
		return ErrStatus
	}
	delete(r.products, id)
	return nil
}

func (r *fakeProductRepo) ChangePrice(context.Context, string, string, int64, *Trail) (*Product, error) {
	return nil, nil
}

func (r *fakeProductRepo) List(context.Context, ListFilter) ([]Card, bool, error) {
	return nil, false, nil
}

func (r *fakeProductRepo) Search(_ context.Context, _, orderBy string, _, _ int) ([]Card, bool, error) {
	r.lastOrder = orderBy
	return nil, false, nil
}

func (r *fakeProductRepo) Skus(context.Context, []string) ([]SkuView, error) { return nil, nil }

func (r *fakeProductRepo) AddSales(context.Context, string, string, string, int64) error { return nil }

func (r *fakeProductRepo) Update(ctx context.Context, p *Product, check UpdateCheck) (*Product, error) {
	current, err := r.GetFresh(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	replace, trail, err := check(ctx, current)
	if err != nil {
		return nil, err
	}
	r.lastTrail = trail
	r.lastReplace = replace
	cp := *p
	cp.Status = current.Status
	r.products[p.ID] = &cp
	return &cp, nil
}

func (r *fakeProductRepo) ListAudits(context.Context, string, int, int) ([]Audit, bool, error) {
	return nil, false, nil
}

func (r *fakeProductRepo) ListLogs(context.Context, string, int, int) ([]OpLog, bool, error) {
	return nil, false, nil
}

type fakeBrand struct {
	item *brand.Brand
}

func (f fakeBrand) Find(context.Context, string) (*brand.Brand, error) {
	if f.item == nil {
		return nil, brand.ErrNotFound
	}
	return f.item, nil
}

type fakeCategory struct {
	item *category.Category
}

func (f fakeCategory) Find(context.Context, string) (*category.Category, error) {
	if f.item == nil {
		return nil, category.ErrNotFound
	}
	return f.item, nil
}

func (f fakeCategory) ChildIDs(context.Context, string) ([]string, error) { return nil, nil }

func TestCreateRejectsSpecOutsideTemplate(t *testing.T) {
	cat := leaf()
	cat.TemplateID = "t1"
	uc := NewProductUsecase(newFakeProductRepo(), fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: cat}, nil, fakeTemplates{})
	_, err := uc.Create(context.Background(), &Product{
		Name: "牛奶", CategoryID: "c2", BrandID: "b1",
		Skus: []Sku{{SpecsJSON: `{"口味":"原味"}`, Price: 100, Enabled: true}},
	})
	if err == nil || !strings.Contains(err.Error(), "口味") {
		t.Fatalf("got %v", err)
	}
}

func TestSubmitThenPublish(t *testing.T) {
	repo := newFakeProductRepo()
	repo.products["p1"] = &Product{
		ID: "p1", CategoryID: "c2", BrandID: "b1", Name: "牛奶", Status: StatusDraft,
		Images: []string{"https://img.local/a.jpg"},
		Skus:   []Sku{{Enabled: true, Price: 100}},
	}
	uc := NewProductUsecase(repo, fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	if _, err := uc.Submit(context.Background(), "p1", "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Approve(context.Background(), "p1", "admin"); err != nil {
		t.Fatal(err)
	}
	got, err := uc.Publish(context.Background(), "p1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusOnSale {
		t.Fatalf("status %d", got.Status)
	}
}

func TestPublishRejectsZeroPrice(t *testing.T) {
	repo := newFakeProductRepo()
	repo.products["p1"] = &Product{
		ID: "p1", CategoryID: "c2", BrandID: "b1", Name: "牛奶", Status: StatusApproved,
		Images: []string{"https://img.local/a.jpg"},
		Skus:   []Sku{{ID: "s1", Enabled: true, Price: 100}, {ID: "s0", Enabled: true, Price: 0}},
	}
	uc := NewProductUsecase(repo, fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	_, err := uc.Publish(context.Background(), "p1", "admin")
	if err == nil || !strings.Contains(err.Error(), "greater than 0") {
		t.Fatalf("got %v", err)
	}
}

func TestPublishRejectsStockCreateFailure(t *testing.T) {
	repo := newFakeProductRepo()
	repo.products["p1"] = &Product{
		ID: "p1", CategoryID: "c2", BrandID: "b1", Name: "牛奶", Status: StatusApproved,
		Images: []string{"https://img.local/a.jpg"},
		Skus:   []Sku{{ID: "s1", Enabled: true, Price: 100}},
	}
	uc := NewProductUsecase(repo, fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	uc.UseStocks(failStock{})
	_, err := uc.Publish(context.Background(), "p1", "admin")
	if err == nil || !strings.Contains(err.Error(), "stock create failed") {
		t.Fatalf("got %v", err)
	}
}

type failStock struct{}

func (failStock) Ensure(context.Context, string) error { return errors.New("down") }

func TestRestoreGoesOffShelf(t *testing.T) {
	repo := newFakeProductRepo()
	repo.products["p1"] = &Product{ID: "p1", Status: StatusOnSale}
	uc := NewProductUsecase(repo, fakeBrand{}, fakeCategory{item: leaf()}, nil, nil)
	if err := uc.Delete(context.Background(), "p1", "admin"); err != nil {
		t.Fatal(err)
	}
	got, err := uc.Restore(context.Background(), "p1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusOff {
		t.Fatalf("status %d", got.Status)
	}
}

type fakeTemplates struct{}

func (fakeTemplates) List(context.Context, string) ([]attribute.Attribute, error) {
	return []attribute.Attribute{{Name: "容量", Kind: attribute.KindSpec, Options: []string{"500ml"}}}, nil
}

func TestUpdateLogsSkuPriceChange(t *testing.T) {
	_, hashA, _ := CanonicalSpecs(`{"容量":"500ml"}`)
	_, hashB, _ := CanonicalSpecs(`{"容量":"1L"}`)
	repo := newFakeProductRepo()
	repo.products["p1"] = &Product{
		ID: "p1", CategoryID: "c2", BrandID: "b1", Name: "牛奶", Status: StatusOnSale,
		Skus: []Sku{
			{ID: "s1", SpecsJSON: `{"容量":"500ml"}`, SpecHash: hashA, Price: 100, Enabled: true},
			{ID: "s2", SpecsJSON: `{"容量":"1L"}`, SpecHash: hashB, Price: 200, Enabled: true},
		},
	}
	uc := NewProductUsecase(repo, fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	_, err := uc.Update(context.Background(), &Product{
		ID: "p1", Name: "牛奶",
		Skus: []Sku{
			{ID: "s1", SpecsJSON: `{"容量":"500ml"}`, Price: 150, Enabled: true},
			{ID: "s2", SpecsJSON: `{"容量":"1L"}`, Price: 200, Enabled: true},
		},
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	trail := repo.lastTrail
	if trail == nil || trail.Action != "update" {
		t.Fatalf("trail %+v", trail)
	}
	if !strings.Contains(trail.Before, `"sku_id":"s1"`) || !strings.Contains(trail.Before, `"price":100`) {
		t.Fatalf("before %s", trail.Before)
	}
	if !strings.Contains(trail.After, `"price":150`) || strings.Contains(trail.After, "s2") {
		t.Fatalf("after %s", trail.After)
	}
}

func TestUpdateWithoutPriceChangeLeavesTrailEmpty(t *testing.T) {
	_, hash, _ := CanonicalSpecs(`{"容量":"500ml"}`)
	repo := newFakeProductRepo()
	repo.products["p1"] = &Product{
		ID: "p1", CategoryID: "c2", BrandID: "b1", Name: "牛奶", Status: StatusOnSale,
		Skus: []Sku{{ID: "s1", SpecsJSON: `{"容量":"500ml"}`, SpecHash: hash, Price: 100, Enabled: true}},
	}
	uc := NewProductUsecase(repo, fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	if _, err := uc.Update(context.Background(), &Product{
		ID: "p1", Name: "鲜牛奶",
		Skus: []Sku{{ID: "s1", SpecsJSON: `{"容量":"500ml"}`, Price: 100, Enabled: true}},
	}, "admin"); err != nil {
		t.Fatal(err)
	}
	if repo.lastTrail.Before != "" || repo.lastTrail.After != "" {
		t.Fatalf("trail %+v", repo.lastTrail)
	}
}

func TestSearchOrderBy(t *testing.T) {
	repo := newFakeProductRepo()
	uc := NewProductUsecase(repo, fakeBrand{}, fakeCategory{}, nil, nil)
	for in, want := range map[string]string{"": "latest", "latest": "latest", "price": "price", "price_desc": "price_desc", "sales": "sales"} {
		if _, _, err := uc.Search(context.Background(), "奶", in, "", 10); err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if repo.lastOrder != want {
			t.Fatalf("%q 传给仓库 %q", in, repo.lastOrder)
		}
	}
	if _, _, err := uc.Search(context.Background(), "奶", "hot", "", 10); !errors.Is(err, ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

// 商品已经落库 补库存失败不能再把请求报成失败
func TestCreateSucceedsWhenStockEnsureFails(t *testing.T) {
	uc := NewProductUsecase(newFakeProductRepo(), fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	uc.UseStocks(failStock{})
	got, err := uc.Create(context.Background(), &Product{
		Name: "牛奶", CategoryID: "c2", BrandID: "b1",
		Skus: []Sku{{ID: "s1", SpecsJSON: `{"容量":"500ml"}`, Price: 100, Enabled: true}},
	})
	if err != nil {
		t.Fatalf("got %v", err)
	}
	if got == nil || got.ID == "" {
		t.Fatalf("product %+v", got)
	}
}

func TestUpdateSucceedsWhenStockEnsureFails(t *testing.T) {
	_, hash, _ := CanonicalSpecs(`{"容量":"500ml"}`)
	repo := newFakeProductRepo()
	repo.products["p1"] = &Product{
		ID: "p1", CategoryID: "c2", BrandID: "b1", Name: "牛奶", Status: StatusOnSale,
		Skus: []Sku{{ID: "s1", SpecsJSON: `{"容量":"500ml"}`, SpecHash: hash, Price: 100, Enabled: true}},
	}
	uc := NewProductUsecase(repo, fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	uc.UseStocks(failStock{})
	got, err := uc.Update(context.Background(), &Product{
		ID: "p1", Name: "鲜牛奶",
		Skus: []Sku{{ID: "s1", SpecsJSON: `{"容量":"500ml"}`, Price: 120, Enabled: true}},
	}, "admin")
	if err != nil {
		t.Fatalf("got %v", err)
	}
	if got.Name != "鲜牛奶" {
		t.Fatalf("product %+v", got)
	}
}

// 缓存说已下架 库里其实在售 改规格结构必须被拒
func TestUpdateJudgesOnSaleByLockedRowNotCache(t *testing.T) {
	_, hash, _ := CanonicalSpecs(`{"容量":"500ml"}`)
	row := &Product{
		ID: "p1", CategoryID: "c2", BrandID: "b1", Name: "牛奶", Status: StatusOnSale,
		Skus: []Sku{{ID: "s1", SpecsJSON: `{"容量":"500ml"}`, SpecHash: hash, Price: 100, Enabled: true}},
	}
	stale := *row
	stale.Status = StatusOff
	repo := newFakeProductRepo()
	repo.products["p1"] = row
	repo.cached["p1"] = &stale
	uc := NewProductUsecase(repo, fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	_, err := uc.Update(context.Background(), &Product{
		ID: "p1", Name: "牛奶", CategoryID: "c2", BrandID: "b1",
		Skus: []Sku{
			{ID: "s1", SpecsJSON: `{"容量":"500ml"}`, Price: 100, Enabled: true},
			{SpecsJSON: `{"容量":"1L"}`, Price: 200, Enabled: true},
		},
	}, "admin")
	if err == nil || !strings.Contains(err.Error(), "sku structure") {
		t.Fatalf("got %v", err)
	}
}

// 缓存说在售 库里已下架 改规格结构应当放行并替换规格
func TestUpdateAllowsSpecChangeWhenLockedRowIsOff(t *testing.T) {
	_, hash, _ := CanonicalSpecs(`{"容量":"500ml"}`)
	row := &Product{
		ID: "p1", CategoryID: "c2", BrandID: "b1", Name: "牛奶", Status: StatusOff,
		Skus: []Sku{{ID: "s1", SpecsJSON: `{"容量":"500ml"}`, SpecHash: hash, Price: 100, Enabled: true}},
	}
	stale := *row
	stale.Status = StatusOnSale
	repo := newFakeProductRepo()
	repo.products["p1"] = row
	repo.cached["p1"] = &stale
	uc := NewProductUsecase(repo, fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	if _, err := uc.Update(context.Background(), &Product{
		ID: "p1", Name: "牛奶", CategoryID: "c2", BrandID: "b1",
		Skus: []Sku{
			{ID: "s1", SpecsJSON: `{"容量":"500ml"}`, Price: 100, Enabled: true},
			{SpecsJSON: `{"容量":"1L"}`, Price: 200, Enabled: true},
		},
	}, "admin"); err != nil {
		t.Fatal(err)
	}
	if !repo.lastReplace {
		t.Fatal("spec structure should be replaced")
	}
}

// 缓存里有图 库里没图 上架检查要按库里的数据拒绝
func TestPublishReadyIgnoresCache(t *testing.T) {
	row := &Product{
		ID: "p1", CategoryID: "c2", BrandID: "b1", Name: "牛奶", Status: StatusApproved,
		Skus: []Sku{{ID: "s1", Enabled: true, Price: 100}},
	}
	stale := *row
	stale.Images = []string{"https://img.local/a.jpg"}
	repo := newFakeProductRepo()
	repo.products["p1"] = row
	repo.cached["p1"] = &stale
	uc := NewProductUsecase(repo, fakeBrand{item: &brand.Brand{ID: "b1", Visible: true}}, fakeCategory{item: leaf()}, nil, nil)
	_, err := uc.Publish(context.Background(), "p1", "admin")
	if err == nil || !strings.Contains(err.Error(), "image") {
		t.Fatalf("got %v", err)
	}
}

// 买家只能看到在售和下架 其它状态一律当作不存在
func TestBuyerGetOnlyOnSaleOrOff(t *testing.T) {
	repo := newFakeProductRepo()
	uc := NewProductUsecase(repo, fakeBrand{}, fakeCategory{item: leaf()}, nil, nil)
	for _, status := range []int32{StatusDraft, StatusPending, StatusRejected, StatusApproved, StatusOnSale, StatusOff} {
		repo.products["p1"] = &Product{ID: "p1", Name: "牛奶", Status: status, DetailHTML: "<p>x</p>"}
		_, getErr := uc.Get(context.Background(), "p1", "")
		_, detailErr := uc.Detail(context.Background(), "p1")
		visible := status == StatusOnSale || status == StatusOff
		if visible && (getErr != nil || detailErr != nil) {
			t.Fatalf("status %d get %v detail %v", status, getErr, detailErr)
		}
		if !visible && (!errors.Is(getErr, ErrNotFound) || !errors.Is(detailErr, ErrNotFound)) {
			t.Fatalf("status %d get %v detail %v", status, getErr, detailErr)
		}
	}
	if _, err := uc.AdminGet(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
}
