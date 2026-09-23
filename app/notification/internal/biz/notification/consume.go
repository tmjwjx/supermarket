package notification

import (
	"context"
	"encoding/json"

	"github.com/tmjwjx/supermarket/pkg/kafkaout"

	"github.com/google/uuid"
)

// Accept 把订单和支付事件写成站内信 已处理过的事件直接跳过
func (uc *NotificationUsecase) Accept(ctx context.Context, ev kafkaout.Event) error {
	title, body := noteCopy(ev.Topic)
	if title == "" {
		return nil
	}
	var payload struct {
		OrderID   string `json:"order_id"`
		UserID    string `json:"user_id"`
		PaymentID string `json:"payment_id"`
	}
	if err := json.Unmarshal([]byte(ev.Payload), &payload); err != nil {
		return err
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return nil
	}
	orderID := payload.OrderID
	if orderID == "" {
		orderID = payload.PaymentID
	}
	return uc.repo.SaveOnce(ctx, ev.ID, &Notification{
		UserID: userID, Title: title, Body: body, OrderID: orderID,
	})
}

func noteCopy(topic string) (string, string) {
	switch topic {
	case "order.created":
		return "订单已创建", "你的订单已提交"
	case "order.paid":
		return "支付成功", "订单已支付"
	case "order.cancelled":
		return "订单已取消", "订单已取消"
	case "order.shipped":
		return "订单已发货", "商品已发出"
	case "order.completed":
		return "订单已完成", "订单已完成"
	case "payment.refunded":
		return "款项已退回", "支付已退回"
	default:
		return "", ""
	}
}
