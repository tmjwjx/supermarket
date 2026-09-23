package conf

import (
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/env"
	"github.com/go-kratos/kratos/v3/config/file"
)

// 两把密钥都从同名环境变量读 调 order 的超时要小于网关自身 大于 order 服务端
func TestGatewayConfig(t *testing.T) {
	t.Setenv("AUTH_JWT_SECRET", "buyer-x")
	t.Setenv("ADMIN_JWT_SECRET", "admin-x")
	c := config.New(config.WithSource(file.NewSource("../../configs"), env.NewSource()))
	defer c.Close()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	var bc Bootstrap
	if err := c.Scan(&bc); err != nil {
		t.Fatal(err)
	}
	if bc.Auth.JWTSecret != "buyer-x" || bc.Auth.AdminJWTSecret != "admin-x" {
		t.Fatalf("auth %+v", bc.Auth)
	}
	if bc.Server.HTTP.Timeout() != 7*time.Second || bc.Client.Order.Timeout() != 6*time.Second {
		t.Fatalf("server %v order client %v", bc.Server.HTTP.Timeout(), bc.Client.Order.Timeout())
	}
	if bc.Client.Product.Timeout() != time.Second || bc.Client.Payment.Timeout() != time.Second {
		t.Fatalf("other upstreams %v %v", bc.Client.Product.Timeout(), bc.Client.Payment.Timeout())
	}
}
