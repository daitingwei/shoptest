package data

import (
	"context"

	"bff/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"
	"google.golang.org/grpc"
)

var ProviderSet = wire.NewSet(NewData, NewBFFRepo)

// Data 数据层，持有下游服务 gRPC 连接等资源
type Data struct {
	log       *log.Helper
	discovery registry.Discovery
	pcConn    *grpc.ClientConn // 连 productCenter，启动时建好
	orderConn *grpc.ClientConn // 连 order，启动时建好
}

// NewData 初始化数据层资源，通过 Nacos 连接下游服务
func NewData(c *conf.Data, disc registry.Discovery, logger log.Logger) (*Data, func(), error) {
	pcConn, err := kratosgrpc.DialInsecure(
		context.Background(),
		kratosgrpc.WithEndpoint("discovery:///productCenter"),
		kratosgrpc.WithDiscovery(disc),
	)
	if err != nil {
		return nil, nil, err
	}

	orderConn, err := kratosgrpc.DialInsecure(
		context.Background(),
		kratosgrpc.WithEndpoint("discovery:///order"),
		kratosgrpc.WithDiscovery(disc),
	)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		pcConn.Close()
		orderConn.Close()
	}
	return &Data{log: log.NewHelper(logger), discovery: disc, pcConn: pcConn, orderConn: orderConn}, cleanup, nil
}
