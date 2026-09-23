package review

import (
	"context"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	bizreview "github.com/tmjwjx/supermarket/app/product/internal/biz/review"
	"github.com/tmjwjx/supermarket/app/product/internal/service/principal"
)

type ReviewService struct {
	v1.UnimplementedReviewServiceServer
	uc *bizreview.ReviewUsecase
}

func NewReviewService(uc *bizreview.ReviewUsecase) *ReviewService {
	return &ReviewService{uc: uc}
}

func (s *ReviewService) CreateReview(ctx context.Context, req *v1.CreateReviewRequest) (*v1.CreateReviewResponse, error) {
	userID, err := principal.UserID(ctx)
	if err != nil {
		return nil, err
	}
	created, err := s.uc.Create(ctx, userID, &bizreview.Review{
		ProductID:   req.GetProductId(),
		OrderItemID: req.GetOrderItemId(),
		Stars:       req.GetStars(),
		Content:     req.GetContent(),
		Anonymous:   req.GetAnonymous(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateReviewResponse{Review: toReview(created)}, nil
}

func (s *ReviewService) ListProductReviews(ctx context.Context, req *v1.ListProductReviewsRequest) (*v1.ListProductReviewsResponse, error) {
	rows, next, err := s.uc.List(ctx, req.GetProductId(), req.GetPageToken(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Review, 0, len(rows))
	for _, row := range rows {
		out = append(out, toReview(row))
	}
	return &v1.ListProductReviewsResponse{Reviews: out, NextPageToken: next}, nil
}

func toReview(item *bizreview.Review) *v1.Review {
	if item == nil {
		return nil
	}
	return &v1.Review{
		Id:          item.ID,
		ProductId:   item.ProductID,
		OrderItemId: item.OrderItemID,
		Stars:       item.Stars,
		Content:     item.Content,
		DisplayName: item.DisplayName,
		Anonymous:   item.Anonymous,
		SpecsJson:   item.SpecsJSON,
	}
}
