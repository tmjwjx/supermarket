package service

import (
	usersvc "github.com/tmjwjx/supermarket/app/user/internal/service/user"

	"github.com/google/wire"
)

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(usersvc.NewUserService, usersvc.NewAddressService)
