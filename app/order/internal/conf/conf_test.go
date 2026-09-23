package conf

import (
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/file"
)

// order 服务端超时要比网关调 order 的客户端超时短
func TestOrderServerTimeout(t *testing.T) {
	c := config.New(config.WithSource(file.NewSource("../../configs")))
	defer c.Close()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	var bc Bootstrap
	if err := c.Scan(&bc); err != nil {
		t.Fatal(err)
	}
	if bc.Server.HTTP.Timeout() != 5*time.Second || bc.Server.GRPC.Timeout() != 5*time.Second {
		t.Fatalf("http %v grpc %v", bc.Server.HTTP.Timeout(), bc.Server.GRPC.Timeout())
	}
}
