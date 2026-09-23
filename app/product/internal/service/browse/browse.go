package browse

import (
	"context"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	bizbrowse "github.com/tmjwjx/supermarket/app/product/internal/biz/browse"
	"github.com/tmjwjx/supermarket/app/product/internal/service/principal"
)

type BrowseHistoryService struct {
	v1.UnimplementedBrowseHistoryServiceServer
	uc *bizbrowse.BrowseUsecase
}

func NewBrowseHistoryService(uc *bizbrowse.BrowseUsecase) *BrowseHistoryService {
	return &BrowseHistoryService{uc: uc}
}

func (s *BrowseHistoryService) ListBrowseHistories(ctx context.Context, _ *v1.ListBrowseHistoriesRequest) (*v1.ListBrowseHistoriesResponse, error) {
	userID, err := principal.UserID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.uc.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.BrowseHistory, 0, len(rows))
	for _, row := range rows {
		item := &v1.BrowseHistory{ProductId: row.ProductID}
		if row.Card.ID != "" {
			item.Product = &v1.ProductCard{
				Id:          row.Card.ID,
				Name:        row.Card.Name,
				MainImage:   row.Card.MainImage,
				MinPrice:    row.Card.MinPrice,
				MarketPrice: row.Card.MarketPrice,
				SalesCount:  row.Card.Sales,
				Status:      row.Card.Status,
			}
		}
		out = append(out, item)
	}
	return &v1.ListBrowseHistoriesResponse{Histories: out}, nil
}

func (s *BrowseHistoryService) ClearBrowseHistories(ctx context.Context, _ *v1.ClearBrowseHistoriesRequest) (*v1.ClearBrowseHistoriesResponse, error) {
	userID, err := principal.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.Clear(ctx, userID); err != nil {
		return nil, err
	}
	return &v1.ClearBrowseHistoriesResponse{}, nil
}
