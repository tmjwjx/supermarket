package payment

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	v1 "github.com/tmjwjx/supermarket/api/payment/v1"
	"github.com/tmjwjx/supermarket/app/payment/internal/conf"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

const (
	StatusPending  int32 = 1
	StatusSuccess  int32 = 2
	StatusFailed   int32 = 3
	StatusClosed   int32 = 4
	StatusRefunded int32 = 5

	OrderStatusPending int32 = 1
	ChannelMock              = "mock"
)

var (
	ErrPaymentNotFound        = kerrors.NotFound(v1.ErrorReason_PAYMENT_NOT_FOUND.String(), "payment not found")
	ErrPaymentInvalidArgument = kerrors.BadRequest(v1.ErrorReason_PAYMENT_INVALID_ARGUMENT.String(), "invalid payment argument")
	ErrPaymentStatusConflict  = kerrors.Conflict(v1.ErrorReason_PAYMENT_STATUS_CONFLICT.String(), "payment status conflict")
	ErrPaymentOrderNotPayable = kerrors.Conflict(v1.ErrorReason_PAYMENT_ORDER_NOT_PAYABLE.String(), "order not payable")
	ErrPaymentUpstream        = kerrors.ServiceUnavailable("PAYMENT_UPSTREAM_UNAVAILABLE", "order unavailable")
	ErrPaymentDuplicate       = errors.New("duplicate payment")
	ErrOrderStatusConflict    = errors.New("order status conflict")
)

type Payment struct {
	ID             uuid.UUID
	OrderID        string
	UserID         uuid.UUID
	Amount         int64
	Status         int32
	Channel        string
	ChannelTradeNo string
	Notified       bool
	PaidAt         *time.Time
	RefundedAt     *time.Time
}

type OrderView struct {
	ID        string
	UserID    uuid.UUID
	Status    int32
	PayAmount int64
	ExpiresAt time.Time
}

type OrderClient interface {
	GetOrder(ctx context.Context, userID uuid.UUID, orderID string) (*OrderView, error)
	MarkOrderPaid(ctx context.Context, orderID string, paymentID uuid.UUID) error
	ListPaidOrders(ctx context.Context, from, to time.Time) ([]PaidOrder, error)
}

// PaidOrder 是订单侧已经付过款的摘要
type PaidOrder struct {
	ID        string
	PaymentID string
	Amount    int64
}

type PaymentRepo interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	FindByOrderID(ctx context.Context, orderID string) (*Payment, error)
	Create(ctx context.Context, p *Payment) (*Payment, error)
	ResetFailed(ctx context.Context, id uuid.UUID, amount int64) (*Payment, error)
	MarkResult(ctx context.Context, id uuid.UUID, success bool) (*Payment, bool, error)
	ListUnnotified(ctx context.Context, limit int) ([]*Payment, error)
	MarkNotified(ctx context.Context, id uuid.UUID) error
	MarkRefunded(ctx context.Context, id uuid.UUID) (*Payment, error)
	ListSuccessBetween(ctx context.Context, from, to time.Time) ([]*Payment, error)
	SaveDiffs(ctx context.Context, day string, rows []Diff) error
	ListDiffs(ctx context.Context, day string, size, offset int) ([]Diff, bool, error)
}

// Diff 是对账差异 可按日期查询
type Diff struct {
	ID        string
	Day       string
	Kind      string
	OrderID   string
	PaymentID string
	Detail    string
}

type PaymentUsecase struct {
	repo     PaymentRepo
	orders   OrderClient
	simulate bool
	logger   *slog.Logger
}

func NewPaymentUsecase(repo PaymentRepo, orders OrderClient, cfg *conf.Payment, logger *slog.Logger) *PaymentUsecase {
	on := true
	if cfg != nil {
		on = cfg.SimulateOn()
	}
	return &PaymentUsecase{repo: repo, orders: orders, simulate: on, logger: logger}
}

// 核对订单属于当前用户且待支付未过期 已有待支付单则直接返回 金额以订单为准
func (uc *PaymentUsecase) Create(ctx context.Context, userID uuid.UUID, orderID string) (*Payment, error) {
	orderID = strings.TrimSpace(orderID)
	if userID == uuid.Nil || orderID == "" {
		return nil, ErrPaymentInvalidArgument
	}
	order, err := uc.orders.GetOrder(ctx, userID, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil || order.UserID != userID || order.Status != OrderStatusPending || !time.Now().Before(order.ExpiresAt) {
		return nil, ErrPaymentOrderNotPayable
	}
	existing, err := uc.repo.FindByOrderID(ctx, orderID)
	if err != nil && !errors.Is(err, ErrPaymentNotFound) {
		return nil, err
	}
	if errors.Is(err, ErrPaymentNotFound) {
		created, cerr := uc.repo.Create(ctx, &Payment{
			OrderID: orderID,
			UserID:  userID,
			Amount:  order.PayAmount,
			Status:  StatusPending,
			Channel: ChannelMock,
		})
		if cerr == nil {
			return created, nil
		}
		if !errors.Is(cerr, ErrPaymentDuplicate) {
			return nil, cerr
		}
		existing, err = uc.repo.FindByOrderID(ctx, orderID)
		if err != nil {
			return nil, err
		}
	}
	if existing.UserID != userID {
		return nil, ErrPaymentOrderNotPayable
	}
	switch existing.Status {
	case StatusPending:
		return existing, nil
	case StatusFailed:
		return uc.repo.ResetFailed(ctx, existing.ID, order.PayAmount)
	default:
		return nil, ErrPaymentStatusConflict
	}
}

func (uc *PaymentUsecase) ListDiffs(ctx context.Context, day string, size, offset int) ([]Diff, bool, error) {
	return uc.repo.ListDiffs(ctx, day, size, offset)
}

func (uc *PaymentUsecase) Get(ctx context.Context, userID, id uuid.UUID) (*Payment, error) {
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, ErrPaymentInvalidArgument
	}
	p, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, ErrPaymentNotFound
	}
	return p, nil
}

// 只有待支付能变成成功或失败 重复回调不改状态
func (uc *PaymentUsecase) Simulate(ctx context.Context, userID, id uuid.UUID, success bool) (*Payment, error) {
	if !uc.simulate {
		return nil, ErrPaymentNotFound
	}
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, ErrPaymentInvalidArgument
	}
	current, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.UserID != userID {
		return nil, ErrPaymentNotFound
	}
	if current.Status != StatusPending {
		return current, nil
	}
	updated, changed, err := uc.repo.MarkResult(ctx, id, success)
	if err != nil {
		return nil, err
	}
	if !changed || updated.Status != StatusSuccess {
		return updated, nil
	}
	if nerr := uc.notifyPaid(ctx, updated); nerr != nil {
		uc.log(nerr, updated.ID)
	}
	fresh, ferr := uc.repo.FindByID(ctx, id)
	if ferr != nil {
		return updated, nil
	}
	return fresh, nil
}

func (uc *PaymentUsecase) RetryNotify(ctx context.Context) error {
	rows, err := uc.repo.ListUnnotified(ctx, 100)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if nerr := uc.notifyPaid(ctx, row); nerr != nil {
			uc.log(nerr, row.ID)
		}
	}
	return nil
}

// 通知订单已支付 若订单状态冲突则把支付单标成已退款
func (uc *PaymentUsecase) notifyPaid(ctx context.Context, p *Payment) error {
	err := uc.orders.MarkOrderPaid(ctx, p.OrderID, p.ID)
	if err == nil {
		return uc.repo.MarkNotified(ctx, p.ID)
	}
	if errors.Is(err, ErrOrderStatusConflict) {
		_, rerr := uc.repo.MarkRefunded(ctx, p.ID)
		return rerr
	}
	return err
}

func (uc *PaymentUsecase) log(err error, id uuid.UUID) {
	if uc.logger == nil || err == nil {
		return
	}
	uc.logger.Error("notify order paid", "err", err, "payment_id", id.String())
}
