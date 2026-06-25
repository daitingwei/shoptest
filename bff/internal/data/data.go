package data

import (
	"bff/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewData, NewBFFRepo)

// Data 数据层，持有 ProductCenter 各领域 gRPC 客户端
type Data struct {
	log       *log.Helper
	discovery registry.Discovery
}

// NewData 通过 Nacos 发现 ProductCenter，初始化数据层
func NewData(c *conf.Data, r registry.Registrar, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	return &Data{log: log.NewHelper(logger), discovery: r.(registry.Discovery)}, cleanup, nil
}
