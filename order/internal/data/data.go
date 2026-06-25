package data

import (
	"order/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewData, NewOrderRepo)

type Data struct {
	db *gorm.DB
}

func (d *Data) DB() *gorm.DB {
	return d.db
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	db.AutoMigrate(&Order{}, &OrderItem{})

	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
	return &Data{db: db}, cleanup, nil
}