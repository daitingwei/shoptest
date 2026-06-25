//go:build wireinject
// +build wireinject

package main

import (
	"order/internal/biz"
	"order/internal/conf"
	"order/internal/data"
	"order/internal/server"
	"order/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

func wireApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
