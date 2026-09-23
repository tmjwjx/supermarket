package conf

import (
	"testing"

	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/file"
)

// 密钥按 dev.yaml 里的字面值读取
func TestJWTSecretFromFile(t *testing.T) {
	c := config.New(config.WithSource(file.NewSource("../../configs/dev.yaml")))
	defer c.Close()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	var bc Bootstrap
	if err := c.Scan(&bc); err != nil {
		t.Fatal(err)
	}
	if bc.Auth.JWTSecret != "tmjwjx-admin-jwt-secret" {
		t.Fatalf("jwt_secret %q", bc.Auth.JWTSecret)
	}
	if bc.Data.Database.Source != "root:root@tcp(mysql:3306)/admin?timeout=5s&parseTime=True&loc=Local&charset=utf8mb4" {
		t.Fatalf("source %s", bc.Data.Database.Source)
	}
}
