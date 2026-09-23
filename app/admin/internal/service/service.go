package service

import (
	adminsvc "github.com/tmjwjx/supermarket/app/admin/internal/service/adminuser"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(adminsvc.NewAdminUserService)
