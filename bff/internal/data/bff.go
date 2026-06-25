package data

import (
	"context"

	"bff/internal/biz"
	productv1 "productCenter/api/product/v1"
	mediav1 "productCenter/api/productmedia/v1"
	shopv1 "productCenter/api/shop/v1"
	skuv1 "productCenter/api/sku/v1"

	klog "github.com/go-kratos/kratos/v2/log"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/registry"
	"google.golang.org/grpc"
)

type bffRepo struct {
	log       *klog.Helper
	discovery registry.Discovery
}

// NewBFFRepo 创建 BFFRepo 实例，通过 Nacos 发现 ProductCenter 进行 gRPC 调用
func NewBFFRepo(data *Data, logger klog.Logger) biz.BFFRepo {
	return &bffRepo{
		log:       klog.NewHelper(logger),
		discovery: data.discovery,
	}
}

// getProductCenterConn 通过 Nacos 发现 ProductCenter 并建立 gRPC 连接
func (r *bffRepo) getProductCenterConn(ctx context.Context) (*grpc.ClientConn, error) {
	return kratosgrpc.DialInsecure(
		ctx,
		kratosgrpc.WithEndpoint("discovery:///productCenter"),
		kratosgrpc.WithDiscovery(r.discovery),
	)
}

// GetProductDetail 通过 gRPC 调用 ProductCenter 各领域服务，聚合商品详情
func (r *bffRepo) GetProductDetail(ctx context.Context, productID int64) (*biz.ProductDetail, error) {
	conn, err := r.getProductCenterConn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	productClient := productv1.NewProductClient(conn)
	productResp, err := productClient.GetProduct(ctx, &productv1.GetProductRequest{Id: productID})
	if err != nil {
		return nil, err
	}

	shopClient := shopv1.NewShopClient(conn)
	shopResp, err := shopClient.GetShop(ctx, &shopv1.GetShopRequest{Id: productResp.Product.ShopId})
	if err != nil {
		return nil, err
	}

	skuClient := skuv1.NewSkuClient(conn)
	skuResp, err := skuClient.ListSkus(ctx, &skuv1.ListSkusRequest{ProductId: productID, PageSize: 100})
	if err != nil {
		return nil, err
	}

	mediaClient := mediav1.NewProductMediaClient(conn)
	mediaResp, err := mediaClient.ListMedias(ctx, &mediav1.ListMediasRequest{ProductId: productID, PageSize: 100})
	if err != nil {
		return nil, err
	}

	tags := make([]*biz.ProductTag, 0, len(productResp.Product.TagIds))
	for i, tagID := range productResp.Product.TagIds {
		tagName := ""
		if i < len(productResp.Product.TagNames) {
			tagName = productResp.Product.TagNames[i]
		}
		tags = append(tags, &biz.ProductTag{
			ID:   int64(tagID),
			Name: tagName,
		})
	}

	medias := make([]*biz.ProductMedia, 0, len(mediaResp.Medias))
	for _, m := range mediaResp.Medias {
		medias = append(medias, &biz.ProductMedia{
			ID:        m.Id,
			ProductID: m.ProductId,
			URL:       m.Url,
			Sort:      int(m.Sort),
		})
	}

	skus := make([]*biz.Sku, 0, len(skuResp.Skus))
	for _, s := range skuResp.Skus {
		skus = append(skus, &biz.Sku{
			ID:        s.Id,
			ProductID: s.ProductId,
			Sku:       s.Sku,
			Title:     s.Title,
			Price:     int(s.Price),
			Stock:     int(s.Stock),
			ImgURL:    s.ImgUrl,
		})
	}

	product := &biz.Product{
		ID:             productResp.Product.Id,
		ShopID:         productResp.Product.ShopId,
		ShopName:       shopResp.Shop.ShopName,
		Name:           productResp.Product.Name,
		Type:           productResp.Product.Type,
		Description:    productResp.Product.Description,
		MainImageURL:   productResp.Product.MainImageUrl,
		Price:          int(productResp.Product.Price),
		CompareAtPrice: int(productResp.Product.CompareAtPrice),
		Status:         productResp.Product.Status,
		Sort:           int(productResp.Product.Sort),
		Tags:           tags,
	}

	return &biz.ProductDetail{
		Product: product,
		Shop: &biz.Shop{
			ID:          shopResp.Shop.Id,
			ShopName:    shopResp.Shop.ShopName,
			Description: shopResp.Shop.Description,
		},
		Tags:   tags,
		Skus:   skus,
		Medias: medias,
	}, nil
}

// ListProducts 通过 gRPC 调用 ProductCenter 查询商品列表
func (r *bffRepo) ListProducts(ctx context.Context, page, pageSize int32, shopID int64, status int32) ([]*biz.ProductListItem, int64, error) {
	conn, err := r.getProductCenterConn(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer conn.Close()

	productClient := productv1.NewProductClient(conn)
	resp, err := productClient.ListProducts(ctx, &productv1.ListProductsRequest{
		Page:     page,
		PageSize: pageSize,
		ShopId:   shopID,
		Status:   status,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]*biz.ProductListItem, 0, len(resp.Products))
	for _, p := range resp.Products {
		tags := make([]*biz.ProductTag, 0, len(p.TagIds))
		for i, tagID := range p.TagIds {
			tagName := ""
			if i < len(p.TagNames) {
				tagName = p.TagNames[i]
			}
			tags = append(tags, &biz.ProductTag{
				ID:   int64(tagID),
				Name: tagName,
			})
		}
		items = append(items, &biz.ProductListItem{
			Product: &biz.Product{
				ID:             p.Id,
				ShopID:         p.ShopId,
				ShopName:       p.ShopName,
				Name:           p.Name,
				Type:           p.Type,
				MainImageURL:   p.MainImageUrl,
				Price:          int(p.Price),
				CompareAtPrice: int(p.CompareAtPrice),
				Status:         p.Status,
				Tags:           tags,
			},
			ShopName: p.ShopName,
			Tags:     tags,
		})
	}
	return items, int64(resp.Total), nil
}

// GetShopHome 通过 gRPC 调用 ProductCenter 聚合查询店铺首页
func (r *bffRepo) GetShopHome(ctx context.Context, shopID int64, page, pageSize int32) (*biz.ShopHome, error) {
	conn, err := r.getProductCenterConn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	shopClient := shopv1.NewShopClient(conn)
	shopResp, err := shopClient.GetShop(ctx, &shopv1.GetShopRequest{Id: shopID})
	if err != nil {
		return nil, err
	}

	productClient := productv1.NewProductClient(conn)
	productResp, err := productClient.ListProducts(ctx, &productv1.ListProductsRequest{
		Page:     page,
		PageSize: pageSize,
		ShopId:   shopID,
		Status:   -1,
	})
	if err != nil {
		return nil, err
	}

	products := make([]*biz.ProductListItem, 0, len(productResp.Products))
	for _, p := range productResp.Products {
		tags := make([]*biz.ProductTag, 0, len(p.TagIds))
		for i, tagID := range p.TagIds {
			tagName := ""
			if i < len(p.TagNames) {
				tagName = p.TagNames[i]
			}
			tags = append(tags, &biz.ProductTag{
				ID:   int64(tagID),
				Name: tagName,
			})
		}
		products = append(products, &biz.ProductListItem{
			Product: &biz.Product{
				ID:             p.Id,
				ShopID:         p.ShopId,
				ShopName:       p.ShopName,
				Name:           p.Name,
				Type:           p.Type,
				MainImageURL:   p.MainImageUrl,
				Price:          int(p.Price),
				CompareAtPrice: int(p.CompareAtPrice),
				Status:         p.Status,
				Tags:           tags,
			},
			ShopName: p.ShopName,
			Tags:     tags,
		})
	}

	return &biz.ShopHome{
		Shop: &biz.Shop{
			ID:          shopResp.Shop.Id,
			ShopName:    shopResp.Shop.ShopName,
			Description: shopResp.Shop.Description,
		},
		Products: products,
		Total:    int64(productResp.Total),
	}, nil
}
