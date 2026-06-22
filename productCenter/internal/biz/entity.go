package biz

import "time"

// Shop 店铺业务实体
type Shop struct {
	ID          int64     `json:"id"`
	ShopName    string    `json:"shop_name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Product 商品业务实体
type Product struct {
	ID             int64         `json:"id"`
	ShopID         int64         `json:"shop_id"`
	ShopName       string        `json:"shop_name"`
	Name           string        `json:"name"`
	Type           string        `json:"type"`
	Description    string        `json:"description"`
	MainImageURL   string        `json:"main_image_url"`
	Price          int           `json:"price"`            // 分
	CompareAtPrice int           `json:"compare_at_price"` // 划线价（分）
	Status         int32         `json:"status"`
	Sort           int           `json:"sort"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Tags           []*ProductTag `json:"tags,omitempty"`
}

// ProductTag 商品标签业务实体
type ProductTag struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProductMedia 商品媒体资源业务实体
type ProductMedia struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	URL       string    `json:"url"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Sku 商品SKU业务实体
type Sku struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	Sku       string    `json:"sku"`
	Title     string    `json:"title"`
	Price     int       `json:"price"`
	Stock     int       `json:"stock"`
	ImgURL    string    `json:"img_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProductDetail 商品详情聚合实体（BFF层使用）
type ProductDetail struct {
	Product *Product        `json:"product"`
	Shop    *Shop           `json:"shop"`
	Tags    []*ProductTag   `json:"tags"`
	Skus    []*Sku          `json:"skus"`
	Medias  []*ProductMedia `json:"medias"`
}

// ProductListItem 商品列表项（BFF层使用）
type ProductListItem struct {
	Product  *Product      `json:"product"`
	ShopName string        `json:"shop_name"`
	Tags     []*ProductTag `json:"tags"`
}

// ShopHome 店铺首页聚合实体（BFF层使用）
type ShopHome struct {
	Shop     *Shop              `json:"shop"`
	Products []*ProductListItem `json:"products"`
	Total    int64              `json:"total"`
}
