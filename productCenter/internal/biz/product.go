package biz

import (
	"context"

	v1 "productCenter/api/product/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrProductNotFound 商品未找到
	ErrProductNotFound = errors.NotFound(v1.ErrorReason_PRODUCT_NOT_FOUND.String(), "商品未找到")
	// ErrProductNameEmpty 商品名称不能为空
	ErrProductNameEmpty = errors.BadRequest(v1.ErrorReason_PRODUCT_NAME_EMPTY.String(), "商品名称不能为空")
	// ErrProductPriceInvalid 商品价格不合法
	ErrProductPriceInvalid = errors.BadRequest(v1.ErrorReason_PRODUCT_PRICE_INVALID.String(), "商品价格不合法")
)

// ProductRepo 商品数据仓库接口，由 data 层实现
type ProductRepo interface {
	// Create 创建商品，同时处理标签关联
	Create(ctx context.Context, product *Product, tagIDs []int64) (*Product, error)
	// Update 更新商品信息，同时更新标签关联
	Update(ctx context.Context, product *Product, tagIDs []int64) (*Product, error)
	// Get 根据ID获取商品详情（含标签）
	Get(ctx context.Context, id int64) (*Product, error)
	// List 分页查询商品列表，支持按店铺和状态筛选，返回商品列表和总数
	List(ctx context.Context, page, pageSize int32, shopID int64, status int32) ([]*Product, int64, error)
	// Delete 根据ID删除商品
	Delete(ctx context.Context, id int64) error
}

// ProductUseCase 商品业务用例
type ProductUseCase struct {
	repo ProductRepo
	log  *log.Helper
}

// NewProductUseCase 创建商品业务用例实例
func NewProductUseCase(repo ProductRepo, logger log.Logger) *ProductUseCase {
	return &ProductUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// Create 创建商品，包含参数校验逻辑
func (uc *ProductUseCase) Create(ctx context.Context, product *Product, tagIDs []int64) (*Product, error) {
	if product.Name == "" {
		return nil, ErrProductNameEmpty
	}
	if product.Price < 0 {
		return nil, ErrProductPriceInvalid
	}
	uc.log.WithContext(ctx).Infof("CreateProduct: %s", product.Name)
	return uc.repo.Create(ctx, product, tagIDs)
}

// Update 更新商品信息，包含参数校验逻辑
func (uc *ProductUseCase) Update(ctx context.Context, product *Product, tagIDs []int64) (*Product, error) {
	if product.ID <= 0 {
		return nil, ErrProductNotFound
	}
	if product.Name == "" {
		return nil, ErrProductNameEmpty
	}
	if product.Price < 0 {
		return nil, ErrProductPriceInvalid
	}
	uc.log.WithContext(ctx).Infof("UpdateProduct: id=%d", product.ID)
	return uc.repo.Update(ctx, product, tagIDs)
}

// Get 根据ID获取商品详情
func (uc *ProductUseCase) Get(ctx context.Context, id int64) (*Product, error) {
	uc.log.WithContext(ctx).Infof("GetProduct: id=%d", id)
	return uc.repo.Get(ctx, id)
}

// List 分页查询商品列表，支持按店铺和状态筛选
func (uc *ProductUseCase) List(ctx context.Context, page, pageSize int32, shopID int64, status int32) ([]*Product, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	uc.log.WithContext(ctx).Infof("ListProducts: page=%d, pageSize=%d, shopID=%d, status=%d", page, pageSize, shopID, status)
	return uc.repo.List(ctx, page, pageSize, shopID, status)
}

// Delete 根据ID删除商品
func (uc *ProductUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrProductNotFound
	}
	uc.log.WithContext(ctx).Infof("DeleteProduct: id=%d", id)
	return uc.repo.Delete(ctx, id)
}
