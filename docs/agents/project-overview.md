# 项目概述与目录结构

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

## 项目概述

`ark-go-starter` 是一个前后端一体的全栈工程实践项目：

- **后端**：Gin + GORM，Go workspace 多模块，应用在 `apps/api`、公共层在 `pkg`；
- **前端**：React + Vite + Ant Design，pnpm monorepo，位于 `apps/web` 与 `packages/*`。

Go 以仓库根 `go.work` 作为 workspace，`use ./apps/api ./pkg`（应用逻辑标识仍为 `demo`，目录名为 `api`，二者解耦）。

## 项目结构

```
ark-go-starter/
├── apps/
│   ├── api/                 # Demo Go 应用（逻辑标识 demo），Gin 演示 :8099
│   └── web/                 # 前端应用 :3000，代理 /v1 → http://localhost:8099
├── packages/                # 前端共享包（api/api-client/types/ui/tsconfig），pnpm workspace 成员
├── pkg/                     # Go 公共包（client/code/dbclient/ginhelper/iamlink/operator/testsetup/token）
├── contracts/               # 契约平面（单一事实源）：openapi/openapi.json + verify.sh
├── tests/e2e/               # Playwright 端到端测试
├── deploy/                  # docker/api.Dockerfile、compose/dev.yml
├── scripts/                 # 仓库级脚本（quickstart.sh）与 DDL（sql/table_init.sql）
├── docs/                    # 开发规范（docs/agents/…）与设计语言（docs/design-language.md）
├── go.work                  # Go workspace（use ./apps/api ./pkg）
├── package.json             # 根前端 workspace 清单
├── pnpm-workspace.yaml      # 根 pnpm workspace 定义（apps/web、packages/*、tests/e2e）
├── nx.json                  # Nx 任务编排（project graph / target cache / tags 边界）
├── tsconfig.base.json       # 前端共享 TS 基配置
├── Makefile                 # 根 Makefile（在仓库根执行）
├── AGENTS.md                # AI 代理入口（精简索引）
└── README.md
```
> 运行时产物统一落 `.local/`（日志 `.local/log`、存储 `.local/storage`，均 gitignore）；`APP_LOG_DIR` / `APP_STORAGE_DIR` 可覆盖。
>
> `deploy/` 只放构建与部署资产（不进 `go.work` / `pnpm-workspace` / Nx 图，不参与编译）；代码侧基础设施层统一叫 `internal/infra/`，**当前预留不建**——命名边界与迁移纪律见 [backend-conventions.md](./backend-conventions.md#部署目录与基础设施层的命名边界)。

后端应用内部的分层结构见 [backend-conventions.md](./backend-conventions.md)；
前端包结构与样式规范见 [frontend-conventions.md](./frontend-conventions.md) 与 [`docs/design-language.md`](../../docs/design-language.md)。

## 应用与端口

| 应用 | 后端端口 | 前端 dev 端口 |
|------|----------|---------------|
| `demo`（目录 apps/api） | 8099 | 3000（目录 apps/web） |

根 `Makefile` 中有效 `APP` 取值为 `demo`。命令用法见 [commands.md](./commands.md)。
前端 monorepo 的技术栈、包结构与命令见 [frontend-conventions.md](./frontend-conventions.md)。
