package data

import (
	"context"
	"errors"

	"productCenter/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type productRepo struct {
	data *Data
	log  *log.Helper
}

// NewProductRepo 创建 ProductRepo 实例，实现 biz.ProductRepo 接口
func NewProductRepo(data *Data, logger log.Logger) biz.ProductRepo {
	return &productRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// Create 创建商品，同时处理标签关联（事务）
func (r *productRepo) Create(ctx context.Context, product *biz.Product, tagIDs []int64) (*biz.Product, error) {
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		po := Product{
			ShopID:         product.ShopID,
			Name:           product.Name,
			Type:           product.Type,
			Description:    product.Description,
			MainImageURL:   product.MainImageURL,
			Price:          product.Price,
			CompareAtPrice: product.CompareAtPrice,
			Status:         int8(product.Status),
			Sort:           product.Sort,
		}
		if err := tx.Create(&po).Error; err != nil {
			return err
		}
		product.ID = po.ID
		product.CreatedAt = po.CreatedAt
		product.UpdatedAt = po.UpdatedAt

		// 同步处理 product_tag_mapping 关联
		if len(tagIDs) > 0 {
			mappings := make([]ProductTagMapping, 0, len(tagIDs))
			for _, tagID := range tagIDs {
				mappings = append(mappings, ProductTagMapping{
					ProductID: po.ID,
					TagID:     tagID,
				})
			}
			if err := tx.Create(&mappings).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return product, nil
}

// Update 更新商品信息，同时更新标签关联（事务）
func (r *productRepo) Update(ctx context.Context, product *biz.Product, tagIDs []int64) (*biz.Product, error) {
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		po := Product{
			ShopID:         product.ShopID,
			Name:           product.Name,
			Type:           product.Type,
			Description:    product.Description,
			MainImageURL:   product.MainImageURL,
			Price:          product.Price,
			CompareAtPrice: product.CompareAtPrice,
			Status:         int8(product.Status),
			Sort:           product.Sort,
		}
		if err := tx.Model(&Product{}).Where("id = ?", product.ID).Updates(po).Error; err != nil {
			return err
		}

		// 先删除旧的标签关联，再创建新的
		if err := tx.Where("product_id = ?", product.ID).Delete(&ProductTagMapping{}).Error; err != nil {
			return err
		}
		if len(tagIDs) > 0 {
			mappings := make([]ProductTagMapping, 0, len(tagIDs))
			for _, tagID := range tagIDs {
				mappings = append(mappings, ProductTagMapping{
					ProductID: product.ID,
					TagID:     tagID,
				})
			}
			if err := tx.Create(&mappings).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return product, nil
}

// Get 根据ID获取商品详情（含标签信息）
func (r *productRepo) Get(ctx context.Context, id int64) (*biz.Product, error) {
	var po Product
	if err := r.data.db.WithContext(ctx).Preload("Tags").Preload("Shop").First(&po, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrProductNotFound
		}
		return nil, err
	}

	product := &biz.Product{
		ID:             po.ID,
		ShopID:         po.ShopID,
		ShopName:       po.Shop.ShopName,
		Name:           po.Name,
		Type:           po.Type,
		Description:    po.Description,
		MainImageURL:   po.MainImageURL,
		Price:          po.Price,
		CompareAtPrice: po.CompareAtPrice,
		Status:         int32(po.Status),
		Sort:           po.Sort,
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}

	for _, tag := range po.Tags {
		product.Tags = append(product.Tags, &biz.ProductTag{
			ID:   tag.ID,
			Name: tag.Name,
			Sort: tag.Sort,
		})
	}
	return product, nil
}

// List 分页查询商品列表，支持按店铺和状态筛选
func (r *productRepo) List(ctx context.Context, page, pageSize int32, shopID int64, status int32) ([]*biz.Product, int64, error) {
	var pos []Product
	var total int64

	db := r.data.db.WithContext(ctx).Model(&Product{})
	if shopID > 0 {
		db = db.Where("shop_id = ?", shopID)
	}
	if status >= 0 {
		db = db.Where("status = ?", int8(status))
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (int(page) - 1) * int(pageSize)
	if err := db.Preload("Tags").Preload("Shop").Order("sort asc, id desc").Offset(offset).Limit(int(pageSize)).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	products := make([]*biz.Product, 0, len(pos))
	for _, po := range pos {
		product := &biz.Product{
			ID:             po.ID,
			ShopID:         po.ShopID,
			ShopName:       po.Shop.ShopName,
			Name:           po.Name,
			Type:           po.Type,
			Description:    po.Description,
			MainImageURL:   po.MainImageURL,
			Price:          po.Price,
			CompareAtPrice: po.CompareAtPrice,
			Status:         int32(po.Status),
			Sort:           po.Sort,
			CreatedAt:      po.CreatedAt,
			UpdatedAt:      po.UpdatedAt,
		}
		for _, tag := range po.Tags {
			product.Tags = append(product.Tags, &biz.ProductTag{
				ID:   tag.ID,
				Name: tag.Name,
				Sort: tag.Sort,
			})
		}
		products = append(products, product)
	}
	return products, total, nil
}

// Delete 删除商品（软删除）
func (r *productRepo) Delete(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&Product{}, id).Error
}
