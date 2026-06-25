# 订单服务实现 + 现有模型 gorm.Model 改造计划

## 一、需求概述

### 1.1 新建订单服务（orderCenter）
创建基于 Kratos 框架的订单微服务，包含订单主表和订单项表。

### 1.2 改造现有项目（productCenter）
将 productCenter 中所有模型改为使用 `gorm.Model` 嵌入方式。

### 核心功能
| 功能 | 描述 |
|------|------|
| 创建订单 | 用户提交订单，包含多个订单项 |
| 查询订单详情 | 获取订单完整信息（含订单项） |
| 查询订单列表 | 分页查询用户订单 |
| 更新订单状态 | 支付、发货、确认收货等状态变更 |
| 取消订单 | 取消待支付订单 |

---

## 二、技术方案

### 2.1 gorm.Model 说明

`gorm.Model` 是 GORM 提供的嵌入结构体，包含：
```go
type Model struct {
    ID        uint           `gorm:"primaryKey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt DeletedAt `gorm:"index"`
}
```

使用方式：
```go
type Shop struct {
    gorm.Model           // 嵌入后自动获得 ID, CreatedAt, UpdatedAt, DeletedAt
    ShopName    string   `gorm:"column:shop_name"`
    Description string   `gorm:"column:description"`
}
```

---

## 三、实现步骤与 Git Commit

### 阶段1：改造 productCenter 模型（使用 gorm.Model）

#### 步骤1.1：修改 productCenter Model（使用 gorm.Model）
**文件**: `productCenter/internal/data/model.go`

将所有模型改为嵌入 `gorm.Model`：
- Shop
- ProductTag
- Product
- ProductMedia
- Sku

**Commit**: `refactor: productCenter model 使用 gorm.Model`

#### 步骤1.2：修改 productCenter Entity（使用 gorm.Model）
**文件**: `productCenter/internal/biz/entity.go`

**Commit**: `refactor: productCenter entity 使用 gorm.Model`

#### 步骤1.3：编译验证
```bash
cd productCenter
go build ./...
```

**Commit**: `chore: 验证 productCenter gorm.Model 改造`

---

### 阶段2：新建订单服务（orderCenter）

#### 步骤2.1：生成 Proto 代码
```bash
cd ordercenter
make api
```

**Commit**: `feat: 生成订单服务 proto 代码`

#### 步骤2.2：创建 GORM 模型（使用 gorm.Model）
**文件**: `ordercenter/internal/data/model.go`

```go
type Order struct {
    gorm.Model
    OrderNo     string      `gorm:"column:order_no;unique;not null"`
    UserID      int64       `gorm:"column:user_id;not null"`
    ShopID      int64       `gorm:"column:shop_id;not null"`
    TotalAmount int         `gorm:"column:total_amount;not null"`
    Status      int         `gorm:"column:status;not null;default:0"`
    PayStatus   int         `gorm:"column:pay_status;not null;default:0"`
    PayTime     *time.Time  `gorm:"column:pay_time"`
    ShipTime    *time.Time  `gorm:"column:ship_time"`
    ConfirmTime *time.Time  `gorm:"column:confirm_time"`
    Items       []OrderItem `gorm:"foreignKey:OrderID"`
}

type OrderItem struct {
    gorm.Model
    OrderID     int64  `gorm:"column:order_id;not null;index"`
    ProductID   int64  `gorm:"column:product_id;not null"`
    SKUID       int64  `gorm:"column:sku_id;not null"`
    ProductName string `gorm:"column:product_name;not null"`
    SKUTitle    string `gorm:"column:sku_title"`
    Price       int    `gorm:"column:price;not null"`
    Quantity    int    `gorm:"column:quantity;not null"`
    ImageURL    string `gorm:"column:image_url"`
}
```

**Commit**: `feat: 创建订单服务 GORM 模型`

#### 步骤2.3：创建业务实体（使用 gorm.Model）
**文件**: `ordercenter/internal/biz/entity.go`

**Commit**: `feat: 创建订单服务业务实体`

#### 步骤2.4：创建 Data 层
**文件**: `ordercenter/internal/data/data.go` - DB连接和 AutoMigrate
**文件**: `ordercenter/internal/data/order.go` - OrderRepo 实现

**Commit**: `feat: 创建订单服务 Data 层`

#### 步骤2.5：创建 Biz 层
**文件**: `ordercenter/internal/biz/order.go` - OrderRepo 接口和 OrderUseCase

**Commit**: `feat: 创建订单服务 Biz 层`

#### 步骤2.6：创建 Service 层
**文件**: `ordercenter/internal/service/order.go` - OrderService 实现

**Commit**: `feat: 创建订单服务 Service 层`

#### 步骤2.7：配置 Wire 依赖注入
**文件**: `ordercenter/cmd/ordercenter/wire.go`

**Commit**: `feat: 配置订单服务 Wire 依赖注入`

#### 步骤2.8：配置服务器
**文件**: `ordercenter/internal/server/grpc.go`
**文件**: `ordercenter/internal/server/http.go`

**Commit**: `feat: 配置订单服务 gRPC/HTTP 服务器`

#### 步骤2.9：创建配置文件和入口
**文件**: `ordercenter/configs/config.yaml`
**文件**: `ordercenter/cmd/ordercenter/main.go`

**Commit**: `feat: 创建订单服务配置和入口文件`

#### 步骤2.10：代码生成和编译验证
```bash
cd ordercenter
make api
make generate
go build ./...
```

**Commit**: `chore: 订单服务代码生成和编译验证`

---

## 四、数据库设计

### 4.1 订单服务表结构

#### 表 orders（订单主表）

| 字段名 | 类型 | 说明 | 来源 |
|--------|------|------|------|
| `id` | BIGINT | 主键 | gorm.Model |
| `order_no` | VARCHAR(64) | 订单编号 | 自定义 |
| `user_id` | BIGINT | 用户ID | 自定义 |
| `shop_id` | BIGINT | 店铺ID | 自定义 |
| `total_amount` | INT | 订单总金额（分） | 自定义 |
| `status` | TINYINT | 订单状态 | 自定义 |
| `pay_status` | TINYINT | 支付状态 | 自定义 |
| `pay_time` | DATETIME | 支付时间 | 自定义 |
| `ship_time` | DATETIME | 发货时间 | 自定义 |
| `confirm_time` | DATETIME | 确认收货时间 | 自定义 |
| `created_at` | DATETIME | 创建时间 | gorm.Model |
| `updated_at` | DATETIME | 更新时间 | gorm.Model |
| `deleted_at` | DATETIME | 删除时间 | gorm.Model |

#### 表 order_items（订单项表）

| 字段名 | 类型 | 说明 | 来源 |
|--------|------|------|------|
| `id` | BIGINT | 主键 | gorm.Model |
| `order_id` | BIGINT | 订单ID | 自定义 |
| `product_id` | BIGINT | 商品ID | 自定义 |
| `sku_id` | BIGINT | SKU ID | 自定义 |
| `product_name` | VARCHAR(255) | 商品名称 | 自定义 |
| `sku_title` | VARCHAR(255) | SKU标题 | 自定义 |
| `price` | INT | 单价（分） | 自定义 |
| `quantity` | INT | 数量 | 自定义 |
| `image_url` | VARCHAR(512) | 商品图片 | 自定义 |
| `created_at` | DATETIME | 创建时间 | gorm.Model |
| `updated_at` | DATETIME | 更新时间 | gorm.Model |
| `deleted_at` | DATETIME | 删除时间 | gorm.Model |

#### 状态枚举

**订单状态 (status)**:
- `0` - 待支付
- `1` - 已支付
- `2` - 已发货
- `3` - 已完成
- `4` - 已取消

**支付状态 (pay_status)**:
- `0` - 未支付
- `1` - 已支付
- `2` - 支付失败
- `3` - 退款中
- `4` - 已退款

---

## 五、API接口设计

| RPC方法 | HTTP方法 | 路径 | 描述 |
|---------|----------|------|------|
| CreateOrder | POST | `/api/v1/orders` | 创建订单 |
| GetOrder | GET | `/api/v1/orders/{order_id}` | 查询订单详情 |
| ListOrders | GET | `/api/v1/orders` | 查询订单列表 |
| UpdateOrderStatus | PUT | `/api/v1/orders/{order_id}/status` | 更新订单状态 |
| CancelOrder | DELETE | `/api/v1/orders/{order_id}` | 取消订单 |

---

## 六、错误码定义

| 错误码 | 枚举值 | 说明 |
|--------|--------|------|
| ORDER_NOT_FOUND | 0 | 订单不存在 |
| ORDER_STATUS_INVALID | 1 | 订单状态无效 |
| ORDER_CANCEL_FAILED | 2 | 取消订单失败 |
| ORDER_CREATE_FAILED | 3 | 创建订单失败 |
| INSUFFICIENT_STOCK | 4 | 库存不足 |
| PARAMETER_ERROR | 5 | 参数错误 |

---

## 七、文件清单

### 7.1 productCenter 改造文件

| 文件路径 | 说明 | 状态 |
|----------|------|------|
| `productCenter/internal/data/model.go` | GORM模型改造 | 待修改 |
| `productCenter/internal/biz/entity.go` | 业务实体改造 | 待修改 |

### 7.2 orderCenter 新建文件

| 文件路径 | 说明 | 状态 |
|----------|------|------|
| `ordercenter/api/order/v1/order.proto` | Proto接口定义 | ✅ 已创建 |
| `ordercenter/api/order/v1/error_reason.proto` | 错误码定义 | ✅ 已创建 |
| `ordercenter/internal/data/model.go` | GORM模型 | 待创建 |
| `ordercenter/internal/biz/entity.go` | 业务实体 | 待创建 |
| `ordercenter/internal/data/data.go` | DB连接 | 待创建 |
| `ordercenter/internal/data/order.go` | OrderRepo实现 | 待创建 |
| `ordercenter/internal/biz/order.go` | OrderUseCase | 待创建 |
| `ordercenter/internal/service/order.go` | OrderService | 待创建 |
| `ordercenter/internal/server/grpc.go` | gRPC注册 | 待创建 |
| `ordercenter/internal/server/http.go` | HTTP注册 | 待创建 |
| `ordercenter/cmd/ordercenter/main.go` | 入口文件 | 待创建 |
| `ordercenter/cmd/ordercenter/wire.go` | Wire配置 | 待创建 |
| `ordercenter/configs/config.yaml` | 配置文件 | 待创建 |

---

## 八、验证计划

### 8.1 productCenter 验证
```bash
cd productCenter
go build ./...
```

### 8.2 orderCenter 验证
```bash
cd ordercenter
make api
make generate
go build ./...
go run ./cmd/ordercenter/ -conf ./configs
```

---

## 九、风险与注意事项

| 风险点 | 描述 | 处理方式 |
|--------|------|----------|
| ID类型变更 | uint vs int64 | 需要检查所有使用ID的地方 |
| 数据库迁移 | gorm.Model 使用 uint ID | 需确保数据库表字段类型兼容 |
| 事务一致性 | 创建订单时需同时创建订单项 | 使用 DB 事务 |
| 订单号唯一性 | 订单号需全局唯一 | 使用时间戳+随机数生成 |