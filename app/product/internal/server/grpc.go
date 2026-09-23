package server

import (
	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	"github.com/tmjwjx/supermarket/app/product/internal/conf"
	svcattribute "github.com/tmjwjx/supermarket/app/product/internal/service/attribute"
	svcbrand "github.com/tmjwjx/supermarket/app/product/internal/service/brand"
	svcbrowse "github.com/tmjwjx/supermarket/app/product/internal/service/browse"
	svccategory "github.com/tmjwjx/supermarket/app/product/internal/service/category"
	svcfavorite "github.com/tmjwjx/supermarket/app/product/internal/service/favorite"
	svcproduct "github.com/tmjwjx/supermarket/app/product/internal/service/product"
	svcrecommendation "github.com/tmjwjx/supermarket/app/product/internal/service/recommendation"
	svcreview "github.com/tmjwjx/supermarket/app/product/internal/service/review"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/grpc"
)

func NewGRPCServer(
	c *conf.Server,
	brand *svcbrand.BrandService,
	category *svccategory.CategoryService,
	product *svcproduct.ProductService,
	recommendation *svcrecommendation.RecommendationService,
	favorite *svcfavorite.FavoriteService,
	browse *svcbrowse.BrowseHistoryService,
	review *svcreview.ReviewService,
	attribute *svcattribute.AttributeService,
) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			metadata.Server(),
		),
	}
	if c.GRPC.Network != "" {
		opts = append(opts, grpc.Network(c.GRPC.Network))
	}
	if c.GRPC.Addr != "" {
		opts = append(opts, grpc.Address(c.GRPC.Addr))
	}
	if d := c.GRPC.Timeout(); d != 0 {
		opts = append(opts, grpc.Timeout(d))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterBrandServiceServer(srv, brand)
	v1.RegisterCategoryServiceServer(srv, category)
	v1.RegisterProductServiceServer(srv, product)
	v1.RegisterRecommendationServiceServer(srv, recommendation)
	v1.RegisterFavoriteServiceServer(srv, favorite)
	v1.RegisterBrowseHistoryServiceServer(srv, browse)
	v1.RegisterReviewServiceServer(srv, review)
	v1.RegisterAttributeServiceServer(srv, attribute)
	return srv
}
