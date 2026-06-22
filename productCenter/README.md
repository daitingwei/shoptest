# Product Center

商品中心服务，基于 Kratos v2 构建。

## 项目结构

```
api/                    Proto 定义 + 生成代码
cmd/productcenter/      入口、Wire 依赖注入
configs/                配置文件
internal/
  server/               HTTP / gRPC 服务注册
  service/              接口实现（DTO 转换）
  biz/                  业务逻辑 + 仓储接口
  data/                 数据层实现（GORM + MySQL）
```

## 服务模块

| 服务 | 描述 |
|------|------|
| Product | 商品 CRUD + 列表查询 |
| Shop | 店铺 CRUD |
| SKU | 商品 SKU CRUD |
| ProductTag | 商品标签 CRUD |
| ProductMedia | 商品媒体 CRUD |
| BFF | 前台聚合接口（商品详情、列表页、店铺主页） |

## 启动

```bash
go generate ./...
go build -o ./bin/ ./...
./bin/productcenter -conf ./configs
```

HTTP: `http://localhost:8000`
gRPC: `localhost:9000`

## 生成代码

```bash
make api      # 生成 proto 代码
make generate # 生成 wire + go mod tidy
make all      # 以上全部
```
