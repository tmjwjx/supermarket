package client

import (
	"context"

	productv1 "github.com/tmjwjx/supermarket/api/product/v1"
	"github.com/tmjwjx/supermarket/app/order/internal/biz/cart"
	"github.com/tmjwjx/supermarket/app/order/internal/biz/order"
	"github.com/tmjwjx/supermarket/app/order/internal/conf"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

type productAPI struct {
	cli productv1.ProductServiceClient
}

type skuSnap struct {
	SkuID       string
	ProductID   string
	ProductName string
	SpecsJSON   string
	Price       int64
	Image       string
	Sellable    bool
}

func NewProductAPI(c *conf.Client) (*productAPI, func(), error) {
	conn, err := dial(c.Product.Addr, "127.0.0.1:9001", c.Product.Timeout())
	if err != nil {
		return nil, nil, err
	}
	return &productAPI{cli: productv1.NewProductServiceClient(conn)}, func() { _ = conn.Close() }, nil
}

func NewCartProducts(api *productAPI) cart.ProductGateway { return cartProduct{api} }

func NewOrderCatalog(api *productAPI) order.Catalog { return orderCatalog{api} }

type cartProduct struct{ api *productAPI }

func (c cartProduct) CheckSellable(ctx context.Context, skuIDs []string) ([]cart.SkuView, error) {
	rows, err := c.api.check(ctx, skuIDs)
	if err != nil {
		return nil, err
	}
	return toCartViews(rows), nil
}

func (c cartProduct) BatchGetSkus(ctx context.Context, skuIDs []string) ([]cart.SkuView, error) {
	rows, err := c.api.batch(ctx, skuIDs)
	if err != nil {
		return nil, cart.ErrUpstream
	}
	return toCartViews(rows), nil
}

type orderCatalog struct{ api *productAPI }

func (c orderCatalog) BatchGetSkus(ctx context.Context, skuIDs []string) ([]order.SkuSnap, error) {
	rows, err := c.api.batch(ctx, skuIDs)
	if err != nil {
		return nil, mapOrderErr(err)
	}
	out := make([]order.SkuSnap, 0, len(rows))
	for _, row := range rows {
		out = append(out, order.SkuSnap{
			SkuID: row.SkuID, ProductID: row.ProductID, ProductName: row.ProductName,
			SpecsJSON: row.SpecsJSON, Price: row.Price, Image: row.Image, Sellable: row.Sellable,
		})
	}
	return out, nil
}

func (c orderCatalog) IncreaseSales(ctx context.Context, orderID, productID string, count int64) error {
	_, err := c.api.cli.IncreaseSales(ctx, &productv1.IncreaseSalesRequest{ProductId: productID, Count: count, OrderId: orderID})
	if err != nil {
		return mapOrderErr(err)
	}
	return nil
}

func (p *productAPI) check(ctx context.Context, skuIDs []string) ([]skuSnap, error) {
	resp, err := p.cli.CheckSellable(ctx, &productv1.CheckSellableRequest{SkuIds: skuIDs})
	if err != nil {
		if kerrors.Reason(err) == productv1.ErrorReason_PRODUCT_NOT_SELLABLE.String() {
			return nil, cart.ErrProductNotSellable
		}
		return nil, cart.ErrUpstream
	}
	return fromProto(resp.GetSkus()), nil
}

func (p *productAPI) batch(ctx context.Context, skuIDs []string) ([]skuSnap, error) {
	resp, err := p.cli.BatchGetSkus(ctx, &productv1.BatchGetSkusRequest{SkuIds: skuIDs})
	if err != nil {
		return nil, err
	}
	return fromProto(resp.GetSkus()), nil
}

func fromProto(in []*productv1.SkuSnapshot) []skuSnap {
	out := make([]skuSnap, 0, len(in))
	for _, row := range in {
		if row == nil {
			continue
		}
		out = append(out, skuSnap{
			SkuID: row.GetSkuId(), ProductID: row.GetProductId(), ProductName: row.GetProductName(),
			SpecsJSON: row.GetSpecsJson(), Price: row.GetPrice(), Image: row.GetImage(), Sellable: row.GetSellable(),
		})
	}
	return out
}

func toCartViews(rows []skuSnap) []cart.SkuView {
	out := make([]cart.SkuView, 0, len(rows))
	for _, row := range rows {
		out = append(out, cart.SkuView{
			SkuID: row.SkuID, ProductName: row.ProductName, SpecsJSON: row.SpecsJSON,
			Price: row.Price, Image: row.Image, Sellable: row.Sellable,
		})
	}
	return out
}

func mapOrderErr(err error) error {
	if err == nil {
		return nil
	}
	if kerrors.Reason(err) == productv1.ErrorReason_PRODUCT_NOT_SELLABLE.String() {
		return order.ErrOrderItemNotSellable
	}
	return order.ErrUpstream
}
