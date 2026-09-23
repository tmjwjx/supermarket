package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orderv1 "github.com/tmjwjx/supermarket/api/order/v1"
	paymentv1 "github.com/tmjwjx/supermarket/api/payment/v1"
	productv1 "github.com/tmjwjx/supermarket/api/product/v1"
	userv1 "github.com/tmjwjx/supermarket/api/user/v1"
	"github.com/tmjwjx/supermarket/app/gateway/internal/auth"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestCheckoutFlowOrder(t *testing.T) {
	var seq []string
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gs := grpc.NewServer()
	userv1.RegisterUserServiceServer(gs, &flowUsers{seq: &seq})
	productv1.RegisterProductServiceServer(gs, &flowProducts{seq: &seq})
	orderv1.RegisterCartServiceServer(gs, &flowCart{seq: &seq})
	orderv1.RegisterOrderServiceServer(gs, &flowOrders{seq: &seq})
	paymentv1.RegisterPaymentServiceServer(gs, &flowPays{seq: &seq})
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(func() {
		gs.Stop()
		_ = lis.Close()
	})
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	handler := NewHTTPServer(&conf.Server{}, Services{
		Users:       userv1.NewUserServiceClient(conn),
		Products:    productv1.NewProductServiceClient(conn),
		Carts:       orderv1.NewCartServiceClient(conn),
		Orders:      orderv1.NewOrderServiceClient(conn),
		Payments:    paymentv1.NewPaymentServiceClient(conn),
		Tokens:      auth.NewVerifier(&conf.Auth{JWTSecret: "user-secret"}),
		AdminTokens: auth.NewAdminVerifier(&conf.Auth{AdminJWTSecret: "admin-secret"}),
	})
	user := bearer(t, "user-secret", jwt.RegisteredClaims{
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	steps := []struct {
		method string
		path   string
		body   string
		header string
	}{
		{http.MethodPost, "/v1/users/register", `{"phone":"13800001111","password":"secret1","nickname":"街口"}`, ""},
		{http.MethodGet, "/v1/products", "", ""},
		{http.MethodGet, "/v1/products/p1", "", ""},
		{http.MethodPost, "/v1/cart/items", `{"sku_id":"s1","quantity":1}`, user},
		{http.MethodPost, "/v1/orders", `{"lines":[{"sku_id":"s1","quantity":1}],"address_id":"a1","request_id":"r1"}`, user},
		{http.MethodPost, "/v1/payments/pay1:simulate", `{"success":true}`, user},
	}
	for _, step := range steps {
		code := callOK(t, handler, step.method, step.path, step.body, step.header)
		if code != http.StatusOK {
			t.Fatalf("%s %s status %d seq %v", step.method, step.path, code, seq)
		}
	}
	want := []string{"register", "browse", "detail", "cart", "order", "pay"}
	if strings.Join(seq, ",") != strings.Join(want, ",") {
		t.Fatalf("seq %v", seq)
	}
}

func callOK(t *testing.T, handler http.Handler, method, path, body, header string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res.Code
}

type flowUsers struct {
	userv1.UnimplementedUserServiceServer
	seq *[]string
}

func (s *flowUsers) Register(context.Context, *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	*s.seq = append(*s.seq, "register")
	return &userv1.RegisterResponse{AccessToken: "tok", User: &userv1.User{Id: "user-1"}}, nil
}

type flowProducts struct {
	productv1.UnimplementedProductServiceServer
	seq *[]string
}

func (s *flowProducts) ListProducts(context.Context, *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	*s.seq = append(*s.seq, "browse")
	return &productv1.ListProductsResponse{Products: []*productv1.ProductCard{{Id: "p1", Name: "牛奶"}}}, nil
}

func (s *flowProducts) GetProduct(context.Context, *productv1.GetProductRequest) (*productv1.GetProductResponse, error) {
	*s.seq = append(*s.seq, "detail")
	return &productv1.GetProductResponse{Product: &productv1.Product{Id: "p1", Name: "牛奶"}}, nil
}

type flowCart struct {
	orderv1.UnimplementedCartServiceServer
	seq *[]string
}

func (s *flowCart) AddCartItem(context.Context, *orderv1.AddCartItemRequest) (*orderv1.AddCartItemResponse, error) {
	*s.seq = append(*s.seq, "cart")
	return &orderv1.AddCartItemResponse{Item: &orderv1.CartItem{SkuId: "s1", Quantity: 1}}, nil
}

type flowOrders struct {
	orderv1.UnimplementedOrderServiceServer
	seq *[]string
}

func (s *flowOrders) CreateOrder(context.Context, *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	*s.seq = append(*s.seq, "order")
	return &orderv1.CreateOrderResponse{Order: &orderv1.Order{Id: "o1", Status: 1}}, nil
}

type flowPays struct {
	paymentv1.UnimplementedPaymentServiceServer
	seq *[]string
}

func (s *flowPays) SimulatePayment(context.Context, *paymentv1.SimulatePaymentRequest) (*paymentv1.SimulatePaymentResponse, error) {
	*s.seq = append(*s.seq, "pay")
	return &paymentv1.SimulatePaymentResponse{Payment: &paymentv1.Payment{Id: "pay1", OrderId: "o1", Status: 2}}, nil
}
