package biz

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	RequestID   string       `json:"request_id"`
	OrderNo     string       `json:"order_no"`
	UserID      int64        `json:"user_id"`
	ShopID      int64        `json:"shop_id"`
	TotalAmount int          `json:"total_amount"`
	Status      int          `json:"status"`
	PayStatus   int          `json:"pay_status"`
	PayTime     *time.Time   `json:"pay_time"`
	ShipTime    *time.Time   `json:"ship_time"`
	ConfirmTime *time.Time   `json:"confirm_time"`
	Items       []*OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	gorm.Model
	OrderID     int64  `json:"order_id"`
	ProductID   int64  `json:"product_id"`
	SKUID       int64  `json:"sku_id"`
	ProductName string `json:"product_name"`
	SKUTitle    string `json:"sku_title"`
	Price       int    `json:"price"`
	Quantity    int    `json:"quantity"`
	ImageURL    string `json:"image_url"`
}

type OrderStatus int32

const (
	OrderStatusPending   OrderStatus = 0
	OrderStatusPaid      OrderStatus = 1
	OrderStatusShipped   OrderStatus = 2
	OrderStatusCompleted OrderStatus = 3
	OrderStatusCancelled OrderStatus = 4
)

type PayStatus int32

const (
	PayStatusUnpaid    PayStatus = 0
	PayStatusPaid      PayStatus = 1
	PayStatusFailed    PayStatus = 2
	PayStatusRefunding PayStatus = 3
	PayStatusRefunded  PayStatus = 4
)
