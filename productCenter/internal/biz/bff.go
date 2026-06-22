package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// BFFRepo BFF聚合查询数据仓库接口，由 data 层实现
type BFFRepo interface {
	// GetProductDetail 聚合查询商品详情，包含商品、店铺、标签、SKU、媒体
	GetProductDetail(ctx context.Context, id int64) (*ProductDetail, error)
	// ListProducts 聚合查询商品列表，支持按门店和状态筛选
	ListProducts(ctx context.Context, page, pageSize int32, shopID int64, status int32) ([]*ProductListItem, int64, error)
	// GetShopHome 聚合查询店铺首页，包含店铺信息和商品列表
	GetShopHome(ctx context.Context, id int64, page, pageSize int32) (*ShopHome, error)
}

// BFFUseCase BFF聚合查询业务用例
type BFFUseCase struct {
	repo BFFRepo
	log  *log.Helper
}

// NewBFFUseCase 创建BFF聚合查询业务用例实例
func NewBFFUseCase(repo BFFRepo, logger log.Logger) *BFFUseCase {
	return &BFFUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// GetProductDetail 获取商品详情聚合数据
func (uc *BFFUseCase) GetProductDetail(ctx context.Context, id int64) (*ProductDetail, error) {
	if id <= 0 {
		return nil, ErrProductNotFound
	}
	uc.log.WithContext(ctx).Infof("GetProductDetail: id=%d", id)
	return uc.repo.GetProductDetail(ctx, id)
}

// ListProducts 分页查询商品列表，支持按店铺和状态筛选
func (uc *BFFUseCase) ListProducts(ctx context.Context, page, pageSize int32, shopID int64, status int32) ([]*ProductListItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	uc.log.WithContext(ctx).Infof("ListProducts: page=%d, pageSize=%d, shopID=%d, status=%d", page, pageSize, shopID, status)
	return uc.repo.ListProducts(ctx, page, pageSize, shopID, status)
}

// GetShopHome 获取店铺首页聚合数据
func (uc *BFFUseCase) GetShopHome(ctx context.Context, id int64, page, pageSize int32) (*ShopHome, error) {
	if id <= 0 {
		return nil, ErrShopNotFound
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	uc.log.WithContext(ctx).Infof("GetShopHome: id=%d, page=%d, pageSize=%d", id, page, pageSize)
	return uc.repo.GetShopHome(ctx, id, page, pageSize)
}
