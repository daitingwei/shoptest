//go:build wireinject
// +build wireinject

package main

import (
	"productCenter/internal/biz"
	"productCenter/internal/conf"
	"productCenter/internal/data"
	"productCenter/internal/server"
	"productCenter/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/wire"
)

func wireApp(*conf.Server, *conf.Data, log.Logger, registry.Registrar) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
