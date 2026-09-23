package biz

import (
	bizcart "github.com/tmjwjx/supermarket/app/order/internal/biz/cart"
	bizorder "github.com/tmjwjx/supermarket/app/order/internal/biz/order"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	bizcart.NewCartUsecase,
	bizorder.NewOrderUsecase,
	bizorder.NewSweeper,
	bizorder.NewStreams,
	NewCartCleaner,
)

func NewCartCleaner(repo bizcart.CartRepo) bizorder.CartCleaner {
	return repo
}
