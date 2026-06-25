# BFF 整合订单服务 + gRPC Client 统一写法 + SKU 版本号库存扣减 实现计划

## 目标

1. BFF 和 Order 统一写法：`repo` 存 `*Data`，`Data` 只存 `*grpc.ClientConn`（连接），方法里按需创建轻量 client
2. BFF 新增 `CreateOrder` 接口，自动生成 `request_id`
3. SKU 库存扣减改为版本号乐观锁机制

## request_id 策略

先用 A（BFF 自动 UUID），后续可升级到 B（前端传 idempotency-key）。

## 统一模式

```
ProductCenter:  repo 存 *Data →  r.data.db
Order:          repo 存 *Data →  r.data.db、r.data.rdb、r.data.pcConn
BFF:            repo 存 *Data →  r.data.pcConn、r.data.orderConn
```

Data 只存基础设施资源，方法里按需 New 轻量 client。

## 改动详情

### 阶段一：统一写法（重构，BFF + Order）

#### 1. `bff/internal/data/data.go` — 存 `*grpc.ClientConn`

```go
// 现在
type Data struct {
    log       *log.Helper
    discovery registry.Discovery
}

// 改后
type Data struct {
    log       *log.Helper
    discovery registry.Discovery
    pcConn    *grpc.ClientConn  // 连 productCenter，启动时建好
}
```

`NewData` 里建连接，`cleanup` 里 `pcConn.Close()`。

#### 2. `bff/internal/data/bff.go` — 存 `*Data`

```go
// 现在
type bffRepo struct {
    log       *klog.Helper
    discovery registry.Discovery
}
func NewBFFRepo(data *Data, logger klog.Logger) biz.BFFRepo {
    return &bffRepo{log: klog.NewHelper(logger), discovery: data.discovery}
}

// 改后
type bffRepo struct {
    data *Data
}
func NewBFFRepo(data *Data) biz.BFFRepo {
    return &bffRepo{data: data}
}
```

去掉 `getProductCenterConn`，方法里直接 `r.data.pcConn`：

```go
func (r *bffRepo) GetProductDetail(ctx context.Context, productID int64) (*biz.ProductDetail, error) {
    productClient := productv1.NewProductClient(r.data.pcConn)
    shopClient := shopv1.NewShopClient(r.data.pcConn)
    skuClient := skuv1.NewSkuClient(r.data.pcConn)
    mediaClient := mediav1.NewProductMediaClient(r.data.pcConn)
    // ... 聚合逻辑不变
}
```

#### 3. `order/internal/data/data.go` — 存 `*grpc.ClientConn`

```go
// 现在
type Data struct {
    db        *gorm.DB
    rdb       redis.UniversalClient
    skuClient skuv1.SkuClient
}

// 改后
type Data struct {
    db     *gorm.DB
    rdb    redis.UniversalClient
    pcConn *grpc.ClientConn  // 连 productCenter
}
```

`NewData` 里建连接，不再创建具体 client：

```go
pcConn, err := kratosgrpc.DialInsecure(
    context.Background(),
    kratosgrpc.WithEndpoint("discovery:///productCenter"),
    kratosgrpc.WithDiscovery(disc),
)
return &Data{db: db, rdb: rdb, pcConn: pcConn}, cleanup, nil
```

#### 4. `order/internal/data/order.go` — 方法里创建 client

```go
// 现在
func NewOrderRepo(data *Data) biz.OrderRepo {
    return &OrderRepo{data: data}
}
// DeductStock 里用 r.data.skuClient

// 改后 — NewOrderRepo 不变
// DeductStock 改为方法里创建轻量 client
func (r *OrderRepo) DeductStock(ctx context.Context, skuID int64, quantity int) error {
    skuClient := skuv1.NewSkuClient(r.data.pcConn)
    resp, err := skuClient.DeductStock(ctx, &skuv1.DeductStockRequest{...})
    ...
}
```

#### 5. `order/cmd/order/main.go` — 调整

`NewData` 签名从 `(c *conf.Data, rc *conf.Registry, disc registry.Discovery, logger)` 变为 `(c *conf.Data, disc registry.Discovery, logger)`，去掉没用的 `*conf.Registry`。wire.go 同步调整。

### 编译验证（阶段一）

```bash
cd order && go mod tidy && go generate ./... && go build ./...
cd bff && go generate ./... && go build ./...
```

### 阶段二：BFF 新增 CreateOrder

#### 6. `bff/api/bff/v1/bff.proto` — 新增 RPC

```protobuf
rpc CreateOrder (CreateOrderRequest) returns (CreateOrderResponse) {
    option (google.api.http) = {
        post: "/api/v1/bff/orders"
        body: "*"
    };
}

message CreateOrderRequest {
    int64 user_id = 1;
    int64 shop_id = 2;
    repeated OrderItem items = 3;
}
message CreateOrderResponse {
    int64 order_id = 1;
    string order_no = 2;
}
message OrderItem {
    int64 product_id = 1; int64 sku_id = 2;
    string product_name = 3; string sku_title = 4;
    int32 price = 5; int32 quantity = 6;
    string image_url = 7;
}
```

#### 7. `bff/go.mod` — 新增 Order 依赖

```
require order v0.0.0
replace order => ../order
go get order
```

#### 8. `bff/internal/data/data.go` — 加 orderConn

```go
type Data struct {
    log       *log.Helper
    discovery registry.Discovery
    pcConn    *grpc.ClientConn    // 连 productCenter
    orderConn *grpc.ClientConn    // 连 order（新增）
}
```

#### 9. `bff/internal/data/bff.go` — 新增 CreateOrder

```go
func (r *bffRepo) CreateOrder(ctx context.Context, requestID string, userID, shopID int64, items []*biz.OrderItem) (*biz.CreateOrderResult, error) {
    orderClient := orderv1.NewOrderServiceClient(r.data.orderConn)
    protoItems := make([]*orderv1.OrderItem, len(items))
    for i := range items {
        protoItems[i] = &orderv1.OrderItem{
            ProductId: items[i].ProductID, SkuId: items[i].SKUID,
            ProductName: items[i].ProductName, SkuTitle: items[i].SKUTitle,
            Price: int32(items[i].Price), Quantity: int32(items[i].Quantity),
            ImageUrl: items[i].ImageURL,
        }
    }
    resp, err := orderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
        RequestId: requestID, UserId: userID, ShopId: shopID, Items: protoItems,
    })
    if err != nil { return nil, err }
    return &biz.CreateOrderResult{OrderID: resp.OrderId, OrderNo: resp.OrderNo}, nil
}
```

#### 10. `bff/internal/biz/bff.go` — 新增接口和方法

```go
// BFFRepo 接口新增
CreateOrder(ctx context.Context, requestID string, userID, shopID int64, items []*OrderItem) (*CreateOrderResult, error)

// BFFUseCase 新增
func (uc *BFFUseCase) CreateOrder(ctx context.Context, userID, shopID int64, items []*OrderItem) (*CreateOrderResult, error) {
    requestID := uuid.NewString()
    uc.log.WithContext(ctx).Infof("CreateOrder: requestID=%s", requestID)
    return uc.repo.CreateOrder(ctx, requestID, userID, shopID, items)
}
```

#### 11. `bff/internal/biz/entity.go` — 新增实体

```go
type OrderItem struct {
    ProductID   int64  `json:"product_id"`
    SKUID       int64  `json:"sku_id"`
    ProductName string `json:"product_name"`
    SKUTitle    string `json:"sku_title"`
    Price       int    `json:"price"`
    Quantity    int    `json:"quantity"`
    ImageURL    string `json:"image_url"`
}
type CreateOrderResult struct {
    OrderID int64  `json:"order_id"`
    OrderNo string `json:"order_no"`
}
```

#### 12. `bff/internal/service/bff.go` — 新增 handler

```go
func (s *BFFService) CreateOrder(ctx context.Context, req *v1.CreateOrderRequest) (*v1.CreateOrderResponse, error) {
    items := make([]*biz.OrderItem, len(req.Items))
    for i := range req.Items {
        items[i] = &biz.OrderItem{
            ProductID: req.Items[i].ProductId, SKUID: req.Items[i].SkuId,
            ProductName: req.Items[i].ProductName, SKUTitle: req.Items[i].SkuTitle,
            Price: int(req.Items[i].Price), Quantity: int(req.Items[i].Quantity),
            ImageURL: req.Items[i].ImageUrl,
        }
    }
    result, err := s.uc.CreateOrder(ctx, req.UserId, req.ShopId, items)
    if err != nil { return nil, err }
    return &v1.CreateOrderResponse{OrderId: result.OrderID, OrderNo: result.OrderNo}, nil
}
```

### 编译验证（阶段二）

```bash
cd bff && make api && go mod tidy && go generate ./... && go build ./...
```

### 阶段三：SKU 库存扣减改为版本号乐观锁

当前扣减方式：

```sql
-- 现在：检查 stock >= quantity
UPDATE sku SET stock = stock - ? WHERE id = ? AND stock >= ?
```

问题：两个并发请求都读到 stock=10，都判断 stock>=5 为真，都更新成功 → 超卖。

改为版本号机制：

```sql
-- 改后：检查版本号
UPDATE sku SET stock = stock - ?, version = version + 1 WHERE id = ? AND version = ?
```

流程：Order 扣库存前先 `GetSku` 获取当前 version，传给 `DeductStock`，版本不匹配就重试。

#### 13. `schema.sql` — sku 表加 version 字段

```sql
alter table sku add column version int not null default 0 comment '版本号（乐观锁）' after stock;
```

#### 14. `productCenter/internal/data/model.go` — Sku model 加 Version

```go
type Sku struct {
    // ... 现有字段
    Version   int            `gorm:"column:version;type:int;not null;default:0" json:"version"`  // 新增
}
```

#### 15. `productCenter/api/sku/v1/sku.proto` — SkuInfo 和 DeductStockRequest 加 version

```protobuf
message SkuInfo {
    // ... 现有字段
    int64  version = 10;  // 新增：版本号（乐观锁）
}

message DeductStockRequest {
    int64 id = 1;
    int64 quantity = 2;
    int64 version = 3;  // 新增：Order 传当前版本号过来
}
```

#### 16. `productCenter/internal/data/sku.go` — DeductStock 改为版本号乐观锁

```go
func (r *skuRepo) DeductStock(ctx context.Context, id int64, quantity int, version int) (int, int, error) {
    result := r.data.db.WithContext(ctx).Exec(
        "UPDATE sku SET stock = stock - ?, version = version + 1 WHERE id = ? AND version = ? AND stock >= ?",
        quantity, id, version, quantity,
    )
    if result.Error != nil {
        return 0, 0, result.Error
    }
    if result.RowsAffected == 0 {
        return 0, 0, biz.ErrSkuVersionConflict  // 版本冲突
    }

    var stock, newVersion int
    r.data.db.WithContext(ctx).Model(&Sku{}).Where("id = ?", id).Select("stock", "version").Row().Scan(&stock, &newVersion)
    return stock, newVersion, nil
}
```

返回 `(newStock, newVersion, error)`，Order 拿到新 version 后下次重试用。

#### 17. `productCenter/internal/biz/sku.go` — 接口和错误调整

```go
// 新增 ErrSkuVersionConflict
var ErrSkuVersionConflict = errors.New(409, "SKU_VERSION_CONFLICT", "版本号冲突，请重试")

// SkuRepo 接口调整：DeductStock 多加 version 参数，返回值加 newVersion
type SkuRepo interface {
    DeductStock(ctx context.Context, id int64, quantity int, version int) (int, int, error)
    // ...
}

// SkuUseCase.DeductStock 签名调整
func (uc *SkuUseCase) DeductStock(ctx context.Context, id int64, quantity int, version int) (int, int, error) {
    return uc.repo.DeductStock(ctx, id, quantity, version)
}
```

#### 18. `productCenter/internal/biz/entity.go` — Sku 实体加 Version

```go
type Sku struct {
    // ... 现有字段
    Version   int       `json:"version"`  // 新增
}
```

#### 19. `productCenter/internal/service/sku.go` — DeductStock handler 调整

```go
func (s *SkuService) DeductStock(ctx context.Context, req *v1.DeductStockRequest) (*v1.DeductStockResponse, error) {
    newStock, newVersion, err := s.uc.DeductStock(ctx, req.Id, int(req.Quantity), int(req.Version))
    if err != nil { return nil, err }
    return &v1.DeductStockResponse{Success: true, NewStock: int64(newStock), NewVersion: int64(newVersion)}, nil
}
```

DeductStockResponse 也加 `int64 new_version` 字段。

#### 20. `order/internal/data/order.go` — DeductStock 带版本号 + 重试

```go
func (r *OrderRepo) DeductStock(ctx context.Context, skuID int64, quantity int) error {
    skuClient := skuv1.NewSkuClient(r.data.pcConn)

    for retry := 0; retry < 3; retry++ {
        // 1. 先拿当前 version
        skuResp, err := skuClient.GetSku(ctx, &skuv1.GetSkuRequest{Id: skuID})
        if err != nil { return err }

        // 2. 带 version 扣减
        resp, err := skuClient.DeductStock(ctx, &skuv1.DeductStockRequest{
            Id: skuID, Quantity: int64(quantity), Version: skuResp.Sku.Version,
        })
        if err == nil && resp.Success {
            return nil
        }
        // 3. 版本冲突 → 重试
    }
    return biz.ErrInsufficientStock
}
```

### 编译验证（阶段三）

```bash
cd productCenter && make api && go generate ./... && go build ./...
cd order && go generate ./... && go build ./...
cd bff && go build ./...
```

## 调用链路

```
前端 → POST /api/v1/bff/orders
  ├─ BFF Biz: requestID = uuid.NewString()
  ├─ BFF Repo: NewOrderServiceClient(r.data.orderConn).CreateOrder(...)
  │     └─ gRPC → Order.CreateOrder
  │           ├─ Redis SET NX + DB 查询幂等
  │           ├─ for each item:
  │           │     ├─ GetSku(skuID) → 拿到 version
  │           │     └─ DeductStock(skuID, quantity, version)
  │           │           ├─ 版本匹配 → 扣减成功
  │           │           └─ 版本冲突 → 重试（最多3次）
  │           └─ 创建订单 + 订单项
  └─ 返回 {order_id, order_no}
```

## 不涉及

- 不修改 ProductCenter 商品/SKU 的核心业务逻辑（只改扣减方式）
- 本次只整合 CreateOrder
