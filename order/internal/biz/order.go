package biz

import (
	"context"
	"fmt"

	v1 "order/api/order/v1"
	"github.com/go-kratos/kratos/v2/errors"
)

// OrderRepo 订单数据仓库接口，由 data 层实现
type OrderRepo interface {
	CreateOrder(ctx context.Context, order *Order) (*Order, error)
	GetOrder(ctx context.Context, orderID int64) (*Order, error)
	GetOrderByRequestID(ctx context.Context, requestID string) (*Order, error)
	IdempotentSetNX(ctx context.Context, requestID string) (bool, error)
	IdempotentDel(ctx context.Context, requestID string) error
	ListOrders(ctx context.Context, userID int64, page, pageSize int, status OrderStatus) ([]*Order, int32, error)
	UpdateOrderStatus(ctx context.Context, orderID int64, status OrderStatus) error
	CancelOrder(ctx context.Context, orderID int64) error
	// DeductStock 扣减库存，调用ProductCenter gRPC
	DeductStock(ctx context.Context, skuID int64, quantity int) error
	// RestoreStock 回补库存，调用ProductCenter gRPC
	RestoreStock(ctx context.Context, skuID int64, quantity int) error
}

// ErrInsufficientStock 库存不足错误
var ErrInsufficientStock = errors.New(404, "INSUFFICIENT_STOCK", "库存不足")

// OrderUseCase 订单业务用例
type OrderUseCase struct {
	repo OrderRepo
}

// NewOrderUseCase 创建订单业务用例实例
func NewOrderUseCase(repo OrderRepo) *OrderUseCase {
	return &OrderUseCase{repo: repo}
}

// CreateOrder 创建订单，包含三级幂等保护（Redis SET NX → DB查询 → 唯一索引兜底）
func (uc *OrderUseCase) CreateOrder(ctx context.Context, requestID string, userID, shopID int64, items []*OrderItem) (*Order, error) {
	// 1. Redis SET NX 幂等判断（异常降级不阻塞下单）
	ok, err := uc.repo.IdempotentSetNX(ctx, requestID)
	if err == nil && !ok {
		return uc.repo.GetOrderByRequestID(ctx, requestID)
	}

	// 2. 数据库查询兜底
	existing, err := uc.repo.GetOrderByRequestID(ctx, requestID)
	if err == nil && existing != nil {
		return existing, nil
	}

	// 3. 校验订单项
	if len(items) == 0 {
		return nil, fmt.Errorf(v1.ErrorReason_PARAMETER_ERROR.String())
	}

	// 4. 计算总价
	var totalAmount int
	for _, item := range items {
		totalAmount += item.Price * item.Quantity
	}

	// 5. 创建订单
	order := &Order{
		RequestID:   requestID,
		UserID:      userID,
		ShopID:      shopID,
		TotalAmount: totalAmount,
		Status:      int(OrderStatusPending),
		PayStatus:   int(PayStatusUnpaid),
		Items:       items,
	}

	created, err := uc.repo.CreateOrder(ctx, order)
	if err != nil {
		// 创建失败 → 删除 Redis key，允许重试
		_ = uc.repo.IdempotentDel(ctx, requestID)
		return nil, err
	}

	return created, nil
}

// GetOrder 根据ID获取订单
func (uc *OrderUseCase) GetOrder(ctx context.Context, orderID int64) (*Order, error) {
	return uc.repo.GetOrder(ctx, orderID)
}

// ListOrders 分页查询用户订单列表
func (uc *OrderUseCase) ListOrders(ctx context.Context, userID int64, page, pageSize int, status OrderStatus) ([]*Order, int32, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return uc.repo.ListOrders(ctx, userID, page, pageSize, status)
}

// UpdateOrderStatus 更新订单状态
func (uc *OrderUseCase) UpdateOrderStatus(ctx context.Context, orderID int64, status OrderStatus) error {
	return uc.repo.UpdateOrderStatus(ctx, orderID, status)
}

// CancelOrder 取消订单
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
