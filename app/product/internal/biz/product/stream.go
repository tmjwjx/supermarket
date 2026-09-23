package product

import (
	"context"

	"github.com/tmjwjx/supermarket/pkg/kafkaout"
)

// Streams 拉起发件箱投递和订单完成消费
type Streams struct{}

func NewStreams(store kafkaout.Store, uc *ProductUsecase) (*Streams, func()) {
	stopPub := kafkaout.StartPublisher(store)
	stopSub := kafkaout.StartConsumer("product-sales", []string{"order.completed"}, func(ctx context.Context, ev kafkaout.Event) error {
		if ev.Topic != "order.completed" {
			return nil
		}
		return uc.AcceptCompleted(ctx, ev.ID, ev.Payload)
	})
	return &Streams{}, func() {
		stopPub()
		stopSub()
	}
}
