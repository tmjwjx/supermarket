package service

import (
	cartsvc "github.com/tmjwjx/supermarket/app/order/internal/service/cart"
	ordersvc "github.com/tmjwjx/supermarket/app/order/internal/service/order"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(cartsvc.NewCartService, ordersvc.NewOrderService)
