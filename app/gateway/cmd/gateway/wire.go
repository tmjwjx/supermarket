//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"log/slog"

	"github.com/tmjwjx/supermarket/app/gateway/internal/auth"
	"github.com/tmjwjx/supermarket/app/gateway/internal/client"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"
	"github.com/tmjwjx/supermarket/app/gateway/internal/server"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Client, *conf.Auth, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, client.ProviderSet, auth.ProviderSet, newApp))
}
