package stock

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v3/log"
)

type Sweeper struct{}

// NewSweeper 每分钟释放过期仍占住的预占
func NewSweeper(uc *StockUsecase) (*Sweeper, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		run := func() {
			c, stop := context.WithTimeout(context.Background(), 30*time.Second)
			defer stop()
			if err := uc.ReleaseExpired(c); err != nil {
				log.Error("release expired reservations", "err", err)
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
