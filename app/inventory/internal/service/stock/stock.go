package stock

import (
	"context"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/inventory/v1"
	bizstock "github.com/tmjwjx/supermarket/app/inventory/internal/biz/stock"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	kmd "github.com/go-kratos/kratos/v3/metadata"
)

type StockService struct {
	v1.UnimplementedStockServiceServer
	uc *bizstock.StockUsecase
}

func NewStockService(uc *bizstock.StockUsecase) *StockService {
	return &StockService{uc: uc}
}

func (s *StockService) CreateStock(ctx context.Context, req *v1.CreateStockRequest) (*v1.CreateStockResponse, error) {
	created, err := s.uc.Create(ctx, strings.TrimSpace(req.GetSkuId()))
	if err != nil {
		return nil, err
	}
	return &v1.CreateStockResponse{Stock: toStock(created)}, nil
}

func (s *StockService) AdjustStock(ctx context.Context, req *v1.AdjustStockRequest) (*v1.AdjustStockResponse, error) {
	if err := requireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	updated, err := s.uc.Adjust(ctx, strings.TrimSpace(req.GetSkuId()), req.GetDelta(), strings.TrimSpace(req.GetReason()))
	if err != nil {
		return nil, err
	}
	return &v1.AdjustStockResponse{Stock: toStock(updated)}, nil
}

func (s *StockService) GetStocks(ctx context.Context, req *v1.GetStocksRequest) (*v1.GetStocksResponse, error) {
	rows, err := s.uc.Get(ctx, req.GetSkuIds())
	if err != nil {
		return nil, err
	}
	out := &v1.GetStocksResponse{}
	for _, row := range rows {
		out.Stocks = append(out.Stocks, toStock(row))
	}
	return out, nil
}

func (s *StockService) AdminGetStocks(ctx context.Context, req *v1.AdminGetStocksRequest) (*v1.AdminGetStocksResponse, error) {
	if err := requireCatalogAdmin(ctx); err != nil {
		return nil, err
	}
	rows, err := s.uc.Get(ctx, req.GetSkuIds())
	if err != nil {
		return nil, err
	}
	out := &v1.AdminGetStocksResponse{}
	for _, row := range rows {
		out.Stocks = append(out.Stocks, toStock(row))
	}
	return out, nil
}

func (s *StockService) ReserveStock(ctx context.Context, req *v1.ReserveStockRequest) (*v1.ReserveStockResponse, error) {
	items := make([]bizstock.Item, 0, len(req.GetItems()))
	for _, it := range req.GetItems() {
		items = append(items, bizstock.Item{SkuID: it.GetSkuId(), Quantity: it.GetQuantity()})
	}
	if err := s.uc.Reserve(ctx, req.GetOrderId(), items, req.GetExpireUnix()); err != nil {
		return nil, err
	}
	return &v1.ReserveStockResponse{}, nil
}

func (s *StockService) ConfirmReservation(ctx context.Context, req *v1.ConfirmReservationRequest) (*v1.ConfirmReservationResponse, error) {
	if err := s.uc.Confirm(ctx, req.GetOrderId()); err != nil {
		return nil, err
	}
	return &v1.ConfirmReservationResponse{}, nil
}

func (s *StockService) ReleaseReservation(ctx context.Context, req *v1.ReleaseReservationRequest) (*v1.ReleaseReservationResponse, error) {
	if err := s.uc.Release(ctx, req.GetOrderId()); err != nil {
		return nil, err
	}
	return &v1.ReleaseReservationResponse{}, nil
}

func requireCatalogAdmin(ctx context.Context) error {
	md, ok := kmd.FromServerContext(ctx)
	if !ok {
		return kerrors.Unauthorized("STOCK_UNAUTHENTICATED", "unauthenticated")
	}
	id := strings.TrimSpace(md.Get("x-md-global-admin-id"))
	role := strings.TrimSpace(md.Get("x-md-global-admin-role"))
	if id == "" || role == "" {
		return kerrors.Unauthorized("STOCK_UNAUTHENTICATED", "unauthenticated")
	}
	if role != "product" && role != "super" {
		return kerrors.Forbidden("STOCK_FORBIDDEN", "forbidden")
	}
	return nil
}

func toStock(in *bizstock.Stock) *v1.Stock {
	if in == nil {
		return nil
	}
	return &v1.Stock{
		SkuId: in.SkuID, OnHand: in.OnHand, Reserved: in.Reserved, Available: in.Available,
	}
}
