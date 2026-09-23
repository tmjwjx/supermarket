package conf

import (
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/file"
)

// 密钥和超时都按 dev.yaml 的字面值读取
func TestGatewayConfig(t *testing.T) {
	c := config.New(config.WithSource(file.NewSource("../../configs/dev.yaml")))
	defer c.Close()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	var bc Bootstrap
	if err := c.Scan(&bc); err != nil {
		t.Fatal(err)
	}
	if bc.Auth.JWTSecret != "tmjwjx-user-jwt-secret" || bc.Auth.AdminJWTSecret != "tmjwjx-admin-jwt-secret" {
		t.Fatalf("auth %+v", bc.Auth)
	}
	if bc.Client.User.Addr != "user:9000" || bc.Client.Product.Addr != "product:9001" || bc.Client.Inventory.Addr != "inventory:9002" || bc.Client.Order.Addr != "order:9003" || bc.Client.Payment.Addr != "payment:9004" || bc.Client.Notification.Addr != "notification:9005" || bc.Client.Admin.Addr != "admin:9006" {
		t.Fatalf("upstream %+v", bc.Client)
	}
	if bc.Server.HTTP.Timeout() != 7*time.Second || bc.Client.Order.Timeout() != 6*time.Second {
		t.Fatalf("server %v order client %v", bc.Server.HTTP.Timeout(), bc.Client.Order.Timeout())
	}
	if bc.Client.Product.Timeout() != time.Second || bc.Client.Payment.Timeout() != time.Second {
		t.Fatalf("other upstreams %v %v", bc.Client.Product.Timeout(), bc.Client.Payment.Timeout())
	}
}
