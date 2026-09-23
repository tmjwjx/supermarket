package stock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// memRepo 用互斥锁模拟条件更新 并发预占不会卖超
type memRepo struct {
	mu    sync.Mutex
	stock map[string]*Stock
	hold  map[string][]memHold
}

type memHold struct {
	skuID    string
	quantity int64
	status   int32
	expires  time.Time
}

func newMemRepo() *memRepo {
	return &memRepo{stock: map[string]*Stock{}, hold: map[string][]memHold{}}
}

func (m *memRepo) Create(_ context.Context, skuID string) (*Stock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if row, ok := m.stock[skuID]; ok {
		cp := *row
		return &cp, nil
	}
	row := &Stock{SkuID: skuID}
	m.stock[skuID] = row
	cp := *row
	return &cp, nil
}

func (m *memRepo) Adjust(_ context.Context, skuID string, delta int64, _ string) (*Stock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.stock[skuID]
	if !ok {
		return nil, ErrStockNotFound
	}
	if row.OnHand+delta-row.Reserved < 0 {
		return nil, ErrStockInsufficient
	}
	row.OnHand += delta
	row.Version++
	row.Available = row.OnHand - row.Reserved
	cp := *row
	return &cp, nil
}

func (m *memRepo) Get(_ context.Context, skuIDs []string) ([]*Stock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Stock, 0, len(skuIDs))
	for _, id := range skuIDs {
		row, ok := m.stock[id]
		if !ok {
			continue
		}
		cp := *row
		cp.Available = cp.OnHand - cp.Reserved
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memRepo) Reserve(_ context.Context, orderID string, items []Item, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.hold[orderID]) > 0 {
		return nil
	}
	var bad []string
	for _, it := range items {
		row := m.stock[it.SkuID]
		if row == nil || row.OnHand-row.Reserved < it.Quantity {
			bad = append(bad, it.SkuID)
		}
	}
	if len(bad) > 0 {
		return Insufficient(bad)
	}
	for _, it := range items {
		row := m.stock[it.SkuID]
		row.Reserved += it.Quantity
		row.Version++
		row.Available = row.OnHand - row.Reserved
		m.hold[orderID] = append(m.hold[orderID], memHold{
			skuID: it.SkuID, quantity: it.Quantity, status: ReservationHeld, expires: expiresAt,
		})
	}
	return nil
}

func (m *memRepo) Confirm(_ context.Context, orderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := m.hold[orderID]
	if len(rows) == 0 {
		return ErrReservationNotFound
	}
	for _, row := range rows {
		if row.status == ReservationReleased {
			return ErrReservationStatusConflict
		}
	}
	for i := range rows {
		if rows[i].status != ReservationHeld {
			continue
		}
		st := m.stock[rows[i].skuID]
		if st == nil || st.OnHand < rows[i].quantity || st.Reserved < rows[i].quantity {
			return ErrStockInsufficient
		}
		st.OnHand -= rows[i].quantity
		st.Reserved -= rows[i].quantity
		st.Version++
		st.Available = st.OnHand - st.Reserved
		rows[i].status = ReservationConfirmed
	}
	m.hold[orderID] = rows
	return nil
}

func (m *memRepo) Release(_ context.Context, orderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := m.hold[orderID]
	if len(rows) == 0 {
		return ErrReservationNotFound
	}
	for _, row := range rows {
		if row.status == ReservationConfirmed {
			return ErrReservationStatusConflict
		}
	}
	for i := range rows {
		if rows[i].status != ReservationHeld {
			continue
		}
		st := m.stock[rows[i].skuID]
		if st == nil || st.Reserved < rows[i].quantity {
			return ErrStockInsufficient
		}
		st.Reserved -= rows[i].quantity
		st.Version++
		st.Available = st.OnHand - st.Reserved
		rows[i].status = ReservationReleased
	}
	m.hold[orderID] = rows
	return nil
}

func (m *memRepo) ListExpiredOrderIDs(_ context.Context, now time.Time, limit int) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var ids []string
	for id, rows := range m.hold {
		for _, row := range rows {
			if row.status == ReservationHeld && !row.expires.After(now) {
				ids = append(ids, id)
				break
			}
		}
		if len(ids) >= limit {
			break
		}
	}
	return ids, nil
}

func TestCreateIsIdempotent(t *testing.T) {
	uc := NewStockUsecase(newMemRepo())
	a, err := uc.Create(context.Background(), "sku-1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := uc.Create(context.Background(), "sku-1")
	if err != nil {
		t.Fatal(err)
	}
	if a.SkuID != b.SkuID || b.OnHand != 0 {
		t.Fatalf("got %+v %+v", a, b)
	}
}

func TestAdjustRejectsNegativeAvailable(t *testing.T) {
	uc := NewStockUsecase(newMemRepo())
	if _, err := uc.Create(context.Background(), "sku-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Adjust(context.Background(), "sku-1", 2, "inbound"); err != nil {
		t.Fatal(err)
	}
	_, err := uc.Adjust(context.Background(), "sku-1", -3, "outbound")
	if !errors.Is(err, ErrStockInsufficient) {
		t.Fatalf("got %v", err)
	}
}

func TestReserveDoesNotOversell(t *testing.T) {
	repo := newMemRepo()
	uc := NewStockUsecase(repo)
	if _, err := uc.Create(context.Background(), "sku-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Adjust(context.Background(), "sku-1", 5, "inbound"); err != nil {
		t.Fatal(err)
	}
	var okCount, failCount atomic.Int64
	var winner atomic.Value
	var wg sync.WaitGroup
	expire := time.Now().Add(time.Hour).Unix()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			orderID := fmt.Sprintf("order-%d", i)
			err := uc.Reserve(context.Background(), orderID, []Item{{SkuID: "sku-1", Quantity: 1}}, expire)
			if err == nil {
				okCount.Add(1)
				winner.Store(orderID)
				return
			}
			if errors.Is(err, ErrStockInsufficient) {
				failCount.Add(1)
				return
			}
			t.Errorf("reserve: %v", err)
		}(i)
	}
	wg.Wait()
	if okCount.Load() != 5 || failCount.Load() != 5 {
		t.Fatalf("ok=%d fail=%d", okCount.Load(), failCount.Load())
	}
	rows, err := uc.Get(context.Background(), []string{"sku-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Available != 0 || rows[0].Reserved != 5 || rows[0].OnHand != 5 {
		t.Fatalf("stock %+v", rows[0])
	}
	if err := uc.Reserve(context.Background(), winner.Load().(string), []Item{{SkuID: "sku-1", Quantity: 1}}, expire); err != nil {
		t.Fatal(err)
	}
	rows, err = uc.Get(context.Background(), []string{"sku-1"})
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Reserved != 5 {
		t.Fatalf("second reserve changed reserved to %d", rows[0].Reserved)
	}
}

func TestConfirmAndReleaseStatus(t *testing.T) {
	repo := newMemRepo()
	uc := NewStockUsecase(repo)
	ctx := context.Background()
	if _, err := uc.Create(ctx, "sku-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Adjust(ctx, "sku-1", 4, "inbound"); err != nil {
		t.Fatal(err)
	}
	expire := time.Now().Add(time.Hour).Unix()
	if err := uc.Reserve(ctx, "order-1", []Item{{SkuID: "sku-1", Quantity: 2}}, expire); err != nil {
		t.Fatal(err)
	}
	if err := uc.Confirm(ctx, "order-1"); err != nil {
		t.Fatal(err)
	}
	if err := uc.Confirm(ctx, "order-1"); err != nil {
		t.Fatal(err)
	}
	rows, err := uc.Get(ctx, []string{"sku-1"})
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].OnHand != 2 || rows[0].Reserved != 0 || rows[0].Available != 2 {
		t.Fatalf("after confirm %+v", rows[0])
	}
	if err := uc.Release(ctx, "order-1"); !errors.Is(err, ErrReservationStatusConflict) {
		t.Fatalf("release confirmed: %v", err)
	}

	if err := uc.Reserve(ctx, "order-2", []Item{{SkuID: "sku-1", Quantity: 1}}, expire); err != nil {
		t.Fatal(err)
	}
	if err := uc.Release(ctx, "order-2"); err != nil {
		t.Fatal(err)
	}
	if err := uc.Release(ctx, "order-2"); err != nil {
		t.Fatal(err)
	}
	if err := uc.Confirm(ctx, "order-2"); !errors.Is(err, ErrReservationStatusConflict) {
		t.Fatalf("confirm released: %v", err)
	}
	rows, err = uc.Get(ctx, []string{"sku-1"})
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].OnHand != 2 || rows[0].Reserved != 0 {
		t.Fatalf("after release %+v", rows[0])
	}
}
