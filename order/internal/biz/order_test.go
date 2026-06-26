package biz

import (
	"context"
	"strings"
	"testing"
	"time"
)

// mockOrderRepo 模拟 OrderRepo 接口，用于单元测试
type mockOrderRepo struct {
	orders            map[int64]*Order
	ordersByRequestID map[string]*Order
	nextID            int64
	idempotentKEYS    map[string]bool

	createOrderErr     error
	getOrderErr        error
	idempotentSetNXErr error
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{
		orders:            make(map[int64]*Order),
		ordersByRequestID: make(map[string]*Order),
		idempotentKEYS:    make(map[string]bool),
		nextID:            1,
	}
}

func (m *mockOrderRepo) CreateOrder(ctx context.Context, order *Order) (*Order, error) {
	if m.createOrderErr != nil {
		return nil, m.createOrderErr
	}
	order.ID = uint(m.nextID)
	order.OrderNo = "ORD20260626000001"
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	m.nextID++
	m.orders[int64(order.ID)] = order
	m.ordersByRequestID[order.RequestID] = order
	return order, nil
}

func (m *mockOrderRepo) GetOrder(ctx context.Context, orderID int64) (*Order, error) {
	if m.getOrderErr != nil {
		return nil, m.getOrderErr
	}
	order, ok := m.orders[orderID]
	if !ok {
		return nil, &testError{"NOT_FOUND", "订单未找到"}
	}
	return order, nil
}

func (m *mockOrderRepo) GetOrderByRequestID(ctx context.Context, requestID string) (*Order, error) {
	order, ok := m.ordersByRequestID[requestID]
	if !ok {
		return nil, &testError{"NOT_FOUND", "订单未找到"}
	}
	return order, nil
}

func (m *mockOrderRepo) IdempotentSetNX(ctx context.Context, requestID string) (bool, error) {
	if m.idempotentSetNXErr != nil {
		return false, m.idempotentSetNXErr
	}
	if _, exists := m.idempotentKEYS[requestID]; exists {
		return false, nil
	}
	m.idempotentKEYS[requestID] = true
	return true, nil
}

func (m *mockOrderRepo) IdempotentDel(ctx context.Context, requestID string) error {
	delete(m.idempotentKEYS, requestID)
	return nil
}

func (m *mockOrderRepo) ListOrders(ctx context.Context, userID int64, page, pageSize int, status OrderStatus) ([]*Order, int32, error) {
	var result []*Order
	for _, order := range m.orders {
		if order.UserID == userID {
			if status < 0 || OrderStatus(order.Status) == status {
				result = append(result, order)
			}
		}
	}
	return result, int32(len(result)), nil
}

func (m *mockOrderRepo) UpdateOrderStatus(ctx context.Context, orderID int64, status OrderStatus) error {
	order, ok := m.orders[orderID]
	if !ok {
		return &testError{"NOT_FOUND", "订单未找到"}
	}
	order.Status = int(status)
	if status == OrderStatusAwaitingShipment {
		now := time.Now()
		order.PayStatus = int(PayStatusPaid)
		order.PayTime = &now
	}
	if status == OrderStatusAwaitingCompleted {
		now := time.Now()
		order.ShipTime = &now
	}
	if status == OrderStatusCompleted {
		now := time.Now()
		order.ConfirmTime = &now
	}
	return nil
}

func (m *mockOrderRepo) CancelOrder(ctx context.Context, orderID int64) error {
	order, ok := m.orders[orderID]
	if !ok {
		return &testError{"NOT_FOUND", "订单未找到"}
	}
	order.Status = int(OrderStatusCancelled)
	return nil
}

func (m *mockOrderRepo) DeductStock(ctx context.Context, skuID int64, quantity int) error {
	return nil
}

func (m *mockOrderRepo) RestoreStock(ctx context.Context, skuID int64, quantity int) error {
	return nil
}

type testError struct {
	code    string
	message string
}

func (e *testError) Error() string {
	return e.message
}

// buildTestItems 构建测试用订单项
func buildTestItems() []*OrderItem {
	return []*OrderItem{
		{ProductID: 1, SKUID: 1, ProductName: "商品A", SKUTitle: "标准版", Price: 100, Quantity: 2},
		{ProductID: 1, SKUID: 2, ProductName: "商品A", SKUTitle: "Pro版", Price: 200, Quantity: 1},
	}
}

// TestCreateOrder 测试创建订单
func TestCreateOrder(t *testing.T) {
	t.Run("成功创建订单", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		order, err := uc.CreateOrder(context.Background(), "req-001", 1, 1, buildTestItems())
		if err != nil {
			t.Fatalf("创建订单失败: %v", err)
		}
		if order.ID == 0 {
			t.Error("订单ID不应为0")
		}
		if order.OrderNo == "" {
			t.Error("订单编号不应为空")
		}
		if order.Status != int(OrderStatusPending) {
			t.Errorf("新订单状态应为待处理(0), 实际=%d", order.Status)
		}
		if order.PayStatus != int(PayStatusUnpaid) {
			t.Errorf("新订单支付状态应为未支付(0), 实际=%d", order.PayStatus)
		}
		expectedTotal := 100*2 + 200*1
		if order.TotalAmount != expectedTotal {
			t.Errorf("总金额应为%d, 实际=%d", expectedTotal, order.TotalAmount)
		}
	})

	t.Run("空订单项校验", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		_, err := uc.CreateOrder(context.Background(), "req-002", 1, 1, nil)
		if err == nil {
			t.Error("空订单项应该返回错误")
		}
	})

	t.Run("幂等性-重复请求", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		order1, err := uc.CreateOrder(context.Background(), "req-idempotent", 1, 1, buildTestItems())
		if err != nil {
			t.Fatalf("首次创建失败: %v", err)
		}

		order2, err := uc.CreateOrder(context.Background(), "req-idempotent", 1, 1, buildTestItems())
		if err != nil {
			t.Fatalf("幂等请求失败: %v", err)
		}
		if order1.ID != order2.ID {
			t.Errorf("幂等请求应返回相同订单, 首次ID=%d, 再次ID=%d", order1.ID, order2.ID)
		}
	})

	t.Run("精确计算带折扣混合项的总金额", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		items := []*OrderItem{
			{ProductID: 1, SKUID: 1, Price: 299, Quantity: 3},
			{ProductID: 2, SKUID: 3, Price: 1599, Quantity: 1},
			{ProductID: 3, SKUID: 5, Price: 4990, Quantity: 2},
		}
		order, err := uc.CreateOrder(context.Background(), "req-price-calc", 1, 1, items)
		if err != nil {
			t.Fatalf("创建失败: %v", err)
		}
		expected := 299*3 + 1599*1 + 4990*2
		if order.TotalAmount != expected {
			t.Errorf("总金额计算错误: 期望=%d, 实际=%d", expected, order.TotalAmount)
		}
	})
}

// TestGetOrder 测试查看订单详情
func TestGetOrder(t *testing.T) {
	t.Run("成功获取订单", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		created, _ := uc.CreateOrder(context.Background(), "req-get-1", 1, 1, buildTestItems())
		order, err := uc.GetOrder(context.Background(), int64(created.ID))
		if err != nil {
			t.Fatalf("获取订单失败: %v", err)
		}
		if order.OrderNo != created.OrderNo {
			t.Errorf("订单号不匹配: 期望=%s, 实际=%s", created.OrderNo, order.OrderNo)
		}
	})

	t.Run("订单不存在", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		_, err := uc.GetOrder(context.Background(), 99999)
		if err == nil {
			t.Error("不存在的订单应返回错误")
		}
	})
}

// TestListOrders 测试分页查询订单列表
func TestListOrders(t *testing.T) {
	t.Run("查询用户全部订单", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		items := buildTestItems()
		uc.CreateOrder(context.Background(), "req-list-1", 1, 1, items)
		uc.CreateOrder(context.Background(), "req-list-2", 1, 1, items)
		uc.CreateOrder(context.Background(), "req-list-3", 2, 4, items)

		orders, total, err := uc.ListOrders(context.Background(), 1, 1, 10, -1)
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}
		if total != 2 {
			t.Errorf("用户1应有2个订单, 实际=%d", total)
		}
		if len(orders) != 2 {
			t.Errorf("返回订单数应为2, 实际=%d", len(orders))
		}
	})

	t.Run("按状态筛选", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		items := buildTestItems()
		order1, _ := uc.CreateOrder(context.Background(), "req-filter-1", 1, 1, items)
		uc.CreateOrder(context.Background(), "req-filter-2", 1, 1, items)

		uc.UpdateOrderStatus(context.Background(), int64(order1.ID), OrderStatusAwaitingShipment)

		orders, total, err := uc.ListOrders(context.Background(), 1, 1, 10, OrderStatusPending)
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}
		if total != 1 {
			t.Errorf("待处理状态应有1个订单, 实际=%d", total)
		}
		if len(orders) != 1 {
			t.Errorf("返回订单数应为1, 实际=%d", len(orders))
		}
	})

	t.Run("页码自动修正", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		items := buildTestItems()
		uc.CreateOrder(context.Background(), "req-page-1", 1, 1, items)

		_, _, err := uc.ListOrders(context.Background(), 1, 0, 10, -1)
		if err != nil {
			t.Fatalf("页码默认值修正失败: %v", err)
		}
	})
}

// TestUpdateOrderStatus 测试更新订单状态
func TestUpdateOrderStatus(t *testing.T) {
	t.Run("待处理→待发货", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		order, _ := uc.CreateOrder(context.Background(), "req-status-1", 1, 1, buildTestItems())

		err := uc.UpdateOrderStatus(context.Background(), int64(order.ID), OrderStatusAwaitingShipment)
		if err != nil {
			t.Fatalf("更新状态失败: %v", err)
		}

		updated, _ := uc.GetOrder(context.Background(), int64(order.ID))
		if updated.Status != int(OrderStatusAwaitingShipment) {
			t.Errorf("状态应为待发货(2), 实际=%d", updated.Status)
		}
		if updated.PayTime == nil {
			t.Error("设为待发货时应写入支付时间")
		}
	})

	t.Run("待发货→待完成", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		order, _ := uc.CreateOrder(context.Background(), "req-status-2", 1, 1, buildTestItems())
		uc.UpdateOrderStatus(context.Background(), int64(order.ID), OrderStatusAwaitingShipment)

		err := uc.UpdateOrderStatus(context.Background(), int64(order.ID), OrderStatusAwaitingCompleted)
		if err != nil {
			t.Fatalf("更新状态失败: %v", err)
		}

		updated, _ := uc.GetOrder(context.Background(), int64(order.ID))
		if updated.Status != int(OrderStatusAwaitingCompleted) {
			t.Errorf("状态应为待完成(3), 实际=%d", updated.Status)
		}
		if updated.ShipTime == nil {
			t.Error("设为待完成时应写入发货时间")
		}
	})

	t.Run("待完成→已完成", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		order, _ := uc.CreateOrder(context.Background(), "req-status-3", 1, 1, buildTestItems())
		uc.UpdateOrderStatus(context.Background(), int64(order.ID), OrderStatusAwaitingShipment)
		uc.UpdateOrderStatus(context.Background(), int64(order.ID), OrderStatusAwaitingCompleted)

		err := uc.UpdateOrderStatus(context.Background(), int64(order.ID), OrderStatusCompleted)
		if err != nil {
			t.Fatalf("更新状态失败: %v", err)
		}

		updated, _ := uc.GetOrder(context.Background(), int64(order.ID))
		if updated.Status != int(OrderStatusCompleted) {
			t.Errorf("状态应为已完成(4), 实际=%d", updated.Status)
		}
		if updated.ConfirmTime == nil {
			t.Error("设为已完成时应写入确认收货时间")
		}
	})
}

// TestCancelOrder 测试取消订单
func TestCancelOrder(t *testing.T) {
	t.Run("取消待处理订单", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		order, _ := uc.CreateOrder(context.Background(), "req-cancel-1", 1, 1, buildTestItems())

		err := uc.CancelOrder(context.Background(), int64(order.ID))
		if err != nil {
			t.Fatalf("取消订单失败: %v", err)
		}

		updated, _ := uc.GetOrder(context.Background(), int64(order.ID))
		if updated.Status != int(OrderStatusCancelled) {
			t.Errorf("状态应为已取消(5), 实际=%d", updated.Status)
		}
	})

	t.Run("取消已发货订单应失败", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		order, _ := uc.CreateOrder(context.Background(), "req-cancel-2", 1, 1, buildTestItems())
		uc.UpdateOrderStatus(context.Background(), int64(order.ID), OrderStatusAwaitingShipment)

		err := uc.CancelOrder(context.Background(), int64(order.ID))
		if err == nil {
			t.Error("已发货订单不应允许取消")
		}
		if !strings.Contains(err.Error(), "ORDER_CANCEL_FAILED") {
			t.Errorf("错误信息应包含 ORDER_CANCEL_FAILED, 实际=%s", err.Error())
		}
	})

	t.Run("取消不存在的订单", func(t *testing.T) {
		repo := newMockOrderRepo()
		uc := NewOrderUseCase(repo)

		err := uc.CancelOrder(context.Background(), 99999)
		if err == nil {
			t.Error("取消不存在订单应返回错误")
		}
	})
}
