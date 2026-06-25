package data

import (
	"context"

	"order/internal/conf"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewData, NewOrderRepo)

type Data struct {
	db     *gorm.DB
	rdb    redis.UniversalClient
	pcConn *grpc.ClientConn
}

func (d *Data) DB() *gorm.DB {
	return d.db
}

func NewData(c *conf.Data, logger klog.Logger) (*Data, func(), error) {
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

	pcConn, err := kratosgrpc.DialInsecure(
		context.Background(),
		kratosgrpc.WithEndpoint("127.0.0.1:9003"),
	)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		klog.NewHelper(logger).Info("closing the data resources")
		rdb.Close()
		pcConn.Close()
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
	return &Data{db: db, rdb: rdb, pcConn: pcConn}, cleanup, nil
}
