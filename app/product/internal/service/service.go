package service

import (
	svcattribute "github.com/tmjwjx/supermarket/app/product/internal/service/attribute"
	svcbrand "github.com/tmjwjx/supermarket/app/product/internal/service/brand"
	svcbrowse "github.com/tmjwjx/supermarket/app/product/internal/service/browse"
	svccategory "github.com/tmjwjx/supermarket/app/product/internal/service/category"
	svcfavorite "github.com/tmjwjx/supermarket/app/product/internal/service/favorite"
	svcproduct "github.com/tmjwjx/supermarket/app/product/internal/service/product"
	svcrecommendation "github.com/tmjwjx/supermarket/app/product/internal/service/recommendation"
	svcreview "github.com/tmjwjx/supermarket/app/product/internal/service/review"

	"github.com/google/wire"
)

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
	svcbrand.NewBrandService,
	svccategory.NewCategoryService,
	svcattribute.NewAttributeService,
	svcproduct.NewProductService,
	svcrecommendation.NewRecommendationService,
	svcfavorite.NewFavoriteService,
	svcbrowse.NewBrowseHistoryService,
	svcreview.NewReviewService,
)
