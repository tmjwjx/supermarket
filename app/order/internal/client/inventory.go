package client

import (
	"context"

	inventoryv1 "github.com/tmjwjx/supermarket/api/inventory/v1"
	orderv1 "github.com/tmjwjx/supermarket/api/order/v1"
	"github.com/tmjwjx/supermarket/app/order/internal/biz/cart"
	"github.com/tmjwjx/supermarket/app/order/internal/biz/order"
	"github.com/tmjwjx/supermarket/app/order/internal/conf"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

type inventoryAPI struct {
	cli inventoryv1.StockServiceClient
}

func NewInventoryAPI(c *conf.Client) (*inventoryAPI, func(), error) {
	conn, err := dial(c.Inventory.Addr, "127.0.0.1:9002", c.Inventory.Timeout())
	if err != nil {
		return nil, nil, err
	}
	return &inventoryAPI{cli: inventoryv1.NewStockServiceClient(conn)}, func() { _ = conn.Close() }, nil
}

func NewCartStock(api *inventoryAPI) cart.StockGateway { return cartStock{api} }

func NewOrderStock(api *inventoryAPI) order.Stocker { return orderStock{api} }

type cartStock struct{ api *inventoryAPI }

func (c cartStock) GetAvailable(ctx context.Context, skuIDs []string) (map[string]int64, error) {
	resp, err := c.api.cli.GetStocks(ctx, &inventoryv1.GetStocksRequest{SkuIds: skuIDs})
	if err != nil {
		return nil, cart.ErrUpstream
	}
	out := make(map[string]int64, len(resp.GetStocks()))
	for _, row := range resp.GetStocks() {
		if row == nil {
			continue
		}
		out[row.GetSkuId()] = row.GetAvailable()
	}
	return out, nil
}

type orderStock struct{ api *inventoryAPI }

func (o orderStock) Reserve(ctx context.Context, orderID string, lines []order.StockLine, expireUnix int64) error {
	items := make([]*inventoryv1.StockItem, 0, len(lines))
	for _, ln := range lines {
		items = append(items, &inventoryv1.StockItem{SkuId: ln.SkuID, Quantity: ln.Quantity})
	}
	_, err := o.api.cli.ReserveStock(ctx, &inventoryv1.ReserveStockRequest{
		OrderId: orderID, Items: items, ExpireUnix: expireUnix,
	})
	if err == nil {
		return nil
	}
	se := kerrors.FromError(err)
	if se.Reason == inventoryv1.ErrorReason_STOCK_INSUFFICIENT.String() {
		return kerrors.Conflict(orderv1.ErrorReason_ORDER_STOCK_INSUFFICIENT.String(), se.Message)
	}
	return order.ErrUpstream
}

func (o orderStock) Confirm(ctx context.Context, orderID string) error {
	_, err := o.api.cli.ConfirmReservation(ctx, &inventoryv1.ConfirmReservationRequest{OrderId: orderID})
	return mapReservation(err)
}

func (o orderStock) Release(ctx context.Context, orderID string) error {
	_, err := o.api.cli.ReleaseReservation(ctx, &inventoryv1.ReleaseReservationRequest{OrderId: orderID})
	if err == nil {
		return nil
	}
	if kerrors.Reason(err) == inventoryv1.ErrorReason_RESERVATION_NOT_FOUND.String() {
		return nil
	}
	return mapReservation(err)
}

func mapReservation(err error) error {
	if err == nil {
		return nil
	}
	switch kerrors.Reason(err) {
	case inventoryv1.ErrorReason_RESERVATION_NOT_FOUND.String():
		return order.ErrReservationMissing
	case inventoryv1.ErrorReason_RESERVATION_STATUS_CONFLICT.String():
		return order.ErrOrderStatusConflict
	default:
		return order.ErrUpstream
	}
}
