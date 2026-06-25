package service

import (
	"context"
	"time"

	v1 "order/api/order/v1"
	"order/internal/biz"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderService struct {
	v1.UnimplementedOrderServiceServer

	uc *biz.OrderUseCase
}

func NewOrderService(uc *biz.OrderUseCase) *OrderService {
	return &OrderService{uc: uc}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *v1.CreateOrderRequest) (*v1.CreateOrderResponse, error) {
	items := make([]*biz.OrderItem, len(req.Items))
	for i := range req.Items {
		items[i] = &biz.OrderItem{
			ProductID:   req.Items[i].ProductId,
			SKUID:       req.Items[i].SkuId,
			ProductName: req.Items[i].ProductName,
			SKUTitle:    req.Items[i].SkuTitle,
			Price:       int(req.Items[i].Price),
			Quantity:    int(req.Items[i].Quantity),
			ImageURL:    req.Items[i].ImageUrl,
		}
	}

	order, err := s.uc.CreateOrder(ctx, req.UserId, req.ShopId, items)
	if err != nil {
		return nil, err
	}

	return &v1.CreateOrderResponse{
		OrderId: int64(order.ID),
		OrderNo: order.OrderNo,
	}, nil
}

func (s *OrderService) GetOrder(ctx context.Context, req *v1.GetOrderRequest) (*v1.GetOrderResponse, error) {
	order, err := s.uc.GetOrder(ctx, req.OrderId)
	if err != nil {
		return nil, err
	}

	return &v1.GetOrderResponse{
		Order: convertOrderToProto(order),
	}, nil
}

func (s *OrderService) ListOrders(ctx context.Context, req *v1.ListOrdersRequest) (*v1.ListOrdersResponse, error) {
	orders, total, err := s.uc.ListOrders(ctx, req.UserId, int(req.Page), int(req.PageSize), biz.OrderStatus(req.Status))
	if err != nil {
		return nil, err
	}

	protoOrders := make([]*v1.Order, len(orders))
	for i := range orders {
		protoOrders[i] = convertOrderToProto(orders[i])
	}

	return &v1.ListOrdersResponse{
		Orders: protoOrders,
		Total:  total,
	}, nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, req *v1.UpdateOrderStatusRequest) (*v1.UpdateOrderStatusResponse, error) {
	err := s.uc.UpdateOrderStatus(ctx, req.OrderId, biz.OrderStatus(req.Status))
	if err != nil {
		return &v1.UpdateOrderStatusResponse{Success: false}, err
	}

	return &v1.UpdateOrderStatusResponse{Success: true}, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, req *v1.CancelOrderRequest) (*v1.CancelOrderResponse, error) {
	err := s.uc.CancelOrder(ctx, req.OrderId)
	if err != nil {
		return &v1.CancelOrderResponse{Success: false}, err
	}

	return &v1.CancelOrderResponse{Success: true}, nil
}

func convertOrderToProto(order *biz.Order) *v1.Order {
	if order == nil {
		return nil
	}

	protoOrder := &v1.Order{
		Id:          int64(order.ID),
		OrderNo:     order.OrderNo,
		UserId:      order.UserID,
		ShopId:      order.ShopID,
		TotalAmount: int32(order.TotalAmount),
		Status:      v1.OrderStatus(order.Status),
		PayStatus:   v1.PayStatus(order.PayStatus),
	}

	if order.PayTime != nil {
		protoOrder.PayTime = timestampToProto(order.PayTime)
	}
	if order.ShipTime != nil {
		protoOrder.ShipTime = timestampToProto(order.ShipTime)
	}
	if order.ConfirmTime != nil {
		protoOrder.ConfirmTime = timestampToProto(order.ConfirmTime)
	}
	if !order.CreatedAt.IsZero() {
		protoOrder.CreatedAt = timestampToProto(&order.CreatedAt)
	}

	if len(order.Items) > 0 {
		protoOrder.Items = make([]*v1.OrderItem, len(order.Items))
		for i := range order.Items {
			protoOrder.Items[i] = &v1.OrderItem{
				Id:          int64(order.Items[i].ID),
				OrderId:     order.Items[i].OrderID,
				ProductId:   order.Items[i].ProductID,
				SkuId:       order.Items[i].SKUID,
				ProductName: order.Items[i].ProductName,
				SkuTitle:    order.Items[i].SKUTitle,
				Price:       int32(order.Items[i].Price),
				Quantity:    int32(order.Items[i].Quantity),
				ImageUrl:    order.Items[i].ImageURL,
			}
		}
	}

	return protoOrder
}

func timestampToProto(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}