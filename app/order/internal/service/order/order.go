package order

import (
	"context"
	"strconv"
	"strings"
	"time"

	v1 "github.com/tmjwjx/supermarket/api/order/v1"
	bizorder "github.com/tmjwjx/supermarket/app/order/internal/biz/order"

	kmetadata "github.com/go-kratos/kratos/v3/metadata"
	"github.com/google/uuid"
	"go.einride.tech/aip/pagination"
)

type OrderService struct {
	v1.UnimplementedOrderServiceServer
	uc *bizorder.OrderUsecase
}

func NewOrderService(uc *bizorder.OrderUsecase) *OrderService {
	return &OrderService{uc: uc}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *v1.CreateOrderRequest) (*v1.CreateOrderResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	lines := make([]bizorder.Line, 0, len(req.GetLines()))
	for _, ln := range req.GetLines() {
		lines = append(lines, bizorder.Line{SkuID: ln.GetSkuId(), Quantity: ln.GetQuantity()})
	}
	created, err := s.uc.Create(ctx, uid, bizorder.CreateInput{
		Lines: lines, AddressID: req.GetAddressId(), Remark: req.GetRemark(), RequestID: req.GetRequestId(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateOrderResponse{Order: toOrder(created)}, nil
}

func (s *OrderService) ListOrders(ctx context.Context, req *v1.ListOrdersRequest) (*v1.ListOrdersResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	size := req.GetPageSize()
	if size <= 0 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	req.PageSize = size
	token, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, bizorder.ErrOrderInvalidArgument
	}
	rows, err := s.uc.List(ctx, uid, req.GetStatus(), int(token.Offset), int(size)+1)
	if err != nil {
		return nil, err
	}
	var next string
	if len(rows) > int(size) {
		rows = rows[:size]
		next = token.Next(req).String()
	}
	out := &v1.ListOrdersResponse{NextPageToken: next}
	for _, row := range rows {
		out.Orders = append(out.Orders, toOrder(row))
	}
	return out, nil
}

func (s *OrderService) GetOrder(ctx context.Context, req *v1.GetOrderRequest) (*v1.GetOrderResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	row, err := s.uc.Get(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetOrderResponse{Order: toOrder(row)}, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, req *v1.CancelOrderRequest) (*v1.CancelOrderResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	row, err := s.uc.Cancel(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	return &v1.CancelOrderResponse{Order: toOrder(row)}, nil
}

func (s *OrderService) ConfirmOrder(ctx context.Context, req *v1.ConfirmOrderRequest) (*v1.ConfirmOrderResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	row, err := s.uc.ConfirmReceipt(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	return &v1.ConfirmOrderResponse{Order: toOrder(row)}, nil
}

func (s *OrderService) MarkOrderPaid(ctx context.Context, req *v1.MarkOrderPaidRequest) (*v1.MarkOrderPaidResponse, error) {
	if err := s.uc.MarkPaid(ctx, req.GetOrderId(), req.GetPaymentId()); err != nil {
		return nil, err
	}
	return &v1.MarkOrderPaidResponse{}, nil
}

func (s *OrderService) GetOrderItem(ctx context.Context, req *v1.GetOrderItemRequest) (*v1.GetOrderItemResponse, error) {
	itemID, err := parseID(req.GetOrderItemId())
	if err != nil {
		return nil, err
	}
	userID, err := parseID(req.GetUserId())
	if err != nil {
		return nil, err
	}
	view, err := s.uc.GetItem(ctx, itemID, userID)
	if err != nil {
		return nil, err
	}
	return &v1.GetOrderItemResponse{
		OrderItemId: view.ID.String(), ProductId: view.ProductID, UserId: view.UserID.String(),
		Completed: view.Completed, Reviewed: view.Reviewed, SpecsJson: view.SpecsJSON,
	}, nil
}

func (s *OrderService) MarkOrderItemReviewed(ctx context.Context, req *v1.MarkOrderItemReviewedRequest) (*v1.MarkOrderItemReviewedResponse, error) {
	itemID, err := parseID(req.GetOrderItemId())
	if err != nil {
		return nil, err
	}
	if err := s.uc.MarkItemReviewed(ctx, itemID); err != nil {
		return nil, err
	}
	return &v1.MarkOrderItemReviewedResponse{}, nil
}

func (s *OrderService) AdminListOrders(ctx context.Context, req *v1.AdminListOrdersRequest) (*v1.AdminListOrdersResponse, error) {
	if err := requireOperator(ctx); err != nil {
		return nil, err
	}
	offset := 0
	if token := strings.TrimSpace(req.GetPageToken()); token != "" {
		n, err := strconv.Atoi(token)
		if err != nil || n < 0 {
			return nil, bizorder.ErrOrderInvalidArgument
		}
		offset = n
	}
	limit := int(req.GetPageSize())
	rows, err := s.uc.ListAdmin(ctx, req.GetStatus(), offset, limit)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	out := &v1.AdminListOrdersResponse{}
	for _, row := range rows {
		out.Orders = append(out.Orders, toOrder(row))
	}
	if len(rows) == limit {
		out.NextPageToken = strconv.Itoa(offset + limit)
	}
	return out, nil
}

func (s *OrderService) AdminShipOrder(ctx context.Context, req *v1.AdminShipOrderRequest) (*v1.AdminShipOrderResponse, error) {
	if err := requireOperator(ctx); err != nil {
		return nil, err
	}
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	row, err := s.uc.Ship(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v1.AdminShipOrderResponse{Order: toOrder(row)}, nil
}

func (s *OrderService) ListPaidOrders(ctx context.Context, req *v1.ListPaidOrdersRequest) (*v1.ListPaidOrdersResponse, error) {
	from := time.Unix(req.GetPaidFromUnix(), 0)
	to := time.Unix(req.GetPaidToUnix(), 0)
	rows, err := s.uc.ListPaid(ctx, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.PaidOrderRef, 0, len(rows))
	for _, row := range rows {
		out = append(out, &v1.PaidOrderRef{Id: row.ID, PaymentId: row.PaymentID, PayAmount: row.Amount})
	}
	return &v1.ListPaidOrdersResponse{Orders: out}, nil
}

func toOrder(in *bizorder.Order) *v1.Order {
	if in == nil {
		return nil
	}
	out := &v1.Order{
		Id: in.ID.String(), OrderNo: in.OrderNo, Status: in.Status,
		ItemsAmount: in.ItemsAmount, PayAmount: in.PayAmount,
		Receiver: in.Receiver, Phone: in.Phone,
		Address:       in.Province + in.City + in.District + in.Detail,
		ExpiresAtUnix: in.ExpiresAt.Unix(),
	}
	for _, it := range in.Items {
		if it == nil {
			continue
		}
		out.Items = append(out.Items, &v1.OrderItem{
			Id: it.ID.String(), SkuId: it.SkuID, ProductId: it.ProductID, ProductName: it.ProductName,
			SpecsJson: it.SpecsJSON, Image: it.Image, Price: it.Price, Quantity: it.Quantity,
			Amount: it.Amount, Reviewed: it.Reviewed,
		})
	}
	return out
}

func requireOperator(ctx context.Context) error {
	md, ok := kmetadata.FromServerContext(ctx)
	if !ok {
		return bizorder.ErrUnauthenticated
	}
	id := strings.TrimSpace(md.Get("x-md-global-admin-id"))
	role := strings.TrimSpace(md.Get("x-md-global-admin-role"))
	if id == "" || role == "" {
		return bizorder.ErrUnauthenticated
	}
	if role != "order" && role != "super" {
		return bizorder.ErrForbidden
	}
	return nil
}

func callerID(ctx context.Context) (uuid.UUID, error) {
	md, ok := kmetadata.FromServerContext(ctx)
	if !ok {
		return uuid.Nil, bizorder.ErrUnauthenticated
	}
	return parseCaller(md.Get("x-md-global-user-id"))
}

func parseCaller(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return uuid.Nil, bizorder.ErrUnauthenticated
	}
	return parsed, nil
}

func parseID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return uuid.Nil, bizorder.ErrOrderInvalidArgument
	}
	return parsed, nil
}
