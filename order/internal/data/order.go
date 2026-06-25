package data

import (
	"context"
	"fmt"
	"time"

	"order/internal/biz"
	"gorm.io/gorm"
)

type OrderRepo struct {
	data *Data
}

func NewOrderRepo(data *Data) biz.OrderRepo {
	return &OrderRepo{data: data}
}

func (r *OrderRepo) CreateOrder(ctx context.Context, order *biz.Order) (*biz.Order, error) {
	order.OrderNo = generateOrderNo()

	err := r.data.db.Transaction(func(tx *gorm.DB) error {
		orderModel := &Order{
			OrderNo:     order.OrderNo,
			UserID:      order.UserID,
			ShopID:      order.ShopID,
			TotalAmount: order.TotalAmount,
			Status:      order.Status,
			PayStatus:   order.PayStatus,
		}

		if err := tx.Create(orderModel).Error; err != nil {
			return err
		}

		for _, item := range order.Items {
			orderItemModel := &OrderItem{
				OrderID:     int64(orderModel.ID),
				ProductID:   item.ProductID,
				SKUID:       item.SKUID,
				ProductName: item.ProductName,
				SKUTitle:    item.SKUTitle,
				Price:       item.Price,
				Quantity:    item.Quantity,
				ImageURL:    item.ImageURL,
			}
			if err := tx.Create(orderItemModel).Error; err != nil {
				return err
			}
		}

		order.ID = orderModel.ID
		return nil
	})

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (r *OrderRepo) GetOrder(ctx context.Context, orderID int64) (*biz.Order, error) {
	var orderModel Order
	if err := r.data.db.Preload("Items").First(&orderModel, orderID).Error; err != nil {
		return nil, err
	}

	return convertOrderToBiz(&orderModel), nil
}

func (r *OrderRepo) ListOrders(ctx context.Context, userID int64, page, pageSize int, status biz.OrderStatus) ([]*biz.Order, int32, error) {
	var orders []Order
	var total int64

	query := r.data.db.Model(&Order{}).Where("user_id = ?", userID)
	if status >= 0 {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("Items").Offset(offset).Limit(pageSize).Order("id DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.Order, len(orders))
	for i := range orders {
		result[i] = convertOrderToBiz(&orders[i])
	}

	return result, int32(total), nil
}

func (r *OrderRepo) UpdateOrderStatus(ctx context.Context, orderID int64, status biz.OrderStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	switch status {
	case biz.OrderStatusPaid:
		now := time.Now()
		updates["pay_time"] = now
	case biz.OrderStatusShipped:
		now := time.Now()
		updates["ship_time"] = now
	case biz.OrderStatusCompleted:
		now := time.Now()
		updates["confirm_time"] = now
	}

	return r.data.db.Model(&Order{}).Where("id = ?", orderID).Updates(updates).Error
}

func (r *OrderRepo) CancelOrder(ctx context.Context, orderID int64) error {
	return r.data.db.Model(&Order{}).Where("id = ? AND status = ?", orderID, biz.OrderStatusPending).Update("status", biz.OrderStatusCancelled).Error
}

func convertOrderToBiz(orderModel *Order) *biz.Order {
	if orderModel == nil {
		return nil
	}

	order := &biz.Order{
		OrderNo:     orderModel.OrderNo,
		UserID:      orderModel.UserID,
		ShopID:      orderModel.ShopID,
		TotalAmount: orderModel.TotalAmount,
		Status:      orderModel.Status,
		PayStatus:   orderModel.PayStatus,
		PayTime:     orderModel.PayTime,
		ShipTime:    orderModel.ShipTime,
		ConfirmTime: orderModel.ConfirmTime,
	}

	if len(orderModel.Items) > 0 {
		order.Items = make([]*biz.OrderItem, len(orderModel.Items))
		for i := range orderModel.Items {
			order.Items[i] = &biz.OrderItem{
				OrderID:     orderModel.Items[i].OrderID,
				ProductID:   orderModel.Items[i].ProductID,
				SKUID:       orderModel.Items[i].SKUID,
				ProductName: orderModel.Items[i].ProductName,
				SKUTitle:    orderModel.Items[i].SKUTitle,
				Price:       orderModel.Items[i].Price,
				Quantity:    orderModel.Items[i].Quantity,
				ImageURL:    orderModel.Items[i].ImageURL,
			}
		}
	}

	return order
}

func generateOrderNo() string {
	now := time.Now()
	return fmt.Sprintf("ORD%d%d%d%d%d%d%d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second(), now.Nanosecond()%1000000)
}