package biz

import (
	"context"

	v1 "productCenter/api/sku/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrSkuNotFound SKU未找到
	ErrSkuNotFound = errors.NotFound(v1.ErrorReason_SKU_NOT_FOUND.String(), "SKU未找到")
	// ErrSkuProductIDInvalid SKU所属商品ID无效
	ErrSkuProductIDInvalid = errors.BadRequest(v1.ErrorReason_SKU_PRODUCT_ID_INVALID.String(), "商品ID不能为空")
	// ErrSkuPriceInvalid SKU价格不能为负
	ErrSkuPriceInvalid = errors.BadRequest(v1.ErrorReason_SKU_PRICE_INVALID.String(), "SKU价格不能为负")
	// ErrSkuStockInvalid SKU库存不能为负
	ErrSkuStockInvalid = errors.BadRequest(v1.ErrorReason_SKU_STOCK_INVALID.String(), "SKU库存不能为负")
	// ErrSkuStockInsufficient 库存不足
	ErrSkuStockInsufficient = errors.New(404, v1.ErrorReason_SKU_STOCK_INSUFFICIENT.String(), "库存不足")
)

// SkuRepo 商品SKU数据仓库接口，由 data 层实现
type SkuRepo interface {
	// Create 创建SKU
	Create(ctx context.Context, sku *Sku) (*Sku, error)
	// Update 更新SKU
	Update(ctx context.Context, sku *Sku) (*Sku, error)
	// Get 根据ID获取SKU
	Get(ctx context.Context, id int64) (*Sku, error)
	// List 分页查询SKU列表，支持按商品筛选，返回SKU列表和总数
	List(ctx context.Context, page, pageSize int32, productID int64) ([]*Sku, int64, error)
	// Delete 删除SKU
	Delete(ctx context.Context, id int64) error
	// DeductStock 扣减库存，使用乐观锁保证并发安全
	DeductStock(ctx context.Context, id int64, quantity int) (int, error)
	// RestoreStock 回补库存
	RestoreStock(ctx context.Context, id int64, quantity int) (int, error)
}

// SkuUseCase 商品SKU业务用例
type SkuUseCase struct {
	repo SkuRepo
	log  *log.Helper
}

// NewSkuUseCase 创建SKU业务用例实例
func NewSkuUseCase(repo SkuRepo, logger log.Logger) *SkuUseCase {
	return &SkuUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// Create 创建SKU，包含参数校验逻辑
func (uc *SkuUseCase) Create(ctx context.Context, sku *Sku) (*Sku, error) {
	if sku.ProductID <= 0 {
		return nil, ErrSkuProductIDInvalid
	}
	if sku.Price < 0 {
		return nil, ErrSkuPriceInvalid
	}
	if sku.Stock < 0 {
		return nil, ErrSkuStockInvalid
	}
	uc.log.WithContext(ctx).Infof("CreateSku: product_id=%d, sku=%s", sku.ProductID, sku.Sku)
	return uc.repo.Create(ctx, sku)
}

// Update 更新SKU，包含参数校验逻辑
func (uc *SkuUseCase) Update(ctx context.Context, sku *Sku) (*Sku, error) {
	if sku.ID <= 0 {
		return nil, ErrSkuNotFound
	}
	if sku.ProductID <= 0 {
		return nil, ErrSkuProductIDInvalid
	}
	if sku.Price < 0 {
		return nil, ErrSkuPriceInvalid
	}
	if sku.Stock < 0 {
		return nil, ErrSkuStockInvalid
	}
	uc.log.WithContext(ctx).Infof("UpdateSku: id=%d", sku.ID)
	return uc.repo.Update(ctx, sku)
}

// Get 根据ID获取SKU
func (uc *SkuUseCase) Get(ctx context.Context, id int64) (*Sku, error) {
	if id <= 0 {
		return nil, ErrSkuNotFound
	}
	uc.log.WithContext(ctx).Infof("GetSku: id=%d", id)
	return uc.repo.Get(ctx, id)
}

// List 分页查询SKU列表，支持按商品筛选
func (uc *SkuUseCase) List(ctx context.Context, page, pageSize int32, productID int64) ([]*Sku, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	uc.log.WithContext(ctx).Infof("ListSkus: page=%d, pageSize=%d, product_id=%d", page, pageSize, productID)
	return uc.repo.List(ctx, page, pageSize, productID)
}

// Delete 根据ID删除SKU
func (uc *SkuUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrSkuNotFound
	}
	uc.log.WithContext(ctx).Infof("DeleteSku: id=%d", id)
	return uc.repo.Delete(ctx, id)
}

// DeductStock 扣减库存，含参数校验
func (uc *SkuUseCase) DeductStock(ctx context.Context, id int64, quantity int) (int, error) {
	if id <= 0 {
		return 0, ErrSkuNotFound
	}
	if quantity <= 0 {
		return 0, errors.BadRequest("SKU_QUANTITY_INVALID", "扣减数量必须大于0")
	}
	uc.log.WithContext(ctx).Infof("DeductStock: id=%d, quantity=%d", id, quantity)
	return uc.repo.DeductStock(ctx, id, quantity)
}

// RestoreStock 回补库存，含参数校验
func (uc *SkuUseCase) RestoreStock(ctx context.Context, id int64, quantity int) (int, error) {
	if id <= 0 {
		return 0, ErrSkuNotFound
	}
	if quantity <= 0 {
		return 0, errors.BadRequest("SKU_QUANTITY_INVALID", "回补数量必须大于0")
	}
	uc.log.WithContext(ctx).Infof("RestoreStock: id=%d, quantity=%d", id, quantity)
	return uc.repo.RestoreStock(ctx, id, quantity)
}
