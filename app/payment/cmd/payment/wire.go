//go:build wireinject
// +build wireinject

package main

import (
	"log/slog"

	"github.com/tmjwjx/supermarket/app/payment/internal/biz"
	"github.com/tmjwjx/supermarket/app/payment/internal/client"
	"github.com/tmjwjx/supermarket/app/payment/internal/conf"
	"github.com/tmjwjx/supermarket/app/payment/internal/data"
	"github.com/tmjwjx/supermarket/app/payment/internal/server"
	"github.com/tmjwjx/supermarket/app/payment/internal/service"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
)

func wireApp(*conf.Server, *conf.Data, *conf.Client, *conf.Payment, *conf.Auth, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, client.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
