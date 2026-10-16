# 常用命令

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

所有命令默认在**项目根目录**下执行。有效 `APP` 取值为 `demo`。

## 构建与运行

```bash
# 列出所有可用应用
make list-apps

# 构建指定应用
make build APP=demo

# 运行指定应用（开发调试）
make run APP=demo

# 下载依赖
make deps

# 清理构建产物
make clean

# 同时启动前后端开发服务（后端后台运行、前端前台日志，Ctrl+C 一并停止）
make dev-all APP=demo

# 启动前端开发服务
make dev-frontend

# 停止前端开发服务
make stop-frontend
```

应用端口：demo 后端 8099、前端 demo-web dev 3000。

## 测试命令

```bash
# 运行指定应用的测试（推荐）
make test APP=demo

# 运行单个测试函数
go test ./apps/demo/internal/service/svcuser -run TestGeneratePassword -v
```

> **注意**：后端模块的 go 命令需在 `cd backend` 后执行。
> `pkg/dbclient/dbcheck_test.go` 依赖本地 PostgreSQL(5432) / Redis(6379) / ES(9200) 服务，未启动对应服务时该包测试会失败（属环境依赖，非代码问题）。
> 数据库初始化脚本见 `backend/scripts/sql/table_init.sql`（PostgreSQL DDL）。

测试写法约定见 [testing.md](./testing.md)。

## Lint 和代码检查

```bash
# 运行 golangci-lint（在 workspace 各模块内分别执行）
make lint

# 仅运行特定 linter
golangci-lint run ./... --disable-all -E golint,errcheck,staticcheck
```

## 代码生成

项目使用 `gocli` 工具进行代码生成：

```bash
# 生成 API 路由和控制器
make codegen APP=demo COMMAND=api

# 生成模块代码
make codegen APP=demo COMMAND=module

# 生成模型代码
make codegen APP=demo COMMAND=model
```

## Swagger 文档

```bash
make swag APP=demo
```

注解写法见 [api-conventions.md](./api-conventions.md#swagger-文档)。

## Docker 支持

```bash
# 构建 Docker 镜像
make docker-build APP=demo

# 运行 Docker 容器
make docker-run APP=demo
```
