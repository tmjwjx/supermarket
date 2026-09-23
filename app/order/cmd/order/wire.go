//go:build wireinject
// +build wireinject

package main

import (
	"log/slog"

	"github.com/tmjwjx/supermarket/app/order/internal/biz"
	"github.com/tmjwjx/supermarket/app/order/internal/client"
	"github.com/tmjwjx/supermarket/app/order/internal/conf"
	"github.com/tmjwjx/supermarket/app/order/internal/data"
	"github.com/tmjwjx/supermarket/app/order/internal/server"
	"github.com/tmjwjx/supermarket/app/order/internal/service"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
)

func wireApp(*conf.Server, *conf.Data, *conf.Client, *conf.Auth, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, client.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
