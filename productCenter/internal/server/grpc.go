package server

import (
	bffv1 "productCenter/api/bff/v1"
	hellov1 "productCenter/api/helloworld/v1"
	productv1 "productCenter/api/product/v1"
	mediav1 "productCenter/api/productmedia/v1"
	tagv1 "productCenter/api/producttag/v1"
	shopv1 "productCenter/api/shop/v1"
	skuv1 "productCenter/api/sku/v1"
	"productCenter/internal/conf"
	"productCenter/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server,
	greeter *service.GreeterService,
	shop *service.ShopService,
	product *service.ProductService,
	sku *service.SkuService,
	productTag *service.ProductTagService,
	productMedia *service.ProductMediaService,
	bff *service.BFFService,
	logger log.Logger,
) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	hellov1.RegisterGreeterServer(srv, greeter)
	shopv1.RegisterShopServer(srv, shop)
	productv1.RegisterProductServer(srv, product)
	skuv1.RegisterSkuServer(srv, sku)
	tagv1.RegisterProductTagServer(srv, productTag)
	mediav1.RegisterProductMediaServer(srv, productMedia)
	bffv1.RegisterBFFServer(srv, bff)
	return srv
}
