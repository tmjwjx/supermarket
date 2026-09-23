package biz

import (
	"context"

	"github.com/tmjwjx/supermarket/app/product/internal/biz/attribute"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/brand"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/browse"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/category"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/favorite"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/product"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/recommendation"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/review"

	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	brand.NewBrandUsecase,
	category.NewCategoryUsecase,
	attribute.NewAttributeUsecase,
	ProvideProductUsecase,
	product.NewStreams,
	recommendation.NewRecommendationUsecase,
	favorite.NewFavoriteUsecase,
	browse.NewBrowseUsecase,
	review.NewReviewUsecase,
	ProvideBrandReader,
	ProvideCategoryLookup,
	ProvideBrowseTouch,
	ProvideTemplateReader,
	ProvideCategoryBind,
)

func ProvideProductUsecase(repo product.ProductRepo, brands product.BrandReader, categories product.CategoryLookup, browse product.BrowseTouch, templates product.TemplateReader, stocks product.StockGate) *product.ProductUsecase {
	uc := product.NewProductUsecase(repo, brands, categories, browse, templates)
	uc.UseStocks(stocks)
	return uc
}

// ProvideBrandReader 把品牌仓库收成商品上架要的读取接口
func ProvideBrandReader(repo brand.BrandRepo) product.BrandReader { return repo }

// ProvideCategoryLookup 把分类仓库收成商品要的查询接口
func ProvideCategoryLookup(repo category.CategoryRepo) product.CategoryLookup { return repo }

// ProvideBrowseTouch 把浏览仓库收成详情页记浏览的接口
func ProvideBrowseTouch(repo browse.BrowseRepo) product.BrowseTouch { return repo }

type templateReader struct{ repo attribute.TemplateRepo }

func (t templateReader) List(ctx context.Context, templateID string) ([]attribute.Attribute, error) {
	rows, err := t.repo.ListAttributes(ctx, templateID)
	if err != nil {
		return nil, err
	}
	out := make([]attribute.Attribute, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, *row)
	}
	return out, nil
}

// ProvideTemplateReader 把属性仓库收成商品保存时的模板读取
func ProvideTemplateReader(repo attribute.TemplateRepo) product.TemplateReader {
	return templateReader{repo: repo}
}

type categoryBind struct{ repo category.CategoryRepo }

func (c categoryBind) UsesTemplate(ctx context.Context, templateID string) (bool, error) {
	return c.repo.UsesTemplate(ctx, templateID)
}

// ProvideCategoryBind 把分类仓库收成模板删除前的占用检查
func ProvideCategoryBind(repo category.CategoryRepo) attribute.CategoryBind {
	return categoryBind{repo: repo}
}
