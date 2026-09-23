package order

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/google/uuid"
)

type Sweeper struct{}

// NewSweeper 每分钟关闭支付超时的待支付订单并补偿库存
func NewSweeper(uc *OrderUsecase) (*Sweeper, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		run := func() {
			c, stop := context.WithTimeout(context.Background(), 30*time.Second)
			defer stop()
			if err := uc.Sweep(c); err != nil {
				log.Error("sweep orders", "err", err)
			}
		}
		run()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
	return &Sweeper{}, cancel
}

// Sweep 关超时单 补释放 补确认 并发货满 7 天自动收货
func (uc *OrderUsecase) Sweep(ctx context.Context) error {
	now := time.Now()
	expired, err := uc.orders.ListExpiredPending(ctx, now, scanBatch)
	if err != nil {
		return err
	}
	for _, o := range expired {
		if err := uc.closeExpired(ctx, o.ID); err != nil {
			log.Error("close expired order", "err", err, "order_id", o.ID.String())
		}
	}
	unreleased, err := uc.orders.ListUnreleased(ctx, scanBatch)
	if err != nil {
		return err
	}
	for _, o := range unreleased {
		_ = uc.releaseStock(ctx, o.ID)
	}
	unconfirmed, err := uc.orders.ListUnconfirmed(ctx, scanBatch)
	if err != nil {
		return err
	}
	for _, o := range unconfirmed {
		_ = uc.confirmStock(ctx, o)
	}
	due, err := uc.orders.ListShippedBefore(ctx, now.Add(-autoReceiveAfter), scanBatch)
	if err != nil {
		return err
	}
	for _, o := range due {
		if _, err := uc.finish(ctx, o.UserID, o.ID); err != nil && !errors.Is(err, ErrOrderStatusConflict) && !errors.Is(err, ErrOrderNotFound) {
			log.Error("auto receive order", "err", err, "order_id", o.ID.String())
		}
	}
	return nil
}

func (uc *OrderUsecase) closeExpired(ctx context.Context, id uuid.UUID) error {
	ok, err := uc.orders.CancelIfPending(ctx, id, CancelTimeout, time.Now())
	if err != nil || !ok {
		return err
	}
	return uc.releaseStock(ctx, id)
}
