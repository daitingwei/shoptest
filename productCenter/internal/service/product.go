package service

import (
	"context"

	v1 "productCenter/api/product/v1"
	"productCenter/internal/biz"
)

// ProductService 商品服务，实现 gRPC 和 HTTP 接口
type ProductService struct {
	v1.UnimplementedProductServer
	uc *biz.ProductUseCase
}

// NewProductService 创建商品服务实例
func NewProductService(uc *biz.ProductUseCase) *ProductService {
	return &ProductService{uc: uc}
}

// CreateProduct 创建商品
func (s *ProductService) CreateProduct(ctx context.Context, req *v1.CreateProductRequest) (*v1.CreateProductResponse, error) {
	product, err := s.uc.Create(ctx, &biz.Product{
		ShopID:         req.ShopId,
		Name:           req.Name,
		Type:           req.Type,
		Description:    req.Description,
		MainImageURL:   req.MainImageUrl,
		Price:          int(req.Price),
		CompareAtPrice: int(req.CompareAtPrice),
		Status:         req.Status,
		Sort:           int(req.Sort),
	}, toInt64Slice(req.TagIds))
	if err != nil {
		return nil, err
	}
	return &v1.CreateProductResponse{Product: productEntityToProto(product)}, nil
}

// UpdateProduct 更新商品
func (s *ProductService) UpdateProduct(ctx context.Context, req *v1.UpdateProductRequest) (*v1.UpdateProductResponse, error) {
	product, err := s.uc.Update(ctx, &biz.Product{
		ID:             req.Id,
		Name:           req.Name,
		Type:           req.Type,
		Description:    req.Description,
		MainImageURL:   req.MainImageUrl,
		Price:          int(req.Price),
		CompareAtPrice: int(req.CompareAtPrice),
		Status:         req.Status,
		Sort:           int(req.Sort),
	}, toInt64Slice(req.TagIds))
	if err != nil {
		return nil, err
	}
	return &v1.UpdateProductResponse{Product: productEntityToProto(product)}, nil
}

// GetProduct 获取商品详情
func (s *ProductService) GetProduct(ctx context.Context, req *v1.GetProductRequest) (*v1.GetProductResponse, error) {
	product, err := s.uc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetProductResponse{Product: productEntityToProto(product)}, nil
}

// ListProducts 分页查询商品列表，支持按店铺筛选
func (s *ProductService) ListProducts(ctx context.Context, req *v1.ListProductsRequest) (*v1.ListProductsResponse, error) {
	products, total, err := s.uc.List(ctx, req.Page, req.PageSize, req.ShopId, req.Status)
	if err != nil {
		return nil, err
	}

	productInfos := make([]*v1.ProductInfo, 0, len(products))
	for _, product := range products {
		productInfos = append(productInfos, productEntityToProto(product))
	}

	return &v1.ListProductsResponse{
		Products: productInfos,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// DeleteProduct 删除商品
func (s *ProductService) DeleteProduct(ctx context.Context, req *v1.DeleteProductRequest) (*v1.DeleteProductResponse, error) {
	if err := s.uc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteProductResponse{Success: true}, nil
}

// productEntityToProto 将 biz 层 Product 实体转换为 proto ProductInfo
func productEntityToProto(product *biz.Product) *v1.ProductInfo {
	if product == nil {
		return nil
	}

	tagIDs := make([]int32, 0, len(product.Tags))
	tagNames := make([]string, 0, len(product.Tags))
	for _, tag := range product.Tags {
		tagIDs = append(tagIDs, int32(tag.ID))
		tagNames = append(tagNames, tag.Name)
	}

	return &v1.ProductInfo{
		Id:             product.ID,
		ShopId:         product.ShopID,
		ShopName:       product.ShopName,
		Name:           product.Name,
		Type:           product.Type,
		Description:    product.Description,
		MainImageUrl:   product.MainImageURL,
		Price:          int64(product.Price),
		CompareAtPrice: int64(product.CompareAtPrice),
		Status:         product.Status,
		Sort:           int32(product.Sort),
		TagIds:         tagIDs,
		TagNames:       tagNames,
		CreatedAt:      product.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:      product.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// toInt64Slice 将 []int32 转换为 []int64
func toInt64Slice(items []int32) []int64 {
	if items == nil {
		return nil
	}
	result := make([]int64, 0, len(items))
	for _, item := range items {
		result = append(result, int64(item))
	}
	return result
}
