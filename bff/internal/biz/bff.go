package biz

import (
	"context"
	"errors"

	v1 "bff/api/bff/v1"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

var (
	ErrProductNotFound = errors.New(v1.ErrorReason_PRODUCT_NOT_FOUND.String())
	ErrShopNotFound    = errors.New(v1.ErrorReason_SHOP_NOT_FOUND.String())
)

// BFFRepo BFF聚合查询数据仓库接口，由 data 层实现
type BFFRepo interface {
	GetProductDetail(ctx context.Context, id int64) (*ProductDetail, error)
	ListProducts(ctx context.Context, page, pageSize int32, shopID int64, status int32) ([]*ProductListItem, int64, error)
	GetShopHome(ctx context.Context, id int64, page, pageSize int32) (*ShopHome, error)
	CreateOrder(ctx context.Context, requestID string, userID, shopID int64, items []*OrderItem) (*CreateOrderResult, error)
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

// CreateOrder 创建订单，自动生成 request_id
func (uc *BFFUseCase) CreateOrder(ctx context.Context, userID, shopID int64, items []*OrderItem) (*CreateOrderResult, error) {
	requestID := uuid.NewString()
	uc.log.WithContext(ctx).Infof("CreateOrder: requestID=%s, userID=%d, shopID=%d", requestID, userID, shopID)
	return uc.repo.CreateOrder(ctx, requestID, userID, shopID, items)
}
