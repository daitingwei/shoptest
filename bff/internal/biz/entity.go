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
	Price          int           `json:"price"`
	CompareAtPrice int           `json:"compare_at_price"`
	Status         int32         `json:"status"`
	Sort           int           `json:"sort"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Tags           []*ProductTag `json:"tags,omitempty"`
}

// ProductTag 商品标签业务实体
type ProductTag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Sort int    `json:"sort"`
}

// ProductMedia 商品媒体资源业务实体
type ProductMedia struct {
	ID        int64  `json:"id"`
	ProductID int64  `json:"product_id"`
	URL       string `json:"url"`
	Sort      int    `json:"sort"`
}

// Sku 商品SKU业务实体
type Sku struct {
	ID        int64  `json:"id"`
	ProductID int64  `json:"product_id"`
	Sku       string `json:"sku"`
	Title     string `json:"title"`
	Price     int    `json:"price"`
	Stock     int    `json:"stock"`
	ImgURL    string `json:"img_url"`
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

// OrderItem 订单项（BFF层）
type OrderItem struct {
	ProductID   int64  `json:"product_id"`
	SKUID       int64  `json:"sku_id"`
	ProductName string `json:"product_name"`
	SKUTitle    string `json:"sku_title"`
	Price       int    `json:"price"`
	Quantity    int    `json:"quantity"`
	ImageURL    string `json:"image_url"`
}

// CreateOrderResult 创建订单结果（BFF层）
type CreateOrderResult struct {
	OrderID int64  `json:"order_id"`
	OrderNo string `json:"order_no"`
}
