package brand

import (
	"context"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	bizbrand "github.com/tmjwjx/supermarket/app/product/internal/biz/brand"
	"github.com/tmjwjx/supermarket/app/product/internal/service/principal"
)

type BrandService struct {
	v1.UnimplementedBrandServiceServer
	uc *bizbrand.BrandUsecase
}

func NewBrandService(uc *bizbrand.BrandUsecase) *BrandService {
	return &BrandService{uc: uc}
}

func (s *BrandService) CreateBrand(ctx context.Context, req *v1.CreateBrandRequest) (*v1.CreateBrandResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	created, err := s.uc.Create(ctx, &bizbrand.Brand{
		Name:        req.GetName(),
		Initial:     req.GetInitial(),
		LogoURL:     req.GetLogoUrl(),
		Description: req.GetDescription(),
		Visible:     req.GetVisible(),
		Sort:        req.GetSort(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateBrandResponse{Brand: toBrand(created)}, nil
}

func (s *BrandService) UpdateBrand(ctx context.Context, req *v1.UpdateBrandRequest) (*v1.UpdateBrandResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	updated, err := s.uc.Update(ctx, bizbrand.Patch{
		ID:          req.GetId(),
		Name:        req.Name,
		Initial:     req.Initial,
		LogoURL:     req.LogoUrl,
		Description: req.Description,
		Visible:     req.Visible,
		Sort:        req.Sort,
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateBrandResponse{Brand: toBrand(updated)}, nil
}

func (s *BrandService) DeleteBrand(ctx context.Context, req *v1.DeleteBrandRequest) (*v1.DeleteBrandResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteBrandResponse{}, nil
}

func (s *BrandService) ListBrands(ctx context.Context, _ *v1.ListBrandsRequest) (*v1.ListBrandsResponse, error) {
	rows, err := s.uc.List(ctx, true)
	if err != nil {
		return nil, err
	}
	return &v1.ListBrandsResponse{Brands: toBrands(rows)}, nil
}

func (s *BrandService) AdminListBrands(ctx context.Context, _ *v1.AdminListBrandsRequest) (*v1.AdminListBrandsResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	rows, err := s.uc.List(ctx, false)
	if err != nil {
		return nil, err
	}
	return &v1.AdminListBrandsResponse{Brands: toBrands(rows)}, nil
}

func toBrands(rows []*bizbrand.Brand) []*v1.Brand {
	out := make([]*v1.Brand, 0, len(rows))
	for _, row := range rows {
		out = append(out, toBrand(row))
	}
	return out
}

func toBrand(b *bizbrand.Brand) *v1.Brand {
	if b == nil {
		return nil
	}
	return &v1.Brand{
		Id:          b.ID,
		Name:        b.Name,
		Initial:     b.Initial,
		LogoUrl:     b.LogoURL,
		Description: b.Description,
		Visible:     b.Visible,
		Sort:        b.Sort,
	}
}
