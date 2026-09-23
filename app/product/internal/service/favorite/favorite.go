package favorite

import (
	"context"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	bizfav "github.com/tmjwjx/supermarket/app/product/internal/biz/favorite"
	"github.com/tmjwjx/supermarket/app/product/internal/service/principal"
)

type FavoriteService struct {
	v1.UnimplementedFavoriteServiceServer
	uc *bizfav.FavoriteUsecase
}

func NewFavoriteService(uc *bizfav.FavoriteUsecase) *FavoriteService {
	return &FavoriteService{uc: uc}
}

func (s *FavoriteService) AddFavorite(ctx context.Context, req *v1.AddFavoriteRequest) (*v1.AddFavoriteResponse, error) {
	userID, err := principal.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.Add(ctx, userID, req.GetProductId()); err != nil {
		return nil, err
	}
	return &v1.AddFavoriteResponse{}, nil
}

func (s *FavoriteService) RemoveFavorite(ctx context.Context, req *v1.RemoveFavoriteRequest) (*v1.RemoveFavoriteResponse, error) {
	userID, err := principal.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.Remove(ctx, userID, req.GetProductId()); err != nil {
		return nil, err
	}
	return &v1.RemoveFavoriteResponse{}, nil
}

func (s *FavoriteService) ListFavorites(ctx context.Context, _ *v1.ListFavoritesRequest) (*v1.ListFavoritesResponse, error) {
	userID, err := principal.UserID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.uc.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Favorite, 0, len(rows))
	for _, row := range rows {
		item := &v1.Favorite{ProductId: row.ProductID, Invalid: row.Invalid}
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
	return &v1.ListFavoritesResponse{Favorites: out}, nil
}
