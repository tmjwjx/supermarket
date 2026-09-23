package payment

import (
	"context"
	"log/slog"
	"strconv"
	"time"
)

// Reconcile 比对某一天已成功支付单和已支付订单 只记差异不改数据
func (uc *PaymentUsecase) Reconcile(ctx context.Context, day time.Time) error {
	if day.IsZero() {
		return ErrPaymentInvalidArgument
	}
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.Add(24 * time.Hour)
	pays, err := uc.repo.ListSuccessBetween(ctx, start, end)
	if err != nil {
		return err
	}
	orders, err := uc.orders.ListPaidOrders(ctx, start, end)
	if err != nil {
		return err
	}
	byOrder := make(map[string]*Payment, len(pays))
	for _, pay := range pays {
		if pay == nil {
			continue
		}
		byOrder[pay.OrderID] = pay
	}
	byID := make(map[string]PaidOrder, len(orders))
	for _, order := range orders {
		byID[order.ID] = order
	}
	logger := uc.logger
	if logger == nil {
		logger = slog.Default()
	}
	dayText := start.Format("2006-01-02")
	var diffs []Diff
	for id, pay := range byOrder {
		order, ok := byID[id]
		if !ok {
			logger.Error("reconcile mismatch", "day", dayText, "kind", "payment_without_order", "order_id", id, "payment_id", pay.ID.String(), "amount", pay.Amount)
			diffs = append(diffs, Diff{Day: dayText, Kind: "payment_without_order", OrderID: id, PaymentID: pay.ID.String(), Detail: "amount " + itoa(pay.Amount)})
			continue
		}
		if order.PaymentID != "" && order.PaymentID != pay.ID.String() {
			logger.Error("reconcile mismatch", "day", dayText, "kind", "payment_id", "order_id", id, "payment_id", pay.ID.String(), "order_payment_id", order.PaymentID)
			diffs = append(diffs, Diff{Day: dayText, Kind: "payment_id", OrderID: id, PaymentID: pay.ID.String(), Detail: "order payment " + order.PaymentID})
		}
		if order.Amount != pay.Amount {
			logger.Error("reconcile mismatch", "day", dayText, "kind", "amount", "order_id", id, "payment_amount", pay.Amount, "order_amount", order.Amount)
			diffs = append(diffs, Diff{Day: dayText, Kind: "amount", OrderID: id, PaymentID: pay.ID.String(), Detail: "payment " + itoa(pay.Amount) + " order " + itoa(order.Amount)})
		}
	}
	for id, order := range byID {
		if _, ok := byOrder[id]; ok {
			continue
		}
		logger.Error("reconcile mismatch", "day", dayText, "kind", "order_without_payment", "order_id", id, "payment_id", order.PaymentID, "amount", order.Amount)
		diffs = append(diffs, Diff{Day: dayText, Kind: "order_without_payment", OrderID: id, PaymentID: order.PaymentID, Detail: "amount " + itoa(order.Amount)})
	}
	if err := uc.repo.SaveDiffs(ctx, dayText, diffs); err != nil {
		return err
	}
	logger.Info("reconcile done", "day", dayText, "payments", len(byOrder), "orders", len(byID), "diffs", len(diffs))
	return nil
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

type ReconcileLoop struct{}

// NewReconcileLoop 每天凌晨 2 点对前一天 启动失败不会挡住进程
func NewReconcileLoop(uc *PaymentUsecase, logger *slog.Logger) (*ReconcileLoop, func()) {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			timer := time.NewTimer(time.Until(next))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				runCtx, runCancel := context.WithTimeout(ctx, time.Minute)
				if err := uc.Reconcile(runCtx, time.Now().Add(-24*time.Hour)); err != nil {
					logger.Error("reconcile payments", "err", err)
				}
				runCancel()
			}
		}
	}()
	return &ReconcileLoop{}, cancel
}
