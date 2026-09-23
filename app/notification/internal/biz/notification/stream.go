package notification

import (
	"context"

	"github.com/tmjwjx/supermarket/pkg/kafkaout"
)

// Streams 投递本库发件箱并消费订单与支付事件
type Streams struct{}

func NewStreams(store kafkaout.Store, uc *NotificationUsecase) (*Streams, func()) {
	stopPub := kafkaout.StartPublisher(store)
	stopSub := kafkaout.StartConsumer("notification", []string{
		"order.created", "order.paid", "order.cancelled", "order.shipped", "order.completed", "payment.refunded",
	}, func(ctx context.Context, ev kafkaout.Event) error {
		return uc.Accept(ctx, ev)
	})
	return &Streams{}, func() {
		stopPub()
		stopSub()
	}
}
