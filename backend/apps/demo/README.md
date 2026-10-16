# Demo 应用

demo 应用接口调用示例。

## 健康检查

数据源连通性检查（PostgreSQL / Redis / ES），请求成功即返回各数据源连通状态。

```bash
curl http://127.0.0.1:8099/v1/demo/health
```

成功返回：

```json
{"code":0,"requestID":"...","msg":"success","data":{"postgres":"ok","redis":"ok","es":"ok"}}
```

字段说明：

- `postgres`：PostgreSQL 数据源连通状态（本项目样板库）
- `redis`：Redis 数据源连通状态
- `es`：Elasticsearch 数据源连通状态

各字段正常值为 `ok`，异常时为对应的错误信息。
