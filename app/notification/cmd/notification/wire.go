//go:build wireinject
// +build wireinject

package main

import (
	"log/slog"

	"github.com/tmjwjx/supermarket/app/notification/internal/biz"
	"github.com/tmjwjx/supermarket/app/notification/internal/conf"
	"github.com/tmjwjx/supermarket/app/notification/internal/data"
	"github.com/tmjwjx/supermarket/app/notification/internal/server"
	"github.com/tmjwjx/supermarket/app/notification/internal/service"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
)

func wireApp(*conf.Server, *conf.Data, *conf.Auth, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
