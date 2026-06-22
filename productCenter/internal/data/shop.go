package data

import (
	"context"
	"errors"

	"productCenter/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type shopRepo struct {
	data *Data
	log  *log.Helper
}

// NewShopRepo 创建 ShopRepo 实例，实现 biz.ShopRepo 接口
func NewShopRepo(data *Data, logger log.Logger) biz.ShopRepo {
	return &shopRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// Create 创建店铺
func (r *shopRepo) Create(ctx context.Context, shop *biz.Shop) (*biz.Shop, error) {
	po := &Shop{
		ShopName:    shop.ShopName,
		Description: shop.Description,
	}
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	shop.ID = po.ID
	shop.CreatedAt = po.CreatedAt
	shop.UpdatedAt = po.UpdatedAt
	return shop, nil
}

// Update 更新店铺信息
func (r *shopRepo) Update(ctx context.Context, shop *biz.Shop) (*biz.Shop, error) {
	po := &Shop{
		ShopName:    shop.ShopName,
		Description: shop.Description,
	}
	if err := r.data.db.WithContext(ctx).Model(&Shop{}).Where("id = ?", shop.ID).Updates(po).Error; err != nil {
		return nil, err
	}
	return shop, nil
}

// Get 根据ID获取店铺详情
func (r *shopRepo) Get(ctx context.Context, id int64) (*biz.Shop, error) {
	var po Shop
	if err := r.data.db.WithContext(ctx).First(&po, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrShopNotFound
		}
		return nil, err
	}
	return &biz.Shop{
		ID:          po.ID,
		ShopName:    po.ShopName,
		Description: po.Description,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}, nil
}

// List 分页获取店铺列表
func (r *shopRepo) List(ctx context.Context, page, pageSize int32) ([]*biz.Shop, int64, error) {
	var pos []Shop
	var total int64

	db := r.data.db.WithContext(ctx).Model(&Shop{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (int(page) - 1) * int(pageSize)
	if err := db.Order("id desc").Offset(offset).Limit(int(pageSize)).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	shops := make([]*biz.Shop, 0, len(pos))
	for _, po := range pos {
		shops = append(shops, &biz.Shop{
			ID:          po.ID,
			ShopName:    po.ShopName,
			Description: po.Description,
			CreatedAt:   po.CreatedAt,
			UpdatedAt:   po.UpdatedAt,
		})
	}
	return shops, total, nil
}

// Delete 删除店铺（软删除），先检查是否有未删除的关联商品
func (r *shopRepo) Delete(ctx context.Context, id int64) error {
	var count int64
	if err := r.data.db.WithContext(ctx).Model(&Product{}).Where("shop_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该店铺下存在关联商品，无法删除")
	}
	return r.data.db.WithContext(ctx).Delete(&Shop{}, id).Error
}
