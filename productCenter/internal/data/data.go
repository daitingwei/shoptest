package data

import (
	"time"

	"productCenter/internal/conf"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlog "gorm.io/gorm/logger"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewGreeterRepo,
	NewShopRepo,
	NewProductTagRepo,
	NewProductRepo,
	NewProductMediaRepo,
	NewSkuRepo,
)

// Data 数据层封装结构体，持有数据库连接等资源
type Data struct {
	db *gorm.DB
}

// NewData 初始化数据层资源，建立 MySQL 数据库连接
func NewData(c *conf.Data, logger klog.Logger) (*Data, func(), error) {
	logHelper := klog.NewHelper(logger)

	gormLogger := gormlog.Default.LogMode(gormlog.Info)
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		logHelper.Errorf("failed to connect mysql: %v", err)
		return nil, nil, err
	}

	// 自动迁移：启动时自动创建/更新数据表结构
	if err := db.AutoMigrate(
		&Shop{},
		&ProductTag{},
		&Product{},
		&ProductMedia{},
		&ProductTagMapping{},
		&Sku{},
	); err != nil {
		logHelper.Errorf("failed to auto migrate: %v", err)
		return nil, nil, err
	}
	logHelper.Info("auto migrate done")

	sqlDB, err := db.DB()
	if err != nil {
		logHelper.Errorf("failed to get underlying sql.DB: %v", err)
		return nil, nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	cleanup := func() {
		logHelper.Info("closing the data resources")
		sqlDB.Close()
	}

	return &Data{db: db}, cleanup, nil
}
