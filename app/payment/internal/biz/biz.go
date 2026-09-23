package biz

import (
	bizpayment "github.com/tmjwjx/supermarket/app/payment/internal/biz/payment"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(bizpayment.NewPaymentUsecase, bizpayment.NewRetryLoop, bizpayment.NewReconcileLoop, bizpayment.NewStreams)
