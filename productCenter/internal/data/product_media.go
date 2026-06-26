package data

import (
	"context"
	"errors"

	"productCenter/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type productMediaRepo struct {
	data *Data
	log  *log.Helper
}

// NewProductMediaRepo 创建 ProductMediaRepo 实例，实现 biz.ProductMediaRepo 接口
func NewProductMediaRepo(data *Data, logger log.Logger) biz.ProductMediaRepo {
	return &productMediaRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// Create 创建商品副图
func (r *productMediaRepo) Create(ctx context.Context, media *biz.ProductMedia) (*biz.ProductMedia, error) {
	po := &ProductMedia{
		ProductID: media.ProductID,
		URL:       media.URL,
		Sort:      media.Sort,
	}
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	media.ID = po.ID
	media.CreatedAt = po.CreatedAt
	media.UpdatedAt = po.UpdatedAt
	return media, nil
}

// Update 更新商品副图
func (r *productMediaRepo) Update(ctx context.Context, media *biz.ProductMedia) (*biz.ProductMedia, error) {
	po := &ProductMedia{
		ProductID: media.ProductID,
		URL:       media.URL,
		Sort:      media.Sort,
	}
	if err := r.data.db.WithContext(ctx).Model(&ProductMedia{}).Where("id = ?", media.ID).Updates(po).Error; err != nil {
		return nil, err
	}
	// 更新后重新从数据库读取，获取真实的 created_at、updated_at
	if err := r.data.db.WithContext(ctx).First(&po, media.ID).Error; err != nil {
		return nil, err
	}
	media.CreatedAt = po.CreatedAt
	media.UpdatedAt = po.UpdatedAt
	return media, nil
}

// Get 根据ID获取商品副图
func (r *productMediaRepo) Get(ctx context.Context, id int64) (*biz.ProductMedia, error) {
	var po ProductMedia
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrProductMediaNotFound
		}
		return nil, err
	}
	return &biz.ProductMedia{
		ID:        po.ID,
		ProductID: po.ProductID,
		URL:       po.URL,
		Sort:      po.Sort,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}, nil
}

// List 分页查询商品副图列表，支持按商品筛选
func (r *productMediaRepo) List(ctx context.Context, page, pageSize int32, productID int64) ([]*biz.ProductMedia, int64, error) {
	var pos []ProductMedia
	var total int64

	db := r.data.db.WithContext(ctx).Model(&ProductMedia{})
	if productID > 0 {
		db = db.Where("product_id = ?", productID)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (int(page) - 1) * int(pageSize)
	if err := db.Order("sort asc, id desc").Offset(offset).Limit(int(pageSize)).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	medias := make([]*biz.ProductMedia, 0, len(pos))
	for _, po := range pos {
		medias = append(medias, &biz.ProductMedia{
			ID:        po.ID,
			ProductID: po.ProductID,
			URL:       po.URL,
			Sort:      po.Sort,
			CreatedAt: po.CreatedAt,
			UpdatedAt: po.UpdatedAt,
		})
	}
	return medias, total, nil
}

// Delete 删除商品副图（软删除）
func (r *productMediaRepo) Delete(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&ProductMedia{}, id).Error
}
