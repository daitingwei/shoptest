package data

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	OrderNo     string      `gorm:"column:order_no;type:varchar(64);uniqueIndex;not null" json:"order_no"`
	UserID      int64       `gorm:"column:user_id;type:bigint;not null;index" json:"user_id"`
	ShopID      int64       `gorm:"column:shop_id;type:bigint;not null;index" json:"shop_id"`
	TotalAmount int         `gorm:"column:total_amount;type:int;not null;default:0" json:"total_amount"`
	Status      int         `gorm:"column:status;type:tinyint;not null;default:0" json:"status"`
	PayStatus   int         `gorm:"column:pay_status;type:tinyint;not null;default:0" json:"pay_status"`
	PayTime     *time.Time  `gorm:"column:pay_time" json:"pay_time"`
	ShipTime    *time.Time  `gorm:"column:ship_time" json:"ship_time"`
	ConfirmTime *time.Time  `gorm:"column:confirm_time" json:"confirm_time"`
	Items       []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	gorm.Model
	OrderID     int64  `gorm:"column:order_id;type:bigint;not null;index" json:"order_id"`
	ProductID   int64  `gorm:"column:product_id;type:bigint;not null" json:"product_id"`
	SKUID       int64  `gorm:"column:sku_id;type:bigint;not null" json:"sku_id"`
	ProductName string `gorm:"column:product_name;type:varchar(255);not null" json:"product_name"`
	SKUTitle    string `gorm:"column:sku_title;type:varchar(255)" json:"sku_title"`
	Price       int    `gorm:"column:price;type:int;not null" json:"price"`
	Quantity    int    `gorm:"column:quantity;type:int;not null" json:"quantity"`
	ImageURL    string `gorm:"column:image_url;type:varchar(512)" json:"image_url"`
}

func (OrderItem) TableName() string {
	return "order_items"
}