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
	OrderStatusPending           OrderStatus = 0 // 待处理
	OrderStatusUnpaid            OrderStatus = 1 // 待支付
	OrderStatusAwaitingShipment  OrderStatus = 2 // 待发货
	OrderStatusAwaitingCompleted OrderStatus = 3 // 待完成
	OrderStatusCompleted         OrderStatus = 4 // 已完成
	OrderStatusCancelled         OrderStatus = 5 // 已取消
	OrderStatusFailed            OrderStatus = 6 // 失败的订单
)

type PayStatus int32

const (
	PayStatusUnpaid    PayStatus = 0 // 未支付
	PayStatusPaid      PayStatus = 1 // 已支付
	PayStatusFailed    PayStatus = 2 // 支付失败
	PayStatusRefunding PayStatus = 3 // 退款中
	PayStatusRefunded  PayStatus = 4 // 已退款
)
