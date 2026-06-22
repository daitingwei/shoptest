package biz

import (
	"context"

	v1 "productCenter/api/producttag/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrProductTagNotFound 商品标签未找到
	ErrProductTagNotFound = errors.NotFound(v1.ErrorReason_PRODUCT_TAG_NOT_FOUND.String(), "商品标签未找到")
	// ErrProductTagNameEmpty 商品标签名称不能为空
	ErrProductTagNameEmpty = errors.BadRequest(v1.ErrorReason_PRODUCT_TAG_NAME_EMPTY.String(), "商品标签名称不能为空")
)

// ProductTagRepo 商品标签数据仓库接口，由 data 层实现
type ProductTagRepo interface {
	// Create 创建商品标签
	Create(ctx context.Context, tag *ProductTag) (*ProductTag, error)
	// Update 更新商品标签
	Update(ctx context.Context, tag *ProductTag) (*ProductTag, error)
	// Get 根据ID获取商品标签
	Get(ctx context.Context, id int64) (*ProductTag, error)
	// List 分页查询商品标签列表，返回标签列表和总数
	List(ctx context.Context, page, pageSize int32) ([]*ProductTag, int64, error)
	// Delete 删除商品标签
	Delete(ctx context.Context, id int64) error
}

// ProductTagUseCase 商品标签业务用例
type ProductTagUseCase struct {
	repo ProductTagRepo
	log  *log.Helper
}

// NewProductTagUseCase 创建商品标签业务用例实例
func NewProductTagUseCase(repo ProductTagRepo, logger log.Logger) *ProductTagUseCase {
	return &ProductTagUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// Create 创建商品标签，包含参数校验逻辑
func (uc *ProductTagUseCase) Create(ctx context.Context, tag *ProductTag) (*ProductTag, error) {
	if tag.Name == "" {
		return nil, ErrProductTagNameEmpty
	}
	uc.log.WithContext(ctx).Infof("CreateProductTag: %s", tag.Name)
	return uc.repo.Create(ctx, tag)
}

// Update 更新商品标签，包含参数校验逻辑
func (uc *ProductTagUseCase) Update(ctx context.Context, tag *ProductTag) (*ProductTag, error) {
	if tag.ID <= 0 {
		return nil, ErrProductTagNotFound
	}
	if tag.Name == "" {
		return nil, ErrProductTagNameEmpty
	}
	uc.log.WithContext(ctx).Infof("UpdateProductTag: id=%d", tag.ID)
	return uc.repo.Update(ctx, tag)
}

// Get 根据ID获取商品标签
func (uc *ProductTagUseCase) Get(ctx context.Context, id int64) (*ProductTag, error) {
	uc.log.WithContext(ctx).Infof("GetProductTag: id=%d", id)
	return uc.repo.Get(ctx, id)
}

// List 分页查询商品标签列表
func (uc *ProductTagUseCase) List(ctx context.Context, page, pageSize int32) ([]*ProductTag, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	uc.log.WithContext(ctx).Infof("ListProductTags: page=%d, pageSize=%d", page, pageSize)
	return uc.repo.List(ctx, page, pageSize)
}

// Delete 根据ID删除商品标签
func (uc *ProductTagUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrProductTagNotFound
	}
	uc.log.WithContext(ctx).Infof("DeleteProductTag: id=%d", id)
	return uc.repo.Delete(ctx, id)
}
