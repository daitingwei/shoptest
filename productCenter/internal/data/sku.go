package data

import (
	"context"
	"errors"

	"productCenter/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type skuRepo struct {
	data *Data
	log  *log.Helper
}

// NewSkuRepo 创建 SkuRepo 实例，实现 biz.SkuRepo 接口
func NewSkuRepo(data *Data, logger log.Logger) biz.SkuRepo {
	return &skuRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// Create 创建SKU
func (r *skuRepo) Create(ctx context.Context, sku *biz.Sku) (*biz.Sku, error) {
	po := &Sku{
		ProductID: sku.ProductID,
		Sku:       sku.Sku,
		Title:     sku.Title,
		Price:     sku.Price,
		Stock:     sku.Stock,
		ImgURL:    sku.ImgURL,
	}
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	sku.ID = po.ID
	sku.CreatedAt = po.CreatedAt
	sku.UpdatedAt = po.UpdatedAt
	return sku, nil
}

// Update 更新SKU
func (r *skuRepo) Update(ctx context.Context, sku *biz.Sku) (*biz.Sku, error) {
	po := &Sku{
		ProductID: sku.ProductID,
		Sku:       sku.Sku,
		Title:     sku.Title,
		Price:     sku.Price,
		Stock:     sku.Stock,
		ImgURL:    sku.ImgURL,
	}
	if err := r.data.db.WithContext(ctx).Model(&Sku{}).Where("id = ?", sku.ID).Updates(po).Error; err != nil {
		return nil, err
	}
	return sku, nil
}

// Get 根据ID获取SKU
func (r *skuRepo) Get(ctx context.Context, id int64) (*biz.Sku, error) {
	var po Sku
	if err := r.data.db.WithContext(ctx).First(&po, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrSkuNotFound
		}
		return nil, err
	}
	return &biz.Sku{
		ID:        po.ID,
		ProductID: po.ProductID,
		Sku:       po.Sku,
		Title:     po.Title,
		Price:     po.Price,
		Stock:     po.Stock,
		ImgURL:    po.ImgURL,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}, nil
}

// List 分页查询SKU列表，支持按商品筛选
func (r *skuRepo) List(ctx context.Context, page, pageSize int32, productID int64) ([]*biz.Sku, int64, error) {
	var pos []Sku
	var total int64

	db := r.data.db.WithContext(ctx).Model(&Sku{})
	if productID > 0 {
		db = db.Where("product_id = ?", productID)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (int(page) - 1) * int(pageSize)
	if err := db.Order("id desc").Offset(offset).Limit(int(pageSize)).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	skus := make([]*biz.Sku, 0, len(pos))
	for _, po := range pos {
		skus = append(skus, &biz.Sku{
			ID:        po.ID,
			ProductID: po.ProductID,
			Sku:       po.Sku,
			Title:     po.Title,
			Price:     po.Price,
			Stock:     po.Stock,
			ImgURL:    po.ImgURL,
			CreatedAt: po.CreatedAt,
			UpdatedAt: po.UpdatedAt,
		})
	}
	return skus, total, nil
}

// Delete 删除SKU（软删除）
func (r *skuRepo) Delete(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&Sku{}, id).Error
}
