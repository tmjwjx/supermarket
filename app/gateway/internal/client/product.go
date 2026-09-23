package client

import (
	productv1 "github.com/tmjwjx/supermarket/api/product/v1"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"
)

// 同一条 product 连接上的目录收藏浏览和评价客户端
type ProductClients struct {
	Products        productv1.ProductServiceClient
	Brands          productv1.BrandServiceClient
	Categories      productv1.CategoryServiceClient
	Favorites       productv1.FavoriteServiceClient
	Histories       productv1.BrowseHistoryServiceClient
	Reviews         productv1.ReviewServiceClient
	Recommendations productv1.RecommendationServiceClient
	Attributes      productv1.AttributeServiceClient
}

// 拨到 product 的 gRPC 供目录收藏浏览和评价转发
func NewProductClient(c *conf.Client) (*ProductClients, func(), error) {
	conn, err := dial(c.Product)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = conn.Close() }
	return &ProductClients{
		Products:        productv1.NewProductServiceClient(conn),
		Brands:          productv1.NewBrandServiceClient(conn),
		Categories:      productv1.NewCategoryServiceClient(conn),
		Favorites:       productv1.NewFavoriteServiceClient(conn),
		Histories:       productv1.NewBrowseHistoryServiceClient(conn),
		Reviews:         productv1.NewReviewServiceClient(conn),
		Recommendations: productv1.NewRecommendationServiceClient(conn),
		Attributes:      productv1.NewAttributeServiceClient(conn),
	}, cleanup, nil
}
