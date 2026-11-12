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

# 运行单个测试函数（go.work 在仓库根，需带模块路径）
go test ./apps/api/internal/service/svcproduct -run TestProductList -v
```

> **注意**：Go workspace 在仓库根 `go.work`（`use ./apps/api ./pkg`），go 命令在**仓库根**执行并用模块路径（如 `./apps/api/...`、`./pkg/...`）；`make` 目标已封装好，通常无需手敲。
> `pkg/dbclient/dbcheck_test.go` 依赖本地 PostgreSQL(5432) / Redis(6379) 服务，未启动对应服务时该包测试会失败（属环境依赖，非代码问题）。
> 本项目不连接 Elasticsearch，`health` 接口只返回 `postgres` / `redis` 两项。
> 数据库初始化脚本见 `apps/api/db/table_init.sql`（PostgreSQL DDL）。

测试写法约定见 [testing.md](./testing.md)。

## Lint 和代码检查

```bash
# 运行 golangci-lint（在 workspace 各模块内分别执行）
make lint

# 仅运行特定 linter
golangci-lint run ./... --disable-all -E golint,errcheck,staticcheck
```

前端 ESLint（flat config：[`eslint.config.mjs`](../../eslint.config.mjs)，覆盖 `apps/web` 与 `packages/*` 源码）：

```bash
# 根目录等价命令
make lint-frontend

# 或在仓库根直接跑（自动修复用 pnpm lint:fix）
pnpm lint
```

> 提交前自检口径：前端 `pnpm lint` 必须 **0 error**；当前基线另有 7 条 `react-hooks` warning（`set-state-in-effect` /
> `static-components`，属既有页面写法，已显式降级为 warn 并在配置注释里记录收紧条件）。

> **安装 golangci-lint（v2，勿用 `go install`）**
>
> 本仓库 `.golangci.yml` 是 **v2 schema**，必须用 golangci-lint **v2**（`golangci-lint --version` 应 ≥ 2）。
>
> - ❌ **不要 `go install`**：二进制会落到当前 gvm pkgset 的 `$GOPATH/bin`，切换 Go 版本（如 1.26 → 1.27）后就不在 PATH，且可能与 Go toolchain 版本不匹配。
> - ❌ **不要裸 `brew install golangci-lint`**：其 formula `depends_on "go"`，会顺带装 Homebrew 的 go；而 `/opt/homebrew/bin` 在 PATH 中通常早于 gvm，会**劫持全局 `which go`**。
> - ✅ **推荐**：`brew install --ignore-dependencies golangci-lint`（跳过 Homebrew go），或从 [GitHub Releases](https://github.com/golangci/golangci-lint/releases) 下预编译二进制放到独立目录（如 `~/.local/bin`）。
>
> 装完用 `which -a go` 确认仍只有 gvm 那条路径，即未污染本地 Go 环境。

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
