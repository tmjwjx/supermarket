package attribute

import (
	"context"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	bizattr "github.com/tmjwjx/supermarket/app/product/internal/biz/attribute"
	"github.com/tmjwjx/supermarket/app/product/internal/service/principal"
)

type AttributeService struct {
	v1.UnimplementedAttributeServiceServer
	uc *bizattr.AttributeUsecase
}

func NewAttributeService(uc *bizattr.AttributeUsecase) *AttributeService {
	return &AttributeService{uc: uc}
}

func (s *AttributeService) CreateAttributeTemplate(ctx context.Context, req *v1.CreateAttributeTemplateRequest) (*v1.CreateAttributeTemplateResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	row, err := s.uc.CreateTemplate(ctx, req.GetName())
	if err != nil {
		return nil, err
	}
	return &v1.CreateAttributeTemplateResponse{Template: toTemplate(row)}, nil
}

func (s *AttributeService) DeleteAttributeTemplate(ctx context.Context, req *v1.DeleteAttributeTemplateRequest) (*v1.DeleteAttributeTemplateResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.DeleteTemplate(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteAttributeTemplateResponse{}, nil
}

func (s *AttributeService) ListAttributeTemplates(ctx context.Context, _ *v1.ListAttributeTemplatesRequest) (*v1.ListAttributeTemplatesResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	rows, err := s.uc.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AttributeTemplate, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTemplate(row))
	}
	return &v1.ListAttributeTemplatesResponse{Templates: out}, nil
}

func (s *AttributeService) CreateAttribute(ctx context.Context, req *v1.CreateAttributeRequest) (*v1.CreateAttributeResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	row, err := s.uc.CreateAttribute(ctx, &bizattr.Attribute{
		TemplateID:  req.GetTemplateId(),
		Name:        req.GetName(),
		Kind:        req.GetKind(),
		Options:     append([]string(nil), req.GetOptions()...),
		AllowCustom: req.GetAllowCustom(),
		Sort:        req.GetSort(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateAttributeResponse{Attribute: toAttribute(row)}, nil
}

func (s *AttributeService) UpdateAttribute(ctx context.Context, req *v1.UpdateAttributeRequest) (*v1.UpdateAttributeResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	row, err := s.uc.UpdateAttribute(ctx, bizattr.Patch{
		ID:           req.GetId(),
		Name:         req.Name,
		Kind:         req.Kind,
		Options:      append([]string(nil), req.GetOptions()...),
		ClearOptions: req.GetClearOptions(),
		AllowCustom:  req.AllowCustom,
		Sort:         req.Sort,
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateAttributeResponse{Attribute: toAttribute(row)}, nil
}

func (s *AttributeService) DeleteAttribute(ctx context.Context, req *v1.DeleteAttributeRequest) (*v1.DeleteAttributeResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.DeleteAttribute(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteAttributeResponse{}, nil
}

func (s *AttributeService) ListAttributes(ctx context.Context, req *v1.ListAttributesRequest) (*v1.ListAttributesResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	rows, err := s.uc.ListAttributes(ctx, req.GetTemplateId())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Attribute, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAttribute(row))
	}
	return &v1.ListAttributesResponse{Attributes: out}, nil
}

func toTemplate(row *bizattr.Template) *v1.AttributeTemplate {
	if row == nil {
		return nil
	}
	return &v1.AttributeTemplate{Id: row.ID, Name: row.Name}
}

func toAttribute(row *bizattr.Attribute) *v1.Attribute {
	if row == nil {
		return nil
	}
	return &v1.Attribute{
		Id: row.ID, TemplateId: row.TemplateID, Name: row.Name, Kind: row.Kind,
		Options: append([]string(nil), row.Options...), AllowCustom: row.AllowCustom, Sort: row.Sort,
	}
}
