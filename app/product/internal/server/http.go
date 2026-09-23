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
	"github.com/tmjwjx/supermarket/pkg/httpauth"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/http"
)

func NewHTTPServer(
	c *conf.Server,
	auth *conf.Auth,
	brand *svcbrand.BrandService,
	category *svccategory.CategoryService,
	product *svcproduct.ProductService,
	recommendation *svcrecommendation.RecommendationService,
	favorite *svcfavorite.FavoriteService,
	browse *svcbrowse.BrowseHistoryService,
	review *svcreview.ReviewService,
	attribute *svcattribute.AttributeService,
) *http.Server {
	var userSecret, adminSecret string
	if auth != nil {
		userSecret, adminSecret = auth.JWTSecret, auth.AdminJWTSecret
	}
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			metadata.Server(),
			httpauth.Server(httpauth.Options{
				UserSecret:  userSecret,
				AdminSecret: adminSecret,
				Public: []string{
					v1.OperationBrandServiceListBrands,
					v1.OperationCategoryServiceListCategories,
					v1.OperationProductServiceListProducts,
					v1.OperationProductServiceSearchProducts,
					v1.OperationProductServiceGetProductDetail,
					v1.OperationRecommendationServiceListRecommendations,
					v1.OperationReviewServiceListProductReviews,
				},
				Optional: []string{v1.OperationProductServiceGetProduct},
			}),
		),
	}
	if c.HTTP.Network != "" {
		opts = append(opts, http.Network(c.HTTP.Network))
	}
	if c.HTTP.Addr != "" {
		opts = append(opts, http.Address(c.HTTP.Addr))
	}
	if d := c.HTTP.Timeout(); d != 0 {
		opts = append(opts, http.Timeout(d))
	}
	srv := http.NewServer(opts...)
	v1.RegisterBrandServiceHTTPServer(srv, brand)
	v1.RegisterCategoryServiceHTTPServer(srv, category)
	v1.RegisterProductServiceHTTPServer(srv, product)
	v1.RegisterRecommendationServiceHTTPServer(srv, recommendation)
	v1.RegisterFavoriteServiceHTTPServer(srv, favorite)
	v1.RegisterBrowseHistoryServiceHTTPServer(srv, browse)
	v1.RegisterReviewServiceHTTPServer(srv, review)
	v1.RegisterAttributeServiceHTTPServer(srv, attribute)
	return srv
}
