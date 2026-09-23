package server

import (
	paymentv1 "github.com/tmjwjx/supermarket/api/payment/v1"

	"github.com/go-kratos/kratos/v3/transport/http"
)

type paymentProxy struct {
	gate
	payments paymentv1.PaymentServiceClient
}

func (p *paymentProxy) routes(r *http.Router) {
	// 登录
	r.POST("/v1/payments", p.requireLogin(p.createPayment))
	// 登录
	r.GET("/v1/payments/{id}", p.requireLogin(p.getPayment))
	// 登录
	r.POST("/v1/payments/{id}:simulate", p.requireLogin(p.simulatePayment))
	// 后台
	r.GET("/v1/admin/payments/reconcile-diffs", p.requireAdmin(p.listReconcileDiffs))
}

func (p *paymentProxy) createPayment(ctx http.Context) error {
	var in paymentv1.CreatePaymentRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, paymentv1.OperationPaymentServiceCreatePayment, &in, rpc(p.payments.CreatePayment))
}

func (p *paymentProxy) getPayment(ctx http.Context) error {
	var in paymentv1.GetPaymentRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, paymentv1.OperationPaymentServiceGetPayment, &in, rpc(p.payments.GetPayment))
}

func (p *paymentProxy) listReconcileDiffs(ctx http.Context) error {
	var in paymentv1.ListReconcileDiffsRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, paymentv1.OperationPaymentServiceListReconcileDiffs, &in, rpc(p.payments.ListReconcileDiffs))
}

func (p *paymentProxy) simulatePayment(ctx http.Context) error {
	var in paymentv1.SimulatePaymentRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, paymentv1.OperationPaymentServiceSimulatePayment, &in, rpc(p.payments.SimulatePayment))
}
