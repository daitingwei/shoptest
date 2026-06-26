package data

import (
	"context"
	"errors"

	"productCenter/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type productTagRepo struct {
	data *Data
	log  *log.Helper
}

// NewProductTagRepo 创建 ProductTagRepo 实例，实现 biz.ProductTagRepo 接口
func NewProductTagRepo(data *Data, logger log.Logger) biz.ProductTagRepo {
	return &productTagRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// Create 创建商品标签
func (r *productTagRepo) Create(ctx context.Context, tag *biz.ProductTag) (*biz.ProductTag, error) {
	po := &ProductTag{
		Name: tag.Name,
		Sort: tag.Sort,
	}
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	tag.ID = po.ID
	tag.CreatedAt = po.CreatedAt
	tag.UpdatedAt = po.UpdatedAt
	return tag, nil
}

// Update 更新商品标签
func (r *productTagRepo) Update(ctx context.Context, tag *biz.ProductTag) (*biz.ProductTag, error) {
	po := &ProductTag{
		Name: tag.Name,
		Sort: tag.Sort,
	}
	if err := r.data.db.WithContext(ctx).Model(&ProductTag{}).Where("id = ?", tag.ID).Updates(po).Error; err != nil {
		return nil, err
	}
	// 更新后重新从数据库读取，获取真实的 created_at、updated_at
	if err := r.data.db.WithContext(ctx).First(&po, tag.ID).Error; err != nil {
		return nil, err
	}
	tag.CreatedAt = po.CreatedAt
	tag.UpdatedAt = po.UpdatedAt
	return tag, nil
}

// Get 根据ID获取商品标签
func (r *productTagRepo) Get(ctx context.Context, id int64) (*biz.ProductTag, error) {
	var po ProductTag
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrProductTagNotFound
		}
		return nil, err
	}
	return &biz.ProductTag{
		ID:        po.ID,
		Name:      po.Name,
		Sort:      po.Sort,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}, nil
}

// List 分页查询商品标签列表
func (r *productTagRepo) List(ctx context.Context, page, pageSize int32) ([]*biz.ProductTag, int64, error) {
	var pos []ProductTag
	var total int64

	db := r.data.db.WithContext(ctx).Model(&ProductTag{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (int(page) - 1) * int(pageSize)
	if err := db.Order("sort asc, id desc").Offset(offset).Limit(int(pageSize)).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	tags := make([]*biz.ProductTag, 0, len(pos))
	for _, po := range pos {
		tags = append(tags, &biz.ProductTag{
			ID:        po.ID,
			Name:      po.Name,
			Sort:      po.Sort,
			CreatedAt: po.CreatedAt,
			UpdatedAt: po.UpdatedAt,
		})
	}
	return tags, total, nil
}

// Delete 删除商品标签（软删除）
func (r *productTagRepo) Delete(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&ProductTag{}, id).Error
}
