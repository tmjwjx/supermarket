package biz

import (
	biznote "github.com/tmjwjx/supermarket/app/notification/internal/biz/notification"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(biznote.NewNotificationUsecase, biznote.NewStreams)
