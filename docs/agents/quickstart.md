# 快速开始（clone 即用，一条命令跑通全流程）

> 目标读者：第一次 clone 本仓库、想在**最短时间内**把前后端跑起来、并能在页面上走完
> 「登录 → 商品增删改查 → 字典 → 文件上传 → 异步任务 → 定时任务」主流程的开发者。
>
> 本页只讲「怎么最快跑起来」；分层规范、命令大全等见 [docs/agents/README.md](./README.md)。
>
> ⚠️ 本章命令**仅用于本地开发（dev）**。生产部署涉及 `iam.mode` / 签名密钥 / cookie_secure 等
> 安全项（非 dev 走 fail-closed），不在本章范围。

---

## 0. 你需要准备什么

- **Go** **1.27.4+**（`go.work` 与各 `go.mod` 均为 `go 1.27.4` 指令，无 `toolchain` 行；低于此版本 `go work`/构建会直接报 `requires go >= 1.27.4`）
- **Node** **≥26** 与 **pnpm** **12**（根 `package.json` 的 `engines.node >=26`、`packageManager: pnpm@12.10.1`，`.nvmrc` 为 26.13.0；Node 20/22 会因引擎校验导致 `pnpm install` 失败）
- 依赖二选一：
  - **有 Docker（推荐）**：无需自备 PG/Redis，`make quickstart` 会自动拉起。
  - **无 Docker**：自备本机 **PostgreSQL(5432)** 与 **Redis(6379)**，用 `make quickstart-local`。
    - ⚠️ **本机 PG 账号须与配置对齐**：默认走 `postgres` / 密码 `123456` / 库 `demo`（即 `config.yaml` 的 DSN）。
      若你本机 PG 密码不是 `123456`，需在启动前用环境变量覆盖（`DB_USER` / `DB_PASS` / `DB_NAME`），
      否则建库建表会因 `psql` 认证失败而中断。Redis 需**无密码**（与 `redis_config.password` 为空一致）。

> 首次运行会自动执行 `pnpm install` / `go mod download`（需要联网），稍慢属正常。

---

## 1. 一条命令跑起来

在**仓库根目录**执行：

```bash
# 有 Docker（推荐）：自动起 PG/Redis + 建库建表 + 初始化 IAM + 起前后端
make quickstart

# 无 Docker：复用本机已在运行的 PG(5432) / Redis(6379)
make quickstart-local
```

脚本会依次：

1. 探测/拉起依赖（PostgreSQL + Redis），端口已被占则自动降级为「复用现有实例」；
2. `createdb demo`（不存在才建）→ 执行 `apps/api/db/table_init.sql`（幂等建表）；
3. 后台起后端（:8099），等待 `/iam/install/status` 就绪；
4. 首次运行时自动调用 `/iam/install/initialize` 创建管理员（**已初始化则跳过，不重置数据**）；
5. 前台起前端（:3000），并在终端打印访问地址与登录账号密码。

> **幂等**：整条命令可重复执行，已初始化/已建表/已起容器都会跳过，不会重置你的数据。
>
> **端口冲突**：若本机 5432/6379 已被别的服务占用，脚本自动按「外部依赖已就绪」处理，
> 不需要你手动改配置。

---

## 2. 打开浏览器

打开 **http://localhost:3000**：

- 未登录会自动跳转到内嵌登录页 `/iam/login`；
- 用 `make quickstart` 终端打印的账号密码登录（dev 缺省：`admin` / `Adm1n@Strong!Pass`，
  可在启动时用 `IAM_ADMIN_USER` / `IAM_ADMIN_PASS` 覆盖）；
- 登录成功回跳到业务页，即可开始操作。

---

## 3. 全流程操作清单（登录后逐项验证）

| 步骤 | 页面 | 操作 | 预期 |
|------|------|------|------|
| 1 | 登录页 | 输入账号密码 → 登录 | 回跳业务首页，左侧出现菜单 |
| 2 | 商品管理 | 新增 / 编辑 / 删除 / 列表查询 | 增删改查生效，列表刷新正确 |
| 3 | 字典管理 | 打开字典页 | 字典数据正常展示 |
| 4 | 文件上传 | 上传一个文件 → 列表可见 | 上传成功、可访问 |
| 5 | 异步任务 | 触发一次异步任务 | 任务执行、状态可查 |
| 6 | 定时任务 | 打开定时任务页 | 任务列表/执行记录正常 |

---

## 4. 验收：跑一遍端到端测试

环境跑起来后（后端 8099 + 前端 3000 已在运行、IAM 已初始化），执行：

```bash
# 若管理员密码不是 dev 缺省，显式传入
make quickstart-verify IAM_ADMIN_USER=admin IAM_ADMIN_PASS='你的密码'
```

它会以 `E2E_IAM=1` 运行前端 Playwright 的 **IAM 登录主链路**用例
（`specs/iam-login.spec.ts`：未登录 → 跳内嵌登录页 → 账密登录 → 回跳业务页 → 带 token
调受保护接口 200），全绿即代表「快速开始（op 模式登录）」确实可跑通。

> 说明：`quickstart` 以 `iam.mode: op` 启动（登录强制）。其余业务页（商品/字典/文件/异步/
> 定时）的细粒度回归用例是**免登录（disabled 模式）**设计，用 `make e2e` 在 OFF 全量下跑；
> 两套用例分别对齐两种模式，互不冲突。
>
> 首次跑 e2e 需要浏览器内核：`make e2e-install`（仅一次）。

---

## 5. 常见坑（Troubleshooting）

- **登录后一直停在 8099 的「接口不存在」/404**：`iam.issuer` 必须指向前端 origin
  `http://localhost:3000/iam/oidc`（`apps/api/config/config.yaml` 已配好）。
  这是因为登录页 SPA 跑在前端 3000，issuer 若按后端端口推导会去 8099 找登录页。
- **`/iam/install/status` 的 `schemaReady=false`**：业务表没建出来。确认 `table_init.sql`
  已执行（`make quickstart` 会自动执行；手动跑见下）。
- **初始化报 401/`tokenRequired`**：引导令牌不一致。令牌优先级「环境变量 `BOOTSTRAP_TOKEN`
  > `config.yaml` 的 `install.bootstrapToken`」，quickstart 会把 config 里的值通过环境变量
  注入后端，前后端需一致。
- **端口被占（5432/6379）**：quickstart 会探测到端口已被占用并**跳过起容器、直接复用现有实例**
  （等价 `make quickstart-local`）。注意此时用的是**你本机那个 PG/Redis**，而非 compose 里的
  PG16 —— 若本机 PG **密码不是 `123456`**（或版本差异导致迁移不兼容），建库建表会因认证失败中断；
  此时用 `DB_USER`/`DB_PASS`/`DB_NAME` 覆盖为你的实际账号，或先 `make stop-dev`、停掉本机 PG/Redis
  再让 quickstart 自己起容器。若要彻底重来，`make stop-dev` 清理残留进程。
- **本机无 psql**：quickstart 会通过 `docker exec` 调用 PG 容器里的 psql 执行 SQL；也可用
  `PG_CONTAINER=<容器名>` 指定容器。若既无 psql 也无任何 postgres 容器，脚本会直接报错退出。
- **忘记管理员密码**：口令以 bcrypt 存储无法反查。开发期最简单是靠 quickstart 的幂等跳过，
  或清库（`DROP DATABASE demo`）后重跑 `make quickstart` 重新初始化。

---

## 6. 手动分步（不使用 quickstart 时）

供想了解内部步骤、或不方便用脚本的开发者参考：

```bash
# 1) 起依赖（或自备本机 PG/Redis）
docker compose -f deploy/compose/dev.yml up -d

# 2) 建库建表
docker exec -i ark-go-starter-postgres psql -U postgres -c 'CREATE DATABASE demo'   # 已存在可忽略
docker exec -i ark-go-starter-postgres psql -U postgres -d demo -f - < apps/api/db/table_init.sql

# 3) 起后端（BOOTSTRAP_TOKEN 与 config.yaml 的 install.bootstrapToken 一致即可）
BOOTSTRAP_TOKEN=dev-bootstrap-token make run APP=demo   # :8099

# 4) 初始化 IAM（仅首次；status 里 initialized=false 才需要）
curl -s http://localhost:8099/iam/install/status
curl -s -X POST http://localhost:8099/iam/install/initialize \
  -H 'Content-Type: application/json' -H 'X-Bootstrap-Token: dev-bootstrap-token' \
  -d '{"adminUsername":"admin","adminPassword":"Adm1n@Strong!Pass","adminEmail":"admin@example.com"}'

# 5) 起前端
make dev-frontend   # 或 make dev-all 一起起
```

---

## 7. 安全说明（重要）

- config 里的 `install.bootstrapToken: dev-bootstrap-token` 与默认管理员口令**仅供本地 dev**，
  **生产环境绝不可沿用**：引导令牌请用环境变量/密钥管理注入，初始化完成后移除；
  管理员口令由初始化时由你指定为强口令。
- 本章脚本不会把真实管理员密码写死进 `config.yaml` / README；默认口令仅作为 dev 便捷示例，
  且可用环境变量覆盖。

---

## 相关文档

- [docs/agents/iam.md](./iam.md)：可选 IAM（disabled/op/rp 三态）接入与登录排查
- [docs/agents/commands.md](./commands.md)：完整命令大全
- [docs/design-language.md](../../docs/design-language.md)：前端设计语言（改样式前必读）
