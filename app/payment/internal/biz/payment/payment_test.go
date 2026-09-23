package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tmjwjx/supermarket/app/payment/internal/conf"

	"github.com/google/uuid"
)

type fakeOrders struct {
	view    *OrderView
	markErr error
	marked  int
	paid    []PaidOrder
}

func (f *fakeOrders) GetOrder(context.Context, uuid.UUID, string) (*OrderView, error) {
	if f.view == nil {
		return nil, ErrPaymentOrderNotPayable
	}
	cp := *f.view
	return &cp, nil
}

func (f *fakeOrders) MarkOrderPaid(context.Context, string, uuid.UUID) error {
	f.marked++
	return f.markErr
}

func (f *fakeOrders) ListPaidOrders(context.Context, time.Time, time.Time) ([]PaidOrder, error) {
	return f.paid, nil
}

type fakeRepo struct {
	rows map[uuid.UUID]*Payment
}

func (f *fakeRepo) put(p *Payment) {
	if f.rows == nil {
		f.rows = map[uuid.UUID]*Payment{}
	}
	cp := *p
	f.rows[p.ID] = &cp
}

func (f *fakeRepo) FindByID(_ context.Context, id uuid.UUID) (*Payment, error) {
	p, ok := f.rows[id]
	if !ok {
		return nil, ErrPaymentNotFound
	}
	cp := *p
	return &cp, nil
}

func (f *fakeRepo) FindByOrderID(_ context.Context, orderID string) (*Payment, error) {
	for _, p := range f.rows {
		if p.OrderID == orderID {
			cp := *p
			return &cp, nil
		}
	}
	return nil, ErrPaymentNotFound
}

func (f *fakeRepo) Create(_ context.Context, p *Payment) (*Payment, error) {
	if _, err := f.FindByOrderID(context.Background(), p.OrderID); err == nil {
		return nil, ErrPaymentDuplicate
	}
	id := uuid.Must(uuid.NewV7())
	p.ID = id
	if p.Status == 0 {
		p.Status = StatusPending
	}
	f.put(p)
	return f.FindByID(context.Background(), id)
}

func (f *fakeRepo) ResetFailed(_ context.Context, id uuid.UUID, amount int64) (*Payment, error) {
	p, ok := f.rows[id]
	if !ok {
		return nil, ErrPaymentNotFound
	}
	if p.Status != StatusFailed {
		if p.Status == StatusPending {
			cp := *p
			return &cp, nil
		}
		return nil, ErrPaymentStatusConflict
	}
	p.Status = StatusPending
	p.Amount = amount
	p.Notified = false
	cp := *p
	return &cp, nil
}

func (f *fakeRepo) MarkResult(_ context.Context, id uuid.UUID, success bool) (*Payment, bool, error) {
	p, ok := f.rows[id]
	if !ok {
		return nil, false, ErrPaymentNotFound
	}
	if p.Status != StatusPending {
		cp := *p
		return &cp, false, nil
	}
	if success {
		now := time.Now()
		p.Status = StatusSuccess
		p.PaidAt = &now
		p.ChannelTradeNo = "trade"
	} else {
		p.Status = StatusFailed
	}
	cp := *p
	return &cp, true, nil
}

func (f *fakeRepo) ListUnnotified(context.Context, int) ([]*Payment, error) {
	var out []*Payment
	for _, p := range f.rows {
		if p.Status == StatusSuccess && !p.Notified {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) MarkNotified(_ context.Context, id uuid.UUID) error {
	p, ok := f.rows[id]
	if !ok {
		return ErrPaymentNotFound
	}
	if p.Status == StatusSuccess {
		p.Notified = true
	}
	return nil
}

func (f *fakeRepo) MarkRefunded(_ context.Context, id uuid.UUID) (*Payment, error) {
	p, ok := f.rows[id]
	if !ok {
		return nil, ErrPaymentNotFound
	}
	if p.Status == StatusSuccess {
		now := time.Now()
		p.Status = StatusRefunded
		p.Notified = true
		p.RefundedAt = &now
	}
	cp := *p
	return &cp, nil
}

func (f *fakeRepo) SaveDiffs(context.Context, string, []Diff) error { return nil }

func (f *fakeRepo) ListDiffs(context.Context, string, int, int) ([]Diff, bool, error) {
	return nil, false, nil
}

func (f *fakeRepo) ListSuccessBetween(_ context.Context, from, to time.Time) ([]*Payment, error) {
	out := []*Payment{}
	for _, p := range f.rows {
		if p.Status != StatusSuccess || p.PaidAt == nil {
			continue
		}
		if !p.PaidAt.Before(from) && p.PaidAt.Before(to) {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out, nil
}

func payable(user uuid.UUID, orderID string, amount int64) *OrderView {
	return &OrderView{
		ID:        orderID,
		UserID:    user,
		Status:    OrderStatusPending,
		PayAmount: amount,
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func TestCreateUsesOrderAmountAndReturnsPending(t *testing.T) {
	user := uuid.Must(uuid.NewV7())
	repo := &fakeRepo{}
	orders := &fakeOrders{view: payable(user, "order-1", 1990)}
	uc := NewPaymentUsecase(repo, orders, nil, nil)
	first, err := uc.Create(context.Background(), user, "order-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.Amount != 1990 || first.Status != StatusPending || first.Channel != ChannelMock {
		t.Fatalf("created %+v", first)
	}
	second, err := uc.Create(context.Background(), user, "order-1")
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("want same payment %s got %s", first.ID, second.ID)
	}
}

func TestCreateRejectsUnpayableOrder(t *testing.T) {
	user := uuid.Must(uuid.NewV7())
	other := uuid.Must(uuid.NewV7())
	repo := &fakeRepo{}
	orders := &fakeOrders{view: payable(other, "order-1", 10)}
	uc := NewPaymentUsecase(repo, orders, nil, nil)
	if _, err := uc.Create(context.Background(), user, "order-1"); !errors.Is(err, ErrPaymentOrderNotPayable) {
		t.Fatalf("wrong user: %v", err)
	}
	orders.view = payable(user, "order-1", 10)
	orders.view.Status = 2
	if _, err := uc.Create(context.Background(), user, "order-1"); !errors.Is(err, ErrPaymentOrderNotPayable) {
		t.Fatalf("paid order: %v", err)
	}
	orders.view = payable(user, "order-1", 10)
	orders.view.ExpiresAt = time.Now().Add(-time.Minute)
	if _, err := uc.Create(context.Background(), user, "order-1"); !errors.Is(err, ErrPaymentOrderNotPayable) {
		t.Fatalf("expired: %v", err)
	}
}

func TestCreateResetsFailed(t *testing.T) {
	user := uuid.Must(uuid.NewV7())
	id := uuid.Must(uuid.NewV7())
	repo := &fakeRepo{}
	repo.put(&Payment{ID: id, OrderID: "order-1", UserID: user, Amount: 1, Status: StatusFailed, Channel: ChannelMock})
	orders := &fakeOrders{view: payable(user, "order-1", 50)}
	uc := NewPaymentUsecase(repo, orders, nil, nil)
	got, err := uc.Create(context.Background(), user, "order-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id || got.Status != StatusPending || got.Amount != 50 {
		t.Fatalf("reset %+v", got)
	}
}

func TestSimulateSuccessFailDuplicateAndRefund(t *testing.T) {
	user := uuid.Must(uuid.NewV7())
	repo := &fakeRepo{}
	orders := &fakeOrders{view: payable(user, "order-1", 80)}
	uc := NewPaymentUsecase(repo, orders, nil, nil)
	created, err := uc.Create(context.Background(), user, "order-1")
	if err != nil {
		t.Fatal(err)
	}
	failed, err := uc.Simulate(context.Background(), user, created.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != StatusFailed || orders.marked != 0 {
		t.Fatalf("fail %+v marked %d", failed, orders.marked)
	}
	again, err := uc.Simulate(context.Background(), user, created.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != StatusFailed || orders.marked != 0 {
		t.Fatalf("duplicate %+v marked %d", again, orders.marked)
	}

	repo.put(&Payment{ID: created.ID, OrderID: "order-1", UserID: user, Amount: 80, Status: StatusPending, Channel: ChannelMock})
	okPay, err := uc.Simulate(context.Background(), user, created.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if okPay.Status != StatusSuccess || !okPay.Notified || orders.marked != 1 {
		t.Fatalf("success %+v marked %d", okPay, orders.marked)
	}
	dup, err := uc.Simulate(context.Background(), user, created.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if dup.Status != StatusSuccess || orders.marked != 1 {
		t.Fatalf("second callback %+v marked %d", dup, orders.marked)
	}

	id2 := uuid.Must(uuid.NewV7())
	repo.put(&Payment{ID: id2, OrderID: "order-2", UserID: user, Amount: 1, Status: StatusPending, Channel: ChannelMock})
	orders.markErr = ErrOrderStatusConflict
	refunded, err := uc.Simulate(context.Background(), user, id2, true)
	if err != nil {
		t.Fatal(err)
	}
	if refunded.Status != StatusRefunded || !refunded.Notified {
		t.Fatalf("refund %+v", refunded)
	}
}

func TestSimulateDisabled(t *testing.T) {
	off := false
	user := uuid.Must(uuid.NewV7())
	id := uuid.Must(uuid.NewV7())
	repo := &fakeRepo{}
	repo.put(&Payment{ID: id, OrderID: "o", UserID: user, Status: StatusPending, Channel: ChannelMock})
	uc := NewPaymentUsecase(repo, &fakeOrders{}, &conf.Payment{SimulateEnabled: &off}, nil)
	if _, err := uc.Simulate(context.Background(), user, id, true); !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("disabled: %v", err)
	}
}

func TestRetryNotifiesPendingSuccess(t *testing.T) {
	user := uuid.Must(uuid.NewV7())
	id := uuid.Must(uuid.NewV7())
	repo := &fakeRepo{}
	repo.put(&Payment{ID: id, OrderID: "order-9", UserID: user, Status: StatusSuccess, Channel: ChannelMock})
	orders := &fakeOrders{}
	uc := NewPaymentUsecase(repo, orders, nil, nil)
	if err := uc.RetryNotify(context.Background()); err != nil {
		t.Fatal(err)
	}
	if orders.marked != 1 || !repo.rows[id].Notified {
		t.Fatalf("marked %d notified %v", orders.marked, repo.rows[id].Notified)
	}
}

func TestReconcileLeavesRowsUntouched(t *testing.T) {
	day := time.Date(2026, 9, 22, 15, 0, 0, 0, time.Local)
	id := uuid.Must(uuid.NewV7())
	repo := &fakeRepo{}
	repo.put(&Payment{ID: id, OrderID: "order-a", Status: StatusSuccess, Amount: 100, PaidAt: &day})
	orders := &fakeOrders{paid: []PaidOrder{{ID: "order-b", PaymentID: "other", Amount: 80}}}
	uc := NewPaymentUsecase(repo, orders, nil, nil)
	if err := uc.Reconcile(context.Background(), day); err != nil {
		t.Fatal(err)
	}
	if repo.rows[id].Status != StatusSuccess || repo.rows[id].Amount != 100 {
		t.Fatalf("payment changed %+v", repo.rows[id])
	}
}
