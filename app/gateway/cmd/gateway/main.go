package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"
	"github.com/tmjwjx/supermarket/pkg/secretcheck"
	"github.com/tmjwjx/supermarket/pkg/trace"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/env"
	"github.com/go-kratos/kratos/v3/config/file"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport/http"

	_ "go.uber.org/automaxprocs"
)

// Name 与 Version 由 ldflags 注入 flagconf 指向配置目录
var (
	Name     string
	Version  string
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

// 网关只挂 HTTP
func newApp(logger *slog.Logger, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			hs,
		),
	)
}

func main() {
	flag.Parse()
	defer trace.Setup()()
	logger := log.NewLogger(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}),
		log.WithExtractor(tracing.TraceAttrs),
	).With(
		slog.String("service.id", id),
		slog.String("service.name", Name),
		slog.String("service.version", Version),
	)
	log.SetDefault(logger)
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
			env.NewSource(),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}
	if err := secretcheck.Check(
		secretcheck.Key{Name: "AUTH_JWT_SECRET", Value: bc.Auth.JWTSecret},
		secretcheck.Key{Name: "ADMIN_JWT_SECRET", Value: bc.Auth.AdminJWTSecret},
	); err != nil {
		log.Error("invalid jwt secrets", "err", err)
		os.Exit(1)
	}

	app, cleanup, err := wireApp(&bc.Server, &bc.Client, &bc.Auth, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// 启动后阻塞直到收到停止信号
	if err := app.Run(); err != nil {
		panic(err)
	}
}
