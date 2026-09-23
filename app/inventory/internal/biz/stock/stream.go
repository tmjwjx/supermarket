package stock

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/tmjwjx/supermarket/pkg/kafkaout"
)

// Streams 消费商品发出的 SKU 创建事件并补库存记录
type Streams struct{}

func NewStreams(uc *StockUsecase) (*Streams, func()) {
	stop := kafkaout.StartConsumer("inventory-sku", []string{"product.sku.created"}, func(ctx context.Context, ev kafkaout.Event) error {
		id := strings.TrimSpace(ev.Key)
		var body struct {
			SkuID string `json:"sku_id"`
		}
		if err := json.Unmarshal([]byte(ev.Payload), &body); err == nil && strings.TrimSpace(body.SkuID) != "" {
			id = strings.TrimSpace(body.SkuID)
		}
		if id == "" {
			return errors.New("sku id missing")
		}
		_, err := uc.Create(ctx, id)
		return err
	})
	return &Streams{}, stop
}
