package main

import (
	"flag"
	"log/slog"
	"os"

	bizpayment "github.com/tmjwjx/supermarket/app/payment/internal/biz/payment"
	"github.com/tmjwjx/supermarket/app/payment/internal/conf"
	"github.com/tmjwjx/supermarket/pkg/kafkaout"
	"github.com/tmjwjx/supermarket/pkg/secretcheck"
	"github.com/tmjwjx/supermarket/pkg/trace"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/file"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/go-kratos/kratos/v3/transport/http"

	_ "go.uber.org/automaxprocs"
)

var (
	Name     string
	Version  string
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/dev.yaml", "config path, eg: -conf config.yaml")
}

func newApp(logger *slog.Logger, gs *grpc.Server, hs *http.Server, _ *bizpayment.RetryLoop, _ *bizpayment.ReconcileLoop, _ *bizpayment.Streams) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
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
		secretcheck.Key{Name: "jwt_secret", Value: bc.Auth.JWTSecret},
		secretcheck.Key{Name: "admin_jwt_secret", Value: bc.Auth.AdminJWTSecret},
	); err != nil {
		log.Error("invalid jwt secrets", "err", err)
		os.Exit(1)
	}
	kafkaout.UseBrokers(bc.Kafka.Brokers)

	app, cleanup, err := wireApp(&bc.Server, &bc.Data, &bc.Client, &bc.Payment, &bc.Auth, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		panic(err)
	}
}
