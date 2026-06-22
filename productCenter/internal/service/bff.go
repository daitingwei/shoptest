package service

import (
	"context"

	v1 "productCenter/api/bff/v1"
	"productCenter/internal/biz"
)

// BFFService BFF 聚合服务，提供前端统一入口
type BFFService struct {
	v1.UnimplementedBFFServer
	uc *biz.BFFUseCase
}

// NewBFFService 创建 BFF 聚合服务实例
func NewBFFService(uc *biz.BFFUseCase) *BFFService {
	return &BFFService{uc: uc}
}

// GetProductDetail 商品详情页 - 聚合商品 + 店铺 + 标签 + SKU + 副图
func (s *BFFService) GetProductDetail(ctx context.Context, req *v1.GetProductDetailRequest) (*v1.GetProductDetailResponse, error) {
	detail, err := s.uc.GetProductDetail(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetProductDetailResponse{Product: productDetailToProto(detail)}, nil
}

// ListProducts 商品列表页 - 聚合商品 + 店铺名 + 标签
func (s *BFFService) ListProducts(ctx context.Context, req *v1.ListProductsRequest) (*v1.ListProductsResponse, error) {
	items, total, err := s.uc.ListProducts(ctx, req.Page, req.PageSize, req.ShopId, req.Status)
	if err != nil {
		return nil, err
	}

	productListItems := make([]*v1.ProductListItem, 0, len(items))
	for _, item := range items {
		productListItems = append(productListItems, productListItemToProto(item))
	}

	return &v1.ListProductsResponse{
		Products: productListItems,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetShopHome 店铺主页 - 聚合店铺 + 商品列表
func (s *BFFService) GetShopHome(ctx context.Context, req *v1.GetShopHomeRequest) (*v1.GetShopHomeResponse, error) {
	shopHome, err := s.uc.GetShopHome(ctx, req.Id, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	productListItems := make([]*v1.ProductListItem, 0, len(shopHome.Products))
	for _, item := range shopHome.Products {
		productListItems = append(productListItems, productListItemToProto(item))
	}

	return &v1.GetShopHomeResponse{
		Shop: &v1.ShopHome{
			Id:          shopHome.Shop.ID,
			ShopName:    shopHome.Shop.ShopName,
			Description: shopHome.Shop.Description,
			Products:    productListItems,
		},
	}, nil
}

// productDetailToProto 将 biz 层 ProductDetail 转换为 proto ProductDetail
func productDetailToProto(detail *biz.ProductDetail) *v1.ProductDetail {
	if detail == nil || detail.Product == nil {
		return nil
	}

	tags := make([]*v1.Tag, 0, len(detail.Tags))
	for _, tag := range detail.Tags {
		tags = append(tags, &v1.Tag{
			Id:   int32(tag.ID),
			Name: tag.Name,
		})
	}

	medias := make([]*v1.Media, 0, len(detail.Medias))
	for _, media := range detail.Medias {
		medias = append(medias, &v1.Media{
			Id:   media.ID,
			Url:  media.URL,
			Sort: int32(media.Sort),
		})
	}

	skus := make([]*v1.Sku, 0, len(detail.Skus))
	for _, sku := range detail.Skus {
		skus = append(skus, &v1.Sku{
			Id:     sku.ID,
			Sku:    sku.Sku,
			Title:  sku.Title,
			Price:  int64(sku.Price),
			Stock:  int32(sku.Stock),
			ImgUrl: sku.ImgURL,
		})
	}

	shopName := ""
	if detail.Shop != nil {
		shopName = detail.Shop.ShopName
	}

	return &v1.ProductDetail{
		Id:             detail.Product.ID,
		ShopId:         detail.Product.ShopID,
		ShopName:       shopName,
		Name:           detail.Product.Name,
		Type:           detail.Product.Type,
		Description:    detail.Product.Description,
		MainImageUrl:   detail.Product.MainImageURL,
		Price:          int64(detail.Product.Price),
		CompareAtPrice: int64(detail.Product.CompareAtPrice),
		Status:         detail.Product.Status,
		Sort:           int32(detail.Product.Sort),
		Tags:           tags,
		Medias:         medias,
		Skus:           skus,
		CreatedAt:      detail.Product.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:      detail.Product.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// productListItemToProto 将 biz 层 ProductListItem 转换为 proto ProductListItem
func productListItemToProto(item *biz.ProductListItem) *v1.ProductListItem {
	if item == nil || item.Product == nil {
		return nil
	}

	tags := make([]*v1.Tag, 0, len(item.Tags))
	for _, tag := range item.Tags {
		tags = append(tags, &v1.Tag{
			Id:   int32(tag.ID),
			Name: tag.Name,
		})
	}

	return &v1.ProductListItem{
		Id:             item.Product.ID,
		ShopId:         item.Product.ShopID,
		ShopName:       item.ShopName,
		Name:           item.Product.Name,
		Type:           item.Product.Type,
		MainImageUrl:   item.Product.MainImageURL,
		Price:          int64(item.Product.Price),
		CompareAtPrice: int64(item.Product.CompareAtPrice),
		Status:         item.Product.Status,
		Tags:           tags,
		CreatedAt:      item.Product.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
