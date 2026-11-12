# 常用工具与技术选型

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

| 领域 | 选型 | 备注 |
|------|------|------|
| 依赖管理 | go mod | 后端 workspace 由 `go.work` 管理 |
| API 文档 | swag (Swag Go) | 注解规范见 [api-conventions.md](./api-conventions.md#swagger-文档) |
| 代码生成 | gocli | 命令见 [commands.md](./commands.md#代码生成) |
| 静态检查 | golangci-lint v2 | 配置 `.golangci.yml`（v2 schema）；安装方式与坑见 [commands.md](./commands.md#lint-和代码检查)（**勿 `go install` / 裸 `brew install`**） |
| 数据库 | GORM with PostgreSQL | **唯一目标库**：DDL 见 `tools/sql/table_init.sql`（[字段注释规范](./backend-conventions.md#sql-字段注释硬性要求)）；model tag 禁写其它方言类型（[模型字段类型规范](./backend-conventions.md#模型字段类型规范postgresql)）。MySQL 驱动虽已注册，但没有对应 DDL，不作为目标库 |
| 缓存 | Redis | |
| 链路追踪 | OpenTelemetry | |
| 日志 | golib/glog | 错误日志格式见 [backend-conventions.md](./backend-conventions.md#错误处理)；golib 全部组件能力速查见 [golib.md](./golib.md) |
| 契约平面 | OpenAPI 3.1 + oapi-codegen / Redocly | 契约源 `contracts/openapi/`（单一事实源，**不得被任何模块 import**）；Go client 用 oapi-codegen 生成（`pkg/client`）。见下方[技术债](#技术债todos) |

## 技术债（TODOs）

> 本次仓库结构重构（P0–P6）遗留、**已知但暂不处理**的技术债，每条标注**触发条件**（达到该条件时必须回来处理）。登记于 P6 收尾。

| # | 债务 | 现状 | 触发条件（何时必须处理） |
|---|------|------|--------------------------|
| D1 | **oapi-codegen 不支持 OpenAPI 3.1** | 试点从 `contracts/openapi/openapi.json`（3.1.0）**派生 3.0.3** 再生成（`pkg/client/oapi-codegen.yaml`），生成器 0 warning；契约权威源仍 3.1.0 | oapi-codegen 正式支持 3.1（[issue #373](https://github.com/oapi-codegen/oapi-codegen/issues/373)），或需要严格 3.1 语义（`type: [x,'null']`、`prefixItems` 等）时，去掉 3.0.3 派生步骤 |
| D2 | **swag general info 为空** | `apps/api/app.go` 无 `@title/@version/@BasePath`，生成的 swagger general info 为空，试点在转换后手动补 | 正式启用契约生成流水线（`make contracts`/CI 校验）前，需在 app.go 补 swag 通用注解或由流水线统一注入，避免手动补 |
| D3 | **契约全量生成未做（仅 1 接口试点）** | P5 仅对 `GET /v1/demo/products` 做 swag→OpenAPI3.1→Go client 试点；`swag --parseDependency` 会把 ark-iam SDK 路由也扫进来（全量 110 paths） | 要从契约生成全量客户端时，需先界定生成边界「**只出本应用 `/v1/demo/*`**」，把 SDK/内部路由排除在契约外 |
| D4 | **Go client 引入 `github.com/oapi-codegen/runtime` 运行时依赖** | `pkg/go.mod` 新增该依赖（生成 client 运行时需要，间接 apapsch/go-jsonmerge） | 要求 Go client **零运行时依赖**时，改用无 runtime 的模板 / 自定义生成 / 换生成器 |
| D5 | **手写 api client 尚未被生成物替换** | `packages/api`（手写，原 648 行）仍在用；`packages/api-client` 为生成路线（P5-T2 已铺） | 正式切换「契约驱动客户端」时：让生成物接管调用方，手写客户端移入 `legacy/` 保留一个版本周期后删除 |
| D6 | **`@apidevtools/swagger-cli` 已 abandoned** | P5 试点用过它做校验（可用但 npm 标注 deprecated）；CI/流水线应以 **Redocly CLI** 为主校验器 | 始终生效（记录备查）；CI 正式校验一律用 Redocly，不新增 swagger-cli 依赖 |
| D7 | **`tests/e2e` 仍声明 `typescript: ^7.0.2`** | 全仓 TS 由 `pnpm-workspace.yaml` 的 `catalog: 5.9.3` + `overrides: typescript 5.9.3` 强制收敛，实际安装**单一 `typescript@5.9.3`**；但 `tests/e2e/package.json` 的声明值仍是 ^7.0.2（靠 overrides 兜底，声明与实际不一致） | 下次改 `tests/e2e` package.json 时顺手把声明改成 `catalog:` / 5.9.x，消除声明漂移 |

### 版本收敛现状（P6 复核，2026-11-12）

- **TypeScript**：全仓单一 **5.9.3**。`apps/web`、`packages/{api,api-client,types,ui}` 均用 `catalog:` → 5.9.3；`tests/e2e` 声明 ^7.0.2 但被 `overrides` 强制解析为 5.9.3。`node_modules/.pnpm` 下仅 `typescript@5.9.3`，`tsc --version` = 5.9.3。**无重复 TS 大版本**（lock 中 `typescript@7.x` 匹配均为 `@babel/*-typescript@7`、`openapi-typescript@7` 等包名，非 TS 编译器）。残留：仅 `tests/e2e` 声明值漂移（D7）。
- **Go 依赖**：workspace 由根 `go.work` 统一；新增 `oapi-codegen/runtime`（D4）单版本，无冲突。
