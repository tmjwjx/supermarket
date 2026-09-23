package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// payRepo 只实现支付回调和扫描用到的方法 状态存在内存里
type payRepo struct {
	OrderRepo
	orders      map[uuid.UUID]*Order
	markPaid    int
	cancels     []string
	confirmFail int
}

func newPayRepo(orders ...*Order) *payRepo {
	r := &payRepo{orders: map[uuid.UUID]*Order{}}
	for _, o := range orders {
		r.orders[o.ID] = o
	}
	return r
}

func (r *payRepo) Find(_ context.Context, id uuid.UUID) (*Order, error) {
	o, ok := r.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	cp := *o
	return &cp, nil
}

func (r *payRepo) MarkPaid(_ context.Context, id uuid.UUID, paymentID string, at time.Time) (bool, error) {
	r.markPaid++
	o := r.orders[id]
	if o.Status != StatusPending {
		return false, nil
	}
	o.Status, o.PaymentID, o.PaidAt = StatusPaid, paymentID, &at
	return true, nil
}

func (r *payRepo) CancelIfPending(_ context.Context, id uuid.UUID, reason string, at time.Time) (bool, error) {
	o := r.orders[id]
	if o.Status != StatusPending {
		return false, nil
	}
	r.cancels = append(r.cancels, reason)
	o.Status, o.CancelReason, o.CancelledAt = StatusCancelled, reason, &at
	return true, nil
}

func (r *payRepo) MarkReleased(_ context.Context, id uuid.UUID) error {
	r.orders[id].StockReleased = true
	return nil
}

func (r *payRepo) MarkStockConfirmed(_ context.Context, id uuid.UUID) error {
	r.orders[id].StockConfirmed = true
	return nil
}

func (r *payRepo) MarkStockConfirmFailed(_ context.Context, id uuid.UUID) error {
	r.confirmFail++
	r.orders[id].StockConfirmFailed = true
	return nil
}

func (r *payRepo) ListExpiredPending(context.Context, time.Time, int) ([]*Order, error) {
	return nil, nil
}

func (r *payRepo) ListUnreleased(context.Context, int) ([]*Order, error) { return nil, nil }

func (r *payRepo) ListUnconfirmed(context.Context, int) ([]*Order, error) {
	var out []*Order
	for _, o := range r.orders {
		paid := o.Status == StatusPaid || o.Status == StatusShipped || o.Status == StatusCompleted
		if paid && !o.StockConfirmed && !o.StockConfirmFailed {
			cp := *o
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *payRepo) ListShippedBefore(context.Context, time.Time, int) ([]*Order, error) {
	return nil, nil
}

// payStock 记下确认和释放次数 confirmErr 决定确认的结果
type payStock struct {
	confirmErr error
	confirms   int
	releases   int
}

func (s *payStock) Reserve(context.Context, string, []StockLine, int64) error { return nil }

func (s *payStock) Confirm(context.Context, string) error {
	s.confirms++
	return s.confirmErr
}

func (s *payStock) Release(context.Context, string) error {
	s.releases++
	return nil
}

func paidOrder(status int32, paymentID string) *Order {
	return &Order{ID: uuid.Must(uuid.NewV7()), Status: status, PaymentID: paymentID, ExpiresAt: time.Now().Add(time.Minute)}
}

// 发货或完成后 同一支付单的重复通知直接成功 不能冲突 否则支付侧会退款
func TestMarkPaidDuplicateAfterShipOrCompleteSucceeds(t *testing.T) {
	for _, status := range []int32{StatusShipped, StatusCompleted} {
		o := paidOrder(status, "pay-1")
		o.StockConfirmed = true
		repo := newPayRepo(o)
		stock := &payStock{}
		uc := NewOrderUsecase(repo, nil, stock, nil, nil)
		if err := uc.MarkPaid(context.Background(), o.ID.String(), "pay-1"); err != nil {
			t.Fatalf("status %d got %v", status, err)
		}
		if repo.markPaid != 0 || stock.confirms != 0 {
			t.Fatalf("status %d markPaid=%d confirms=%d", status, repo.markPaid, stock.confirms)
		}
	}
}

// 已支付且库存还没确认时 重复通知顺带补确认
func TestMarkPaidDuplicateOnPaidConfirmsStock(t *testing.T) {
	o := paidOrder(StatusPaid, "pay-1")
	repo := newPayRepo(o)
	stock := &payStock{}
	uc := NewOrderUsecase(repo, nil, stock, nil, nil)
	if err := uc.MarkPaid(context.Background(), o.ID.String(), "pay-1"); err != nil {
		t.Fatal(err)
	}
	if stock.confirms != 1 || !repo.orders[o.ID].StockConfirmed {
		t.Fatalf("confirms=%d confirmed=%v", stock.confirms, repo.orders[o.ID].StockConfirmed)
	}
}

// 只有已取消或支付单不同才冲突
func TestMarkPaidConflictOnlyWhenCancelledOrOtherPayment(t *testing.T) {
	cases := []*Order{
		paidOrder(StatusCancelled, "pay-1"),
		paidOrder(StatusPaid, "pay-2"),
		paidOrder(StatusShipped, "pay-2"),
		paidOrder(StatusCompleted, "pay-2"),
	}
	for _, o := range cases {
		uc := NewOrderUsecase(newPayRepo(o), nil, &payStock{}, nil, nil)
		if err := uc.MarkPaid(context.Background(), o.ID.String(), "pay-1"); !errors.Is(err, ErrOrderStatusConflict) {
			t.Fatalf("status %d payment %s got %v", o.Status, o.PaymentID, err)
		}
	}
}

// 超过支付截止加 5 分钟才到的通知 按超时关单释放库存并冲突
func TestMarkPaidLateClosesOrderAsTimeout(t *testing.T) {
	o := paidOrder(StatusPending, "")
	o.ExpiresAt = time.Now().Add(-reserveSlack - time.Minute)
	repo := newPayRepo(o)
	stock := &payStock{}
	uc := NewOrderUsecase(repo, nil, stock, nil, nil)
	if err := uc.MarkPaid(context.Background(), o.ID.String(), "pay-1"); !errors.Is(err, ErrOrderStatusConflict) {
		t.Fatalf("got %v", err)
	}
	got := repo.orders[o.ID]
	if got.Status != StatusCancelled || got.CancelReason != CancelTimeout || !got.StockReleased {
		t.Fatalf("order %+v", got)
	}
	if repo.markPaid != 0 || stock.releases != 1 || stock.confirms != 0 {
		t.Fatalf("markPaid=%d releases=%d confirms=%d", repo.markPaid, stock.releases, stock.confirms)
	}
}

// 截止后 5 分钟以内预占还在 仍然收款
func TestMarkPaidWithinSlackStillPays(t *testing.T) {
	o := paidOrder(StatusPending, "")
	o.ExpiresAt = time.Now().Add(-2 * time.Minute)
	repo := newPayRepo(o)
	uc := NewOrderUsecase(repo, nil, &payStock{}, nil, nil)
	if err := uc.MarkPaid(context.Background(), o.ID.String(), "pay-1"); err != nil {
		t.Fatal(err)
	}
	if repo.orders[o.ID].Status != StatusPaid {
		t.Fatalf("status %d", repo.orders[o.ID].Status)
	}
}

// 确认遇到冲突或找不到是终态 落标记后扫描不再重试
func TestConfirmTerminalFailureFlagsAndStopsRetry(t *testing.T) {
	for _, cause := range []error{ErrOrderStatusConflict, ErrReservationMissing} {
		o := paidOrder(StatusPending, "")
		repo := newPayRepo(o)
		stock := &payStock{confirmErr: cause}
		uc := NewOrderUsecase(repo, nil, stock, nil, nil)
		if err := uc.MarkPaid(context.Background(), o.ID.String(), "pay-1"); err != nil {
			t.Fatalf("%v: got %v", cause, err)
		}
		if !repo.orders[o.ID].StockConfirmFailed || repo.confirmFail != 1 {
			t.Fatalf("%v: order %+v", cause, repo.orders[o.ID])
		}
		for i := 0; i < 3; i++ {
			if err := uc.Sweep(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		if stock.confirms != 1 {
			t.Fatalf("%v: confirms %d", cause, stock.confirms)
		}
	}
}

// 下游故障不是终态 不落标记 扫描继续重试
func TestConfirmUpstreamFailureKeepsRetrying(t *testing.T) {
	o := paidOrder(StatusPending, "")
	repo := newPayRepo(o)
	stock := &payStock{confirmErr: ErrUpstream}
	uc := NewOrderUsecase(repo, nil, stock, nil, nil)
	if err := uc.MarkPaid(context.Background(), o.ID.String(), "pay-1"); err != nil {
		t.Fatal(err)
	}
	if err := uc.Sweep(context.Background()); err != nil {
		t.Fatal(err)
	}
	if stock.confirms != 2 || repo.orders[o.ID].StockConfirmFailed {
		t.Fatalf("confirms=%d order %+v", stock.confirms, repo.orders[o.ID])
	}
	stock.confirmErr = nil
	if err := uc.Sweep(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !repo.orders[o.ID].StockConfirmed {
		t.Fatal("stock should be confirmed once upstream recovers")
	}
}

// ctxStock 在预占时取消请求 ctx 再记下释放时拿到的 ctx 状态
type ctxStock struct {
	reserveErr  error
	cancel      context.CancelFunc
	releaseErr  error
	releaseLeft time.Duration
	released    bool
}

func (s *ctxStock) Reserve(context.Context, string, []StockLine, int64) error {
	s.cancel()
	return s.reserveErr
}

func (s *ctxStock) Confirm(context.Context, string) error { return nil }

func (s *ctxStock) Release(ctx context.Context, _ string) error {
	s.released = true
	s.releaseErr = ctx.Err()
	if deadline, ok := ctx.Deadline(); ok {
		s.releaseLeft = time.Until(deadline)
	}
	return nil
}

type failCreateOrders struct{ fakeOrders }

func (f *failCreateOrders) Create(context.Context, *Order) (*Order, error) {
	return nil, errors.New("db down")
}

// 请求 ctx 已经取消 补偿释放仍要拿到未取消且带 3 秒超时的 ctx
func TestCreateCompensationUsesDetachedContext(t *testing.T) {
	catalog := &fakeCatalog{snaps: []SkuSnap{{SkuID: "sku-1", ProductID: "p1", Price: 100, Sellable: true}}}
	in := CreateInput{Lines: []Line{{SkuID: "sku-1", Quantity: 1}}, AddressID: "addr-1", RequestID: "req-1"}
	cases := map[string]struct {
		repo       OrderRepo
		reserveErr error
	}{
		"reserve failed": {repo: &fakeOrders{saved: map[string]*Order{}}, reserveErr: ErrUpstream},
		"create failed":  {repo: &failCreateOrders{fakeOrders{saved: map[string]*Order{}}}},
	}
	for name, tc := range cases {
		ctx, cancel := context.WithCancel(context.Background())
		stock := &ctxStock{reserveErr: tc.reserveErr, cancel: cancel}
		uc := NewOrderUsecase(tc.repo, catalog, stock, &fakeAddress{}, &fakeCart{})
		if _, err := uc.Create(ctx, uuid.Must(uuid.NewV7()), in); err == nil {
			t.Fatalf("%s: want error", name)
		}
		if !stock.released || stock.releaseErr != nil {
			t.Fatalf("%s: released=%v ctx err=%v", name, stock.released, stock.releaseErr)
		}
		if stock.releaseLeft <= 0 || stock.releaseLeft > compensateTimeout {
			t.Fatalf("%s: deadline in %v", name, stock.releaseLeft)
		}
		cancel()
	}
}
