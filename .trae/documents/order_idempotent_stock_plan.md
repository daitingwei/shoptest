# 订单服务幂等性与库存扣减实现计划

## 一、需求概述

### 1.1 背景
当前订单服务存在两个关键问题：
1. **幂等性缺失**：重复请求可能生成多个相同订单（如 DSERS 下单重试场景）
2. **库存扣减缺失**：创建订单时不扣减 SKU 库存，可能导致超卖

### 1.2 目标
- 实现订单创建的幂等性保证，同一请求不会生成重复订单
- 实现下单时的库存扣减，使用乐观锁防止超卖
- 不涉及金额交易相关逻辑

---

## 二、当前进度

### 2.1 已完成（4个提交）

| 序号 | 提交哈希 | 提交信息 | 状态 |
|------|----------|----------|------|
| 1 | e4a74de | feat: 订单创建接口添加 request_id 幂等字段 | ✅ 已完成 |
| 2 | e8cbbdb | chore: 生成订单幂等字段 pb 代码 | ✅ 已完成 |
| 3 | 97e0688 | feat: 订单模型添加 request_id 幂等字段 | ✅ 已完成 |
| 4 | 46f988b | feat: 添加 SKU 库存模型用于扣减操作 | ✅ 已完成 |

### 2.2 已修改未提交

| 文件 | 修改内容 | 状态 |
|------|----------|------|
| `order/api/order/v1/error_reason.proto` | 新增 INSUFFICIENT_STOCK、DUPLICATE_REQUEST 错误码 | ⚠️ 未提交 |

### 2.3 待实现

- 生成错误码 pb 代码
- Biz 层接口扩展（幂等查询、库存扣减接口）
- Data 层幂等性实现
- Data 层库存扣减与回补实现
- Service 层适配
- 编译验证

---

## 三、技术方案

### 3.1 幂等性方案

采用 **"request_id 唯一索引 + 数据库兜底"** 的两级幂等策略：

```
用户请求（携带 request_id）
         ↓
【第一级】先查询 request_id 是否存在
         ↓
   ┌─────┴─────┐
   ↓           ↓
 不存在       已存在
   ↓           ↓
创建订单    返回已有订单（幂等）
   ↓
【第二级】数据库唯一索引兜底
   ↓
并发冲突 → 捕获唯一索引异常 → 查询返回已有订单
```

**方案说明**：
- 调用方传入唯一的 `request_id`（如 UUID、雪花ID）
- `orders` 表添加 `request_id` 字段并建立唯一索引
- 创建订单前先查询是否存在该 `request_id` 的订单
- 存在则直接返回（幂等），不存在则创建
- 并发场景下，唯一索引冲突兜底，捕获异常后查询返回

### 3.2 库存扣减方案

采用 **"数据库乐观锁 + 事务"** 方案：

```
开始事务
     ↓
遍历订单项
     ↓
扣减 SKU 库存（乐观锁）
  UPDATE sku SET stock = stock - quantity 
  WHERE id = ? AND stock >= quantity
     ↓
扣减失败 → 回滚事务 → 返回库存不足
     ↓
扣减成功 → 创建订单 → 提交事务
```

**方案说明**：
- 使用 `stock >= quantity` 条件实现乐观锁，防止超卖
- 库存扣减和订单创建在同一个数据库事务中
- 取消订单时回补库存

---

## 四、剩余实现步骤

### 步骤1：提交错误码修改并生成 pb 代码

**操作1**：提交错误码修改
- 提交信息：`feat: 新增库存不足等错误码`

**操作2**：执行 `make api` 生成 pb 代码
- 提交信息：`chore: 生成错误码 pb 代码`

---

### 步骤2：Biz 层修改 - 接口与逻辑扩展

**修改文件**：`order/internal/biz/order.go`

**修改内容**：

1. **OrderRepo 接口新增方法**：
   - `GetOrderByRequestID(ctx context.Context, requestID string) (*Order, error)` - 根据 request_id 查询订单
   - `DeductStock(ctx context.Context, skuID int64, quantity int) error` - 扣减库存
   - `RestoreStock(ctx context.Context, skuID int64, quantity int) error` - 回补库存

2. **OrderUseCase.CreateOrder 方法修改**：
   - 新增 `requestID string` 参数
   - 先调用 `GetOrderByRequestID` 查询是否已存在订单
   - 存在则直接返回已有订单（幂等）
   - 不存在则继续创建流程
   - 库存扣减逻辑在 Data 层事务中处理

3. **OrderUseCase.CancelOrder 方法修改**：
   - 取消订单时需要回补库存
   - 先获取订单详情（含订单项）
   - 遍历订单项调用 `RestoreStock` 回补库存

**提交信息**：`feat: Biz层新增幂等查询和库存扣减接口`

---

### 步骤3：Data 层修改 - 幂等性实现

**修改文件**：`order/internal/data/order.go`

**修改内容**：

1. **实现 GetOrderByRequestID 方法**：
   - 根据 `request_id` 查询订单（Preload Items）
   - 转换为 biz.Order 返回

2. **修改 CreateOrder 方法 - 幂等逻辑**：
   - 新增 `requestID` 参数透传
   - 创建订单时设置 `RequestID` 字段
   - 捕获唯一索引冲突异常（gorm.ErrDuplicatedKey）
   - 冲突时根据 request_id 查询并返回已有订单

3. **convertOrderToBiz 函数补充**：
   - 转换时包含 RequestID 字段

**提交信息**：`feat: Data层实现订单创建幂等性`

---

### 步骤4：Data 层修改 - 库存扣减与回补

**修改文件**：`order/internal/data/order.go`

**修改内容**：

1. **实现 DeductStock 方法**：
   - 使用乐观锁扣减：`UPDATE sku SET stock = stock - ? WHERE id = ? AND stock >= ?`
   - 影响行数为 0 则返回库存不足错误
   - 使用 INSUFFICIENT_STOCK 错误码

2. **实现 RestoreStock 方法**：
   - 回补库存：`UPDATE sku SET stock = stock + ? WHERE id = ?`

3. **修改 CreateOrder 方法 - 事务中扣减库存**：
   - 在事务中，创建订单前遍历订单项扣减库存
   - 库存扣减失败则回滚事务
   - 库存扣减成功后再创建订单和订单项

4. **修改 CancelOrder 方法 - 事务中回补库存**：
   - 查询订单及订单项
   - 在事务中遍历订单项回补库存
   - 更新订单状态为已取消

**提交信息**：`feat: Data层实现库存扣减与回补`

---

### 步骤5：Service 层适配

**修改文件**：`order/internal/service/order.go`

**修改内容**：

1. **CreateOrder 方法**：
   - 从请求中获取 `request_id` 并传递给 UseCase
   - 重复请求（幂等返回）时，正常返回订单信息

2. **convertOrderToProto 函数补充**：
   - 转换时包含 RequestID 字段（可选，proto Order 消息中是否需要增加 request_id 字段）

**提交信息**：`feat: Service层适配幂等性和库存扣减`

---

### 步骤6：编译验证

**操作**：
1. 执行 `make generate` 生成 Wire 代码
2. 执行 `cd order && go build ./...` 验证编译

**提交信息**：`chore: 编译验证订单幂等性与库存扣减`

---

## 五、涉及文件清单

| 层级 | 文件 | 修改类型 | 状态 |
|------|------|----------|------|
| Proto | `api/order/v1/order.proto` | 修改 | ✅ 已完成 |
| Proto | `api/order/v1/error_reason.proto` | 修改 | ⚠️ 未提交 |
| Data | `internal/data/model.go` | 修改 | ✅ 已完成 |
| Data | `internal/data/order.go` | 修改 | ❌ 待实现 |
| Biz | `internal/biz/entity.go` | 修改 | ✅ 已完成 |
| Biz | `internal/biz/order.go` | 修改 | ❌ 待实现 |
| Service | `internal/service/order.go` | 修改 | ❌ 待实现 |

---

## 六、Git 提交规划（剩余）

| 序号 | 提交信息 | 类型 |
|------|----------|------|
| 1 | feat: 新增库存不足等错误码 | feat |
| 2 | chore: 生成错误码 pb 代码 | chore |
| 3 | feat: Biz层新增幂等查询和库存扣减接口 | feat |
| 4 | feat: Data层实现订单创建幂等性 | feat |
| 5 | feat: Data层实现库存扣减与回补 | feat |
| 6 | feat: Service层适配幂等性和库存扣减 | feat |
| 7 | chore: 编译验证订单幂等性与库存扣减 | chore |

---

## 七、关键代码设计说明

### 7.1 幂等性核心逻辑（Data层 CreateOrder）

```go
// 伪代码示意
func CreateOrder(ctx, order, requestID) (*Order, error) {
    // 1. 先查询是否已存在
    existOrder, err := GetOrderByRequestID(ctx, requestID)
    if err == nil && existOrder != nil {
        return existOrder, nil // 幂等返回
    }
    
    // 2. 设置 request_id
    order.RequestID = requestID
    order.OrderNo = generateOrderNo()
    
    // 3. 事务中扣减库存 + 创建订单
    err = db.Transaction(func(tx) error {
        // 扣减库存...
        // 创建订单...
        return nil
    })
    
    // 4. 捕获唯一索引冲突（并发场景兜底）
    if errors.Is(err, gorm.ErrDuplicatedKey) {
        return GetOrderByRequestID(ctx, requestID)
    }
    
    return order, err
}
```

### 7.2 库存扣减核心逻辑（乐观锁）

```go
// 伪代码示意
func DeductStock(ctx, skuID, quantity) error {
    result := db.Exec(
        "UPDATE sku SET stock = stock - ? WHERE id = ? AND stock >= ?",
        quantity, skuID, quantity,
    )
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return errors.New(INSUFFICIENT_STOCK)
    }
    return nil
}
```

---

## 八、风险与注意事项

### 8.1 风险点

| 风险 | 影响 | 应对方案 |
|------|------|----------|
| 数据库表结构变更 | 需新增 request_id 字段和索引 | GORM AutoMigrate 自动处理 |
| 库存扣减性能 | 高并发下乐观锁冲突率高 | 初期可接受，后续可引入 Redis 预扣 |
| request_id 生成 | 调用方不传入怎么办 | 服务端可兜底生成（但失去幂等意义） |
| Sku 表跨服务 | 订单服务直接操作 product 的 sku 表 | 当前阶段直接访问，后续考虑 gRPC 调用 |

### 8.2 注意事项

1. **库存扣减必须在事务中**：确保扣减库存和创建订单原子性
2. **取消订单必须回补库存**：防止库存泄漏
3. **幂等返回的订单状态**：重复请求返回的订单和首次创建状态一致
4. **错误码规范**：使用 kratos 错误码规范，在 proto 中定义
5. **跨服务数据访问**：订单服务直接操作 sku 表是临时方案，长期应通过 product 服务 gRPC 接口

---

## 九、测试验证要点

### 9.1 幂等性测试
- [ ] 同一 request_id 重复请求，只创建一个订单
- [ ] 不同 request_id 创建不同订单
- [ ] 并发相同 request_id 请求，只创建一个订单
- [ ] request_id 为空时的行为（需明确：报错 or 服务端生成）

### 9.2 库存扣减测试
- [ ] 库存充足时，下单成功，库存减少
- [ ] 库存不足时，下单失败，库存不变
- [ ] 取消订单，库存回补
- [ ] 库存为 0 时，下单失败
- [ ] 多个 SKU 部分库存不足，整个订单回滚

### 9.3 事务一致性测试
- [ ] 库存扣减成功但订单创建失败，库存回滚
- [ ] 部分 SKU 库存不足，整个订单回滚
- [ ] 取消订单时库存回补和状态更新的原子性
