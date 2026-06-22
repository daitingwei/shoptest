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
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server,
	greeter *service.GreeterService,
	shop *service.ShopService,
	product *service.ProductService,
	sku *service.SkuService,
	productTag *service.ProductTagService,
	productMedia *service.ProductMediaService,
	bff *service.BFFService,
	logger log.Logger,
) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	hellov1.RegisterGreeterHTTPServer(srv, greeter)
	shopv1.RegisterShopHTTPServer(srv, shop)
	productv1.RegisterProductHTTPServer(srv, product)
	skuv1.RegisterSkuHTTPServer(srv, sku)
	tagv1.RegisterProductTagHTTPServer(srv, productTag)
	mediav1.RegisterProductMediaHTTPServer(srv, productMedia)
	bffv1.RegisterBFFHTTPServer(srv, bff)
	return srv
}
