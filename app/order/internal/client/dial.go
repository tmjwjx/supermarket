package client

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	ggrpc "google.golang.org/grpc"
)

func dial(addr, fallback string, timeout time.Duration) (*ggrpc.ClientConn, error) {
	if addr == "" {
		addr = fallback
	}
	if timeout <= 0 {
		timeout = time.Second
	}
	return grpc.NewClient(context.Background(),
		grpc.WithEndpoint(addr),
		grpc.WithTimeout(timeout),
		grpc.WithMiddleware(tracing.Client(), metadata.Client()),
	)
}
