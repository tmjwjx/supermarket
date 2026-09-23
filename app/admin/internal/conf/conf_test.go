package conf

import (
	"testing"

	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/env"
	"github.com/go-kratos/kratos/v3/config/file"
)

func scan(t *testing.T, sources ...config.Source) Bootstrap {
	t.Helper()
	c := config.New(config.WithSource(sources...))
	t.Cleanup(func() { _ = c.Close() })
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	var bc Bootstrap
	if err := c.Scan(&bc); err != nil {
		t.Fatal(err)
	}
	return bc
}

// 和 main 一样的配置源 占位符要能读到同名环境变量
func TestPlaceholderReadsPlainEnv(t *testing.T) {
	t.Setenv("ADMIN_JWT_SECRET", "x")
	bc := scan(t, file.NewSource("../../configs"), env.NewSource())
	if bc.Auth.JWTSecret != "x" {
		t.Fatalf("jwt_secret %q", bc.Auth.JWTSecret)
	}
}

// 只加载 KRATOS_ 前缀时读不到 这是改成不带前缀的原因
func TestPrefixedEnvMissesPlaceholder(t *testing.T) {
	t.Setenv("ADMIN_JWT_SECRET", "x")
	bc := scan(t, file.NewSource("../../configs"), env.NewSource("KRATOS"))
	if bc.Auth.JWTSecret == "x" {
		t.Fatal("prefixed env source should not resolve plain ADMIN_JWT_SECRET")
	}
}
