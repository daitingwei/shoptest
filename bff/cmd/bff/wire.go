//go:build wireinject
// +build wireinject

package main

import (
	"bff/internal/biz"
	"bff/internal/conf"
	"bff/internal/data"
	"bff/internal/server"
	"bff/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

func wireApp(*conf.Server, *conf.Data, *conf.Registry, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
