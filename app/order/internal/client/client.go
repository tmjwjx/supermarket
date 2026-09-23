package client

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewProductAPI,
	NewInventoryAPI,
	NewUserAPI,
	NewCartProducts,
	NewCartStock,
	NewOrderCatalog,
	NewOrderStock,
	NewAddresses,
)
