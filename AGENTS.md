# AGENTS.md - ark-go-starter 代码库开发指南

本文档是 AI 代理的**入口**：项目概述、命令速查、硬性规则速查。
详细规范已按主题拆分到 [`docs/agents/`](./docs/agents/README.md)，**动手写代码前请点开对应文档**。

## 文档导航（按需加载）

| 主题 | 文档 | 何时读 |
|------|------|--------|
| 快速开始（clone 即用） | [docs/agents/quickstart.md](./docs/agents/quickstart.md) | **第一次把项目跑起来** |
| 项目概述与目录结构 | [docs/agents/project-overview.md](./docs/agents/project-overview.md) | 初次接触代码库 |
| 命令大全 | [docs/agents/commands.md](./docs/agents/commands.md) | 构建 / 运行 / 测试 / 生成代码 |
| 后端代码规范 | [docs/agents/backend-conventions.md](./docs/agents/backend-conventions.md) | **写后端代码前必读** |
| 前端代码规范 | [docs/agents/frontend-conventions.md](./docs/agents/frontend-conventions.md) | **写前端代码前必读** |
| 前端设计语言（视觉规范） | [docs/DESIGN.md](./docs/DESIGN.md) | 改样式 / 配色 / 布局前必读（视觉规范唯一事实源，Ant Design 原生风格；遵循 Google DESIGN.md 规范） |
| API 路由与 Swagger | [docs/agents/api-conventions.md](./docs/agents/api-conventions.md) | **新增或修改接口前必读** |
| 测试规范 | [docs/agents/testing.md](./docs/agents/testing.md) | 编写测试时 |
| golib 能力说明 | [docs/agents/golib.md](./docs/agents/golib.md) | **写新功能 / 引入通用组件前必读（能力复用）** |
| 可选账号认证（ark-iam） | [docs/agents/iam.md](./docs/agents/iam.md) | 三态 disabled/op/rp 接入、排查登录与鉴权 |
| 常用工具与选型 | [docs/agents/toolchain.md](./docs/agents/toolchain.md) | 引入或替换组件时 |

## 项目概述

`ark-go-starter` 是前后端一体的全栈工程实践项目，采用 monorepo（`apps/` + `packages/` + `pkg/` + `contracts/`）布局：后端 Gin + GORM（Go workspace 多模块，根 `go.work` 管理 `apps/api` 与 `pkg`），前端 React + Vite + Ant Design（pnpm monorepo，根 `pnpm-workspace.yaml` 管理 `apps/web` 与 `packages/*`）。任务编排用 Nx（`nx.json`）。

### 双 apps 语义（重要）

- **`apps/api`** = Go 后端应用。其**逻辑应用名是 `demo`**（`AppName = "demo"`，路由前缀 `/v1/demo`、Swagger tag 等沿用 demo）。因此 `make … APP=demo` 中 `demo` 是逻辑标识，**映射到目录 `apps/api`**（Makefile 内 `APP_DIR=$(if $(filter demo,$(APP)),api,$(APP))`）。它不是前端的 `apps/web`。
- **`apps/web`** = React 前端应用（Vite，:3000），无 `APP` 逻辑名概念，前端命令（`pnpm`/`make dev-frontend` 等）不传 `APP`。
- 一句话：**`APP=demo` 指后端 `apps/api`；前端在 `apps/web`，两者不是同层"两个 demo"，而是 api（Go）/ web（React）。**

## 项目结构

```
ark-go-starter/
├── apps/
│   ├── api/                 # Go 后端应用（Gin，:8099；逻辑名 demo→目录 api）
│   └── web/                 # React 前端应用（Vite，:3000，代理 /v1 → http://localhost:8099）
├── packages/                # 前端共享包（api/api-client/types/ui/tsconfig），pnpm workspace 成员
├── pkg/                     # Go 公共层（code/dbclient/testsetup/token/ginhelper/iamlink/operator/client）
├── contracts/               # 契约平面（单一事实源）：openapi/openapi.json + verify.sh（生成物一致性校验）
├── tests/
│   └── e2e/                 # 端到端测试（前端 Playwright specs + A4 golden）
├── scripts/                 # 仓库级脚本（quickstart.sh、sql/ 建表 DDL）
├── deploy/                  # 容器与编排（docker/api.Dockerfile、compose/dev.yml）
├── docs/                    # 文档（agents/ 开发规范、DESIGN.md 视觉规范事实源）
├── go.work                  # 根 Go workspace（use ./apps/api ./pkg）
├── package.json             # 根前端 workspace 清单
├── pnpm-workspace.yaml      # 根 pnpm workspace 定义（apps/web、packages/*、tests/e2e）
├── nx.json                  # Nx 编排配置（project graph / target cache / tags 边界）
├── tsconfig.base.json       # 前端共享 TS 基配置
├── Makefile                 # 根 Makefile（后端 + 前端 + 契约命令）
└── AGENTS.md                # AI 代理入口（即本文件）
```
> 运行时产物（不入库）：`.local/`（日志 `log`、存储 `storage`）、`output/`、各包 `dist/`、`node_modules/`（见 `.gitignore`）。

后端应用内部分层结构见 [backend-conventions.md](./docs/agents/backend-conventions.md#项目结构后端)。

## 命令速查

所有命令在**项目根目录**执行，`APP` 为应用逻辑标识（当前 `demo`，对应 Go 目录 `apps/api`）。前端/契约命令无需 `APP`。

| 目的 | 命令 |
|------|------|
| 列出可用应用 | `make list-apps` |
| 构建 | `make build APP=demo` |
| 运行（开发调试） | `make run APP=demo` |
| 前后端一起起 | `make dev-all APP=demo` |
| 测试（单元，无需 DB） | `make test APP=demo` |
| 集成测试（需 PG/Redis） | `make test-integration APP=demo` |
| 端到端测试 | `make e2e APP=demo`（后端真端口 + 前端 Playwright，需 PG/Redis；首次 `make e2e-install`） |
| Lint | `make lint`（Go golangci-lint）/ `make lint-frontend`（前端 ESLint） |
| 代码生成 | `make codegen APP=demo COMMAND=api\|module\|model` |
| Swagger | `make swag APP=demo` |
| 契约校验 | `make contracts-verify`（spec 合法 + Go/TS 生成物与 spec 一致；CI 门禁） |
| 依赖 / 清理 | `make deps` / `make tidy` / `make clean` |
| 前端开发 | `make dev-frontend` / `make stop-frontend` |
| 一键跑通 | `make quickstart` / `make quickstart-local` / `make quickstart-verify` |
| Docker | `make docker-build APP=demo` / `make docker-run APP=demo` |

**Nx 任务编排**（`pnpm exec nx …`，monorepo 增量/缓存；详见 `nx.json`）：

| 目的 | 命令 |
|------|------|
| 查看全部项目 | `pnpm exec nx show projects` |
| 依赖图 | `pnpm exec nx graph` |
| 跑全部构建 / 类型检查 | `pnpm exec nx run-many -t build` / `pnpm exec nx run-many -t typecheck` |
| 受影响项目 | `pnpm exec nx show projects --affected` |
| 只构建受影响 | `pnpm exec nx affected -t build`（改 `packages/ui` 只跑 ui+web，不跑 Go） |

> 注意：Go 命令由 Makefile 统一在仓库根调度（`go.work` 在根，`use ./apps/api ./pkg`，无需再 `cd backend`）。**默认测试不依赖外部服务**：连库用例已用 `//go:build integration` 隔离，`make test` 无需 PostgreSQL/Redis；跑集成测试用 `make test-integration`。本项目不连接 Elasticsearch。
> `golangci-lint` 需 **v2**（匹配 `.golangci.yml` v2 schema）：**勿 `go install`、勿裸 `brew install golangci-lint`**（formula 依赖 Homebrew go，会劫持 gvm 的 `go`），用 `brew install --ignore-dependencies golangci-lint`；详见 [commands.md](./docs/agents/commands.md#lint-和代码检查)。
> 完整命令与说明见 [commands.md](./docs/agents/commands.md) 与根 `make help`。

## 硬性规则速查（必须遵守）

1. **禁止硬编码字典字符串**：表名、枚举、字典值一律定义为常量；表名/模型枚举放 `model/*.go`，前端专用状态映射放 `internal/constant/`。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#常量定义规范)。
2. **分层与命名**：`controller/ctrxxx`（`ctr` 前缀）、`service/svcxxx`（`svc` 前缀）、`dto/dtoxxx`（`dto` 前缀）；文件名小写下划线；接口用 `UserSvc` 风格，构造函数 `NewXxxSvc()` 注入，并加 `var _ UserSvc = (*userSvc)(nil)` 编译期检查；**缩略词全大写**（`APIKey`/`OIDC`/`URL`，不是 `ApiKey`/`Oidc`/`Url`）。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#命名规范)。
3. **Import 排序**：标准库 → 第三方库 → 项目内部包 → `github.com/morehao/golib/*`，顺序分组、无空行分隔。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#import-排序)。
4. **错误处理**：业务错误统一 `code.GetError(code.XXXError)`（`pkg/code`）；日志用 `glog.Errorf(ctx, "[module.Method] msg, err:%v", err)`。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#错误处理)。
5. **事务**：使用 `dbclient` 封装的 `Transaction(func(tx *gorm.DB) error {...})`，不要自行开事务。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#事务处理)。
6. **Controller 返回**：统一走 `gincontext`，绑定失败 `gincontext.Fail(ctx, err)`，成功 `gincontext.Success(ctx, res)`。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#controller-返回模式)。
7. **路由**：`/{版本}/{app}/{资源复数}`，RESTful + HTTP 方法，如 `GET /v1/demo/products`；路径参数 `:xxxID`；**层级不超过 4 层**。详见 [api-conventions.md](./docs/agents/api-conventions.md)。
8. **Swagger**：新增接口需补 `@Tags/@Summary/@Param/@Success/@Router` 注解，并执行 `make swag APP=demo`。详见 [api-conventions.md](./docs/agents/api-conventions.md#swagger-文档)。
9. **测试**：同包或 `_test` 包，用 `testsetup.Initialize(testsetup.AppNameDemo)` 初始化环境（基于 `pkg/testsetup`，详见 [pkg/testsetup/README.md](./pkg/testsetup/README.md)），标准库 `testing` + `testify`；**依赖 DB/Redis/外网的用例必须加 `//go:build integration`**（否则 `go test ./...` 在无库环境全红）；错误断言用 `assert.NoError` 而非 `assert.Nil`。详见 [testing.md](./docs/agents/testing.md#集成测试必须隔离硬性要求)。
10. **工具链**：数据库 GORM + PostgreSQL（**唯一目标库**；DDL 见 `scripts/sql/table_init.sql`，model tag 禁写其它方言类型；两者逐列一致由 `make test` 的 `ddl_consistency_test.go` 守护）。缓存 Redis，日志 golib/glog，链路 OpenTelemetry。详见 [toolchain.md](./docs/agents/toolchain.md) 与 [模型字段类型规范](./docs/agents/backend-conventions.md#模型字段类型规范postgresql)。
11. **前端**：`main.tsx` 必须用 `AppShell` 包裹，管理页用 `PageContainer`；颜色/圆角/间距一律取 `@ark-go-starter/ui` 的 `tokens` 或 antd token，**禁止裸色值**；卡片无阴影、Table 必须 `tableLayout="fixed"` + 显式列宽。详见 [frontend-conventions.md](./docs/agents/frontend-conventions.md)；视觉规范事实源是 [docs/DESIGN.md](./docs/DESIGN.md)（Ant Design 原生风格，改 `theme.ts` tokens 与它必须同步）。
12. **注释**：每个包必须有 `// Package xxx` 注释，每个导出符号必须有以符号名开头的 godoc；`internal/dto/` 下结构体字段必须有行内中文注释；DDL 字段必须有行尾 `--` 注释 + `COMMENT ON COLUMN`（两份保持一致）。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#注释规范)。
13. **提交前自检**：`gofmt -w .` + `make lint`（必须 0 issues）+ `make test APP=demo` 三项全过再提交；`%w` 包装错误、blank import 写用途注释、连接池显式配置。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#提交前自检硬性要求)。
14. **目录命名**：`deploy/` 只放构建与部署资产（不进 `go.work` / `pnpm-workspace` / Nx 图；**不含建表 SQL**）；建表 DDL 统一放仓库根 `scripts/sql/`（`table_init.sql`，不参与编译、需手动执行，由 `ddl_consistency_test.go` 按仓库根相对路径读取比对）。代码侧基础设施层统一叫 `internal/infra/`（当前预留不建；一旦建，`dao` / `client` / `storage` 必须同 PR 迁完删除）。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#部署目录与基础设施层的命名边界)。
15. **CI 门禁边界**：PR 门禁只做**判定**，且**按变更范围只跑相关 job**（文档/纯 md → 全跳过；前端 → ESLint + Nx affected typecheck；Go → gofmt + golangci-lint + **单元测试**；依赖变更 → 追加 govulncheck；契约面变更 → 追加 contracts-verify）。**不产生产物**（`vite build` / `go build` / `docker build`）、**不现装工具**（`go install` / `npx` 必须有缓存或路径条件）。判定必需的**编译**用 `actions/cache` 显式缓存 `~/.cache/go-build` + `~/go/pkg/mod`：**PR 侧只 restore，save 只在 main push**（缓存按分支写入，PR 各写一份会打爆 10GB 配额）。**集成测试与 `-race` 不在 PR 门禁**，由主干 `Deep test` 与每周 schedule 兜底；主干 push 永远全量。落点见 [`.github/workflows/ci.yml`](./.github/workflows/ci.yml) 顶部注释。

## 维护本文件

- 本文件只保留概述、结构、命令速查与硬性规则速查（每条一句话 + 链接）；
- 扩写规范请改 [`docs/agents/`](./docs/agents/README.md) 下的主题文档，不要堆回本文件。
