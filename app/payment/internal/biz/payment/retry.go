package payment

import (
	"context"
	"log/slog"
	"time"
)

type RetryLoop struct{}

// 每分钟重试已成功但尚未通知订单的支付单
func NewRetryLoop(uc *PaymentUsecase, logger *slog.Logger) (*RetryLoop, func()) {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCtx, runCancel := context.WithTimeout(ctx, 30*time.Second)
				if err := uc.RetryNotify(runCtx); err != nil {
					logger.Error("retry payment notify", "err", err)
				}
				runCancel()
			}
		}
	}()
	return &RetryLoop{}, cancel
}
