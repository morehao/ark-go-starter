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
