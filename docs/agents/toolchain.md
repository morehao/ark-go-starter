# 常用工具与技术选型

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

| 领域 | 选型 | 备注 |
|------|------|------|
| 依赖管理 | go mod | 后端 workspace 由 `backend/go.work` 管理 |
| API 文档 | swag (Swag Go) | 注解规范见 [api-conventions.md](./api-conventions.md#swagger-文档) |
| 代码生成 | gocli | 命令见 [commands.md](./commands.md#代码生成) |
| 数据库 | GORM with PostgreSQL | **唯一目标库**：DDL 见 `backend/scripts/sql/table_init.sql`（[字段注释规范](./backend-conventions.md#sql-字段注释硬性要求)）；model tag 禁写其它方言类型（[模型字段类型规范](./backend-conventions.md#模型字段类型规范postgresql)）。MySQL 驱动虽已注册，但没有对应 DDL，不作为目标库 |
| 缓存 | Redis | |
| 链路追踪 | OpenTelemetry | |
| 日志 | golib/glog | 错误日志格式见 [backend-conventions.md](./backend-conventions.md#错误处理) |
