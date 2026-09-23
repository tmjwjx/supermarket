package service

import (
	stocksvc "github.com/tmjwjx/supermarket/app/inventory/internal/service/stock"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(stocksvc.NewStockService)
