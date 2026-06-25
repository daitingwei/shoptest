package data

import (
	"context"

	"order/internal/conf"

	skuv1 "productCenter/api/sku/v1"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/wire"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewData, NewOrderRepo)

// Data 数据层，持有数据库、Redis和ProductCenter gRPC客户端
type Data struct {
	db        *gorm.DB
	rdb       redis.UniversalClient
	skuClient skuv1.SkuClient
}

// DB 返回数据库实例
func (d *Data) DB() *gorm.DB {
	return d.db
}

// NewData 初始化数据层资源，包括 MySQL、Redis 和通过Nacos发现的ProductCenter gRPC客户端
func NewData(c *conf.Data, rc *conf.Registry, disc registry.Discovery, logger klog.Logger) (*Data, func(), error) {
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	db.AutoMigrate(&Order{}, &OrderItem{})

	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
	})

	conn, err := kratosgrpc.DialInsecure(
		context.Background(),
		kratosgrpc.WithEndpoint("discovery:///productCenter"),
		kratosgrpc.WithDiscovery(disc),
	)
	if err != nil {
		return nil, nil, err
	}

	skuClient := skuv1.NewSkuClient(conn)

	cleanup := func() {
		klog.NewHelper(logger).Info("closing the data resources")
		rdb.Close()
		conn.Close()
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
	return &Data{db: db, rdb: rdb, skuClient: skuClient}, cleanup, nil
}
