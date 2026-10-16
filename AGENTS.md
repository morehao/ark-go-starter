# AGENTS.md - ark-go-starter 代码库开发指南

本文档是 AI 代理的**入口**：项目概述、命令速查、硬性规则速查。
详细规范已按主题拆分到 [`docs/agents/`](./docs/agents/README.md)，**动手写代码前请点开对应文档**。

## 文档导航（按需加载）

| 主题 | 文档 | 何时读 |
|------|------|--------|
| 项目概述与目录结构 | [docs/agents/project-overview.md](./docs/agents/project-overview.md) | 初次接触代码库 |
| 命令大全 | [docs/agents/commands.md](./docs/agents/commands.md) | 构建 / 运行 / 测试 / 生成代码 |
| 后端代码规范 | [docs/agents/backend-conventions.md](./docs/agents/backend-conventions.md) | **写后端代码前必读** |
| 前端代码规范 | [docs/agents/frontend-conventions.md](./docs/agents/frontend-conventions.md) | **写前端代码前必读** |
| 前端设计语言 | [frontend/DESIGN.md](./frontend/DESIGN.md) | 改样式 / 配色 / 布局前必读（视觉规范事实源） |
| API 路由与 Swagger | [docs/agents/api-conventions.md](./docs/agents/api-conventions.md) | **新增或修改接口前必读** |
| 测试规范 | [docs/agents/testing.md](./docs/agents/testing.md) | 编写测试时 |
| golib 能力说明 | [docs/agents/golib.md](./docs/agents/golib.md) | **写新功能 / 引入通用组件前必读（能力复用）** |
| 常用工具与选型 | [docs/agents/toolchain.md](./docs/agents/toolchain.md) | 引入或替换组件时 |

## 项目概述

`ark-go-starter` 是前后端一体的全栈工程实践项目：后端 Gin + GORM（Go workspace 多模块，`backend/`），前端 React + Vite + Ant Design（pnpm monorepo，`frontend/`）。后端以 `backend/go.work` 管理应用模块 `apps/demo` 与公共层 `pkg`。

## 项目结构

```
ark-go-starter/
├── backend/                 # Go 后端项目（go.work 多模块）
│   ├── apps/demo/           # Demo 示例应用（Gin 演示），:8099
│   ├── pkg/                 # 公共包（code/dbclient/testsetup/token）
│   ├── go.work
│   └── output/              # 构建产物（gitignore）
├── frontend/                # React 前端项目（pnpm monorepo）
│   ├── apps/demo-web/       # 演示前端应用 :3000，代理 /v1 → http://localhost:8099
│   ├── packages/            # 共享包（ui/api/types/tsconfig）
│   └── DESIGN.md            # 前端设计语言（视觉规范事实源）
├── docs/agents/             # 开发规范文档（本文件的拆分正文）
├── Makefile                 # 根 Makefile（backend + frontend 双端命令）
└── AGENTS.md                # AI 代理入口（即本文件）
```

后端应用内部的分层结构见 [backend-conventions.md](./docs/agents/backend-conventions.md#项目结构后端)。

## 命令速查

所有命令在**项目根目录**执行，有效 `APP` 取值为 `demo`。

| 目的 | 命令 |
|------|------|
| 列出可用应用 | `make list-apps` |
| 构建 | `make build APP=demo` |
| 运行（开发调试） | `make run APP=demo` |
| 前后端一起起 | `make dev-all APP=demo` |
| 测试 | `make test APP=demo` |
| Lint | `make lint` |
| 代码生成 | `make codegen APP=demo COMMAND=api\|module\|model` |
| Swagger | `make swag APP=demo` |
| 依赖 / 清理 | `make deps` / `make clean` |
| Docker | `make docker-build APP=demo` / `make docker-run APP=demo` |

> 注意：后端模块的 go 命令需先 `cd backend`；`pkg/dbclient/dbcheck_test.go` 依赖本地 PostgreSQL(5432)/Redis(6379)/ES(9200)，服务未起时该包测试失败属环境依赖。
> 完整命令与说明见 [commands.md](./docs/agents/commands.md)。

## 硬性规则速查（必须遵守）

1. **禁止硬编码字典字符串**：表名、枚举、字典值一律定义为常量；表名/模型枚举放 `model/*.go`，前端专用状态映射放 `internal/constant/`。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#常量定义规范)。
2. **分层与命名**：`controller/ctrxxx`（`ctr` 前缀）、`service/svcxxx`（`svc` 前缀）、`dto/dtoxxx`（`dto` 前缀）；文件名小写下划线；接口用 `UserSvc` 风格，构造函数 `NewXxxSvc()` 注入，并加 `var _ UserSvc = (*userSvc)(nil)` 编译期检查。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#命名规范)。
3. **Import 排序**：标准库 → 第三方库 → 项目内部包 → `github.com/morehao/golib/*`，顺序分组、无空行分隔。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#import-排序)。
4. **错误处理**：业务错误统一 `code.GetError(code.XXXError)`（`pkg/code`）；日志用 `glog.Errorf(ctx, "[module.Method] msg, err:%v", err)`。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#错误处理)。
5. **事务**：使用 `dbclient` 封装的 `Transaction(func(tx *gorm.DB) error {...})`，不要自行开事务。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#事务处理)。
6. **Controller 返回**：统一走 `gincontext`，绑定失败 `gincontext.Fail(ctx, err)`，成功 `gincontext.Success(ctx, res)`。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#controller-返回模式)。
7. **路由**：`/{版本}/{app}/{资源复数}`，RESTful + HTTP 方法，如 `GET /v1/demo/users`；路径参数 `:xxxID`；**层级不超过 4 层**。详见 [api-conventions.md](./docs/agents/api-conventions.md)。
8. **Swagger**：新增接口需补 `@Tags/@Summary/@Param/@Success/@Router` 注解，并执行 `make swag APP=demo`。详见 [api-conventions.md](./docs/agents/api-conventions.md#swagger-文档)。
9. **测试**：同包或 `_test` 包，用 `testsetup.Initialize(testsetup.AppNameDemo)` 初始化环境（基于 `pkg/testsetup`，详见 [pkg/testsetup/README.md](./backend/pkg/testsetup/README.md)），标准库 `testing` + `testify/assert`。详见 [testing.md](./docs/agents/testing.md)。
10. **工具链**：数据库 GORM + PostgreSQL（**唯一目标库**；DDL 见 `backend/scripts/sql/table_init.sql`，model tag 禁写其它方言类型）。缓存 Redis，日志 golib/glog，链路 OpenTelemetry。详见 [toolchain.md](./docs/agents/toolchain.md) 与 [模型字段类型规范](./docs/agents/backend-conventions.md#模型字段类型规范postgresql)。
11. **前端**：`main.tsx` 必须用 `AppShell` 包裹，管理页用 `PageContainer`；颜色/圆角/间距一律取 `@ark-go-starter/ui` 的 `tokens` 或 antd token，**禁止裸色值**；卡片无阴影、Table 必须 `tableLayout="fixed"` + 显式列宽。详见 [frontend-conventions.md](./docs/agents/frontend-conventions.md)；设计值事实源是 [frontend/DESIGN.md](./frontend/DESIGN.md)（改 `theme.ts` tokens 与它必须同步）。
12. **注释**：`internal/dto/` 下结构体字段必须有行内中文注释；DDL 字段必须有行尾 `--` 注释 + `COMMENT ON COLUMN`（两份保持一致）。详见 [backend-conventions.md](./docs/agents/backend-conventions.md#注释规范)。

## 维护本文件

- 本文件只保留概述、结构、命令速查与硬性规则速查（每条一句话 + 链接）；
- 扩写规范请改 [`docs/agents/`](./docs/agents/README.md) 下的主题文档，不要堆回本文件。
