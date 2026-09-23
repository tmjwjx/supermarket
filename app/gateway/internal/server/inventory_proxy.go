package server

import (
	inventoryv1 "github.com/tmjwjx/supermarket/api/inventory/v1"

	"github.com/go-kratos/kratos/v3/transport/http"
)

type inventoryProxy struct {
	gate
	stocks inventoryv1.StockServiceClient
}

func (p *inventoryProxy) routes(r *http.Router) {
	// 公开
	r.GET("/v1/stocks", p.getStocks)
	// 后台
	r.GET("/v1/admin/stocks", p.requireAdmin(p.adminGetStocks))
	// 后台
	r.POST("/v1/admin/stocks:adjust", p.requireAdmin(p.adjustStock))
}

func (p *inventoryProxy) adminGetStocks(ctx http.Context) error {
	var in inventoryv1.AdminGetStocksRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, inventoryv1.OperationStockServiceAdminGetStocks, &in, rpc(p.stocks.AdminGetStocks))
}

func (p *inventoryProxy) adjustStock(ctx http.Context) error {
	var in inventoryv1.AdjustStockRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, inventoryv1.OperationStockServiceAdjustStock, &in, rpc(p.stocks.AdjustStock))
}

func (p *inventoryProxy) getStocks(ctx http.Context) error {
	var in inventoryv1.GetStocksRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, inventoryv1.OperationStockServiceGetStocks, &in, rpc(p.stocks.GetStocks))
}
