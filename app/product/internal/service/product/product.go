package product

import (
	"context"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	bizproduct "github.com/tmjwjx/supermarket/app/product/internal/biz/product"
	"github.com/tmjwjx/supermarket/app/product/internal/service/principal"
)

type ProductService struct {
	v1.UnimplementedProductServiceServer
	uc *bizproduct.ProductUsecase
}

func NewProductService(uc *bizproduct.ProductUsecase) *ProductService {
	return &ProductService{uc: uc}
}

func (s *ProductService) CreateProduct(ctx context.Context, req *v1.CreateProductRequest) (*v1.CreateProductResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	skus := make([]bizproduct.Sku, 0, len(req.GetSkus()))
	for _, sku := range req.GetSkus() {
		skus = append(skus, bizproduct.Sku{
			ID:          sku.GetId(),
			SpecsJSON:   sku.GetSpecsJson(),
			Price:       sku.GetPrice(),
			MarketPrice: sku.GetMarketPrice(),
			Image:       sku.GetImage(),
			Barcode:     sku.GetBarcode(),
			Enabled:     sku.GetEnabled(),
		})
	}
	created, err := s.uc.Create(ctx, &bizproduct.Product{
		CategoryID: req.GetCategoryId(),
		BrandID:    req.GetBrandId(),
		Name:       req.GetName(),
		Subtitle:   req.GetSubtitle(),
		Keywords:   req.GetKeywords(),
		Unit:       req.GetUnit(),
		DetailHTML: req.GetDetailHtml(),
		WeightGram: req.GetWeightGram(),
		Images:     append([]string(nil), req.GetImages()...),
		Skus:       skus,
		Params:     toParams(req.GetParams()),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateProductResponse{Product: toProduct(created, true)}, nil
}

func (s *ProductService) PublishProduct(ctx context.Context, req *v1.PublishProductRequest) (*v1.PublishProductResponse, error) {
	op, err := principal.RequireCatalogAdmin(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.uc.Publish(ctx, req.GetId(), op)
	if err != nil {
		return nil, err
	}
	return &v1.PublishProductResponse{Product: toProduct(p, false)}, nil
}

func (s *ProductService) UnpublishProduct(ctx context.Context, req *v1.UnpublishProductRequest) (*v1.UnpublishProductResponse, error) {
	op, err := principal.RequireCatalogAdmin(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.uc.Unpublish(ctx, req.GetId(), op)
	if err != nil {
		return nil, err
	}
	return &v1.UnpublishProductResponse{Product: toProduct(p, false)}, nil
}

func (s *ProductService) ListProducts(ctx context.Context, req *v1.ListProductsRequest) (*v1.ListProductsResponse, error) {
	cards, next, err := s.uc.List(ctx, bizproduct.ListInput{
		CategoryID: req.GetCategoryId(),
		BrandID:    req.GetBrandId(),
		MinPrice:   req.GetMinPrice(),
		MaxPrice:   req.GetMaxPrice(),
		OrderBy:    req.GetOrderBy(),
		PageSize:   req.GetPageSize(),
		PageToken:  req.GetPageToken(),
		OnSaleOnly: true,
	})
	if err != nil {
		return nil, err
	}
	return &v1.ListProductsResponse{Products: toCards(cards), NextPageToken: next}, nil
}

func (s *ProductService) GetProduct(ctx context.Context, req *v1.GetProductRequest) (*v1.GetProductResponse, error) {
	p, err := s.uc.Get(ctx, req.GetId(), principal.OptionalUserID(ctx))
	if err != nil {
		return nil, err
	}
	out := toProduct(p, false)
	for _, opt := range bizproduct.SpecOptions(p.Skus) {
		out.SpecOptions = append(out.SpecOptions, &v1.SpecOption{Name: opt.Name, Values: opt.Values})
	}
	return &v1.GetProductResponse{Product: out}, nil
}

func (s *ProductService) GetProductDetail(ctx context.Context, req *v1.GetProductDetailRequest) (*v1.GetProductDetailResponse, error) {
	html, err := s.uc.Detail(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.GetProductDetailResponse{DetailHtml: html}, nil
}

func (s *ProductService) SearchProducts(ctx context.Context, req *v1.SearchProductsRequest) (*v1.SearchProductsResponse, error) {
	cards, next, err := s.uc.Search(ctx, req.GetQ(), req.GetOrderBy(), req.GetPageToken(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &v1.SearchProductsResponse{Products: toCards(cards), NextPageToken: next}, nil
}

func (s *ProductService) BatchGetSkus(ctx context.Context, req *v1.BatchGetSkusRequest) (*v1.BatchGetSkusResponse, error) {
	rows, err := s.uc.BatchSkus(ctx, req.GetSkuIds())
	if err != nil {
		return nil, err
	}
	return &v1.BatchGetSkusResponse{Skus: toSnapshots(rows)}, nil
}

func (s *ProductService) CheckSellable(ctx context.Context, req *v1.CheckSellableRequest) (*v1.CheckSellableResponse, error) {
	rows, err := s.uc.CheckSellable(ctx, req.GetSkuIds())
	if err != nil {
		return nil, err
	}
	return &v1.CheckSellableResponse{Skus: toSnapshots(rows)}, nil
}

func (s *ProductService) IncreaseSales(ctx context.Context, req *v1.IncreaseSalesRequest) (*v1.IncreaseSalesResponse, error) {
	if err := s.uc.AddSales(ctx, "", req.GetOrderId(), req.GetProductId(), req.GetCount()); err != nil {
		return nil, err
	}
	return &v1.IncreaseSalesResponse{}, nil
}

func (s *ProductService) AdminListProducts(ctx context.Context, req *v1.AdminListProductsRequest) (*v1.AdminListProductsResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	cards, next, err := s.uc.List(ctx, bizproduct.ListInput{
		PageSize:   req.GetPageSize(),
		PageToken:  req.GetPageToken(),
		OnSaleOnly: false,
	})
	if err != nil {
		return nil, err
	}
	return &v1.AdminListProductsResponse{Products: toCards(cards), NextPageToken: next}, nil
}

func toProduct(p *bizproduct.Product, withDetail bool) *v1.Product {
	if p == nil {
		return nil
	}
	out := &v1.Product{
		Id:         p.ID,
		CategoryId: p.CategoryID,
		BrandId:    p.BrandID,
		Name:       p.Name,
		Subtitle:   p.Subtitle,
		Keywords:   p.Keywords,
		MainImage:  p.MainImage,
		Unit:       p.Unit,
		WeightGram: p.WeightGram,
		Status:     p.Status,
		MinPrice:   p.MinPrice,
		MaxPrice:   p.MaxPrice,
		SalesCount: p.Sales,
		Images:     append([]string(nil), p.Images...),
	}
	if withDetail {
		out.DetailHtml = p.DetailHTML
	}
	for _, param := range p.Params {
		out.Params = append(out.Params, &v1.ProductParam{Name: param.Name, Value: param.Value})
	}
	for _, sku := range p.Skus {
		out.Skus = append(out.Skus, &v1.Sku{
			Id:          sku.ID,
			ProductId:   sku.ProductID,
			SpecsJson:   sku.SpecsJSON,
			Price:       sku.Price,
			MarketPrice: sku.MarketPrice,
			Image:       sku.Image,
			Barcode:     sku.Barcode,
			Enabled:     sku.Enabled,
		})
	}
	return out
}

func toCards(rows []bizproduct.Card) []*v1.ProductCard {
	out := make([]*v1.ProductCard, 0, len(rows))
	for _, row := range rows {
		out = append(out, &v1.ProductCard{
			Id:          row.ID,
			Name:        row.Name,
			MainImage:   row.MainImage,
			MinPrice:    row.MinPrice,
			MarketPrice: row.MarketPrice,
			SalesCount:  row.Sales,
			Status:      row.Status,
		})
	}
	return out
}

func toSnapshots(rows []bizproduct.SkuView) []*v1.SkuSnapshot {
	out := make([]*v1.SkuSnapshot, 0, len(rows))
	for _, row := range rows {
		out = append(out, &v1.SkuSnapshot{
			SkuId:       row.SkuID,
			ProductId:   row.ProductID,
			ProductName: row.Name,
			SpecsJson:   row.SpecsJSON,
			Price:       row.Price,
			Image:       row.Image,
			Sellable:    row.Sellable,
		})
	}
	return out
}

func toParams(rows []*v1.ProductParam) []bizproduct.Param {
	out := make([]bizproduct.Param, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, bizproduct.Param{Name: row.GetName(), Value: row.GetValue()})
	}
	return out
}
