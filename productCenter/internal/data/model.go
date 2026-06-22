package data

import (
	"time"

	"gorm.io/gorm"
)

// Shop 店铺数据库模型，对应 shops 表
type Shop struct {
	ID          int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ShopName    string         `gorm:"column:shop_name;type:varchar(255);not null" json:"shop_name"`
	Description string         `gorm:"column:description;type:text" json:"description"`
	CreatedAt   time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 指定 Shop 模型对应的数据库表名
func (Shop) TableName() string { return "shops" }

// ProductTag 商品标签数据库模型，对应 product_tag 表
type ProductTag struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"column:name;type:varchar(500);not null" json:"name"`
	Sort      int            `gorm:"column:sort;type:int;default:0" json:"sort"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 指定 ProductTag 模型对应的数据库表名
func (ProductTag) TableName() string { return "product_tag" }

// Product 商品数据库模型，对应 products 表
type Product struct {
	ID             int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ShopID         int64          `gorm:"column:shop_id;type:bigint;not null;index" json:"shop_id"`
	Name           string         `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Type           string         `gorm:"column:type;type:varchar(255);index" json:"type"`
	Description    string         `gorm:"column:description;type:text" json:"description"`
	MainImageURL   string         `gorm:"column:main_image_url;type:varchar(500)" json:"main_image_url"`
	Price          int            `gorm:"column:price;type:int;default:0" json:"price"`
	CompareAtPrice int            `gorm:"column:compare_at_price;type:int;default:0" json:"compare_at_price"`
	Status         int8           `gorm:"column:status;type:tinyint;default:0" json:"status"`
	Sort           int            `gorm:"column:sort;type:int;default:0" json:"sort"`
	CreatedAt      time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
	Tags           []ProductTag   `gorm:"many2many:product_tag_mapping;" json:"tags,omitempty"`
}

// TableName 指定 Product 模型对应的数据库表名
func (Product) TableName() string { return "products" }

// ProductMedia 商品媒体数据库模型，对应 product_media 表
type ProductMedia struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID int64          `gorm:"column:product_id;type:bigint;not null;index" json:"product_id"`
	URL       string         `gorm:"column:url;type:varchar(500);not null" json:"url"`
	Sort      int            `gorm:"column:sort;type:int;default:0" json:"sort"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 指定 ProductMedia 模型对应的数据库表名
func (ProductMedia) TableName() string { return "product_media" }

// ProductTagMapping 商品与标签关联表模型，对应 product_tag_mapping 表
type ProductTagMapping struct {
	ProductID int64     `gorm:"column:product_id;primaryKey" json:"product_id"`
	TagID     int64     `gorm:"column:tag_id;primaryKey" json:"tag_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

// TableName 指定 ProductTagMapping 模型对应的数据库表名
func (ProductTagMapping) TableName() string { return "product_tag_mapping" }

// Sku 商品SKU数据库模型，对应 sku 表
type Sku struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID int64          `gorm:"column:product_id;type:bigint;not null;index" json:"product_id"`
	Sku       string         `gorm:"column:sku;type:varchar(255);not null" json:"sku"`
	Title     string         `gorm:"column:title;type:varchar(100)" json:"title"`
	Price     int            `gorm:"column:price;type:int;default:0" json:"price"`
	Stock     int            `gorm:"column:stock;type:int;default:0" json:"stock"`
	ImgURL    string         `gorm:"column:img_url;type:varchar(500)" json:"img_url"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 指定 Sku 模型对应的数据库表名
func (Sku) TableName() string { return "sku" }
