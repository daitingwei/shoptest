package service

import (
	"context"

	v1 "productCenter/api/sku/v1"
	"productCenter/internal/biz"
)

// SkuService 商品SKU服务，实现 gRPC 和 HTTP 接口
type SkuService struct {
	v1.UnimplementedSkuServer
	uc *biz.SkuUseCase
}

// NewSkuService 创建商品SKU服务实例
func NewSkuService(uc *biz.SkuUseCase) *SkuService {
	return &SkuService{uc: uc}
}

// CreateSku 创建SKU
func (s *SkuService) CreateSku(ctx context.Context, req *v1.CreateSkuRequest) (*v1.CreateSkuResponse, error) {
	sku, err := s.uc.Create(ctx, &biz.Sku{
		ProductID: req.ProductId,
		Sku:       req.Sku,
		Title:     req.Title,
		Price:     int(req.Price),
		Stock:     int(req.Stock),
		ImgURL:    req.ImgUrl,
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateSkuResponse{Sku: skuEntityToProto(sku)}, nil
}

// UpdateSku 更新SKU
func (s *SkuService) UpdateSku(ctx context.Context, req *v1.UpdateSkuRequest) (*v1.UpdateSkuResponse, error) {
	sku, err := s.uc.Update(ctx, &biz.Sku{
		ID:        req.Id,
		ProductID: req.ProductId,
		Sku:       req.Sku,
		Title:     req.Title,
		Price:     int(req.Price),
		Stock:     int(req.Stock),
		ImgURL:    req.ImgUrl,
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateSkuResponse{Sku: skuEntityToProto(sku)}, nil
}

// GetSku 获取SKU详情
func (s *SkuService) GetSku(ctx context.Context, req *v1.GetSkuRequest) (*v1.GetSkuResponse, error) {
	sku, err := s.uc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetSkuResponse{Sku: skuEntityToProto(sku)}, nil
}

// ListSkus 分页查询SKU列表，支持按商品筛选
func (s *SkuService) ListSkus(ctx context.Context, req *v1.ListSkusRequest) (*v1.ListSkusResponse, error) {
	skus, total, err := s.uc.List(ctx, req.Page, req.PageSize, req.ProductId)
	if err != nil {
		return nil, err
	}

	skuInfos := make([]*v1.SkuInfo, 0, len(skus))
	for _, sku := range skus {
		skuInfos = append(skuInfos, skuEntityToProto(sku))
	}

	return &v1.ListSkusResponse{
		Skus:     skuInfos,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// DeleteSku 删除SKU
func (s *SkuService) DeleteSku(ctx context.Context, req *v1.DeleteSkuRequest) (*v1.DeleteSkuResponse, error) {
	if err := s.uc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteSkuResponse{Success: true}, nil
}

// DeductStock 扣减库存
func (s *SkuService) DeductStock(ctx context.Context, req *v1.DeductStockRequest) (*v1.DeductStockResponse, error) {
	newStock, err := s.uc.DeductStock(ctx, req.Id, int(req.Quantity))
	if err != nil {
		return nil, err
	}
	return &v1.DeductStockResponse{Success: true, NewStock: int64(newStock)}, nil
}

// RestoreStock 回补库存
func (s *SkuService) RestoreStock(ctx context.Context, req *v1.RestoreStockRequest) (*v1.RestoreStockResponse, error) {
	newStock, err := s.uc.RestoreStock(ctx, req.Id, int(req.Quantity))
	if err != nil {
		return nil, err
	}
	return &v1.RestoreStockResponse{Success: true, NewStock: int64(newStock)}, nil
}

// skuEntityToProto 将 biz 层 Sku 实体转换为 proto SkuInfo
func skuEntityToProto(sku *biz.Sku) *v1.SkuInfo {
	if sku == nil {
		return nil
	}
	return &v1.SkuInfo{
		Id:        sku.ID,
		ProductId: sku.ProductID,
		Sku:       sku.Sku,
		Title:     sku.Title,
		Price:     int64(sku.Price),
		Stock:     int64(sku.Stock),
		ImgUrl:    sku.ImgURL,
		CreatedAt: sku.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: sku.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
