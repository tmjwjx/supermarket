package payment

import (
	"context"
	"strconv"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/payment/v1"
	bizpayment "github.com/tmjwjx/supermarket/app/payment/internal/biz/payment"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	kmd "github.com/go-kratos/kratos/v3/metadata"
	"github.com/google/uuid"
)

type PaymentService struct {
	v1.UnimplementedPaymentServiceServer
	uc *bizpayment.PaymentUsecase
}

func NewPaymentService(uc *bizpayment.PaymentUsecase) *PaymentService {
	return &PaymentService{uc: uc}
}

func (s *PaymentService) CreatePayment(ctx context.Context, req *v1.CreatePaymentRequest) (*v1.CreatePaymentResponse, error) {
	userID, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil || req.GetOrderId() == "" {
		return nil, bizpayment.ErrPaymentInvalidArgument
	}
	created, err := s.uc.Create(ctx, userID, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &v1.CreatePaymentResponse{Payment: toProto(created)}, nil
}

func (s *PaymentService) GetPayment(ctx context.Context, req *v1.GetPaymentRequest) (*v1.GetPaymentResponse, error) {
	userID, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	got, err := s.uc.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetPaymentResponse{Payment: toProto(got)}, nil
}

func (s *PaymentService) SimulatePayment(ctx context.Context, req *v1.SimulatePaymentRequest) (*v1.SimulatePaymentResponse, error) {
	userID, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	got, err := s.uc.Simulate(ctx, userID, id, req.GetSuccess())
	if err != nil {
		return nil, err
	}
	return &v1.SimulatePaymentResponse{Payment: toProto(got)}, nil
}

func (s *PaymentService) ListReconcileDiffs(ctx context.Context, req *v1.ListReconcileDiffsRequest) (*v1.ListReconcileDiffsResponse, error) {
	if err := requirePayAdmin(ctx); err != nil {
		return nil, err
	}
	size := int(req.GetPageSize())
	if size <= 0 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	offset := 0
	if token := strings.TrimSpace(req.GetPageToken()); token != "" {
		n, err := strconv.Atoi(token)
		if err != nil || n < 0 {
			return nil, bizpayment.ErrPaymentInvalidArgument
		}
		offset = n
	}
	rows, hasMore, err := s.uc.ListDiffs(ctx, strings.TrimSpace(req.GetDay()), size, offset)
	if err != nil {
		return nil, err
	}
	out := &v1.ListReconcileDiffsResponse{}
	for _, row := range rows {
		out.Diffs = append(out.Diffs, &v1.ReconcileDiff{
			Id: row.ID, Day: row.Day, Kind: row.Kind, OrderId: row.OrderID, PaymentId: row.PaymentID, Detail: row.Detail,
		})
	}
	if hasMore {
		out.NextPageToken = strconv.Itoa(offset + size)
	}
	return out, nil
}

func requirePayAdmin(ctx context.Context) error {
	md, ok := kmd.FromServerContext(ctx)
	if !ok {
		return kerrors.Unauthorized("PAYMENT_UNAUTHENTICATED", "unauthenticated")
	}
	id := strings.TrimSpace(md.Get("x-md-global-admin-id"))
	role := strings.TrimSpace(md.Get("x-md-global-admin-role"))
	if id == "" || role == "" {
		return kerrors.Unauthorized("PAYMENT_UNAUTHENTICATED", "unauthenticated")
	}
	if role != "order" && role != "super" {
		return kerrors.Forbidden("PAYMENT_FORBIDDEN", "forbidden")
	}
	return nil
}

func callerID(ctx context.Context) (uuid.UUID, error) {
	md, ok := kmd.FromServerContext(ctx)
	if !ok {
		return uuid.Nil, kerrors.Unauthorized("PAYMENT_UNAUTHENTICATED", "unauthenticated")
	}
	id, err := uuid.Parse(md.Get("x-md-global-user-id"))
	if err != nil {
		return uuid.Nil, kerrors.Unauthorized("PAYMENT_UNAUTHENTICATED", "unauthenticated")
	}
	return id, nil
}

func parseID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, bizpayment.ErrPaymentInvalidArgument
	}
	return parsed, nil
}

func toProto(p *bizpayment.Payment) *v1.Payment {
	if p == nil {
		return nil
	}
	return &v1.Payment{
		Id:      p.ID.String(),
		OrderId: p.OrderID,
		Amount:  p.Amount,
		Status:  p.Status,
		Channel: p.Channel,
	}
}
