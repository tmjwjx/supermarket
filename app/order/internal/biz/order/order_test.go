package order

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeOrders struct {
	OrderRepo
	saved   map[string]*Order
	creates int
}

func (f *fakeOrders) FindByUserRequest(_ context.Context, userID uuid.UUID, requestID string) (*Order, error) {
	if o, ok := f.saved[userID.String()+"/"+requestID]; ok {
		return o, nil
	}
	return nil, ErrOrderNotFound
}

func (f *fakeOrders) Create(_ context.Context, in *Order) (*Order, error) {
	key := in.UserID.String() + "/" + in.RequestID
	if _, ok := f.saved[key]; ok {
		return nil, ErrDuplicateRequest
	}
	cp := *in
	cp.OrderNo = "no"
	items := make([]*OrderItem, len(in.Items))
	copy(items, in.Items)
	cp.Items = items
	f.saved[key] = &cp
	f.creates++
	return &cp, nil
}

type fakeCatalog struct {
	snaps []SkuSnap
	calls int
	sales int
}

func (f *fakeCatalog) BatchGetSkus(context.Context, []string) ([]SkuSnap, error) {
	f.calls++
	return f.snaps, nil
}

func (f *fakeCatalog) IncreaseSales(context.Context, string, string, int64) error {
	f.sales++
	return nil
}

type fakeStock struct {
	reserve int
	release int
}

func (f *fakeStock) Reserve(context.Context, string, []StockLine, int64) error {
	f.reserve++
	return nil
}

func (f *fakeStock) Confirm(context.Context, string) error { return nil }

func (f *fakeStock) Release(context.Context, string) error {
	f.release++
	return nil
}

type fakeAddress struct{ calls int }

func (f *fakeAddress) Get(context.Context, string, string) (*AddressSnap, error) {
	f.calls++
	return &AddressSnap{Receiver: "Ada", Phone: "13800000000", Province: "Zhejiang", City: "Hangzhou", District: "Xihu", Detail: "1"}, nil
}

type fakeCart struct{ calls int }

func (f *fakeCart) RemoveSKUs(context.Context, uuid.UUID, []string) error {
	f.calls++
	return nil
}

func TestCreateSameRequestIDReturnsSameOrder(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	repo := &fakeOrders{saved: map[string]*Order{}}
	catalog := &fakeCatalog{snaps: []SkuSnap{{
		SkuID: "sku-1", ProductID: "p1", ProductName: "milk", Price: 250, Sellable: true,
	}}}
	stock := &fakeStock{}
	uc := NewOrderUsecase(repo, catalog, stock, &fakeAddress{}, &fakeCart{})
	in := CreateInput{Lines: []Line{{SkuID: "sku-1", Quantity: 2}}, AddressID: "addr-1", RequestID: "req-1"}
	first, err := uc.Create(context.Background(), userID, in)
	if err != nil {
		t.Fatal(err)
	}
	if first.PayAmount != 500 || first.ItemsAmount != 500 {
		t.Fatalf("amount %d %d", first.ItemsAmount, first.PayAmount)
	}
	second, err := uc.Create(context.Background(), userID, in)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("ids %s %s", first.ID, second.ID)
	}
	if repo.creates != 1 || catalog.calls != 1 || stock.reserve != 1 {
		t.Fatalf("creates=%d catalog=%d reserve=%d", repo.creates, catalog.calls, stock.reserve)
	}
}

func TestCreateNotSellableDoesNotWriteOrder(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	repo := &fakeOrders{saved: map[string]*Order{}}
	catalog := &fakeCatalog{snaps: []SkuSnap{{SkuID: "sku-1", Sellable: false}}}
	stock := &fakeStock{}
	addr := &fakeAddress{}
	uc := NewOrderUsecase(repo, catalog, stock, addr, &fakeCart{})
	_, err := uc.Create(context.Background(), userID, CreateInput{
		Lines: []Line{{SkuID: "sku-1", Quantity: 1}}, AddressID: "addr-1", RequestID: "req-2",
	})
	if !errors.Is(err, ErrOrderItemNotSellable) {
		t.Fatalf("got %v", err)
	}
	if repo.creates != 0 || stock.reserve != 0 || addr.calls != 0 {
		t.Fatalf("creates=%d reserve=%d address=%d", repo.creates, stock.reserve, addr.calls)
	}
}
