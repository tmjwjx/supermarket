package biz

import (
	bizuser "github.com/tmjwjx/supermarket/app/user/internal/biz/user"

	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(bizuser.NewUserUsecase, bizuser.NewAddressUsecase)
