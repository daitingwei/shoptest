package data

import (
	"context"

	"productCenter/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type bffRepo struct {
	data *Data
	log  *log.Helper
}

// NewBFFRepo 创建 BFFRepo 实例，实现 biz.BFFRepo 接口
func NewBFFRepo(data *Data, logger log.Logger) biz.BFFRepo {
	return &bffRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// GetProductDetail 聚合查询商品详情，包含商品、店铺、标签、SKU、媒体
func (r *bffRepo) GetProductDetail(ctx context.Context, productID int64) (*biz.ProductDetail, error) {
	// 查询商品（含标签）
	var po Product
	if err := r.data.db.WithContext(ctx).Preload("Tags").First(&po, productID).Error; err != nil {
		return nil, err
	}

	product := &biz.Product{
		ID:             po.ID,
		ShopID:         po.ShopID,
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

	// 查询店铺
	var shopPO Shop
	if err := r.data.db.WithContext(ctx).First(&shopPO, po.ShopID).Error; err != nil {
		return nil, err
	}
	shop := &biz.Shop{
		ID:          shopPO.ID,
		ShopName:    shopPO.ShopName,
		Description: shopPO.Description,
		CreatedAt:   shopPO.CreatedAt,
		UpdatedAt:   shopPO.UpdatedAt,
	}

	// 查询SKU列表
	var skuPOs []Sku
	if err := r.data.db.WithContext(ctx).Where("product_id = ?", productID).Find(&skuPOs).Error; err != nil {
		return nil, err
	}
	skus := make([]*biz.Sku, 0, len(skuPOs))
	for _, s := range skuPOs {
		skus = append(skus, &biz.Sku{
			ID:        s.ID,
			ProductID: s.ProductID,
			Sku:       s.Sku,
			Title:     s.Title,
			Price:     s.Price,
			Stock:     s.Stock,
			ImgURL:    s.ImgURL,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		})
	}

	// 查询媒体列表
	var mediaPOs []ProductMedia
	if err := r.data.db.WithContext(ctx).Where("product_id = ?", productID).Order("sort asc, id asc").Find(&mediaPOs).Error; err != nil {
		return nil, err
	}
	medias := make([]*biz.ProductMedia, 0, len(mediaPOs))
	for _, m := range mediaPOs {
		medias = append(medias, &biz.ProductMedia{
			ID:        m.ID,
			ProductID: m.ProductID,
			URL:       m.URL,
			Sort:      m.Sort,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		})
	}

	return &biz.ProductDetail{
		Product: product,
		Shop:    shop,
		Tags:    product.Tags,
		Skus:    skus,
		Medias:  medias,
	}, nil
}

// ListProducts 聚合查询商品列表，支持按店铺和状态筛选
func (r *bffRepo) ListProducts(ctx context.Context, page, pageSize int32, shopID int64, status int32) ([]*biz.ProductListItem, int64, error) {
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
	if err := db.Preload("Tags").Order("sort asc, id desc").Offset(offset).Limit(int(pageSize)).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	// 收集所有 shopID 并批量查询
	shopIDs := make([]int64, 0)
	for _, po := range pos {
		shopIDs = append(shopIDs, po.ShopID)
	}

	shopMap := make(map[int64]*Shop)
	if len(shopIDs) > 0 {
		var shops []Shop
		if err := r.data.db.WithContext(ctx).Where("id IN ?", shopIDs).Find(&shops).Error; err != nil {
			return nil, 0, err
		}
		for i := range shops {
			shopMap[shops[i].ID] = &shops[i]
		}
	}

	items := make([]*biz.ProductListItem, 0, len(pos))
	for _, po := range pos {
		product := &biz.Product{
			ID:             po.ID,
			ShopID:         po.ShopID,
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

		tags := make([]*biz.ProductTag, 0, len(po.Tags))
		for _, tag := range po.Tags {
			tags = append(tags, &biz.ProductTag{
				ID:   tag.ID,
				Name: tag.Name,
				Sort: tag.Sort,
			})
		}

		shopName := ""
		if s, ok := shopMap[po.ShopID]; ok {
			shopName = s.ShopName
		}
		product.ShopName = shopName

		items = append(items, &biz.ProductListItem{
			Product:  product,
			ShopName: shopName,
			Tags:     tags,
		})
	}
	return items, total, nil
}

// GetShopHome 聚合查询店铺首页，包含店铺信息和商品列表
func (r *bffRepo) GetShopHome(ctx context.Context, shopID int64, page, pageSize int32) (*biz.ShopHome, error) {
	// 查询店铺
	var shopPO Shop
	if err := r.data.db.WithContext(ctx).First(&shopPO, shopID).Error; err != nil {
		return nil, err
	}
	shop := &biz.Shop{
		ID:          shopPO.ID,
		ShopName:    shopPO.ShopName,
		Description: shopPO.Description,
		CreatedAt:   shopPO.CreatedAt,
		UpdatedAt:   shopPO.UpdatedAt,
	}

	// 查询该店铺下的商品列表（含标签）
	var productPOs []Product
	var total int64

	db := r.data.db.WithContext(ctx).Model(&Product{}).Where("shop_id = ?", shopID)
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (int(page) - 1) * int(pageSize)
	if err := db.Preload("Tags").Order("sort asc, id desc").Offset(offset).Limit(int(pageSize)).Find(&productPOs).Error; err != nil {
		return nil, err
	}

	products := make([]*biz.ProductListItem, 0, len(productPOs))
	for _, po := range productPOs {
		product := &biz.Product{
			ID:             po.ID,
			ShopID:         po.ShopID,
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
			ShopName:       shopPO.ShopName,
		}

		tags := make([]*biz.ProductTag, 0, len(po.Tags))
		for _, tag := range po.Tags {
			tags = append(tags, &biz.ProductTag{
				ID:   tag.ID,
				Name: tag.Name,
				Sort: tag.Sort,
			})
		}

		products = append(products, &biz.ProductListItem{
			Product:  product,
			ShopName: shopPO.ShopName,
			Tags:     tags,
		})
	}

	return &biz.ShopHome{
		Shop:     shop,
		Products: products,
		Total:    total,
	}, nil
}
