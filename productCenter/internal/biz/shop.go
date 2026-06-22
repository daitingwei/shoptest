package biz

import (
	"context"

	v1 "productCenter/api/shop/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrShopNotFound 店铺未找到
	ErrShopNotFound = errors.NotFound(v1.ErrorReason_SHOP_NOT_FOUND.String(), "店铺未找到")
	// ErrShopNameEmpty 店铺名称不能为空
	ErrShopNameEmpty = errors.BadRequest(v1.ErrorReason_SHOP_NAME_EMPTY.String(), "店铺名称不能为空")
)

// ShopRepo 店铺数据仓库接口，由 data 层实现
type ShopRepo interface {
	// Create 创建店铺
	Create(ctx context.Context, shop *Shop) (*Shop, error)
	// Update 更新店铺信息
	Update(ctx context.Context, shop *Shop) (*Shop, error)
	// Get 根据ID获取店铺详情
	Get(ctx context.Context, id int64) (*Shop, error)
	// List 分页获取店铺列表，返回店铺列表和总数
	List(ctx context.Context, page, pageSize int32) ([]*Shop, int64, error)
	// Delete 根据ID删除店铺
	Delete(ctx context.Context, id int64) error
}

// ShopUseCase 店铺业务用例
type ShopUseCase struct {
	repo ShopRepo
	log  *log.Helper
}

// NewShopUseCase 创建店铺业务用例实例
func NewShopUseCase(repo ShopRepo, logger log.Logger) *ShopUseCase {
	return &ShopUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// Create 创建店铺，包含参数校验逻辑
func (uc *ShopUseCase) Create(ctx context.Context, shop *Shop) (*Shop, error) {
	if shop.ShopName == "" {
		return nil, ErrShopNameEmpty
	}
	uc.log.WithContext(ctx).Infof("CreateShop: %s", shop.ShopName)
	return uc.repo.Create(ctx, shop)
}

// Update 更新店铺信息，包含参数校验逻辑
func (uc *ShopUseCase) Update(ctx context.Context, shop *Shop) (*Shop, error) {
	if shop.ID <= 0 {
		return nil, ErrShopNotFound
	}
	if shop.ShopName == "" {
		return nil, ErrShopNameEmpty
	}
	uc.log.WithContext(ctx).Infof("UpdateShop: id=%d", shop.ID)
	return uc.repo.Update(ctx, shop)
}

// Get 根据ID获取店铺详情
func (uc *ShopUseCase) Get(ctx context.Context, id int64) (*Shop, error) {
	uc.log.WithContext(ctx).Infof("GetShop: id=%d", id)
	return uc.repo.Get(ctx, id)
}

// List 分页获取店铺列表
func (uc *ShopUseCase) List(ctx context.Context, page, pageSize int32) ([]*Shop, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	uc.log.WithContext(ctx).Infof("ListShops: page=%d, pageSize=%d", page, pageSize)
	return uc.repo.List(ctx, page, pageSize)
}

// Delete 根据ID删除店铺
func (uc *ShopUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrShopNotFound
	}
	uc.log.WithContext(ctx).Infof("DeleteShop: id=%d", id)
	return uc.repo.Delete(ctx, id)
}
