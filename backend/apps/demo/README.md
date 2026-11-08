# Demo 应用

demo 应用接口调用示例。

## 健康检查

数据源连通性检查（PostgreSQL / Redis），请求成功即返回各数据源连通状态。

```bash
curl http://127.0.0.1:8099/v1/demo/health
```

成功返回：

```json
{"code":0,"requestID":"...","msg":"success","data":{"postgres":"ok","redis":"ok"}}
```

字段说明：

- `postgres`：PostgreSQL 数据源连通状态（本项目样板库）
- `redis`：Redis 数据源连通状态

各字段正常值为 `ok`，异常时为对应的错误信息。

## 商品管理示例接口（product）

demo 以「商品管理」作为业务分层（controller / service / dao / model / object / dto）的示例对象，接口位于 `/v1/demo/products` 下，响应字段为驼峰。代码位置：`internal/controller/ctrproduct`、`internal/service/svcproduct`、`internal/dto/dtoproduct`、`internal/router/product.go`、`model/product.go`、`dao/product.go`、`object/objproduct`。

| 操作 | 方法 & 路径 | 说明 |
|------|-------------|------|
| 创建 | `POST /v1/demo/products` | 创建一条商品，返回 `productID` |
| 事务创建 | `POST /v1/demo/products/tx` | 同一事务内创建两条，演示原子性回滚 |
| 列表 | `GET /v1/demo/products` | 分页查询（`page` / `pageSize`，可选 `name`、`categoryID` 过滤） |
| 详情 | `GET /v1/demo/products/{productID}` | 按自增 ID 查详情 |
| 修改 | `PUT /v1/demo/products/{productID}` | 按需更新 `name` / `categoryID` / `price` |
| 删除 | `DELETE /v1/demo/products/{productID}` | 软删除（`deleted_at`） |

示例 curl：

```bash
# 创建商品
curl -X POST http://127.0.0.1:8099/v1/demo/products \
  -H 'Content-Type: application/json' \
  -d '{"name":"机械键盘","categoryID":1,"price":399.00}'

# 事务创建：同一事务内插两条；failAfterFirst=true 时第一条后返回错误、整体回滚
curl -X POST http://127.0.0.1:8099/v1/demo/products/tx \
  -H 'Content-Type: application/json' \
  -d '{"firstName":"商品A","secondName":"商品B","failAfterFirst":false}'

# 分页列表（可带 name / categoryID 过滤）
curl 'http://127.0.0.1:8099/v1/demo/products?page=1&pageSize=10'

# 详情
curl http://127.0.0.1:8099/v1/demo/products/1

# 修改（只传需要更新的字段）
curl -X PUT http://127.0.0.1:8099/v1/demo/products/1 \
  -H 'Content-Type: application/json' \
  -d '{"price":299.00}'

# 删除（软删）
curl -X DELETE http://127.0.0.1:8099/v1/demo/products/1
```

> 说明：路径参数为 `{productID}`（路由里写作 `:productID`）。身份/操作人体系（`pkg/code/user.go`、`operator.Operator` 等）与示例业务对象相互独立——示例对象只是「商品管理」，真实身份体系仍然保留。

## demo-web 页面

前端 `frontend/apps/demo-web` 已从单页列表升级为多页骨架：左侧深色 `Sider` + 顶部 `Header`，菜单用 antd `Menu` + `useState` 切页（未引入 react-router）。当前页面：

| 页面 | 目录 | 说明 |
|------|------|------|
| 商品管理 | `src/pages/Products` | 对应本应用的 `/v1/demo/products` CRUD，含分页列表与新建/编辑弹窗 |
| 数据字典 | `src/pages/Dicts` | 对接 golib `gindict` 字典接口 |
| 配置中心 | `src/pages/Settings` | 对接 golib `ginsetting` 配置接口 |
| 定时任务 | `src/pages/Cron` | 对接 golib `gincron` 定时任务接口 |
| 异步任务 | `src/pages/Async` | 对接 golib `ginasync` 异步任务接口 |
| 健康检查 | `src/pages/Health` | 对接 `/v1/demo/health` 展示数据源连通状态 |

## 内置管理接口（golib gserver 路由模块）

demo 已把 golib 自带的路由模块挂载到 `/v1/demo` 前缀下，开箱即用，无需自行编写 CRUD。组件实例在启动期用 demo 主库（PostgreSQL）初始化，组件表默认 `AutoMigrate` 自动建表。

| 模块 | 用途 | 资源前缀 |
|------|------|----------|
| ginupload | 文件服务 | `/files`、`/files/multipart`、`/objects` |
| gindict | 数据字典 | `/dict-types`、`/dict-codes`、`/dict-items` |
| ginsetting | 配置中心 | `/settings` |
| gincron | 定时任务 | `/cron-tasks`、`/cron-runs` |
| ginasync | 异步任务 | `/async-tasks`、`/async-runs` |

dev 环境可直接打开 `/v1/demo/docs`（Swagger）或 `/v1/demo/redocs`（ReDoc）查看全部接口。

示例：查询字典类型列表、新建一个字典类型（类型编码 `code` 必填且全表唯一）

```bash
# 查询字典类型列表
curl http://127.0.0.1:8099/v1/demo/dict-types

# 新建字典类型
curl -X POST http://127.0.0.1:8099/v1/demo/dict-types \
  -H 'Content-Type: application/json' \
  -d '{"code":"order_status","name":"订单状态"}'
```
