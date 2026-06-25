# ShopTest 微服务电商系统

基于 Go + Kratos v2 + Nacos 的微服务电商系统，包含商品中心、订单服务、BFF 聚合层三个独立服务。

## 项目结构

```
shpotest/
├── productCenter/          # 商品中心服务（端口 gRPC:9000 HTTP:8000）
│   ├── api/                # Proto 定义 + 生成代码
│   │   ├── product/v1/     # 商品接口
│   │   ├── shop/v1/        # 店铺接口
│   │   ├── sku/v1/         # SKU + 库存扣减接口
│   │   ├── producttag/v1/  # 标签接口
│   │   └── productmedia/v1/# 商品副图接口
│   ├── cmd/productcenter/  # 程序入口 + Wire
│   ├── configs/            # 配置文件
│   └── internal/
│       ├── biz/            # 业务层：实体 + UseCase + Repo 接口
│       ├── data/           # 数据层：GORM 模型 + Repo 实现
│       ├── service/        # 服务层：Proto → Biz 适配
│       └── server/         # gRPC + HTTP 服务器
│
├── order/                  # 订单服务（端口 gRPC:9001 HTTP:8001）
│   ├── api/order/v1/       # 订单接口
│   ├── cmd/order/          # 程序入口 + Wire
│   ├── configs/            # 配置文件
│   └── internal/
│       ├── biz/            # 业务层：幂等创建、状态流转、实体定义
│       ├── data/           # 数据层：MySQL + Redis + gRPC client
│       ├── service/        # 服务层：Proto → Biz 适配
│       └── server/         # gRPC + HTTP 服务器
│
├── bff/                    # BFF 聚合层（端口 gRPC:9002 HTTP:8002）
│   ├── api/bff/v1/         # BFF 聚合接口
│   ├── cmd/bff/            # 程序入口 + Wire
│   ├── configs/            # 配置文件
│   └── internal/
│       ├── biz/            # 业务层：请求聚合 + request_id 生成
│       ├── data/           # 数据层：gRPC client 连接管理
│       ├── service/        # 服务层：Proto → Biz 适配
│       └── server/         # gRPC + HTTP 服务器
│
├── schema.sql              # 数据库建表 DDL
├── 库表设计.md              # 数据库设计文档
└── docker-compose.yml      # MySQL + Redis + Nacos 开发环境
```

## 技术栈

| 领域 | 选型 |
|---|---|
| 语言 | Go 1.22 |
| 框架 | [Kratos](https://go-kratos.dev/) v2 |
| RPC/API | Protobuf + gRPC + HTTP（proto annotation 自动生成） |
| ORM | GORM |
| 数据库 | MySQL 8.0+ |
| 缓存 | Redis |
| 服务注册/发现 | Nacos |
| 依赖注入 | Google Wire |
| 幂等性 | Redis SET NX + DB 唯一索引（request_id） |
| 乐观锁 | MySQL 版本号（version 字段） |

## 服务概览

| 服务 | gRPC | HTTP | 职责 |
|---|---|---|---|
| productCenter | 9000 | 8000 | 店铺、商品、标签、SKU、副图 CRUD + 库存扣减/回补 |
| order | 9001 | 8001 | 订单创建（幂等 + 锁库存）、查询、状态管理、取消 |
| bff | 9002 | 8002 | 面向前端的聚合层：商品详情、订单创建（自动生成 request_id） |

## 核心功能

### 1. 商品中心（productCenter）

- 店铺/商品/标签/副图 CRUD
- SKU 管理 + 库存扣减（版本号乐观锁）
- 库存回补

### 2. 订单服务（order）

- 创建订单：三级幂等保护（Redis SET NX → DB 查询 → request_id 唯一索引）+ 锁定库存
- 取消订单：回补库存
- 订单查询、列表、状态流转

### 3. BFF 聚合层（bff）

- 商品详情页（聚合商品 + 店铺 + 标签 + SKU + 副图）
- 商品列表页（商品 + 店铺名 + 标签）
- 店铺首页（店铺信息 + 商品列表）
- 创建订单（自动生成 request_id，前端无感知幂等）

## 订单创建完整链路

```
前端 → POST /api/v1/bff/orders {user_id, shop_id, items}
  │
  ├─ BFF Biz: requestID = uuid.NewString()
  ├─ BFF Repo: gRPC → Order.CreateOrder(requestID, ...)
  │     │
  │     ├─ Redis SET NX + DB 查询 → 幂等判断
  │     │
  │     ├─ for each item:
  │     │   ├─ GetSku(skuID) → 获取当前 version
  │     │   └─ DeductStock(skuID, quantity, version)
  │     │       ├─ SQL: UPDATE sku SET stock=stock-?,
  │     │       │         version=version+1
  │     │       │         WHERE id=? AND version=? AND stock>=?
  │     │       ├─ 版本匹配 → 扣减成功
  │     │       └─ 版本冲突 → 重试（最多3次）
  │     │
  │     └─ 创建订单 + 订单项
  │
  └─ 返回 {order_id, order_no}
```

## 数据库设计

### 表结构概览

| 表名 | 所属服务 | 说明 |
|---|---|---|
| `shops` | productCenter | 店铺表 |
| `products` | productCenter | 商品表 |
| `product_tag` | productCenter | 商品标签表 |
| `product_tag_mapping` | productCenter | 商品-标签关联表 |
| `product_media` | productCenter | 商品副图表 |
| `sku` | productCenter | SKU 表（含 version 乐观锁字段） |
| `orders` | order | 订单表（含 request_id 唯一索引） |
| `order_items` | order | 订单项表 |

### SKU 版本号乐观锁

```sql
-- sku 表
CREATE TABLE sku (
    ...
    stock   INT NOT NULL DEFAULT 0,
    version INT NOT NULL DEFAULT 0,  -- 乐观锁版本号
    ...
);

-- 扣减 SQL（核心保护）
UPDATE sku
SET stock = stock - ?, version = version + 1
WHERE id = ? AND version = ? AND stock >= ?;
```

`WHERE version = ?` 保证并发安全：读到的版本号和更新时不一致则失败重试。

### 订单幂等

```sql
-- orders 表
CREATE TABLE orders (
    ...
    request_id VARCHAR(64) NOT NULL,
    UNIQUE INDEX idx_request_id (request_id),  -- 幂等兜底
    ...
);
```

三层保护：Redis SET NX（快速判断）→ DB 查询（兜底）→ 唯一索引（最终防线）。

## HTTP API

### BFF 聚合接口

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/v1/bff/products/{id}` | 商品详情页（商品 + 店铺 + 标签 + SKU + 副图） |
| GET | `/api/v1/bff/products` | 商品列表页（商品 + 店铺名 + 标签） |
| GET | `/api/v1/bff/shops/{id}` | 店铺首页（店铺信息 + 商品列表） |
| POST | `/api/v1/bff/orders` | 创建订单 |

### 订单相关

| Method | Path | 说明 |
|---|---|---|
| POST | `/api/v1/orders` | 创建订单 |
| GET | `/api/v1/orders/{order_id}` | 获取订单详情 |
| GET | `/api/v1/orders` | 订单列表 |
| PUT | `/api/v1/orders/{order_id}/status` | 更新订单状态 |
| DELETE | `/api/v1/orders/{order_id}` | 取消订单 |

### 商品中心相关

productCenter 的店铺、商品、标签、SKU、副图接口见各 proto 文件定义。

## 快速开始

### 前置条件

- Go 1.22+
- Docker & Docker Compose
- MySQL 8.0+（端口 3309）、Redis（端口 6379）、Nacos（端口 8848）

### 步骤 1：启动基础设施

```bash
docker-compose up -d
```

### 步骤 2：初始化数据库

```bash
mysql -h 127.0.0.1 -P 3309 -u root -p -e "CREATE DATABASE IF NOT EXISTS shpotest DEFAULT CHARSET utf8mb4;"
mysql -h 127.0.0.1 -P 3309 -u root -p shpotest < schema.sql
```

### 步骤 3：启动服务

```bash
# 终端 1：商品中心
cd productCenter && go run ./cmd/productcenter/ -conf ./configs

# 终端 2：订单服务
cd order && go run ./cmd/order/ -conf ./configs

# 终端 3：BFF 聚合层
cd bff && go run ./cmd/bff/ -conf ./configs
```

服务启动后会自动注册到 Nacos（`127.0.0.1:8848`），BFF 通过 Nacos 发现 productCenter 和 order。

### 步骤 4：测试接口

```bash
# 创建店铺
curl -X POST http://127.0.0.1:8000/api/v1/shops \
  -H "Content-Type: application/json" \
  -d '{"shop_name":"测试旗舰店","description":"这是一个测试店铺"}'

# 创建商品
curl -X POST http://127.0.0.1:8000/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{"shop_id":1,"name":"iPhone 15 Pro","main_image_url":"https://example.com/iphone.jpg","price":799900,"status":1}'

# 创建订单（通过 BFF）
curl -X POST http://127.0.0.1:8002/api/v1/bff/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"shop_id":1,"items":[{"product_id":1,"sku_id":1,"product_name":"iPhone 15 Pro","price":799900,"quantity":1}]}'
```

## 分层架构

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  BFF Service │  │ Order Service│  │ProductCenter │
│   (9002)     │  │   (9001)     │  │   (9000)     │
└───┬────┬─────┘  └───┬────┬─────┘  └───┬────┬─────┘
    │    │             │    │             │    │
    │    └──gRPC──────→│    │             │    │
    │                  │    └──gRPC──────→│    │
    │                  │                  │    │
    ▼                  ▼                  ▼    ▼
┌───────┐         ┌──────────┐       ┌──────────┐
│ Nacos │         │ Redis+DB │       │   MySQL  │
└───────┘         └──────────┘       └──────────┘
```

**分层的价值**：

- `biz` 层定义接口，`data` 层负责实现 → 换数据库只需改 `data`，`biz` 不动
- `service` 层只做协议转换 → 业务逻辑集中在 `biz`，便于测试
- Wire 自动生成依赖注入代码 → 无需手动层层 New
- 跨服务 gRPC 调用在 data 层按需创建轻量 client → 连接在 `Data` 结构体启动时建好

## Proto 代码生成

如果修改了 `api/**/*.proto`，需要重新生成代码：

```bash
# productCenter
cd productCenter && make api

# order
cd order && make api

# bff
cd bff && make api
```

## 关键设计决策记录

### 1. 为什么用版本号乐观锁而不是 SELECT FOR UPDATE？

- 乐观锁无锁等待，并发性能高
- 库存扣减冲突率低，适合乐观锁
- MySQL 默认隔离级别 REPEATABLE-READ 下，乐观锁更可控

### 2. 为什么订单需要三级幂等保护？

- Redis SET NX：快速判断，异常降级不影响下单
- DB 查询：Redis 不可用时的兜底
- 唯一索引：最终的原子性保证（request_id UNIQUE）

### 3. 为什么 BFF 自动生成 request_id？

- 前端无感知，不需要管理幂等 Key
- 降低前端接入复杂度
- 可后续升级为前端传 `X-Idempotency-Key` header

### 4. 为什么 gRPC 连接在 Data 结构体启动时建好？

- 避免每次请求建连接的 TCP 握手开销
- 与 DB 连接、Redis 连接的生命周期一致
- Kratos discovery resolver 内置 watcher，下游实例变化自动更新

### 5. 为什么价格用 int 存分？

- 避免浮点精度丢失
- 整型索引效率高于 DECIMAL
- 电商场景足够
