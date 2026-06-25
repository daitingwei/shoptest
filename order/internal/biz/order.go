package biz

import (
	"context"
	"fmt"

	v1 "order/api/order/v1"
)

type OrderRepo interface {
	CreateOrder(ctx context.Context, order *Order) (*Order, error)
	GetOrder(ctx context.Context, orderID int64) (*Order, error)
	ListOrders(ctx context.Context, userID int64, page, pageSize int, status OrderStatus) ([]*Order, int32, error)
	UpdateOrderStatus(ctx context.Context, orderID int64, status OrderStatus) error
	CancelOrder(ctx context.Context, orderID int64) error
}

type OrderUseCase struct {
	repo OrderRepo
}

func NewOrderUseCase(repo OrderRepo) *OrderUseCase {
	return &OrderUseCase{repo: repo}
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, userID, shopID int64, items []*OrderItem) (*Order, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf(v1.ErrorReason_PARAMETER_ERROR.String())
	}

	var totalAmount int
	for _, item := range items {
		totalAmount += item.Price * item.Quantity
	}

	order := &Order{
		UserID:      userID,
		ShopID:      shopID,
		TotalAmount: totalAmount,
		Status:      int(OrderStatusPending),
		PayStatus:   int(PayStatusUnpaid),
		Items:       items,
	}

	return uc.repo.CreateOrder(ctx, order)
}

func (uc *OrderUseCase) GetOrder(ctx context.Context, orderID int64) (*Order, error) {
	return uc.repo.GetOrder(ctx, orderID)
}

func (uc *OrderUseCase) ListOrders(ctx context.Context, userID int64, page, pageSize int, status OrderStatus) ([]*Order, int32, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return uc.repo.ListOrders(ctx, userID, page, pageSize, status)
}

func (uc *OrderUseCase) UpdateOrderStatus(ctx context.Context, orderID int64, status OrderStatus) error {
	return uc.repo.UpdateOrderStatus(ctx, orderID, status)
}

func (uc *OrderUseCase) CancelOrder(ctx context.Context, orderID int64) error {
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	if order.Status != int(OrderStatusPending) {
		return fmt.Errorf(v1.ErrorReason_ORDER_CANCEL_FAILED.String())
	}

	return uc.repo.CancelOrder(ctx, orderID)
}