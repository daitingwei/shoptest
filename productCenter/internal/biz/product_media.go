package biz

import (
	"context"

	v1 "productCenter/api/productmedia/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrProductMediaNotFound 商品副图未找到
	ErrProductMediaNotFound = errors.NotFound(v1.ErrorReason_PRODUCT_MEDIA_NOT_FOUND.String(), "商品副图未找到")
	// ErrProductMediaProductIDInvalid 副图所属商品ID无效
	ErrProductMediaProductIDInvalid = errors.BadRequest(v1.ErrorReason_PRODUCT_MEDIA_PRODUCT_ID_INVALID.String(), "商品ID不能为空")
	// ErrProductMediaURLEmpty 商品副图URL不能为空
	ErrProductMediaURLEmpty = errors.BadRequest(v1.ErrorReason_PRODUCT_MEDIA_URL_EMPTY.String(), "副图URL不能为空")
)

// ProductMediaRepo 商品副图数据仓库接口，由 data 层实现
type ProductMediaRepo interface {
	// Create 创建副图
	Create(ctx context.Context, media *ProductMedia) (*ProductMedia, error)
	// Update 更新副图
	Update(ctx context.Context, media *ProductMedia) (*ProductMedia, error)
	// Get 根据ID获取副图
	Get(ctx context.Context, id int64) (*ProductMedia, error)
	// List 分页查询副图列表，支持按商品筛选，返回副图列表和总数
	List(ctx context.Context, page, pageSize int32, productID int64) ([]*ProductMedia, int64, error)
	// Delete 删除副图
	Delete(ctx context.Context, id int64) error
}

// ProductMediaUseCase 商品副图业务用例
type ProductMediaUseCase struct {
	repo ProductMediaRepo
	log  *log.Helper
}

// NewProductMediaUseCase 创建商品副图业务用例实例
func NewProductMediaUseCase(repo ProductMediaRepo, logger log.Logger) *ProductMediaUseCase {
	return &ProductMediaUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// Create 创建商品副图，包含参数校验逻辑
func (uc *ProductMediaUseCase) Create(ctx context.Context, media *ProductMedia) (*ProductMedia, error) {
	if media.ProductID <= 0 {
		return nil, ErrProductMediaProductIDInvalid
	}
	if media.URL == "" {
		return nil, ErrProductMediaURLEmpty
	}
	uc.log.WithContext(ctx).Infof("CreateProductMedia: product_id=%d, url=%s", media.ProductID, media.URL)
	return uc.repo.Create(ctx, media)
}

// Update 更新商品副图，包含参数校验逻辑
func (uc *ProductMediaUseCase) Update(ctx context.Context, media *ProductMedia) (*ProductMedia, error) {
	if media.ID <= 0 {
		return nil, ErrProductMediaNotFound
	}
	if media.ProductID <= 0 {
		return nil, ErrProductMediaProductIDInvalid
	}
	if media.URL == "" {
		return nil, ErrProductMediaURLEmpty
	}
	uc.log.WithContext(ctx).Infof("UpdateProductMedia: id=%d", media.ID)
	return uc.repo.Update(ctx, media)
}

// Get 根据ID获取商品副图
func (uc *ProductMediaUseCase) Get(ctx context.Context, id int64) (*ProductMedia, error) {
	if id <= 0 {
		return nil, ErrProductMediaNotFound
	}
	uc.log.WithContext(ctx).Infof("GetProductMedia: id=%d", id)
	return uc.repo.Get(ctx, id)
}

// List 分页查询商品副图列表，支持按商品筛选
func (uc *ProductMediaUseCase) List(ctx context.Context, page, pageSize int32, productID int64) ([]*ProductMedia, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	uc.log.WithContext(ctx).Infof("ListProductMedias: page=%d, pageSize=%d, product_id=%d", page, pageSize, productID)
	return uc.repo.List(ctx, page, pageSize, productID)
}

// Delete 根据ID删除商品副图
func (uc *ProductMediaUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrProductMediaNotFound
	}
	uc.log.WithContext(ctx).Infof("DeleteProductMedia: id=%d", id)
	return uc.repo.Delete(ctx, id)
}
