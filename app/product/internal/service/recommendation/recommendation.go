package recommendation

import (
	"context"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	bizrec "github.com/tmjwjx/supermarket/app/product/internal/biz/recommendation"
	"github.com/tmjwjx/supermarket/app/product/internal/service/principal"
)

type RecommendationService struct {
	v1.UnimplementedRecommendationServiceServer
	uc *bizrec.RecommendationUsecase
}

func NewRecommendationService(uc *bizrec.RecommendationUsecase) *RecommendationService {
	return &RecommendationService{uc: uc}
}

func (s *RecommendationService) CreateRecommendation(ctx context.Context, req *v1.CreateRecommendationRequest) (*v1.CreateRecommendationResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	created, err := s.uc.Create(ctx, &bizrec.Recommendation{
		Slot:      req.GetSlot(),
		ProductID: req.GetProductId(),
		Sort:      req.GetSort(),
		StartAt:   req.GetStartAtUnix(),
		EndAt:     req.GetEndAtUnix(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateRecommendationResponse{Recommendation: toRecommendation(created)}, nil
}

func (s *RecommendationService) ListRecommendations(ctx context.Context, req *v1.ListRecommendationsRequest) (*v1.ListRecommendationsResponse, error) {
	rows, err := s.uc.List(ctx, req.GetSlot())
	if err != nil {
		return nil, err
	}
	return &v1.ListRecommendationsResponse{Recommendations: toRecommendations(rows)}, nil
}

func (s *RecommendationService) AdminListRecommendations(ctx context.Context, req *v1.AdminListRecommendationsRequest) (*v1.AdminListRecommendationsResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	rows, err := s.uc.AdminList(ctx, req.GetSlot())
	if err != nil {
		return nil, err
	}
	return &v1.AdminListRecommendationsResponse{Recommendations: toRecommendations(rows)}, nil
}

func (s *RecommendationService) UpdateRecommendation(ctx context.Context, req *v1.UpdateRecommendationRequest) (*v1.UpdateRecommendationResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	updated, err := s.uc.Update(ctx, bizrec.Patch{
		ID:        req.GetId(),
		Slot:      req.Slot,
		ProductID: req.ProductId,
		Sort:      req.Sort,
		StartAt:   req.StartAtUnix,
		EndAt:     req.EndAtUnix,
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateRecommendationResponse{Recommendation: toRecommendation(updated)}, nil
}

func (s *RecommendationService) DeleteRecommendation(ctx context.Context, req *v1.DeleteRecommendationRequest) (*v1.DeleteRecommendationResponse, error) {
	if _, err := principal.RequireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteRecommendationResponse{}, nil
}

func toRecommendations(rows []*bizrec.Recommendation) []*v1.Recommendation {
	out := make([]*v1.Recommendation, 0, len(rows))
	for _, row := range rows {
		out = append(out, toRecommendation(row))
	}
	return out
}

func toRecommendation(item *bizrec.Recommendation) *v1.Recommendation {
	if item == nil {
		return nil
	}
	out := &v1.Recommendation{
		Id:          item.ID,
		Slot:        item.Slot,
		ProductId:   item.ProductID,
		Sort:        item.Sort,
		StartAtUnix: item.StartAt,
		EndAtUnix:   item.EndAt,
	}
	if item.Card.ID != "" {
		out.Product = &v1.ProductCard{
			Id:          item.Card.ID,
			Name:        item.Card.Name,
			MainImage:   item.Card.MainImage,
			MinPrice:    item.Card.MinPrice,
			MarketPrice: item.Card.MarketPrice,
			SalesCount:  item.Card.Sales,
			Status:      item.Card.Status,
		}
	}
	return out
}
