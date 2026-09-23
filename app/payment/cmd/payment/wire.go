//go:build wireinject
// +build wireinject

package main

import (
	"log/slog"

	"github.com/tmjwjx/supermarket/app/payment/internal/conf"
	"github.com/tmjwjx/supermarket/app/payment/internal/server"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
)

func wireApp(*conf.Server, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, newApp))
}
