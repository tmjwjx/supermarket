package category

import (
	"context"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	bizcategory "github.com/tmjwjx/supermarket/app/product/internal/biz/category"
	"github.com/tmjwjx/supermarket/app/product/internal/service/principal"
)

type CategoryService struct {
	v1.UnimplementedCategoryServiceServer
	uc *bizcategory.CategoryUsecase
}

func NewCategoryService(uc *bizcategory.CategoryUsecase) *CategoryService {
	return &CategoryService{uc: uc}
}

func (s *CategoryService) CreateCategory(ctx context.Context, req *v1.CreateCategoryRequest) (*v1.CreateCategoryResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	created, err := s.uc.Create(ctx, &bizcategory.Category{
		ParentID:   req.GetParentId(),
		Name:       req.GetName(),
		IconURL:    req.GetIconUrl(),
		Sort:       req.GetSort(),
		Visible:    req.GetVisible(),
		TemplateID: req.GetAttributeTemplateId(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateCategoryResponse{Category: toCategory(created)}, nil
}

func (s *CategoryService) UpdateCategory(ctx context.Context, req *v1.UpdateCategoryRequest) (*v1.UpdateCategoryResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	updated, err := s.uc.Update(ctx, bizcategory.Patch{
		ID:         req.GetId(),
		Name:       req.Name,
		IconURL:    req.IconUrl,
		Sort:       req.Sort,
		Visible:    req.Visible,
		TemplateID: req.AttributeTemplateId,
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateCategoryResponse{Category: toCategory(updated)}, nil
}

func (s *CategoryService) AdminListCategories(ctx context.Context, _ *v1.AdminListCategoriesRequest) (*v1.AdminListCategoriesResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	rows, err := s.uc.AdminTree(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Category, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCategory(row))
	}
	return &v1.AdminListCategoriesResponse{Categories: out}, nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, req *v1.DeleteCategoryRequest) (*v1.DeleteCategoryResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteCategoryResponse{}, nil
}

func (s *CategoryService) ListCategories(ctx context.Context, _ *v1.ListCategoriesRequest) (*v1.ListCategoriesResponse, error) {
	rows, err := s.uc.Tree(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Category, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCategory(row))
	}
	return &v1.ListCategoriesResponse{Categories: out}, nil
}

func toCategory(c *bizcategory.Category) *v1.Category {
	if c == nil {
		return nil
	}
	out := &v1.Category{
		Id:                  c.ID,
		ParentId:            c.ParentID,
		Name:                c.Name,
		IconUrl:             c.IconURL,
		Sort:                c.Sort,
		Visible:             c.Visible,
		AttributeTemplateId: c.TemplateID,
	}
	for _, child := range c.Children {
		out.Children = append(out.Children, toCategory(child))
	}
	return out
}
