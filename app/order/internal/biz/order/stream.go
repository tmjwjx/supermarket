package order

import (
	"github.com/tmjwjx/supermarket/pkg/kafkaout"
)

// Streams 投递订单发件箱 连不上 Kafka 时只记日志
type Streams struct{}

func NewStreams(store kafkaout.Store) (*Streams, func()) {
	stop := kafkaout.StartPublisher(store)
	return &Streams{}, stop
}
