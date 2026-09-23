package client

import (
	"context"

	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	kgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/google/wire"
	"google.golang.org/grpc"
)

var ProviderSet = wire.NewSet(
	NewUserClient,
	wire.FieldsOf(new(*UserClients), "Users", "Addresses"),
	NewProductClient,
	wire.FieldsOf(new(*ProductClients), "Products", "Brands", "Categories", "Favorites", "Histories", "Reviews", "Recommendations", "Attributes"),
	NewInventoryClient,
	NewOrderClient,
	wire.FieldsOf(new(*OrderClients), "Carts", "Orders"),
	NewPaymentClient,
	NewNotificationClient,
	NewAdminClient,
)

// 拨上游 gRPC 并挂上身份元数据和追踪
func dial(ep conf.Endpoint) (*grpc.ClientConn, error) {
	opts := []kgrpc.ClientOption{
		kgrpc.WithEndpoint(ep.Addr),
		kgrpc.WithMiddleware(metadata.Client(), tracing.Client()),
	}
	if d := ep.Timeout(); d != 0 {
		opts = append(opts, kgrpc.WithTimeout(d))
	}
	return kgrpc.NewClient(context.Background(), opts...)
}
