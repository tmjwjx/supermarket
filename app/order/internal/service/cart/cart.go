package cart

import (
	"context"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/order/v1"
	bizcart "github.com/tmjwjx/supermarket/app/order/internal/biz/cart"

	kmetadata "github.com/go-kratos/kratos/v3/metadata"
	"github.com/google/uuid"
)

type CartService struct {
	v1.UnimplementedCartServiceServer
	uc *bizcart.CartUsecase
}

func NewCartService(uc *bizcart.CartUsecase) *CartService {
	return &CartService{uc: uc}
}

func (s *CartService) AddCartItem(ctx context.Context, req *v1.AddCartItemRequest) (*v1.AddCartItemResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.uc.Add(ctx, uid, strings.TrimSpace(req.GetSkuId()), req.GetQuantity())
	if err != nil {
		return nil, err
	}
	return &v1.AddCartItemResponse{Item: toItem(item)}, nil
}

func (s *CartService) UpdateCartItem(ctx context.Context, req *v1.UpdateCartItemRequest) (*v1.UpdateCartItemResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.uc.Update(ctx, uid, strings.TrimSpace(req.GetSkuId()), req.GetQuantity(), req.GetChecked())
	if err != nil {
		return nil, err
	}
	return &v1.UpdateCartItemResponse{Item: toItem(item)}, nil
}

func (s *CartService) ListCartItems(ctx context.Context, _ *v1.ListCartItemsRequest) (*v1.ListCartItemsResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.uc.List(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := &v1.ListCartItemsResponse{CheckedAmount: list.CheckedAmount}
	for _, item := range list.Items {
		out.Items = append(out.Items, toItem(item))
	}
	return out, nil
}

func (s *CartService) RemoveCartItems(ctx context.Context, req *v1.RemoveCartItemsRequest) (*v1.RemoveCartItemsResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.Remove(ctx, uid, req.GetSkuIds()); err != nil {
		return nil, err
	}
	return &v1.RemoveCartItemsResponse{}, nil
}

func toItem(in *bizcart.CartItem) *v1.CartItem {
	if in == nil {
		return nil
	}
	return &v1.CartItem{
		SkuId: in.SkuID, Quantity: in.Quantity, Checked: in.Checked, ProductName: in.ProductName,
		SpecsJson: in.SpecsJSON, Price: in.Price, Image: in.Image, Invalid: in.Invalid, ShortStock: in.ShortStock,
	}
}

func callerID(ctx context.Context) (uuid.UUID, error) {
	md, ok := kmetadata.FromServerContext(ctx)
	if !ok {
		return uuid.Nil, bizcart.ErrUnauthenticated
	}
	id, err := uuid.Parse(strings.TrimSpace(md.Get("x-md-global-user-id")))
	if err != nil {
		return uuid.Nil, bizcart.ErrUnauthenticated
	}
	return id, nil
}
