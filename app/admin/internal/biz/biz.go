package biz

import (
	bizadmin "github.com/tmjwjx/supermarket/app/admin/internal/biz/adminuser"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(bizadmin.NewAdminUserUsecase)
