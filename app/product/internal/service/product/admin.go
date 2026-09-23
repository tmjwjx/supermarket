package product

import (
	"context"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	bizproduct "github.com/tmjwjx/supermarket/app/product/internal/biz/product"
	"github.com/tmjwjx/supermarket/app/product/internal/service/principal"
)

func (s *ProductService) AdminGetProduct(ctx context.Context, req *v1.AdminGetProductRequest) (*v1.AdminGetProductResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	p, err := s.uc.AdminGet(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.AdminGetProductResponse{Product: toProduct(p, true)}, nil
}

func (s *ProductService) SubmitProduct(ctx context.Context, req *v1.SubmitProductRequest) (*v1.SubmitProductResponse, error) {
	op, err := principal.RequireCatalogAdmin(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.uc.Submit(ctx, req.GetId(), op)
	if err != nil {
		return nil, err
	}
	return &v1.SubmitProductResponse{Product: toProduct(p, false)}, nil
}

func (s *ProductService) ApproveProduct(ctx context.Context, req *v1.ApproveProductRequest) (*v1.ApproveProductResponse, error) {
	op, err := principal.RequireCatalogAdmin(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.uc.Approve(ctx, req.GetId(), op)
	if err != nil {
		return nil, err
	}
	return &v1.ApproveProductResponse{Product: toProduct(p, false)}, nil
}

func (s *ProductService) RejectProduct(ctx context.Context, req *v1.RejectProductRequest) (*v1.RejectProductResponse, error) {
	op, err := principal.RequireCatalogAdmin(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.uc.Reject(ctx, req.GetId(), op, req.GetReason())
	if err != nil {
		return nil, err
	}
	return &v1.RejectProductResponse{Product: toProduct(p, false)}, nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, req *v1.DeleteProductRequest) (*v1.DeleteProductResponse, error) {
	op, err := principal.RequireCatalogAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.Delete(ctx, req.GetId(), op); err != nil {
		return nil, err
	}
	return &v1.DeleteProductResponse{}, nil
}

func (s *ProductService) RestoreProduct(ctx context.Context, req *v1.RestoreProductRequest) (*v1.RestoreProductResponse, error) {
	op, err := principal.RequireCatalogAdmin(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.uc.Restore(ctx, req.GetId(), op)
	if err != nil {
		return nil, err
	}
	return &v1.RestoreProductResponse{Product: toProduct(p, false)}, nil
}

func (s *ProductService) PurgeProduct(ctx context.Context, req *v1.PurgeProductRequest) (*v1.PurgeProductResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.Purge(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.PurgeProductResponse{}, nil
}

func (s *ProductService) BatchPublishProducts(ctx context.Context, req *v1.BatchPublishProductsRequest) (*v1.BatchPublishProductsResponse, error) {
	op, err := principal.RequireCatalogAdmin(ctx)
	if err != nil {
		return nil, err
	}
	ok, failed := s.uc.BatchPublish(ctx, req.GetIds(), op)
	return &v1.BatchPublishProductsResponse{Published: ok, Failed: toFails(failed)}, nil
}

func (s *ProductService) BatchUnpublishProducts(ctx context.Context, req *v1.BatchUnpublishProductsRequest) (*v1.BatchUnpublishProductsResponse, error) {
	op, err := principal.RequireCatalogAdmin(ctx)
	if err != nil {
		return nil, err
	}
	ok, failed := s.uc.BatchUnpublish(ctx, req.GetIds(), op)
	return &v1.BatchUnpublishProductsResponse{Unpublished: ok, Failed: toFails(failed)}, nil
}

func (s *ProductService) UpdateSkuPrice(ctx context.Context, req *v1.UpdateSkuPriceRequest) (*v1.UpdateSkuPriceResponse, error) {
	op, err := principal.RequireCatalogAdmin(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.uc.ChangePrice(ctx, req.GetProductId(), req.GetSkuId(), req.GetPrice(), op)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateSkuPriceResponse{Product: toProduct(p, false)}, nil
}

func (s *ProductService) ListDeletedProducts(ctx context.Context, req *v1.ListDeletedProductsRequest) (*v1.ListDeletedProductsResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	cards, next, err := s.uc.List(ctx, bizproduct.ListInput{
		PageSize:  req.GetPageSize(),
		PageToken: req.GetPageToken(),
		Deleted:   true,
	})
	if err != nil {
		return nil, err
	}
	return &v1.ListDeletedProductsResponse{Products: toCards(cards), NextPageToken: next}, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, req *v1.UpdateProductRequest) (*v1.UpdateProductResponse, error) {
	op, err := principal.RequireCatalogAdmin(ctx)
	if err != nil {
		return nil, err
	}
	skus := make([]bizproduct.Sku, 0, len(req.GetSkus()))
	for _, sku := range req.GetSkus() {
		skus = append(skus, bizproduct.Sku{
			ID: sku.GetId(), SpecsJSON: sku.GetSpecsJson(), Price: sku.GetPrice(), MarketPrice: sku.GetMarketPrice(),
			Image: sku.GetImage(), Barcode: sku.GetBarcode(), Enabled: sku.GetEnabled(),
		})
	}
	updated, err := s.uc.Update(ctx, &bizproduct.Product{
		ID: req.GetId(), CategoryID: req.GetCategoryId(), BrandID: req.GetBrandId(), Name: req.GetName(),
		Subtitle: req.GetSubtitle(), Keywords: req.GetKeywords(), Unit: req.GetUnit(), DetailHTML: req.GetDetailHtml(),
		WeightGram: req.GetWeightGram(), Images: append([]string(nil), req.GetImages()...), Skus: skus, Params: toParams(req.GetParams()),
	}, op)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateProductResponse{Product: toProduct(updated, true)}, nil
}

func (s *ProductService) ListProductAudits(ctx context.Context, req *v1.ListProductAuditsRequest) (*v1.ListProductAuditsResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	rows, next, err := s.uc.ListAudits(ctx, req.GetProductId(), req.GetPageToken(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.ProductAudit, 0, len(rows))
	for _, row := range rows {
		out = append(out, &v1.ProductAudit{
			Id: row.ID, ProductId: row.ProductID, Action: row.Action, Operator: row.Operator,
			Reason: row.Reason, CreatedAtUnix: row.CreatedAt,
		})
	}
	return &v1.ListProductAuditsResponse{Audits: out, NextPageToken: next}, nil
}

func (s *ProductService) ListProductLogs(ctx context.Context, req *v1.ListProductLogsRequest) (*v1.ListProductLogsResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	rows, next, err := s.uc.ListLogs(ctx, req.GetProductId(), req.GetPageToken(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.ProductLog, 0, len(rows))
	for _, row := range rows {
		out = append(out, &v1.ProductLog{
			Id: row.ID, ProductId: row.ProductID, Operator: row.Operator, Action: row.Action,
			Before: row.Before, After: row.After, CreatedAtUnix: row.CreatedAt,
		})
	}
	return &v1.ListProductLogsResponse{Logs: out, NextPageToken: next}, nil
}

func toFails(rows []bizproduct.Fail) []*v1.BatchFailure {
	out := make([]*v1.BatchFailure, 0, len(rows))
	for _, row := range rows {
		out = append(out, &v1.BatchFailure{Id: row.ID, Reason: row.Reason})
	}
	return out
}
