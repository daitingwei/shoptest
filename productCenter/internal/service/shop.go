package service

import (
	"context"

	v1 "productCenter/api/shop/v1"
	"productCenter/internal/biz"
)

// ShopService 店铺服务，实现 gRPC 和 HTTP 接口
type ShopService struct {
	v1.UnimplementedShopServer
	uc *biz.ShopUseCase
}

// NewShopService 创建店铺服务实例
func NewShopService(uc *biz.ShopUseCase) *ShopService {
	return &ShopService{uc: uc}
}

// CreateShop 创建店铺
func (s *ShopService) CreateShop(ctx context.Context, req *v1.CreateShopRequest) (*v1.CreateShopResponse, error) {
	shop, err := s.uc.Create(ctx, &biz.Shop{
		ShopName:    req.ShopName,
		Description: req.Description,
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateShopResponse{Shop: shopEntityToProto(shop)}, nil
}

// UpdateShop 更新店铺
func (s *ShopService) UpdateShop(ctx context.Context, req *v1.UpdateShopRequest) (*v1.UpdateShopResponse, error) {
	shop, err := s.uc.Update(ctx, &biz.Shop{
		ID:          req.Id,
		ShopName:    req.ShopName,
		Description: req.Description,
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateShopResponse{Shop: shopEntityToProto(shop)}, nil
}

// GetShop 获取店铺详情
func (s *ShopService) GetShop(ctx context.Context, req *v1.GetShopRequest) (*v1.GetShopResponse, error) {
	shop, err := s.uc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetShopResponse{Shop: shopEntityToProto(shop)}, nil
}

// ListShops 分页查询店铺列表
func (s *ShopService) ListShops(ctx context.Context, req *v1.ListShopsRequest) (*v1.ListShopsResponse, error) {
	shops, total, err := s.uc.List(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	shopInfos := make([]*v1.ShopInfo, 0, len(shops))
	for _, shop := range shops {
		shopInfos = append(shopInfos, shopEntityToProto(shop))
	}

	return &v1.ListShopsResponse{
		Shops:    shopInfos,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// DeleteShop 删除店铺
func (s *ShopService) DeleteShop(ctx context.Context, req *v1.DeleteShopRequest) (*v1.DeleteShopResponse, error) {
	if err := s.uc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteShopResponse{Success: true}, nil
}

// shopEntityToProto 将 biz 层 Shop 实体转换为 proto ShopInfo
func shopEntityToProto(shop *biz.Shop) *v1.ShopInfo {
	if shop == nil {
		return nil
	}
	return &v1.ShopInfo{
		Id:          shop.ID,
		ShopName:    shop.ShopName,
		Description: shop.Description,
		CreatedAt:   shop.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   shop.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
