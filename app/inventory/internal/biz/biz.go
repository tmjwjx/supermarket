package biz

import (
	bizstock "github.com/tmjwjx/supermarket/app/inventory/internal/biz/stock"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(bizstock.NewStockUsecase, bizstock.NewSweeper, bizstock.NewStreams)
