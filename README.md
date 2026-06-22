# 商品中心（Product Center）

基于 Go + Kratos 框架实现的极简商品管理后端，支持店铺、商品、标签的完整 CRUD。

## 项目结构

```
shpotest/
├── procenter/              # Kratos 微服务版（主力版本）
│   ├── api/                # Proto 定义 + 生成代码
│   │   ├── shop/v1/        # 店铺接口
│   │   ├── product/v1/     # 商品接口
│   │   ├── producttag/v1/  # 标签接口
│   │   └── bff/v1/         # BFF 聚合接口（商品详情 + 店铺首页）
│   ├── cmd/procenter/      # 程序入口 + Wire 依赖注入
│   ├── configs/            # 配置文件（YAML）
│   └── internal/
│       ├── biz/            # 业务层：实体 + UseCase + Repo 接口
│       ├── data/           # 数据层：GORM 模型 + Repo 实现
│       ├── service/        # 服务层：Proto → Biz 适配
│       └── server/         # gRPC + HTTP 服务器
├── schema.sql              # 数据库建表 DDL
└── 库表设计.md              # 数据库设计文档
```

## 技术栈

| 领域 | 选型 |
|---|---|
| 语言 | Go 1.22 |
| 框架 | [Kratos](https://go-kratos.dev/) v2 |
| RPC/API | Protobuf + gRPC + HTTP（基于 proto annotation 自动生成） |
| ORM | GORM |
| 数据库 | MySQL 8.0+ |
| 依赖注入 | Google Wire |

## 核心功能

### 1. 店铺管理（Shops）
- 创建店铺
- 查询店铺详情
- 分页查询店铺列表
- 更新店铺信息
- 软删除店铺

### 2. 商品管理（Products）
- 创建商品（校验所属店铺必须存在）
- 查询商品详情（含店铺名称）
- 分页查询商品列表（含店铺名称，支持按店铺/状态筛选）
- 更新商品信息（支持部分字段更新）
- 软删除商品

### 3. 商品标签（Product Tags）
- 创建标签
- 标签列表
- 商品 ↔ 标签（多对多关联，通过中间表 `product_tag_mapping`）

### 4. BFF 聚合层
- 商品详情页（聚合商品信息 + 店铺名 + 标签 + SKU + 副图）
- 商品列表页（聚合商品 + 店铺名 + 标签）
- 店铺首页（聚合店铺信息 + 商品列表）

## 数据库设计

### 表结构概览

| 表名 | 说明 | 关键字段 |
|---|---|---|
| `shops` | 店铺表 | id, shop_name, description, created_at, updated_at, deleted_at |
| `product_tag` | 商品标签表 | id, name, sort, created_at, updated_at, deleted_at |
| `products` | 商品表 | id, shop_id, name, description, main_image_url, price, compare_at_price, status, sort, created_at, updated_at, deleted_at |
| `product_tag_mapping` | 商品-标签关联表（多对多） | product_id, tag_id |
| `product_media` | 商品副图表 | id, product_id, url, sort, created_at, updated_at, deleted_at |
| `sku` | 商品 SKU 表 | id, product_id, sku, title, price, stock, img_url, created_at, updated_at, deleted_at |

### ER 关系

```
shops ──┬──1:N── products
        │          ├──1:N── product_media
        │          ├──1:N── sku
        │          └──N:M── product_tag_mapping ── product_tag
        │
product_tag ──┘
```

### 设计要点

1. **价格用 int 存分**：避免浮点精度丢失（DECIMAL 虽好但 ORM 开销大，int 存分更简单直接）
2. **软删除**：`deleted_at` 字段，误删可恢复，有回收站概念
3. **商品状态**：`0=草稿` / `1=上架` / `2=下架`（tinyint，省空间）
4. **联合索引**：`products(shop_id, status)` 支持按店铺 + 状态快速筛选
5. **逻辑外键**：不在 DB 层建物理外键，由应用层保证数据一致性（更灵活，方便 shard）

详见 [库表设计.md](./库表设计.md) 和 [schema.sql](./schema.sql)。

## 分层架构

```
┌─────────────────────────────────────────────┐
│                  Client                     │
│    (HTTP / gRPC / curl / Postman)           │
└──────────────────┬──────────────────────────┘
                   │ HTTP / gRPC 请求
                   ▼
┌─────────────────────────────────────────────┐
│               internal/server               │
│   ├── NewGRPCServer  (gRPC 路由注册)        │
│   └── NewHTTPServer (HTTP 路由注册)         │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│              internal/service               │
│   ShopService      解析 proto 请求          │
│   ProductService   参数校验                 │
│   ProductTagService 拼装 proto 响应         │
│   BFFService       调用 biz UseCase         │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│                 internal/biz                │
│   ├── ShopUseCase        (店铺业务逻辑)     │
│   ├── ProductUseCase     (商品业务逻辑)     │
│   ├── ProductTagUseCase  (标签业务逻辑)     │
│   ├── BFFUseCase         (数据聚合)         │
│   ├── entity.go          (业务实体定义)     │
│   └── *Repo              (接口定义, 而非实现)│
└──────────────────┬──────────────────────────┘
                   │ 依赖倒置（面向接口编程）
                   ▼
┌─────────────────────────────────────────────┐
│                internal/data                │
│   ├── shop.go          (ShopRepo 实现)      │
│   ├── product.go       (ProductRepo 实现)   │
│   ├── product_tag.go   (ProductTagRepo 实现)│
│   ├── bff.go           (BFFRepo 实现)       │
│   ├── product_media.go (ProductMediaRepo)   │
│   ├── sku.go           (SkuRepo 实现)       │
│   ├── model.go         (GORM 模型)         │
│   └── data.go          (DB 连接 + AutoMigrate)│
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│                    MySQL                    │
└─────────────────────────────────────────────┘
```

**分层的价值**：
- `biz` 层定义接口，`data` 层负责实现 → 换数据库只需改 `data`，`biz` 不动
- `service` 层只做协议转换 → 业务逻辑集中在 `biz`，便于测试
- Wire 自动生成依赖注入代码 → 无需手动层层 New

## HTTP API

所有接口均支持 HTTP（JSON）。路由由 proto annotation 自动生成。

### 店铺相关

| Method | Path | 说明 |
|---|---|---|
| POST | `/api/v1/shops` | 创建店铺 |
| GET | `/api/v1/shops/{id}` | 获取店铺详情 |
| GET | `/api/v1/shops` | 店铺列表 |
| PUT | `/api/v1/shops/{id}` | 更新店铺 |
| DELETE | `/api/v1/shops/{id}` | 删除店铺 |

### 商品相关

| Method | Path | 说明 |
|---|---|---|
| POST | `/api/v1/products` | 创建商品 |
| GET | `/api/v1/products/{id}` | 获取商品详情（含店铺名称） |
| GET | `/api/v1/products` | 商品列表（支持 `page`, `page_size`, `shop_id`, `status` 查询参数） |
| PUT | `/api/v1/products/{id}` | 更新商品 |
| DELETE | `/api/v1/products/{id}` | 删除商品 |

### 标签相关

| Method | Path | 说明 |
|---|---|---|
| POST | `/api/v1/product-tags` | 创建标签 |
| GET | `/api/v1/product-tags/{id}` | 获取标签详情 |
| GET | `/api/v1/product-tags` | 标签列表 |
| PUT | `/api/v1/product-tags/{id}` | 更新标签 |
| DELETE | `/api/v1/product-tags/{id}` | 删除标签 |

### BFF 聚合接口

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/v1/bff/products/{id}` | 商品详情页（商品 + 店铺 + 标签 + SKU + 副图） |
| GET | `/api/v1/bff/products` | 商品列表页（商品 + 店铺名 + 标签） |
| GET | `/api/v1/bff/shops/{id}` | 店铺首页（店铺信息 + 商品列表） |

## 快速开始

### 前置条件

- Go 1.22+
- MySQL 8.0+ （端口 3309，用户名 root，密码见配置）

### 步骤 1：创建数据库

```bash
mysql -h 127.0.0.1 -P 3309 -u root -p -e "CREATE DATABASE IF NOT EXISTS shpotest DEFAULT CHARSET utf8mb4;"
```

### 步骤 2：编译并运行

```bash
cd procenter
go build -o ./bin/server ./cmd/procenter/
./bin/server -conf ./configs
```

服务启动后会自动：
1. 连接 MySQL（`127.0.0.1:3309/shpotest`）
2. 自动执行 `AutoMigrate` 创建/更新所有数据表
3. 启动 HTTP 服务在 `:8000`，gRPC 服务在 `:9000`

### 步骤 3：测试接口

```bash
# 创建店铺
curl -X POST http://127.0.0.1:8000/api/v1/shops \
  -H "Content-Type: application/json" \
  -d '{"shop_name":"测试旗舰店","description":"这是一个测试店铺"}'

# 创建商品（需要先有店铺）
curl -X POST http://127.0.0.1:8000/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{
    "shop_id": 1,
    "name": "iPhone 15 Pro",
    "description": "苹果旗舰手机",
    "main_image_url": "https://example.com/iphone.jpg",
    "price": 799900,
    "compare_at_price": 899900,
    "status": 1,
    "sort": 10
  }'

# 查询商品列表
curl "http://127.0.0.1:8000/api/v1/products?page=1&page_size=10&shop_id=1&status=1"

# BFF 查询商品详情页
curl "http://127.0.0.1:8000/api/v1/bff/products/1"
```

## 配置文件

`procenter/configs/config.yaml`

```yaml
server:
  http:
    addr: 0.0.0.0:8000
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 1s
data:
  database:
    driver: mysql
    source: root:dtw258989971@tcp(127.0.0.1:3309)/shpotest?parseTime=True&loc=Local
```

修改配置文件后重启服务即可，无需改代码。

## Proto 代码生成（可选）

如果修改了 `api/**/*.proto`，需要重新生成代码：

```bash
# 安装 kratos 工具
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest

# 在项目根目录执行
make api
```

## Wire 依赖注入生成（可选）

```bash
# 安装 wire
go install github.com/google/wire/cmd/wire@latest

# 生成依赖注入代码
cd cmd/procenter && wire
```

## 开发时常用命令

```bash
# 编译所有包
go build ./...

# 运行测试
go test ./...

# 直接运行（开发时推荐，无需编译）
cd procenter && go run ./cmd/procenter/ -conf ./configs
```

## 关键设计决策记录

### 1. 为什么价格用 int 存分而不是 DECIMAL？
- 避免浮点精度丢失（0.1 在二进制中是无限循环小数）
- int 比较简单，GORM 映射直接，无性能开销
- 缺点：小数除法需注意舍入策略，但电商场景足够

### 2. 为什么商品状态用 tinyint 而不是 VARCHAR？
- tinyint 只有 1 字节，比 VARCHAR 省空间
- 索引效率更高（整型比较远快于字符串）
- 缺点：语义不直观，需要记住数字含义（0=草稿, 1=上架, 2=下架）

### 3. 为什么用软删除而不是硬删除？
- 有回收站概念，误删可恢复
- 方便审计追溯
- 关联数据保留上下文

### 4. 为什么 biz 层只定义 Repo 接口，由 data 层实现？
- **依赖倒置**：业务层不依赖具体实现，换数据库/加缓存只需改 data 层
- **可测试性**：单元测试时可以 mock repo，不需要连真实数据库
- **解耦**：业务逻辑和存储技术选型独立演进

### 5. 为什么需要 BFF 层？
- 前端一个页面可能需要多张表的数据（商品详情 = 商品 + 店铺 + 标签 + SKU + 副图）
- 由后端聚合返回，减少前端网络请求
- 如果某个接口只在前端某页使用，可以独立放在 BFF，不污染核心业务接口
