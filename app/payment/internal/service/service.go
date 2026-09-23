package service

import (
	paymentsvc "github.com/tmjwjx/supermarket/app/payment/internal/service/payment"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(paymentsvc.NewPaymentService)
