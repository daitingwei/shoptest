package data

import (
	"order/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewData, NewOrderRepo)

type Data struct {
	db  *gorm.DB
	rdb redis.UniversalClient
}

// DB 返回数据库实例
func (d *Data) DB() *gorm.DB {
	return d.db
}

// NewData 初始化数据层资源，包括 MySQL 和 Redis 连接
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
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

	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		rdb.Close()
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
	return &Data{db: db, rdb: rdb}, cleanup, nil
}
