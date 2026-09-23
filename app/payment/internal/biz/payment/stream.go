package payment

import "github.com/tmjwjx/supermarket/pkg/kafkaout"

// Streams 投递支付发件箱
type Streams struct{}

func NewStreams(store kafkaout.Store) (*Streams, func()) {
	stop := kafkaout.StartPublisher(store)
	return &Streams{}, stop
}
