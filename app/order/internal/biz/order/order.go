package order

import (
	"context"
	"errors"
	"strings"
	"time"

	v1 "github.com/tmjwjx/supermarket/api/order/v1"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/google/uuid"
)

const (
	StatusPending   int32 = 1
	StatusPaid      int32 = 2
	StatusShipped   int32 = 3
	StatusCompleted int32 = 4
	StatusCancelled int32 = 5
)

const (
	CancelBuyer   = "buyer"
	CancelTimeout = "timeout"
)

const (
	payTimeout        = 30 * time.Minute
	reserveSlack      = 5 * time.Minute
	autoReceiveAfter  = 7 * 24 * time.Hour
	scanBatch         = 200
	compensateTimeout = 3 * time.Second
)

var (
	ErrOrderNotFound          = kerrors.NotFound(v1.ErrorReason_ORDER_NOT_FOUND.String(), "order not found")
	ErrOrderInvalidArgument   = kerrors.BadRequest(v1.ErrorReason_ORDER_INVALID_ARGUMENT.String(), "invalid order argument")
	ErrOrderStatusConflict    = kerrors.Conflict(v1.ErrorReason_ORDER_STATUS_CONFLICT.String(), "order status conflict")
	ErrOrderItemNotSellable   = kerrors.BadRequest(v1.ErrorReason_ORDER_ITEM_NOT_SELLABLE.String(), "order item not sellable")
	ErrOrderStockInsufficient = kerrors.Conflict(v1.ErrorReason_ORDER_STOCK_INSUFFICIENT.String(), "order stock insufficient")
	ErrOrderAddressNotFound   = kerrors.BadRequest(v1.ErrorReason_ORDER_ADDRESS_NOT_FOUND.String(), "order address not found")
	ErrUnauthenticated        = kerrors.Unauthorized(v1.ErrorReason_ORDER_INVALID_ARGUMENT.String(), "unauthenticated")
	ErrForbidden              = kerrors.Forbidden("ORDER_FORBIDDEN", "forbidden")
	ErrUpstream               = kerrors.ServiceUnavailable("ORDER_UPSTREAM", "upstream unavailable")
	ErrDuplicateRequest       = errors.New("duplicate order request")
	ErrReservationMissing     = errors.New("reservation missing")
)

func ItemNotSellable(skuIDs []string) error {
	return kerrors.BadRequest(v1.ErrorReason_ORDER_ITEM_NOT_SELLABLE.String(), "not sellable: "+strings.Join(skuIDs, ","))
}

type Line struct {
	SkuID    string
	Quantity int64
}

type CreateInput struct {
	Lines     []Line
	AddressID string
	Remark    string
	RequestID string
}

type SkuSnap struct {
	SkuID       string
	ProductID   string
	ProductName string
	SpecsJSON   string
	Price       int64
	Image       string
	Sellable    bool
}

type StockLine struct {
	SkuID    string
	Quantity int64
}

type AddressSnap struct {
	Receiver string
	Phone    string
	Province string
	City     string
	District string
	Detail   string
}

type OrderItem struct {
	ID          uuid.UUID
	OrderID     uuid.UUID
	SkuID       string
	ProductID   string
	ProductName string
	SpecsJSON   string
	Image       string
	Price       int64
	Quantity    int64
	Amount      int64
	Reviewed    bool
}

type Order struct {
	ID             uuid.UUID
	OrderNo        string
	UserID         uuid.UUID
	RequestID      string
	Status         int32
	ItemsAmount    int64
	FreightAmount  int64
	PayAmount      int64
	Receiver       string
	Phone          string
	Province       string
	City           string
	District       string
	Detail         string
	Remark         string
	ExpiresAt      time.Time
	StockReleased  bool
	StockConfirmed bool
	// StockConfirmFailed 表示确认库存遇到终态失败 库存没扣 需要人工处理
	StockConfirmFailed bool
	PaidAt             *time.Time
	ShippedAt          *time.Time
	CompletedAt        *time.Time
	CancelledAt        *time.Time
	CancelReason       string
	PaymentID          string
	CreatedAt          time.Time
	Items              []*OrderItem
}

type OrderItemView struct {
	ID        uuid.UUID
	ProductID string
	UserID    uuid.UUID
	SpecsJSON string
	Completed bool
	Reviewed  bool
}

type Catalog interface {
	BatchGetSkus(ctx context.Context, skuIDs []string) ([]SkuSnap, error)
	IncreaseSales(ctx context.Context, orderID, productID string, count int64) error
}

type Stocker interface {
	Reserve(ctx context.Context, orderID string, lines []StockLine, expireUnix int64) error
	Confirm(ctx context.Context, orderID string) error
	Release(ctx context.Context, orderID string) error
}

type Addresses interface {
	Get(ctx context.Context, userID, addressID string) (*AddressSnap, error)
}

type CartCleaner interface {
	RemoveSKUs(ctx context.Context, userID uuid.UUID, skuIDs []string) error
}

type OrderRepo interface {
	FindByUserRequest(ctx context.Context, userID uuid.UUID, requestID string) (*Order, error)
	Create(ctx context.Context, order *Order) (*Order, error)
	FindByUser(ctx context.Context, userID, id uuid.UUID) (*Order, error)
	Find(ctx context.Context, id uuid.UUID) (*Order, error)
	List(ctx context.Context, userID uuid.UUID, status int32, offset, limit int) ([]*Order, error)
	ListAdmin(ctx context.Context, status int32, offset, limit int) ([]*Order, error)
	CancelIfPending(ctx context.Context, id uuid.UUID, reason string, at time.Time) (bool, error)
	MarkReleased(ctx context.Context, id uuid.UUID) error
	MarkPaid(ctx context.Context, id uuid.UUID, paymentID string, at time.Time) (bool, error)
	MarkStockConfirmed(ctx context.Context, id uuid.UUID) error
	MarkStockConfirmFailed(ctx context.Context, id uuid.UUID) error
	MarkShipped(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)
	MarkCompleted(ctx context.Context, userID, id uuid.UUID, at time.Time) (bool, error)
	ListExpiredPending(ctx context.Context, now time.Time, limit int) ([]*Order, error)
	ListUnreleased(ctx context.Context, limit int) ([]*Order, error)
	ListUnconfirmed(ctx context.Context, limit int) ([]*Order, error)
	ListShippedBefore(ctx context.Context, before time.Time, limit int) ([]*Order, error)
	GetItem(ctx context.Context, itemID, userID uuid.UUID) (*OrderItemView, error)
	MarkItemReviewed(ctx context.Context, itemID uuid.UUID) error
	ListPaidBetween(ctx context.Context, from, to time.Time) ([]PaidRef, error)
}

// PaidRef 是某一天已经付过款的订单摘要
type PaidRef struct {
	ID        string
	PaymentID string
	Amount    int64
}

type OrderUsecase struct {
	orders    OrderRepo
	catalog   Catalog
	stock     Stocker
	addresses Addresses
	carts     CartCleaner
}

func NewOrderUsecase(orders OrderRepo, catalog Catalog, stock Stocker, addresses Addresses, carts CartCleaner) *OrderUsecase {
	return &OrderUsecase{orders: orders, catalog: catalog, stock: stock, addresses: addresses, carts: carts}
}

// Create 按请求号幂等下单 价格以商品服务为准 写单失败则释放预占
func (uc *OrderUsecase) Create(ctx context.Context, userID uuid.UUID, in CreateInput) (*Order, error) {
	in.RequestID = strings.TrimSpace(in.RequestID)
	in.AddressID = strings.TrimSpace(in.AddressID)
	if userID == uuid.Nil || in.RequestID == "" || in.AddressID == "" {
		return nil, ErrOrderInvalidArgument
	}
	lines, err := mergeLines(in.Lines)
	if err != nil {
		return nil, err
	}
	existing, err := uc.orders.FindByUserRequest(ctx, userID, in.RequestID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrOrderNotFound) {
		return nil, err
	}
	ids := make([]string, len(lines))
	for i, ln := range lines {
		ids[i] = ln.SkuID
	}
	snaps, err := uc.catalog.BatchGetSkus(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]SkuSnap, len(snaps))
	for _, snap := range snaps {
		byID[snap.SkuID] = snap
	}
	var bad []string
	items := make([]*OrderItem, 0, len(lines))
	var total int64
	for _, ln := range lines {
		snap, ok := byID[ln.SkuID]
		if !ok || !snap.Sellable || snap.Price < 0 {
			bad = append(bad, ln.SkuID)
			continue
		}
		amount := snap.Price * ln.Quantity
		total += amount
		items = append(items, &OrderItem{
			SkuID: ln.SkuID, ProductID: snap.ProductID, ProductName: snap.ProductName,
			SpecsJSON: snap.SpecsJSON, Image: snap.Image, Price: snap.Price,
			Quantity: ln.Quantity, Amount: amount,
		})
	}
	if len(bad) > 0 {
		return nil, ItemNotSellable(bad)
	}
	addr, err := uc.addresses.Get(ctx, userID.String(), in.AddressID)
	if err != nil {
		return nil, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	expire := now.Add(payTimeout)
	stockLines := make([]StockLine, len(lines))
	for i, ln := range lines {
		stockLines[i] = StockLine{SkuID: ln.SkuID, Quantity: ln.Quantity}
	}
	if err := uc.stock.Reserve(ctx, id.String(), stockLines, expire.Add(reserveSlack).Unix()); err != nil {
		if !errors.Is(err, ErrOrderStockInsufficient) {
			uc.compensate(ctx, id.String())
		}
		return nil, err
	}
	created, err := uc.orders.Create(ctx, &Order{
		ID: id, UserID: userID, RequestID: in.RequestID, Status: StatusPending,
		ItemsAmount: total, FreightAmount: 0, PayAmount: total,
		Receiver: addr.Receiver, Phone: addr.Phone, Province: addr.Province,
		City: addr.City, District: addr.District, Detail: addr.Detail,
		Remark: strings.TrimSpace(in.Remark), ExpiresAt: expire, Items: items,
	})
	if err != nil {
		uc.compensate(ctx, id.String())
		if errors.Is(err, ErrDuplicateRequest) {
			again, ferr := uc.orders.FindByUserRequest(ctx, userID, in.RequestID)
			if ferr != nil {
				return nil, ferr
			}
			return again, nil
		}
		return nil, err
	}
	if err := uc.carts.RemoveSKUs(ctx, userID, ids); err != nil {
		log.Error("clear cart after order", "err", err, "order_id", id.String())
	}
	return created, nil
}

// compensate 释放下单失败的预占 请求 ctx 可能已超时或取消 所以脱离它另给 3 秒
func (uc *OrderUsecase) compensate(ctx context.Context, orderID string) {
	c, cancel := context.WithTimeout(context.WithoutCancel(ctx), compensateTimeout)
	defer cancel()
	if err := uc.stock.Release(c, orderID); err != nil {
		log.Error("release reservation after failed create", "err", err, "order_id", orderID)
	}
}

func (uc *OrderUsecase) Get(ctx context.Context, userID, id uuid.UUID) (*Order, error) {
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderInvalidArgument
	}
	return uc.orders.FindByUser(ctx, userID, id)
}

func (uc *OrderUsecase) List(ctx context.Context, userID uuid.UUID, status int32, offset, limit int) ([]*Order, error) {
	if userID == uuid.Nil || offset < 0 || limit <= 0 {
		return nil, ErrOrderInvalidArgument
	}
	if status < 0 || status > StatusCancelled {
		return nil, ErrOrderInvalidArgument
	}
	return uc.orders.List(ctx, userID, status, offset, limit)
}

// Cancel 把待支付订单改成已取消 再释放库存
func (uc *OrderUsecase) Cancel(ctx context.Context, userID, id uuid.UUID) (*Order, error) {
	o, err := uc.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if o.Status != StatusPending {
		return nil, ErrOrderStatusConflict
	}
	ok, err := uc.orders.CancelIfPending(ctx, id, CancelBuyer, time.Now())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrOrderStatusConflict
	}
	_ = uc.releaseStock(ctx, id)
	return uc.orders.FindByUser(ctx, userID, id)
}

// ConfirmReceipt 已发货改为已完成 并通知商品增加销量
func (uc *OrderUsecase) ConfirmReceipt(ctx context.Context, userID, id uuid.UUID) (*Order, error) {
	return uc.finish(ctx, userID, id)
}

// MarkPaid 待支付变为已支付 同一支付单的重复通知在已支付已发货已完成时都直接成功
// 只有已取消或支付单不同才冲突 超过支付截止加预占余量才到的通知按超时关单并冲突 让支付侧退款
func (uc *OrderUsecase) MarkPaid(ctx context.Context, orderID, paymentID string) error {
	orderID = strings.TrimSpace(orderID)
	paymentID = strings.TrimSpace(paymentID)
	if orderID == "" || paymentID == "" {
		return ErrOrderInvalidArgument
	}
	id, err := uuid.Parse(orderID)
	if err != nil {
		return ErrOrderInvalidArgument
	}
	o, err := uc.orders.Find(ctx, id)
	if err != nil {
		return err
	}
	if done, err := uc.samePayment(ctx, o, paymentID); done {
		return err
	}
	if o.Status != StatusPending {
		return ErrOrderStatusConflict
	}
	now := time.Now()
	if now.After(o.ExpiresAt.Add(reserveSlack)) {
		closed, err := uc.orders.CancelIfPending(ctx, id, CancelTimeout, now)
		if err != nil {
			return err
		}
		if closed {
			// 释放失败由扫描里的未释放补偿兜底
			_ = uc.releaseStock(ctx, id)
		}
		return ErrOrderStatusConflict
	}
	ok, err := uc.orders.MarkPaid(ctx, id, paymentID, now)
	if err != nil {
		return err
	}
	if !ok {
		again, ferr := uc.orders.Find(ctx, id)
		if ferr != nil {
			return ferr
		}
		if done, err := uc.samePayment(ctx, again, paymentID); done {
			return err
		}
		return ErrOrderStatusConflict
	}
	o.Status = StatusPaid
	o.PaymentID = paymentID
	o.StockConfirmed = false
	return uc.confirmStock(ctx, o)
}

// samePayment 判断是不是同一支付单的重复通知 是则返回 true 已支付时顺带补确认库存
func (uc *OrderUsecase) samePayment(ctx context.Context, o *Order, paymentID string) (bool, error) {
	if o.PaymentID != paymentID {
		return false, nil
	}
	switch o.Status {
	case StatusPaid:
		return true, uc.confirmStock(ctx, o)
	case StatusShipped, StatusCompleted:
		return true, nil
	default:
		return false, nil
	}
}

func (uc *OrderUsecase) ListAdmin(ctx context.Context, status int32, offset, limit int) ([]*Order, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		return nil, ErrOrderInvalidArgument
	}
	return uc.orders.ListAdmin(ctx, status, offset, limit)
}

func (uc *OrderUsecase) ListPaid(ctx context.Context, from, to time.Time) ([]PaidRef, error) {
	if !from.Before(to) {
		return nil, ErrOrderInvalidArgument
	}
	return uc.orders.ListPaidBetween(ctx, from, to)
}

func (uc *OrderUsecase) GetItem(ctx context.Context, itemID, userID uuid.UUID) (*OrderItemView, error) {
	if itemID == uuid.Nil || userID == uuid.Nil {
		return nil, ErrOrderInvalidArgument
	}
	return uc.orders.GetItem(ctx, itemID, userID)
}

func (uc *OrderUsecase) MarkItemReviewed(ctx context.Context, itemID uuid.UUID) error {
	if itemID == uuid.Nil {
		return ErrOrderInvalidArgument
	}
	return uc.orders.MarkItemReviewed(ctx, itemID)
}

// Ship 后台把已支付订单改为已发货
func (uc *OrderUsecase) Ship(ctx context.Context, id uuid.UUID) (*Order, error) {
	if id == uuid.Nil {
		return nil, ErrOrderInvalidArgument
	}
	o, err := uc.orders.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if o.Status != StatusPaid {
		return nil, ErrOrderStatusConflict
	}
	ok, err := uc.orders.MarkShipped(ctx, id, time.Now())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrOrderStatusConflict
	}
	return uc.orders.Find(ctx, id)
}

func (uc *OrderUsecase) finish(ctx context.Context, userID, id uuid.UUID) (*Order, error) {
	o, err := uc.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if o.Status != StatusShipped {
		return nil, ErrOrderStatusConflict
	}
	ok, err := uc.orders.MarkCompleted(ctx, userID, id, time.Now())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrOrderStatusConflict
	}
	updated, err := uc.orders.FindByUser(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	uc.addSales(ctx, updated)
	return updated, nil
}

func (uc *OrderUsecase) addSales(ctx context.Context, o *Order) {
	if o == nil {
		return
	}
	counts := map[string]int64{}
	for _, it := range o.Items {
		if it == nil || it.ProductID == "" || it.Quantity <= 0 {
			continue
		}
		counts[it.ProductID] += it.Quantity
	}
	for id, n := range counts {
		if err := uc.catalog.IncreaseSales(ctx, o.ID.String(), id, n); err != nil {
			log.Error("increase sales", "err", err, "product_id", id, "order_id", o.ID.String())
		}
	}
}

func (uc *OrderUsecase) releaseStock(ctx context.Context, id uuid.UUID) error {
	if err := uc.stock.Release(ctx, id.String()); err != nil {
		log.Error("release reservation", "err", err, "order_id", id.String())
		return err
	}
	if err := uc.orders.MarkReleased(ctx, id); err != nil {
		log.Error("mark stock released", "err", err, "order_id", id.String())
		return err
	}
	return nil
}

// confirmStock 把预占转成扣减 下游故障留给扫描重试
// 预占已释放或找不到是终态 重试也不会扣到库存 落标记打告警后不再重试
func (uc *OrderUsecase) confirmStock(ctx context.Context, o *Order) error {
	if o == nil || o.StockConfirmed || o.StockConfirmFailed {
		return nil
	}
	err := uc.stock.Confirm(ctx, o.ID.String())
	switch {
	case err == nil:
		if err := uc.orders.MarkStockConfirmed(ctx, o.ID); err != nil {
			log.Error("mark stock confirmed", "err", err, "order_id", o.ID.String())
		}
	case errors.Is(err, ErrOrderStatusConflict) || errors.Is(err, ErrReservationMissing):
		log.Error("ALERT stock confirm failed permanently, stock not deducted", "err", err, "order_id", o.ID.String(), "alert", true)
		if err := uc.orders.MarkStockConfirmFailed(ctx, o.ID); err != nil {
			log.Error("mark stock confirm failed", "err", err, "order_id", o.ID.String())
		}
	default:
		log.Error("confirm reservation", "err", err, "order_id", o.ID.String())
	}
	return nil
}

func mergeLines(lines []Line) ([]Line, error) {
	if len(lines) == 0 {
		return nil, ErrOrderInvalidArgument
	}
	qty := make(map[string]int64, len(lines))
	order := make([]string, 0, len(lines))
	for _, ln := range lines {
		id := strings.TrimSpace(ln.SkuID)
		if id == "" || ln.Quantity <= 0 {
			return nil, ErrOrderInvalidArgument
		}
		if _, ok := qty[id]; !ok {
			order = append(order, id)
		}
		qty[id] += ln.Quantity
	}
	out := make([]Line, 0, len(order))
	for _, id := range order {
		out = append(out, Line{SkuID: id, Quantity: qty[id]})
	}
	return out, nil
}
