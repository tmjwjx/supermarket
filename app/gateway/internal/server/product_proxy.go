package server

import (
	productv1 "github.com/tmjwjx/supermarket/api/product/v1"

	"github.com/go-kratos/kratos/v3/transport/http"
)

type productProxy struct {
	gate
	products        productv1.ProductServiceClient
	brands          productv1.BrandServiceClient
	categories      productv1.CategoryServiceClient
	favorites       productv1.FavoriteServiceClient
	histories       productv1.BrowseHistoryServiceClient
	reviews         productv1.ReviewServiceClient
	recommendations productv1.RecommendationServiceClient
	attributes      productv1.AttributeServiceClient
}

func (p *productProxy) routes(r *http.Router) {
	// 公开
	r.GET("/v1/brands", p.listBrands)
	// 公开
	r.GET("/v1/categories", p.listCategories)
	// 公开 列表条件原样转发 契约里没有状态过滤
	r.GET("/v1/products", p.listProducts)
	// 公开
	r.GET("/v1/products:search", p.searchProducts)
	// 公开
	r.GET("/v1/products/{id}/detail", p.getProductDetail)
	// 可选
	r.GET("/v1/products/{id}", p.optionalLogin(p.getProduct))
	// 公开
	r.GET("/v1/recommendations", p.listRecommendations)
	// 公开
	r.GET("/v1/products/{product_id}/reviews", p.listProductReviews)
	// 登录
	r.POST("/v1/favorites", p.requireLogin(p.addFavorite))
	// 登录
	r.DELETE("/v1/favorites/{product_id}", p.requireLogin(p.removeFavorite))
	// 登录
	r.GET("/v1/favorites", p.requireLogin(p.listFavorites))
	// 登录
	r.GET("/v1/browse-histories", p.requireLogin(p.listBrowseHistories))
	// 登录
	r.DELETE("/v1/browse-histories", p.requireLogin(p.clearBrowseHistories))
	// 登录
	r.POST("/v1/reviews", p.requireLogin(p.createReview))
	// 后台
	r.POST("/v1/admin/products", p.requireAdmin(p.createProduct))
	// 后台
	r.PUT("/v1/admin/products/{id}", p.requireAdmin(p.updateProduct))
	// 后台
	r.GET("/v1/admin/products", p.requireAdmin(p.adminListProducts))
	// 后台
	r.GET("/v1/admin/products:deleted", p.requireAdmin(p.listDeletedProducts))
	// 后台
	r.GET("/v1/admin/products/{id}", p.requireAdmin(p.adminGetProduct))
	// 后台
	r.POST("/v1/admin/products/{id}:submit", p.requireAdmin(p.submitProduct))
	// 后台
	r.POST("/v1/admin/products/{id}:approve", p.requireAdmin(p.approveProduct))
	// 后台
	r.POST("/v1/admin/products/{id}:reject", p.requireAdmin(p.rejectProduct))
	// 后台
	r.POST("/v1/admin/products/{id}:publish", p.requireAdmin(p.publishProduct))
	// 后台
	r.POST("/v1/admin/products/{id}:unpublish", p.requireAdmin(p.unpublishProduct))
	// 后台
	r.DELETE("/v1/admin/products/{id}", p.requireAdmin(p.deleteProduct))
	// 后台
	r.POST("/v1/admin/products/{id}:restore", p.requireAdmin(p.restoreProduct))
	// 后台
	r.POST("/v1/admin/products/{id}:purge", p.requireAdmin(p.purgeProduct))
	// 后台
	r.POST("/v1/admin/products:batchPublish", p.requireAdmin(p.batchPublishProducts))
	// 后台
	r.POST("/v1/admin/products:batchUnpublish", p.requireAdmin(p.batchUnpublishProducts))
	// 后台
	r.POST("/v1/admin/products/{product_id}/skus/{sku_id}:price", p.requireAdmin(p.updateSkuPrice))
	// 后台
	r.GET("/v1/admin/products/{product_id}/audits", p.requireAdmin(p.listProductAudits))
	// 后台
	r.GET("/v1/admin/product-logs", p.requireAdmin(p.listProductLogs))
	// 后台
	r.GET("/v1/admin/brands", p.requireAdmin(p.adminListBrands))
	// 后台
	r.POST("/v1/admin/brands", p.requireAdmin(p.createBrand))
	// 后台
	r.PATCH("/v1/admin/brands/{id}", p.requireAdmin(p.updateBrand))
	// 后台
	r.DELETE("/v1/admin/brands/{id}", p.requireAdmin(p.deleteBrand))
	// 后台
	r.GET("/v1/admin/categories", p.requireAdmin(p.adminListCategories))
	// 后台
	r.POST("/v1/admin/categories", p.requireAdmin(p.createCategory))
	// 后台
	r.PATCH("/v1/admin/categories/{id}", p.requireAdmin(p.updateCategory))
	// 后台
	r.DELETE("/v1/admin/categories/{id}", p.requireAdmin(p.deleteCategory))
	// 后台
	r.GET("/v1/admin/recommendations", p.requireAdmin(p.adminListRecommendations))
	// 后台
	r.POST("/v1/admin/recommendations", p.requireAdmin(p.createRecommendation))
	// 后台
	r.PATCH("/v1/admin/recommendations/{id}", p.requireAdmin(p.updateRecommendation))
	// 后台
	r.DELETE("/v1/admin/recommendations/{id}", p.requireAdmin(p.deleteRecommendation))
	// 后台
	r.GET("/v1/admin/attribute-templates", p.requireAdmin(p.listAttributeTemplates))
	// 后台
	r.POST("/v1/admin/attribute-templates", p.requireAdmin(p.createAttributeTemplate))
	// 后台
	r.DELETE("/v1/admin/attribute-templates/{id}", p.requireAdmin(p.deleteAttributeTemplate))
	// 后台
	r.GET("/v1/admin/attribute-templates/{template_id}/attributes", p.requireAdmin(p.listAttributes))
	// 后台
	r.POST("/v1/admin/attribute-templates/{template_id}/attributes", p.requireAdmin(p.createAttribute))
	// 后台
	r.PATCH("/v1/admin/attributes/{id}", p.requireAdmin(p.updateAttribute))
	// 后台
	r.DELETE("/v1/admin/attributes/{id}", p.requireAdmin(p.deleteAttribute))
}

func (p *productProxy) listBrands(ctx http.Context) error {
	var in productv1.ListBrandsRequest
	return call(ctx, productv1.OperationBrandServiceListBrands, &in, rpc(p.brands.ListBrands))
}

func (p *productProxy) listCategories(ctx http.Context) error {
	var in productv1.ListCategoriesRequest
	return call(ctx, productv1.OperationCategoryServiceListCategories, &in, rpc(p.categories.ListCategories))
}

func (p *productProxy) listProducts(ctx http.Context) error {
	var in productv1.ListProductsRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceListProducts, &in, rpc(p.products.ListProducts))
}

func (p *productProxy) searchProducts(ctx http.Context) error {
	var in productv1.SearchProductsRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceSearchProducts, &in, rpc(p.products.SearchProducts))
}

func (p *productProxy) getProductDetail(ctx http.Context) error {
	var in productv1.GetProductDetailRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceGetProductDetail, &in, rpc(p.products.GetProductDetail))
}

func (p *productProxy) getProduct(ctx http.Context) error {
	var in productv1.GetProductRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceGetProduct, &in, rpc(p.products.GetProduct))
}

func (p *productProxy) listRecommendations(ctx http.Context) error {
	var in productv1.ListRecommendationsRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationRecommendationServiceListRecommendations, &in, rpc(p.recommendations.ListRecommendations))
}

func (p *productProxy) listProductReviews(ctx http.Context) error {
	var in productv1.ListProductReviewsRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationReviewServiceListProductReviews, &in, rpc(p.reviews.ListProductReviews))
}

func (p *productProxy) addFavorite(ctx http.Context) error {
	var in productv1.AddFavoriteRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationFavoriteServiceAddFavorite, &in, rpc(p.favorites.AddFavorite))
}

func (p *productProxy) removeFavorite(ctx http.Context) error {
	var in productv1.RemoveFavoriteRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationFavoriteServiceRemoveFavorite, &in, rpc(p.favorites.RemoveFavorite))
}

func (p *productProxy) listFavorites(ctx http.Context) error {
	var in productv1.ListFavoritesRequest
	return call(ctx, productv1.OperationFavoriteServiceListFavorites, &in, rpc(p.favorites.ListFavorites))
}

func (p *productProxy) listBrowseHistories(ctx http.Context) error {
	var in productv1.ListBrowseHistoriesRequest
	return call(ctx, productv1.OperationBrowseHistoryServiceListBrowseHistories, &in, rpc(p.histories.ListBrowseHistories))
}

func (p *productProxy) clearBrowseHistories(ctx http.Context) error {
	var in productv1.ClearBrowseHistoriesRequest
	return call(ctx, productv1.OperationBrowseHistoryServiceClearBrowseHistories, &in, rpc(p.histories.ClearBrowseHistories))
}

func (p *productProxy) createReview(ctx http.Context) error {
	var in productv1.CreateReviewRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationReviewServiceCreateReview, &in, rpc(p.reviews.CreateReview))
}

func (p *productProxy) createProduct(ctx http.Context) error {
	var in productv1.CreateProductRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceCreateProduct, &in, rpc(p.products.CreateProduct))
}

func (p *productProxy) updateProduct(ctx http.Context) error {
	var in productv1.UpdateProductRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceUpdateProduct, &in, rpc(p.products.UpdateProduct))
}

func (p *productProxy) listProductAudits(ctx http.Context) error {
	var in productv1.ListProductAuditsRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceListProductAudits, &in, rpc(p.products.ListProductAudits))
}

func (p *productProxy) listProductLogs(ctx http.Context) error {
	var in productv1.ListProductLogsRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceListProductLogs, &in, rpc(p.products.ListProductLogs))
}

func (p *productProxy) adminListBrands(ctx http.Context) error {
	var in productv1.AdminListBrandsRequest
	return call(ctx, productv1.OperationBrandServiceAdminListBrands, &in, rpc(p.brands.AdminListBrands))
}

func (p *productProxy) createBrand(ctx http.Context) error {
	var in productv1.CreateBrandRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationBrandServiceCreateBrand, &in, rpc(p.brands.CreateBrand))
}

func (p *productProxy) updateBrand(ctx http.Context) error {
	var in productv1.UpdateBrandRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationBrandServiceUpdateBrand, &in, rpc(p.brands.UpdateBrand))
}

func (p *productProxy) deleteBrand(ctx http.Context) error {
	var in productv1.DeleteBrandRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationBrandServiceDeleteBrand, &in, rpc(p.brands.DeleteBrand))
}

func (p *productProxy) createCategory(ctx http.Context) error {
	var in productv1.CreateCategoryRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationCategoryServiceCreateCategory, &in, rpc(p.categories.CreateCategory))
}

func (p *productProxy) deleteCategory(ctx http.Context) error {
	var in productv1.DeleteCategoryRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationCategoryServiceDeleteCategory, &in, rpc(p.categories.DeleteCategory))
}

func (p *productProxy) createRecommendation(ctx http.Context) error {
	var in productv1.CreateRecommendationRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationRecommendationServiceCreateRecommendation, &in, rpc(p.recommendations.CreateRecommendation))
}

func (p *productProxy) adminListProducts(ctx http.Context) error {
	var in productv1.AdminListProductsRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceAdminListProducts, &in, rpc(p.products.AdminListProducts))
}

func (p *productProxy) listDeletedProducts(ctx http.Context) error {
	var in productv1.ListDeletedProductsRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceListDeletedProducts, &in, rpc(p.products.ListDeletedProducts))
}

func (p *productProxy) submitProduct(ctx http.Context) error {
	var in productv1.SubmitProductRequest
	if err := bindBody(ctx, &in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceSubmitProduct, &in, rpc(p.products.SubmitProduct))
}

func (p *productProxy) approveProduct(ctx http.Context) error {
	var in productv1.ApproveProductRequest
	if err := bindBody(ctx, &in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceApproveProduct, &in, rpc(p.products.ApproveProduct))
}

func (p *productProxy) rejectProduct(ctx http.Context) error {
	var in productv1.RejectProductRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceRejectProduct, &in, rpc(p.products.RejectProduct))
}

func (p *productProxy) publishProduct(ctx http.Context) error {
	var in productv1.PublishProductRequest
	if err := bindBody(ctx, &in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServicePublishProduct, &in, rpc(p.products.PublishProduct))
}

func (p *productProxy) unpublishProduct(ctx http.Context) error {
	var in productv1.UnpublishProductRequest
	if err := bindBody(ctx, &in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceUnpublishProduct, &in, rpc(p.products.UnpublishProduct))
}

func (p *productProxy) deleteProduct(ctx http.Context) error {
	var in productv1.DeleteProductRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceDeleteProduct, &in, rpc(p.products.DeleteProduct))
}

func (p *productProxy) restoreProduct(ctx http.Context) error {
	var in productv1.RestoreProductRequest
	if err := bindBody(ctx, &in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceRestoreProduct, &in, rpc(p.products.RestoreProduct))
}

func (p *productProxy) purgeProduct(ctx http.Context) error {
	var in productv1.PurgeProductRequest
	if err := bindBody(ctx, &in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServicePurgeProduct, &in, rpc(p.products.PurgeProduct))
}

func (p *productProxy) batchPublishProducts(ctx http.Context) error {
	var in productv1.BatchPublishProductsRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceBatchPublishProducts, &in, rpc(p.products.BatchPublishProducts))
}

func (p *productProxy) batchUnpublishProducts(ctx http.Context) error {
	var in productv1.BatchUnpublishProductsRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceBatchUnpublishProducts, &in, rpc(p.products.BatchUnpublishProducts))
}

func (p *productProxy) updateSkuPrice(ctx http.Context) error {
	var in productv1.UpdateSkuPriceRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceUpdateSkuPrice, &in, rpc(p.products.UpdateSkuPrice))
}

func (p *productProxy) listAttributeTemplates(ctx http.Context) error {
	var in productv1.ListAttributeTemplatesRequest
	return call(ctx, productv1.OperationAttributeServiceListAttributeTemplates, &in, rpc(p.attributes.ListAttributeTemplates))
}

func (p *productProxy) createAttributeTemplate(ctx http.Context) error {
	var in productv1.CreateAttributeTemplateRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationAttributeServiceCreateAttributeTemplate, &in, rpc(p.attributes.CreateAttributeTemplate))
}

func (p *productProxy) deleteAttributeTemplate(ctx http.Context) error {
	var in productv1.DeleteAttributeTemplateRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationAttributeServiceDeleteAttributeTemplate, &in, rpc(p.attributes.DeleteAttributeTemplate))
}

func (p *productProxy) listAttributes(ctx http.Context) error {
	var in productv1.ListAttributesRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationAttributeServiceListAttributes, &in, rpc(p.attributes.ListAttributes))
}

func (p *productProxy) createAttribute(ctx http.Context) error {
	var in productv1.CreateAttributeRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationAttributeServiceCreateAttribute, &in, rpc(p.attributes.CreateAttribute))
}

func (p *productProxy) deleteAttribute(ctx http.Context) error {
	var in productv1.DeleteAttributeRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationAttributeServiceDeleteAttribute, &in, rpc(p.attributes.DeleteAttribute))
}

func (p *productProxy) adminGetProduct(ctx http.Context) error {
	var in productv1.AdminGetProductRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationProductServiceAdminGetProduct, &in, rpc(p.products.AdminGetProduct))
}

func (p *productProxy) adminListCategories(ctx http.Context) error {
	var in productv1.AdminListCategoriesRequest
	return call(ctx, productv1.OperationCategoryServiceAdminListCategories, &in, rpc(p.categories.AdminListCategories))
}

func (p *productProxy) updateCategory(ctx http.Context) error {
	var in productv1.UpdateCategoryRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationCategoryServiceUpdateCategory, &in, rpc(p.categories.UpdateCategory))
}

func (p *productProxy) updateAttribute(ctx http.Context) error {
	var in productv1.UpdateAttributeRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationAttributeServiceUpdateAttribute, &in, rpc(p.attributes.UpdateAttribute))
}

func (p *productProxy) adminListRecommendations(ctx http.Context) error {
	var in productv1.AdminListRecommendationsRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationRecommendationServiceAdminListRecommendations, &in, rpc(p.recommendations.AdminListRecommendations))
}

func (p *productProxy) updateRecommendation(ctx http.Context) error {
	var in productv1.UpdateRecommendationRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationRecommendationServiceUpdateRecommendation, &in, rpc(p.recommendations.UpdateRecommendation))
}

func (p *productProxy) deleteRecommendation(ctx http.Context) error {
	var in productv1.DeleteRecommendationRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, productv1.OperationRecommendationServiceDeleteRecommendation, &in, rpc(p.recommendations.DeleteRecommendation))
}
