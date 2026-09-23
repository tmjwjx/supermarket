package service

import (
	notesvc "github.com/tmjwjx/supermarket/app/notification/internal/service/notification"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(notesvc.NewNotificationService)
