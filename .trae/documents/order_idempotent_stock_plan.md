# 订单服务幂等性与库存一致性实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 完成订单服务的幂等性和库存一致性功能，同时将 BFF 从 productCenter 独立为 gRPC 调用服务

**Architecture:** 四层架构（api/service/biz/data），BFF 独立部署后通过 gRPC 调用 Order 和 ProductCenter，Order 通过 gRPC 调用 ProductCenter 的库存接口

**Tech Stack:** Kratos v2, GORM, Wire, go-redis, gRPC, MySQL, Nacos

---

## 一、需求概述

1. **BFF 独立部署**：将 BFF 聚合层从 productCenter 迁出为独立服务，通过 gRPC 调用底层服务
2. **幂等性**：BFF 自动生成 request_id，确保同一请求多次调用只生成一个订单
3. **库存一致性**：下单时通过 gRPC 调用扣减库存（乐观锁），取消订单时回补库存

## 二、当前状态分析

### 架构与调用链路

```
现状：
前端 → ProductCenter（HTTP/gRPC，无服务发现，写死地址）
     → Order（HTTP/gRPC，无服务发现，写死地址）

目标：
前端 → BFF（独立服务）→ Nacos 发现 ProductCenter（gRPC）
                       → Nacos 发现 Order（gRPC）
```

### Order 服务（当前状态）
- `order/api/order/v1/order.proto`：CreateOrderRequest 没有 request_id 字段
- `order/internal/data/model.go`：Order 模型没有 RequestID 字段，没有 Sku 模型
- `order/internal/biz/order.go`：OrderRepo 没有库存相关接口
- `order/internal/data/order.go`：CreateOrder 直接在事务中写入，无幂等/库存逻辑
- `order/internal/conf/conf.proto`：已有 Redis 配置定义
- `order/configs/config.yaml`：已有 Redis 连接配置
- 无 Redis 客户端初始化
- **表结构**：通过 GORM AutoMigrate 自动管理，当前无 DDL 文件

### ProductCenter 服务（当前状态）
- `productCenter/api/sku/v1/sku.proto`：有 Sku CRUD，但没有库存扣减/回补 RPC
- `productCenter/internal/data/sku.go`：有基础的 CRUD，没有乐观锁扣库存方法
- `productCenter/internal/biz/sku.go`：有基础的业务逻辑，没有库存扣减/回补
- `productCenter/internal/data/model.go`：Sku 模型有 Stock 字段

### ProductCenter 内的 BFF 层（当前状态，待迁出）
- `productCenter/api/bff/v1/bff.proto`：3 个 RPC（GetProductDetail/ListProducts/GetShopHome）
- `productCenter/internal/service/bff.go`：BFF 聚合服务实现
- `productCenter/internal/biz/bff.go`：BFF 业务用例
- `productCenter/internal/data/bff.go`：**直接查数据库**做聚合，未通过 gRPC

## 三、关键设计决策

### 3.1 request_id 由谁生成

| 阶段 | 生成方 | 方式 | 说明 |
|------|--------|------|------|
| 迁移后 | **BFF 服务** | `uuid.NewString()` | BFF 在调用 Order.CreateOrder 前自动注入 |
| 不需要前端关心 | - | - | 前端只需正常调用 BFF 接口 |

### 3.2 ID 生成方式

| ID 类型 | 生成方 | 生成方式 |
|---------|--------|----------|
| 主键 ID（int64） | GORM AutoIncrement | 数据库自增，无需关心 |
| 订单号 order_no | Order 服务 | `ORD` + 时间戳 + 随机数（展示/对账用） |
| request_id | BFF 服务 | UUID v4（幂等用） |

### 3.3 数据库变更

全部通过 GORM AutoMigrate 自动管理，无需手动执行 DDL：

| 表 | 变更内容 | 方式 |
|----|----------|------|
| `orders`（新增） | AutoMigrate 首次启动自动创建 | GORM 自动 |
| `order_items`（新增） | AutoMigrate 首次启动自动创建 | GORM 自动 |
| `orders.request_id`（新增字段） | 新增 `varchar(64) uniqueIndex not null` 字段 | AutoMigrate 自动加列 + 建索引 |

**注意**：AutoMigrate 只会新增字段/表，不会删除已存在的字段或表。

### 3.4 BFF 迁移后的 gRPC 调用链路

```
BFF.GetProductDetail 时：
  BFF → 通过 Nacos 发现 ProductCenter
      → gRPC 调用 ProductCenter.GetProduct(id)
      → gRPC 调用 ProductCenter.GetShop(id)
      → gRPC 调用 ProductCenter.ListSkus(productID)
      → gRPC 调用 ProductCenter.ListMedias(productID)
      → 聚合后返回前端

BFF.CreateOrder 时（后续实现）：
  BFF → 自动生成 request_id
      → 通过 Nacos 发现 Order
      → gRPC 调用 Order.CreateOrder(request_id, user_id, items...)
```

### 3.5 Nacos 服务发现

#### 为什么需要 Nacos

三个服务之间需要通过 gRPC 互相调用，写死地址不靠谱：
- BFF → ProductCenter（查商品/店铺/SKU/媒体）
- Order → ProductCenter（扣减/回补库存）
- BFF → Order（创建订单）

#### 服务注册方案

每个服务在启动时通过 Nacos 注册自己的 gRPC 地址：

| 服务 | Nacos 服务名 | 注册地址 |
|------|-------------|----------|
| ProductCenter | `productCenter` | gRPC 9000 |
| Order | `order` | gRPC 9001 |
| BFF | `bff` | gRPC 9002 |

#### gRPC 客户端发现方式

不再写死 `127.0.0.1:9000`，改为通过 Nacos 发现：

```go
// 原来（写死地址）：
conn, _ := grpc.Dial("127.0.0.1:9000", ...)

// 改为（Nacos 发现）：
import kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"

conn, _ := kratosgrpc.DialInsecure(
    context.Background(),
    kratosgrpc.WithEndpoint("discovery:///productCenter"),
    kratosgrpc.WithDiscovery(nacosRegistry),
)
```

#### 依赖

每引入一个需要 Nacos 的服务，`go.mod` 需新增：
- `github.com/nacos-group/nacos-sdk-go/v2`
- `github.com/go-kratos/kratos/contrib/registry/nacos/v2`

#### 配置结构（所有服务统一）

**conf.proto 新增**：
```protobuf
message Registry {
    message Nacos {
        string address = 1;           // Nacos 地址，如 "127.0.0.1:8848"
        string namespace_id = 2;      // 命名空间，默认 "public"
        int64 timeout_ms = 3;         // 超时时间，默认 5000
        string log_dir = 4;           // 日志目录
        string cache_dir = 5;         // 缓存目录
    }
    Nacos nacos = 1;
}

message Bootstrap {
    Server server = 1;
    Data data = 2;
    Registry registry = 3;
}
```

**config.yaml 新增**：
```yaml
registry:
  nacos:
    address: 127.0.0.1:8848
    namespace_id: public
    timeout_ms: 5000
    log_dir: /tmp/nacos/log
    cache_dir: /tmp/nacos/cache
```

#### 服务端注册流程（每个服务的 main.go 都要改）

```go
// 1. 创建 Nacos 客户端
sc := []constant.ServerConfig{
    *constant.NewServerConfig(nacosAddr, 8848),
}
cc := constant.ClientConfig{
    NamespaceId:         "public",
    TimeoutMs:           5000,
    NotLoadCacheAtStart: true,
    LogDir:              "/tmp/nacos/log",
    CacheDir:            "/tmp/nacos/cache",
}
client, _ := clients.NewNamingClient(vo.NacosClientParam{
    ClientConfig:  &cc,
    ServerConfigs: sc,
})

// 2. 创建 Kratos Registry
r := nacosRegistry.New(client)

// 3. App 启动时注册到 Nacos
app := kratos.New(
    kratos.ID(id),
    kratos.Name("productCenter"),     // 服务名
    kratos.Version(Version),
    kratos.Metadata(map[string]string{}),
    kratos.Logger(logger),
    kratos.Server(gs, hs),
    kratos.Registrar(r),               // 注册到 Nacos
)
```

## 四、技术方案

### 4.1 幂等性方案

**三级保护，按请求顺序执行**：

```
请求进来（带 request_id，由 BFF 自动生成）
    │
    ▼
① Redis SET NX 判断（24小时过期自动清理）
   · 已存在 → 直接返回已有订单
   · 不存在 → 继续执行
   · Redis 异常 → 自动降级，不阻塞下单
    │
    ▼
② 数据库查询校验（按 request_id 查询）
   · 已存在 → 直接返回
   · 不存在 → 继续创建
    │
    ▼
③ 数据库唯一索引兜底（request_id 字段 uniqueIndex）
   · 并发极端情况下强制不重复
   · 捕获唯一索引冲突后再查询返回
```

### 4.2 库存一致性方案

```
创建订单流程（事务内）：
① 调用 productCenter gRPC：DeductStock(sku_id, quantity)
   · productCenter 内部用 UPDATE sku SET stock = stock - ? WHERE id = ? AND stock >= ?
   · 影响行数 = 0 → 库存不足，返回错误
   · 影响行数 > 0 → 扣减成功，返回最新库存
② 创建订单（Order + OrderItem）
③ 事务提交（任一步失败全部回滚）

取消订单流程（事务内）：
① 查询订单项，获取已扣减的 sku_id + quantity
② 调用 productCenter gRPC：RestoreStock(sku_id, quantity)
   · UPDATE sku SET stock = stock + ? WHERE id = ?
③ 更新订单状态为 CANCELLED
④ 事务提交
```

## 五、实现步骤

### 第一阶段：创建独立 BFF 服务 + ProductCenter 接入 Nacos

#### 5.1 ProductCenter 先接入 Nacos 注册

**文件**：[main.go](file:///Users/daitingwei/Desktop/shpotest/productCenter/cmd/productcenter/main.go)

- `conf.proto` 新增 Registry + Nacos 配置（参照 3.5 节）
- `config.yaml` 新增 registry.nacos 字段
- `main.go` 新增 Nacos 客户端初始化 + `kratos.Registrar(r)`
- `wire.go` 新增 Registry 参数注入
- `go.mod` 新增 nacos 依赖
- `make config` 重新生成 conf.pb.go

```go
// main.go 的 newApp 函数改为：
func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server, r registry.Registrar) *kratos.App {
    return kratos.New(
        kratos.ID(id),
        kratos.Name("productCenter"),
        kratos.Version(Version),
        kratos.Metadata(map[string]string{}),
        kratos.Logger(logger),
        kratos.Server(gs, hs),
        kratos.Registrar(r),
    )
}

// wireApp 新增 registry 参数
func wireApp(*conf.Server, *conf.Data, *conf.Registry, log.Logger) (*kratos.App, func(), error)

// main 函数改动：
app, cleanup, err := wireApp(bc.Server, bc.Data, bc.Registry, logger)
```

#### 5.2 创建 BFF 服务骨架

```bash
mkdir -p bff
```

创建标准 Kratos 目录结构：
- `bff/api/bff/v1/bff.proto` + `error_reason.proto`
- `bff/internal/service/`
- `bff/internal/biz/`
- `bff/internal/data/`
- `bff/internal/conf/conf.proto`
- `bff/configs/config.yaml`
- `bff/cmd/bff/main.go` + `wire.go`
- `bff/go.mod` + `Makefile` + `.gitignore`

#### 5.3 BFF Proto 定义

**文件**：`bff/api/bff/v1/bff.proto`（从 productCenter 复制，改 go_package）

```
- 原 go_package: productCenter/api/bff/v1;v1
+ 新 go_package: bff/api/bff/v1;v1
```

**文件**：`bff/api/bff/v1/error_reason.proto`（新建）

```protobuf
syntax = "proto3";
package bff.v1;
option go_package = "bff/api/bff/v1;v1";
enum ErrorReason {
    PRODUCT_NOT_FOUND = 0;
    SHOP_NOT_FOUND = 1;
}
```

#### 5.4 BFF Service + Biz 层迁移

**文件**：`bff/internal/service/bff.go`（新建）→ 从 productCenter 复制迁移
**文件**：`bff/internal/service/service.go`（新建）→ Wire ProviderSet
**文件**：`bff/internal/biz/bff.go`（新建）→ 从 productCenter 复制迁移
**文件**：`bff/internal/biz/entity.go`（新建）→ 聚合实体
**文件**：`bff/internal/biz/biz.go`（新建）→ Wire ProviderSet

#### 5.5 BFF Data 层 — 通过 Nacos 发现调用 ProductCenter（核心变更）

**文件**：`bff/internal/data/data.go`（新建）

```go
package data

import (
    "bff/internal/biz"
    "bff/internal/conf"
    productv1 "productCenter/api/product/v1"
    shopv1 "productCenter/api/shop/v1"
    skuv1 "productCenter/api/sku/v1"
    mediav1 "productCenter/api/productmedia/v1"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/google/wire"
    kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
    "github.com/go-kratos/kratos/v2/registry"
)

var ProviderSet = wire.NewSet(NewData, NewBFFRepo)

type Data struct {
    productClient productv1.ProductClient
    shopClient    shopv1.ShopClient
    skuClient     skuv1.SkuClient
    mediaClient   mediav1.ProductMediaClient
}

func NewData(r registry.Registrar, logger log.Logger) (*Data, func(), error) {
    // 通过 Nacos 发现 ProductCenter
    conn, err := kratosgrpc.DialInsecure(
        context.Background(),
        kratosgrpc.WithEndpoint("discovery:///productCenter"),
        kratosgrpc.WithDiscovery(r),
    )
    if err != nil { return nil, nil, err }

    cleanup := func() { conn.Close() }

    return &Data{
        productClient: productv1.NewProductClient(conn),
        shopClient:    shopv1.NewShopClient(conn),
        skuClient:     skuv1.NewSkuClient(conn),
        mediaClient:   mediav1.NewProductMediaClient(conn),
    }, cleanup, nil
}
```

**文件**：`bff/internal/data/bff.go`（新建）

关键变更：**从直接查数据库改为 gRPC 调用**：

```go
func (r *bffRepo) GetProductDetail(ctx context.Context, productID int64) (*biz.ProductDetail, error) {
    // 1. gRPC 查商品
    productResp, _ := r.data.productClient.GetProduct(ctx, &productv1.GetProductRequest{Id: productID})
    // 2. gRPC 查店铺
    shopResp, _ := r.data.shopClient.GetShop(ctx, &shopv1.GetShopRequest{Id: productResp.Product.ShopId})
    // 3. gRPC 查 SKU 列表
    skuResp, _ := r.data.skuClient.ListSkus(ctx, &skuv1.ListSkusRequest{ProductId: productID, PageSize: 100})
    // 4. gRPC 查媒体列表
    mediaResp, _ := r.data.mediaClient.ListProductMedias(ctx, &mediav1.ListProductMediasRequest{ProductId: productID, PageSize: 100})
    // 5. 聚合返回（proto 转 biz 实体）
}
```

`ListProducts` 和 `GetShopHome` 同理。

#### 5.6 BFF 配置 + Nacos 注册

**文件**：`bff/internal/conf/conf.proto`（和 3.5 节统一结构）

```protobuf
message Registry {
    message Nacos {
        string address = 1;
        string namespace_id = 2;
        int64 timeout_ms = 3;
        string log_dir = 4;
        string cache_dir = 5;
    }
    Nacos nacos = 1;
}

message Bootstrap {
    Server server = 1;
    Data data = 2;
    Registry registry = 3;
}
```

**文件**：`bff/configs/config.yaml`

```yaml
server:
  http:
    addr: 0.0.0.0:8002
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9002
    timeout: 1s
registry:
  nacos:
    address: 127.0.0.1:8848
    namespace_id: public
```

**文件**：`bff/cmd/bff/main.go` — 参照 ProductCenter 的 main.go，使用 `kratos.Registrar(r)`
**文件**：`bff/cmd/bff/wire.go` — 新增 registry 参数

#### 5.7 生成 + 编译

```bash
cd bff && make api && make config && go build ./...
cd productCenter && make config && go build ./...
```

#### 5.8 移除 ProductCenter 中的 BFF 代码

- `productCenter/api/bff/v1/` → 删除整个目录
- `productCenter/internal/service/bff.go` → 删除
- `productCenter/internal/biz/bff.go` → 删除
- `productCenter/internal/data/bff.go` → 删除
- `productCenter/internal/server/grpc.go` → 移除 `bffv1.RegisterBFFServer`
- `productCenter/internal/server/http.go` → 移除 BFF handler
- `productCenter/internal/service/service.go` → 移除 `NewBFFService`
- `productCenter/internal/biz/entity.go` → 移除 BFF 聚合实体

```bash
cd productCenter && go build ./...
```

### 第二阶段：ProductCenter 新增库存 RPC

#### 5.10 sku.proto 新增 DeductStock / RestoreStock RPC

**文件**：[sku.proto](file:///Users/daitingwei/Desktop/shpotest/productCenter/api/sku/v1/sku.proto)

```protobuf
message DeductStockRequest {
  int64 id = 1;        // sku_id
  int64 quantity = 2;  // 扣减数量
}
message DeductStockResponse {
  bool success = 1;
  int64 new_stock = 2;
}

message RestoreStockRequest {
  int64 id = 1;
  int64 quantity = 2;
}
message RestoreStockResponse {
  bool success = 1;
  int64 new_stock = 2;
}

service Sku {
  // 原有 RPC...
  rpc DeductStock(DeductStockRequest) returns (DeductStockResponse);
  rpc RestoreStock(RestoreStockRequest) returns (RestoreStockResponse);
}
```

#### 5.11 sku error_reason.proto 新增错误码

**文件**：[error_reason.proto](file:///Users/daitingwei/Desktop/shpotest/productCenter/api/sku/v1/error_reason.proto)

```protobuf
SKU_STOCK_INSUFFICIENT = 5;
```

#### 5.12 Data 层实现乐观锁扣减与回补

**文件**：[sku.go](file:///Users/daitingwei/Desktop/shpotest/productCenter/internal/data/sku.go)

新增方法：
```go
func (r *skuRepo) DeductStock(ctx context.Context, id int64, quantity int) (int, error) {
    // 使用原生 SQL 执行乐观锁扣减，因为 GORM 的 Updates 不支持 SET stock = stock - ?
    result := r.data.db.WithContext(ctx).Exec(
        "UPDATE sku SET stock = stock - ? WHERE id = ? AND stock >= ?",
        quantity, id, quantity,
    )
    if result.Error != nil { return 0, result.Error }
    if result.RowsAffected == 0 { return 0, biz.ErrSkuStockInsufficient }
    
    // 查询扣减后的最新库存
    var sku biz.Sku
    if err := r.data.db.WithContext(ctx).Model(&Sku{}).Where("id = ?", id).Select("stock").Scan(&sku.Stock).Error; err != nil {
        return 0, err
    }
    return sku.Stock, nil
}

func (r *skuRepo) RestoreStock(ctx context.Context, id int64, quantity int) (int, error) {
    result := r.data.db.WithContext(ctx).Exec(
        "UPDATE sku SET stock = stock + ? WHERE id = ?",
        quantity, id,
    )
    if result.Error != nil { return 0, result.Error }
    
    var stock int
    if err := r.data.db.WithContext(ctx).Model(&Sku{}).Where("id = ?", id).Select("stock").Scan(&stock).Error; err != nil {
        return 0, err
    }
    return stock, nil
}
```

#### 5.13 Biz 层新增业务接口

**文件**：[sku.go](file:///Users/daitingwei/Desktop/shpotest/productCenter/internal/biz/sku.go)

```go
type SkuRepo interface {
    // 原有 CRUD...
    DeductStock(ctx context.Context, id int64, quantity int) (int, error)
    RestoreStock(ctx context.Context, id int64, quantity int) (int, error)
}

var ErrSkuStockInsufficient = errors.NotFound("SKU_STOCK_INSUFFICIENT", "库存不足")

func (uc *SkuUseCase) DeductStock(ctx context.Context, id int64, quantity int) (int, error) {
    if id <= 0 { return 0, ErrSkuNotFound }
    if quantity <= 0 { return 0, errors.BadRequest("SKU_QUANTITY_INVALID", "扣减数量必须大于0") }
    return uc.repo.DeductStock(ctx, id, quantity)
}

func (uc *SkuUseCase) RestoreStock(ctx context.Context, id int64, quantity int) (int, error) {
    if id <= 0 { return 0, ErrSkuNotFound }
    if quantity <= 0 { return 0, errors.BadRequest("SKU_QUANTITY_INVALID", "回补数量必须大于0") }
    return uc.repo.RestoreStock(ctx, id, quantity)
}
```

#### 5.14 Service 层实现 gRPC/HTTP 接口

**文件**：[sku.go](file:///Users/daitingwei/Desktop/shpotest/productCenter/internal/service/sku.go)

```go
func (s *SkuService) DeductStock(ctx context.Context, req *v1.DeductStockRequest) (*v1.DeductStockResponse, error) {
    newStock, err := s.uc.DeductStock(ctx, req.Id, req.Quantity)
    if err != nil { return nil, err }
    return &v1.DeductStockResponse{Success: true, NewStock: int64(newStock)}, nil
}

func (s *SkuService) RestoreStock(ctx context.Context, req *v1.RestoreStockRequest) (*v1.RestoreStockResponse, error) {
    newStock, err := s.uc.RestoreStock(ctx, req.Id, req.Quantity)
    if err != nil { return nil, err }
    return &v1.RestoreStockResponse{Success: true, NewStock: int64(newStock)}, nil
}
```

#### 5.15 生成 pb 代码

```bash
cd productCenter && make api
cd productCenter && go build ./...
```

### 第三阶段：Order 幂等性 + Redis 初始化

#### 5.16 Redis 初始化

**文件**：[data.go](file:///Users/daitingwei/Desktop/shpotest/order/internal/data/data.go)

```go
package data

import (
    "order/internal/biz"
    "order/internal/conf"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/google/wire"
    "github.com/redis/go-redis/v9"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewData, NewOrderRepo)

type Data struct {
    db    *gorm.DB
    rdb   redis.UniversalClient
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
    // 原有 MySQL 初始化...
    db, err := gorm.Open(...)
    
    // 新增 Redis 初始化
    rdb := redis.NewClient(&redis.Options{
        Addr: c.Redis.Addr,
    })
    
    cleanup := func() {
        rdb.Close()
        sqlDB.Close()
    }
    
    return &Data{db: db, rdb: rdb}, cleanup, nil
}
```

**文件**：[config.yaml](file:///Users/daitingwei/Desktop/shpotest/order/configs/config.yaml)
- Redis 配置已有，无需改动

**文件**：`order/go.mod`
- 需新增依赖：`github.com/redis/go-redis/v9`

#### 5.17 Proto 新增 request_id 字段

**文件**：[order.proto](file:///Users/daitingwei/Desktop/shpotest/order/api/order/v1/order.proto)

```protobuf
message CreateOrderRequest {
    string request_id = 1;
    int64 user_id = 2;
    int64 shop_id = 3;
    repeated OrderItem items = 4;
}
```

#### 5.18 新增错误码

**文件**：[error_reason.proto](file:///Users/daitingwei/Desktop/shpotest/order/api/order/v1/error_reason.proto)

```protobuf
DUPLICATE_REQUEST = 6;
```

#### 5.19 Data 层模型新增 RequestID + 唯一索引

**文件**：[model.go](file:///Users/daitingwei/Desktop/shpotest/order/internal/data/model.go)

```go
type Order struct {
    gorm.Model
    RequestID   string      `gorm:"column:request_id;type:varchar(64);uniqueIndex;not null"`
    OrderNo     string      `gorm:"column:order_no;type:varchar(64);uniqueIndex;not null"`
    // ... 原有字段不变
}
```

#### 5.20 Biz 实体新增 RequestID 字段

**文件**：[entity.go](file:///Users/daitingwei/Desktop/shpotest/order/internal/biz/entity.go)

```go
type Order struct {
    gorm.Model
    RequestID   string      `json:"request_id"`
    OrderNo     string      `json:"order_no"`
    // ... 原有字段
}
```

#### 5.21 Data 层实现幂等查询 + Redis 操作

**文件**：[order.go](file:///Users/daitingwei/Desktop/shpotest/order/internal/data/order.go)

新增方法：
```go
func (r *OrderRepo) GetOrderByRequestID(ctx context.Context, requestID string) (*Order, error) {
    var order Order
    err := r.data.db.WithContext(ctx).Preload("Items").Where("request_id = ?", requestID).First(&order).Error
    if err != nil { return nil, err }
    return &order, nil
}

func (r *OrderRepo) IdempotentSetNX(ctx context.Context, requestID string) (bool, error) {
    // SET order:idempotent:{request_id} 1 NX EX 86400
    ok, err := r.data.rdb.SetNX(ctx, "order:idempotent:"+requestID, "1", 24*time.Hour).Result()
    if err != nil { return false, err }
    return ok, nil
}

func (r *OrderRepo) IdempotentDel(ctx context.Context, requestID string) error {
    return r.data.rdb.Del(ctx, "order:idempotent:"+requestID).Err()
}
```

`CreateOrder` 方法设置 `RequestID` 字段。

#### 5.22 Biz 层实现幂等逻辑

**文件**：[order.go](file:///Users/daitingwei/Desktop/shpotest/order/internal/biz/order.go)

```go
type OrderRepo interface {
    // 原有接口...
    GetOrderByRequestID(ctx context.Context, requestID string) (*Order, error)
    IdempotentSetNX(ctx context.Context, requestID string) (bool, error)
    IdempotentDel(ctx context.Context, requestID string) error
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, requestID string, userID, shopID int64, items []*OrderItem) (*Order, error) {
    // 1. Redis SET NX 幂等判断
    ok, err := uc.repo.IdempotentSetNX(ctx, requestID)
    if err == nil && !ok {
        // Redis 中存在 → 查数据库返回已有订单
        return uc.repo.GetOrderByRequestID(ctx, requestID)
    }
    // 注意：Redis 报错时 err != nil，直接降级继续（不阻塞下单）
    
    // 2. 数据库查询兜底
    existing, err := uc.repo.GetOrderByRequestID(ctx, requestID)
    if err == nil && existing != nil {
        return existing, nil
    }
    
    // 3. 校验 + 计算总价
    if len(items) == 0 {
        return nil, fmt.Errorf(v1.ErrorReason_PARAMETER_ERROR.String())
    }
    var totalAmount int
    for _, item := range items {
        totalAmount += item.Price * item.Quantity
    }
    
    // 4. 创建订单
    order := &Order{
        RequestID:   requestID,
        UserID:      userID,
        ShopID:      shopID,
        TotalAmount: totalAmount,
        Status:      int(OrderStatusPending),
        PayStatus:   int(PayStatusUnpaid),
        Items:       items,
    }
    
    created, err := uc.repo.CreateOrder(ctx, order)
    if err != nil {
        // 创建失败 → 删除 Redis key，允许重试
        uc.repo.IdempotentDel(ctx, requestID)
        return nil, err
    }
    
    return created, nil
}
```

#### 5.23 Service 层传递 request_id

**文件**：[order.go](file:///Users/daitingwei/Desktop/shpotest/order/internal/service/order.go)

```go
func (s *OrderService) CreateOrder(ctx context.Context, req *v1.CreateOrderRequest) (*v1.CreateOrderResponse, error) {
    items := convertItemsToBiz(req.Items)
    order, err := s.uc.CreateOrder(ctx, req.RequestId, req.UserId, req.ShopId, items)
    // ...
}
```

#### 5.24 生成 pb 代码

```bash
cd order && make api
cd order && go build ./...
```

### 第四阶段：Order 库存扣减 + Order 接入 Nacos

#### 5.25 Order 接入 Nacos 注册

**文件**：[main.go](file:///Users/daitingwei/Desktop/shpotest/order/cmd/order/main.go)

- `conf.proto` 新增 Registry + Nacos 配置（和 ProductCenter/BFF 统一结构）
- `config.yaml` 新增 registry.nacos 字段
- `main.go` 新增 Nacos 客户端初始化 + `kratos.Registrar(r)`
- `wire.go` 新增 Registry 参数注入
- `go.mod` 新增 nacos 依赖
- `make config` 重新生成 conf.pb.go

#### 5.26 Order 引入 productCenter sku proto

**文件**：`order/go.mod`

```go
require productCenter v0.0.0
replace productCenter => ../productCenter
```

然后在代码中 `import skuv1 "productCenter/api/sku/v1"`

#### 5.27 Data 层新增 Nacos 发现 gRPC 客户端

**文件**：[data.go](file:///Users/daitingwei/Desktop/shpotest/order/internal/data/data.go)

```go
type Data struct {
    db        *gorm.DB
    rdb       redis.UniversalClient
    skuClient skuv1.SkuClient
}

func NewData(c *conf.Data, registry *conf.Registry, r registry.Registrar, logger log.Logger) (*Data, func(), error) {
    // 原有 MySQL + Redis 初始化...
    
    // 通过 Nacos 发现 ProductCenter
    conn, err := kratosgrpc.DialInsecure(
        context.Background(),
        kratosgrpc.WithEndpoint("discovery:///productCenter"),
        kratosgrpc.WithDiscovery(r),
    )
    if err != nil { return nil, nil, err }
    
    skuClient := skuv1.NewSkuClient(conn)
    
    cleanup := func() {
        rdb.Close()
        sqlDB.Close()
        conn.Close()
    }
    
    return &Data{db: db, rdb: rdb, skuClient: skuClient}, cleanup, nil
}
```

#### 5.28 Data 层实现库存扣减/回补的 gRPC 调用

**文件**：[order.go](file:///Users/daitingwei/Desktop/shpotest/order/internal/data/order.go)

```go
func (r *OrderRepo) DeductStock(ctx context.Context, skuID int64, quantity int) error {
    resp, err := r.data.skuClient.DeductStock(ctx, &skuv1.DeductStockRequest{
        Id: skuID, Quantity: int64(quantity),
    })
    if err != nil { return err }
    if !resp.Success { return biz.ErrInsufficientStock }
    return nil
}

func (r *OrderRepo) RestoreStock(ctx context.Context, skuID int64, quantity int) error {
    resp, err := r.data.skuClient.RestoreStock(ctx, &skuv1.RestoreStockRequest{
        Id: skuID, Quantity: int64(quantity),
    })
    if err != nil { return err }
    if !resp.Success { return err }
    return nil
}
```

`CreateOrder` 方法在事务中增加库存扣减：
```go
for _, item := range order.Items {
    if err := r.DeductStock(ctx, int64(item.SKUID), item.Quantity); err != nil {
        tx.Rollback()
        return nil, err
    }
}
```

`CancelOrder` 方法在事务中增加库存回补：
```go
var items []OrderItem
tx.Where("order_id = ?", order.ID).Find(&items)
for _, item := range items {
    if err := r.RestoreStock(ctx, int64(item.SKUID), item.Quantity); err != nil {
        tx.Rollback()
        return err
    }
}
```

#### 5.29 Biz 层新增库存相关接口

**文件**：[order.go](file:///Users/daitingwei/Desktop/shpotest/order/internal/biz/order.go)

```go
type OrderRepo interface {
    // 原有 + 幂等接口...
    DeductStock(ctx context.Context, skuID int64, quantity int) error
    RestoreStock(ctx context.Context, skuID int64, quantity int) error
}

var ErrInsufficientStock = errors.NotFound("INSUFFICIENT_STOCK", "库存不足")
```

#### 5.30 Wire 调整 + 编译验证

**文件**：[wire.go](file:///Users/daitingwei/Desktop/shpotest/order/cmd/order/wire.go)

```go
func wireApp(*conf.Server, *conf.Data, *conf.Registry, log.Logger) (*kratos.App, func(), error) {
    panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
```

```bash
cd order && make config && go generate ./... && go build ./...
```

## 六、涉及文件清单

### 第一阶段：创建独立 BFF 服务 + ProductCenter 接入 Nacos

| 文件 | 操作 | 说明 |
|------|------|------|
| `productCenter/internal/conf/conf.proto` | 修改 | 新增 Registry + Nacos 配置 |
| `productCenter/internal/conf/conf.pb.go` | 生成 | `make config` |
| `productCenter/configs/config.yaml` | 修改 | 新增 registry.nacos 字段 |
| `productCenter/cmd/productcenter/main.go` | 修改 | 新增 Nacos 初始化 + Registrar |
| `productCenter/cmd/productcenter/wire.go` | 修改 | 新增 Registry 参数 |
| `productCenter/go.mod` | 修改 | 新增 nacos 依赖 |
| `bff/api/bff/v1/bff.proto` | 新建 | 从 productCenter 复制，改 go_package |
| `bff/api/bff/v1/error_reason.proto` | 新建 | BFF 错误码 |
| `bff/internal/service/bff.go` | 新建 | BFF 聚合服务 |
| `bff/internal/service/service.go` | 新建 | Wire ProviderSet |
| `bff/internal/biz/bff.go` | 新建 | BFF 业务用例 |
| `bff/internal/biz/entity.go` | 新建 | 聚合实体定义 |
| `bff/internal/biz/biz.go` | 新建 | Wire ProviderSet |
| `bff/internal/data/bff.go` | 新建 | **gRPC 调用（非直接查库）** |
| `bff/internal/data/data.go` | 新建 | Data 初始化 + Nacos 发现 gRPC |
| `bff/internal/conf/conf.proto` | 新建 | Nacos 配置 |
| `bff/configs/config.yaml` | 新建 | BFF + Nacos 配置 |
| `bff/cmd/bff/main.go` | 新建 | 入口 + Nacos 注册 |
| `bff/cmd/bff/wire.go` | 新建 | Wire 注入 |
| `bff/go.mod` | 新建 | go module + nacos 依赖 |
| `bff/.gitignore` + `Makefile` | 新建 | 项目基础设施 |
| `productCenter/api/bff/v1/` | 删除 | 移除 BFF proto |
| `productCenter/internal/service/bff.go` | 删除 | |
| `productCenter/internal/biz/bff.go` | 删除 | |
| `productCenter/internal/data/bff.go` | 删除 | |
| `productCenter/internal/server/grpc.go` | 修改 | 移除 BFF 注册 |
| `productCenter/internal/server/http.go` | 修改 | 移除 BFF 注册 |
| `productCenter/internal/service/service.go` | 修改 | 移除 BFF ProviderSet |
| `productCenter/internal/biz/entity.go` | 修改 | 移除 BFF 聚合实体 |

### 第二阶段：ProductCenter 新增库存 RPC

| 文件 | 操作 | 说明 |
|------|------|------|
| `productCenter/api/sku/v1/sku.proto` | 修改 | 新增 DeductStock/RestoreStock |
| `productCenter/api/sku/v1/error_reason.proto` | 修改 | 新增 SKU_STOCK_INSUFFICIENT |
| `productCenter/internal/biz/sku.go` | 修改 | 新增接口 + 业务逻辑 |
| `productCenter/internal/data/sku.go` | 修改 | 乐观锁扣减/回补实现 |
| `productCenter/internal/service/sku.go` | 修改 | gRPC/HTTP 接口 |
| pb 生成代码 | 生成 | `make api` |

### 第三阶段：Order 幂等性 + Redis

| 文件 | 操作 | 说明 |
|------|------|------|
| `order/api/order/v1/order.proto` | 修改 | CreateOrderRequest 新增 request_id |
| `order/api/order/v1/error_reason.proto` | 修改 | 新增 DUPLICATE_REQUEST |
| `order/internal/data/model.go` | 修改 | Order 新增 RequestID + 唯一索引 |
| `order/internal/data/data.go` | 修改 | Data 新增 Redis 客户端和 ProviderSet |
| `order/internal/data/order.go` | 修改 | 新增幂等查询、Redis 操作 |
| `order/internal/biz/entity.go` | 修改 | Order 新增 RequestID |
| `order/internal/biz/order.go` | 修改 | 幂等逻辑 |
| `order/internal/service/order.go` | 修改 | 传递 request_id |
| `order/go.mod` | 修改 | 新增 go-redis 依赖 |
| pb 生成代码 | 生成 | `make api` |

### 第四阶段：Order 库存扣减 + Order 接入 Nacos

| 文件 | 操作 | 说明 |
|------|------|------|
| `order/internal/conf/conf.proto` | 修改 | 新增 Registry + Nacos 配置 |
| `order/internal/conf/conf.pb.go` | 生成 | `make config` |
| `order/configs/config.yaml` | 修改 | 新增 registry.nacos 字段 |
| `order/cmd/order/main.go` | 修改 | 新增 Nacos 初始化 + Registrar |
| `order/cmd/order/wire.go` | 修改 | 新增 Registry + Registrar 参数 |
| `order/go.mod` | 修改 | 新增 nacos + productCenter 依赖 |
| `order/internal/data/data.go` | 修改 | Data 新增 Nacos 发现 gRPC 客户端 |
| `order/internal/data/order.go` | 修改 | DeductStock/RestoreStock gRPC 调用 |
| `order/internal/biz/order.go` | 修改 | 新增库存接口 + 事务逻辑 |

## 七、实施顺序

### 提交 1/5: `feat: 三个服务接入Nacos` (productCenter + order + bff)

1. ProductCenter conf.proto/config.yaml 新增 Nacos 配置
2. ProductCenter main.go 新增 Nacos 初始化 + `kratos.Registrar(r)`
3. ProductCenter wire.go 新增 Registry 参数
4. ProductCenter go.mod 新增 nacos 依赖
5. `cd productCenter && make config && go build ./...`
6. Order conf.proto/config.yaml 新增 Nacos 配置
7. Order main.go 新增 Nacos 初始化 + `kratos.Registrar(r)`
8. Order wire.go 新增 Registry + Registrar 参数
9. Order go.mod 新增 nacos 依赖
10. `cd order && make config && go build ./...`
11. 创建 BFF 服务骨架（kratos new bff，自带 Nacos 配置 + 注册）
12. BFF conf.proto + config.yaml 配好 Nacos（用于发现 ProductCenter/Order）
13. BFF go.mod 新建后可添加 nacos 依赖
14. `cd bff && make api && make config && go build ./...`
15. Commit

### 提交 2/5: `feat: BFF聚合层迁移` (bff + productCenter)

1. 复制 bff.proto，改 go_package
2. 创建 error_reason.proto
3. 创建 service/biz 层（从 productCenter 迁移）
4. **BFF Data 层改为 Nacos 发现 gRPC 调用**
5. `make api && go build ./...`
6. 从 productCenter 移除所有 BFF 代码
7. `cd productCenter && go build ./...`
8. Commit

### 提交 3/5: `feat: SKU新增库存扣减与回补RPC` (productCenter)

1. sku.proto 新增 DeductStock/RestoreStock
2. error_reason.proto 新增 SKU_STOCK_INSUFFICIENT
3. data/sku.go 实现乐观锁扣减/回补
4. biz/sku.go 新增业务逻辑
5. service/sku.go 实现 gRPC/HTTP 接口
6. `make api`
7. `go build ./...`
8. Commit

### 提交 4/5: `feat: 订单服务实现幂等性` (order)

1. 初始化 Redis（data/data.go）
2. order.proto 新增 request_id
3. error_reason.proto 新增 DUPLICATE_REQUEST
4. data/model.go 新增 RequestID + 唯一索引
5. biz/entity.go 新增 RequestID
6. data/order.go 新增幂等查询 + Redis 操作
7. biz/order.go CreateOrder 幂等逻辑
8. service/order.go 传递 request_id
9. `make api`
10. `go build ./...`
11. Commit

### 提交 5/5: `feat: 订单服务通过Nacos发现ProductCenter扣库存` (order)

1. go.mod 引入 productCenter
2. data/data.go 新增 Nacos 发现 gRPC 客户端（Data 结构体 + NewData）
3. data/order.go 实现 DeductStock/RestoreStock
4. biz/order.go 新增库存接口 + ErrInsufficientStock
5. CreateOrder 事务中扣库存，CancelOrder 事务中回补
6. `go mod tidy && go build ./...`
7. Commit

## 八、验证方式

0. **Nacos 验证**：三个服务启动后，在 Nacos 控制台都能看到对应的服务实例
1. **BFF 迁移验证**：
   - BFF 独立启动后，`GetProductDetail/ListProducts/GetShopHome` 返回正确
   - ProductCenter 移除 BFF 后正常启动
2. **`make api` + `make config`**：proto 编译通过
3. **`go build ./...`**：代码编译通过（bff / order / productCenter 三个目录）
4. **幂等性验证**：
   - BFF 自动生成 request_id 调两次 CreateOrder，返回同一个订单
   - 不同请求正常创建不同订单
   - Redis 异常时自动降级不影响下单
5. **库存验证**：
   - 正常下单 → 库存扣减
   - 库存不足 → 返回错误，订单不创建
   - 取消订单 → 库存回补
