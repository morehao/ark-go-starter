# 项目概述与目录结构

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

## 项目概述

`ark-go-starter` 是一个前后端一体的全栈工程实践项目：

- **后端**：Gin + GORM，Go workspace 多模块，位于 `backend/`；
- **前端**：React + Vite + Ant Design，pnpm monorepo，位于 `frontend/`。

后端以 `backend/go.work` 作为 workspace，管理应用模块 `apps/demo` 与公共层 `pkg`。

## 项目结构

```
ark-go-starter/
├── backend/                 # Go 后端项目（go.work 多模块）
│   ├── apps/
│   │   ├── demo/            # Demo 示例应用（Gin 演示），:8099
│   │   └── log/             # 运行时日志（gitignore）
│   ├── pkg/                 # 公共包（code/dbclient/testsetup/token）
│   ├── go.work
│   └── output/              # 构建产物（gitignore）
├── frontend/                # React 前端项目（pnpm monorepo）
│   ├── apps/demo-web/       # 演示前端应用 :3000，代理 /v1 → http://localhost:8099
│   ├── packages/            # 共享包（ui/api/types/tsconfig）
│   └── DESIGN.md            # 前端设计语言（视觉规范事实源）
├── docs/agents/             # 开发规范文档（由 AGENTS.md 拆分而来）
├── Makefile                 # 根 Makefile（backend + frontend 双端命令）
├── AGENTS.md                # AI 代理入口（精简索引）
└── README.md
```

后端应用内部的分层结构见 [backend-conventions.md](./backend-conventions.md)；
前端包结构与样式规范见 [frontend-conventions.md](./frontend-conventions.md) 与 [`frontend/DESIGN.md`](../../frontend/DESIGN.md)。

## 应用与端口

| 应用 | 后端端口 | 前端 dev 端口 |
|------|----------|---------------|
| `demo` | 8099 | 3000（`demo-web`） |

根 `Makefile` 中有效 `APP` 取值为 `demo`。命令用法见 [commands.md](./commands.md)。
前端 monorepo 的技术栈、包结构与命令见 [frontend-conventions.md](./frontend-conventions.md)。
