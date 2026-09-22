//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"log/slog"

	"github.com/tmjwjx/supermarket/app/user/internal/biz"
	"github.com/tmjwjx/supermarket/app/user/internal/conf"
	"github.com/tmjwjx/supermarket/app/user/internal/data"
	"github.com/tmjwjx/supermarket/app/user/internal/server"
	"github.com/tmjwjx/supermarket/app/user/internal/service"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.Auth, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
