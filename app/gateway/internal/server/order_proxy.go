package server

import (
	orderv1 "github.com/tmjwjx/supermarket/api/order/v1"

	"github.com/go-kratos/kratos/v3/transport/http"
)

type orderProxy struct {
	gate
	carts  orderv1.CartServiceClient
	orders orderv1.OrderServiceClient
}

func (p *orderProxy) routes(r *http.Router) {
	// 登录
	r.POST("/v1/cart/items", p.requireLogin(p.addCartItem))
	// 登录
	r.PATCH("/v1/cart/items/{sku_id}", p.requireLogin(p.updateCartItem))
	// 登录
	r.GET("/v1/cart/items", p.requireLogin(p.listCartItems))
	// 登录
	r.POST("/v1/cart/items:batchDelete", p.requireLogin(p.removeCartItems))
	// 登录
	r.POST("/v1/orders", p.requireLogin(p.createOrder))
	// 登录
	r.GET("/v1/orders", p.requireLogin(p.listOrders))
	// 登录
	r.GET("/v1/orders/{id}", p.requireLogin(p.getOrder))
	// 登录
	r.POST("/v1/orders/{id}:cancel", p.requireLogin(p.cancelOrder))
	// 登录
	r.POST("/v1/orders/{id}:confirm", p.requireLogin(p.confirmOrder))
	// 后台
	r.GET("/v1/admin/orders", p.requireAdmin(p.adminListOrders))
	// 后台
	r.POST("/v1/admin/orders/{id}:ship", p.requireAdmin(p.adminShipOrder))
}

func (p *orderProxy) addCartItem(ctx http.Context) error {
	var in orderv1.AddCartItemRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, orderv1.OperationCartServiceAddCartItem, &in, rpc(p.carts.AddCartItem))
}

func (p *orderProxy) updateCartItem(ctx http.Context) error {
	var in orderv1.UpdateCartItemRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, orderv1.OperationCartServiceUpdateCartItem, &in, rpc(p.carts.UpdateCartItem))
}

func (p *orderProxy) listCartItems(ctx http.Context) error {
	var in orderv1.ListCartItemsRequest
	return call(ctx, orderv1.OperationCartServiceListCartItems, &in, rpc(p.carts.ListCartItems))
}

func (p *orderProxy) removeCartItems(ctx http.Context) error {
	var in orderv1.RemoveCartItemsRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, orderv1.OperationCartServiceRemoveCartItems, &in, rpc(p.carts.RemoveCartItems))
}

func (p *orderProxy) createOrder(ctx http.Context) error {
	var in orderv1.CreateOrderRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, orderv1.OperationOrderServiceCreateOrder, &in, rpc(p.orders.CreateOrder))
}

func (p *orderProxy) listOrders(ctx http.Context) error {
	var in orderv1.ListOrdersRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, orderv1.OperationOrderServiceListOrders, &in, rpc(p.orders.ListOrders))
}

func (p *orderProxy) getOrder(ctx http.Context) error {
	var in orderv1.GetOrderRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, orderv1.OperationOrderServiceGetOrder, &in, rpc(p.orders.GetOrder))
}

func (p *orderProxy) cancelOrder(ctx http.Context) error {
	var in orderv1.CancelOrderRequest
	if err := bindBody(ctx, &in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, orderv1.OperationOrderServiceCancelOrder, &in, rpc(p.orders.CancelOrder))
}

func (p *orderProxy) confirmOrder(ctx http.Context) error {
	var in orderv1.ConfirmOrderRequest
	if err := bindBody(ctx, &in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, orderv1.OperationOrderServiceConfirmOrder, &in, rpc(p.orders.ConfirmOrder))
}

func (p *orderProxy) adminListOrders(ctx http.Context) error {
	var in orderv1.AdminListOrdersRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, orderv1.OperationOrderServiceAdminListOrders, &in, rpc(p.orders.AdminListOrders))
}

func (p *orderProxy) adminShipOrder(ctx http.Context) error {
	var in orderv1.AdminShipOrderRequest
	if err := bindBody(ctx, &in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, orderv1.OperationOrderServiceAdminShipOrder, &in, rpc(p.orders.AdminShipOrder))
}
