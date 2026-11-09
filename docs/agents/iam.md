# 可选内嵌 IAM（ark-iam 形态 2）

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

本项目（demo 应用）支持**可选内嵌** [ark-iam](https://github.com/morehao/ark-iam)：一个进程同时是业务后端和 IAM 的 OIDC Provider。
是否启用由 `config.iam.enable` 开关控制，**默认关闭**。关闭时对现有业务零影响；打开后业务接口全部走 OIDC 真鉴权。

相关源码（均在 `backend/apps/demo`）：

| 文件 | 职责 |
|------|------|
| `config/config.go`（`IAMConfig`） | iam 配置结构与 `GetPrefix()` 前缀归一 |
| `iam_setup.go`（`SetupIAM`） | 按开关启动/挂载内嵌 IAM，返回 `io.Closer` |
| `app.go`（`Routers` / `buildAuthMiddleware`） | 装配业务中间件，按开关决定鉴权方式 |
| `internal/middleware/oidc_auth.go` | 业务接口的 OIDC 本地验签鉴权中间件 |
| `backend/pkg/iamlink/*` | IAM string 身份 ↔ 本地 BIGINT 用户ID 的映射层 |
| `backend/scripts/sql/table_init.sql` | 业务表 DDL（含 `biz_iam_user_map`） |

---

## 1. 开关语义（`iam.enable`）

`config.yaml` 默认 `iam.enable: false`。两种状态的行为：

| | `enable: false`（默认，现状） | `enable: true` |
|---|---|---|
| 是否 `iam.New` / 建表 / 挂路由 | **否**，零副作用，直接返回 `(nil, nil)` | 是 |
| `/iam/*` 路由 | 不存在（404） | 挂载 `/install`、`/oidc`、`/v1/{platform,tenant,auth}`、内嵌登录页 |
| 业务鉴权 | dev 回落固定演示身份（信任请求头 `X-User-ID`，**生产禁用**）；非 dev 无身份 | 全部业务接口挂 OIDC 本地验签中间件，无 token 一律 401 |
| 依赖 | ark-iam 依赖仍编入二进制，但运行期不初始化 | 运行期初始化 DB/Redis（复用主库/缓存，或按 iam.dsn / iam.redis_addr 独立） |

> **关闭即无 IAM**：`SetupIAM` 在 `enable=false` 时显式把 `currentIAM` 复位为 nil，避免上一次启用残留。

`iam.enable=true` 时 `SetupIAM` 顺序：`iam.New` → `RunMigrations` → `Mount(prefix)` → `MountFrontend(engine, prefix)`。
`Mount` 与 `MountFrontend` 用同一 `prefix`（`GetPrefix()` 归一，缺省 `/iam`），否则前端页面相对路径 404。

---

## 2. 表前缀与迁移边界（重要）

内嵌 IAM **自带**一批 `iam_` 前缀的表（如 `iam_person`、`iam_user`、`iam_tenant`、`iam_application_client` 等），
**全部由 IAM 的 `RunMigrations`（AutoMigrate）在启用时自动创建**——宿主项目**不要**把它们写进 `table_init.sql`。

| 表 | 建表方式 | 归属 |
|----|----------|------|
| `iam_*`（全部 IAM 领域表） | `SetupIAM` 内 `app.RunMigrations(ctx)`（AutoMigrate），`enable=true` 时执行 | ark-iam 库 |
| 业务表（`product`、`biz_*` 等） | `backend/scripts/sql/table_init.sql` 手工执行 | 宿主项目 |
| `biz_iam_user_map`（身份映射） | `table_init.sql`（**不参与 AutoMigrate**） | 宿主项目 |

**边界约定**：

- **IAM 域表**（`iam_` 前缀）：由 `RunMigrations` 管，随 ark-iam 版本演进；升级 ark-iam 后启动会自动补列/补表。
- **业务表 + 映射表**（无 `iam_` 前缀）：由 `table_init.sql` 管，语义是宿主 DDL 事实源。
  `biz_iam_user_map` 特意**不设为 AutoMigrate**，因为它含独立序列 `biz_iam_user_map_local_uid_seq` 和
  唯一索引，交由 `table_init.sql` 精确控制（首次部署务必先执行该脚本）。
- 同库共存：`SameDatabase: true`，IAM 表与业务表在同一 PG 库（`demo`）。ark-iam 有同库撞名检测，届时需显式配置。

> 首次在空库上启用：先执行 `table_init.sql`（建业务表 + `biz_iam_user_map`），再以 `enable=true` 启动（`RunMigrations` 建 `iam_` 表）。

---

## 3. 身份映射表 `biz_iam_user_map`

**为什么需要**：ark-iam 的 `rp.Identity.UserID` 是 **string**（如 `"user-2222"`），而本站 `operator.Operator.UserID`
与业务表审计列（`created_by/updated_by/deleted_by`）均为 **BIGINT（uint）**，且 golib 的 `gincontext.GetUserID()` 按 uint 解析（库函数不可改）。
直接把 string 写进 `gcontext.KeyUserID` 会导致审计列全解析成 0。

**解决**：一张映射表把「外部 string 身份」与「本地 uint 主体」解耦（计划里的 ID-1 方案）：

```
biz_iam_user_map
  iam_user_id VARCHAR(128) PRIMARY KEY   -- IAM 侧 rp.Identity.UserID
  local_uid   BIGINT NOT NULL DEFAULT nextval('biz_iam_user_map_local_uid_seq')  -- 本地 uint 用户ID
  person_id   VARCHAR(128) NOT NULL DEFAULT ''
  created_at / updated_at TIMESTAMPTZ
  UNIQUE INDEX uk_biz_iam_user_map_local_uid(local_uid)
```

写入走幂等 upsert（`ON CONFLICT (iam_user_id) DO UPDATE ... RETURNING local_uid`），单语句原子、并发安全；
首次登录自动分配 `local_uid`，已存在则原样返回。`pkg/iamlink.Svc.EnsureLocalUser` 是唯一入口，带 30 分钟 Redis 缓存（命中省一次 DB）。

**边界**：本地 `local_uid` 与业务表主键是**两套 ID 空间**——映射表只负责把「谁」投影成本地 uint，
不修改 `operator/业务表/前端`，业务侧对 IAM 零感知。

---

## 4. OIDC 鉴权中间件如何保护业务接口

`enable=true` 时，`app.go` 的 `buildAuthMiddleware` 返回 `middleware.OIDCAuth(...)`，挂在 `/{AppName}/v1` 路由组上
（即所有 `/v1/demo/*` 业务接口）。流程：

1. 取 `Authorization: Bearer <access_token>`；缺失 → **401**（fail-closed）。
2. 用内嵌 OP 的**进程内公钥**（`app.KeySource()`）本地验签（不访问 OP），并校验 `iss == issuer`；失败/过期 → 401。
3. 得到 `rp.Identity`（string UserID）→ 经 `iamlink.EnsureLocalUser` 映射出 `local_uid`（uint）→ 写 `gcontext.KeyUserID`（uint 可解析字符串）、
   `KeyPersonID`、`KeyAuthToken`；映射失败 → 401。
4. 业务侧 `ginhelper.FillOperator` 照旧读这些 gin Keys 并按 uint 解析，**审计列非 0**。

**白名单**（免鉴权）：`/demo/docs`（swag）、`/v1/demo/health`（健康检查）。`/iam/*` 不在业务路由组内，天然放行。

**验收口径**：无 token 调 `/v1/demo/products` → 401；有效 token → 200 且审计列 = 该用户 `local_uid`。

---

## 5. 配置项（`config.IAMConfig`）与生产要求

```yaml
iam:
  enable: false               # 是否启用内嵌 IAM（默认 false）
  prefix: /iam                # 挂载前缀，须与前端 basePath 一致
  dsn: ""                     # 空则复用 db_configs 第一个库
  redis_addr: ""              # 空则复用 redis_config
  redis_password: ""          # 空则复用 redis_config
  issuer: ""                  # 空则按端口推导：http://localhost:{port}{prefix}/oidc
  signing_key: ""             # 签名私钥 PEM（非 dev 必填）
  encryption_key: ""          # 加密密钥（非 dev 必填）
  cookie_secure: false        # SSO cookie Secure，生产（HTTPS）必须 true
```

- **dev**：`signing_key`/`encryption_key` 可留空（IAM 自动生成/放行）。
- **非 dev（prod）**：ark-iam `validateNonDevSecurityConfig` **fail-closed**，必须显式配置
  `signing_key`、`encryption_key`，且 `cookie_secure: true`，否则 `iam.New` 直接报错、进程起不来。
  `config.prod.yaml` 已补 iam 段（占位值请用真实密钥/密钥管理注入，**勿把真实私钥提交入库**）。
- **issuer**：务必与 `Mount` 前缀一致（`{prefix}/oidc`），否则 OP 公布的 issuer 与端点不符，RP 验签因 `iss` 不匹配失败。
  内嵌时 `SetupIAM` 若未显式配置会按 `http://localhost:{port}{prefix}/oidc` 推导。

---

## 6. 前端接入要点（demo-web）

- 开关探测：`src/auth/config.ts` 的 `detectIAMEnabled()`——`VITE_IAM_ENABLE` 显式优先，否则运行时 `GET /iam/config.json` 可达即视为开启。
- 开启时 `main.tsx` 用 `IAMAuthProvider`（门面包 `createAuthProvider`，内含 `BrowserRouter`）+ `AuthGuard`（`useAuthGuard`）包裹；
  令牌经 `AuthGuard` 注入本站 `@ark-go-starter/api`（`setTokenProvider`），401 时 `removeUser` + `signinRedirect`。
- **redirect_uri 白名单**用 `http://localhost:3000/auth/callback`（门面包默认 `redirectPath=/auth/callback`，**不是** `/callback`）。
- **clientID**：建议用门面包 `useRuntimeConfig: true`，从 `/iam/config.json` 下发；也可 `VITE_OIDC_CLIENT_ID` 兜底。
  在内嵌 IAM 控制台建 **public client（PKCE，tokenEndpointAuthMethod=none）** 后，把其 `code` 配到 clientId。
- vite 已代理 `/v1` 与 `/iam` → `http://localhost:8099`（`frontend/apps/demo-web/vite.config.ts`）。

---

## 7. 部署：demo-web(:3000) 与内嵌 IAM 的 redirect_uri 对齐（踩坑必读）

内嵌 IAM 的 OIDC 回调走 `redirect_uri = {前端 origin}/auth/callback`（demo-web 即
`http://localhost:3000/auth/callback`）。**关键前提**：该地址必须登记在所用 OIDC 客户端的 `redirect_uris` 白名单里，
否则 `/iam/oidc/authorize` 会返回 400「The requested redirect_uri is missing in the client configuration」。
内置种子客户端 `platform_admin_web` 默认只登记了控制台的 `http://localhost:4001/auth/callback`，**不含 3000**，
所以直接拿它跑 demo-web 会卡在回调对齐。二选一处理：

### 方案 A：复用种子客户端 `platform_admin_web`，把 3000 回调登记进去

`/iam/config.json` 默认下发 `clientId=platform_admin_web`（`model.SeedBuiltinClientPlatformAdminWeb`），
复用它最省事，只需给该客户端补一条 `http://localhost:3000/auth/callback` 回调。两种做法：

- **改接口**（推荐，可控）：`DefaultConsoleDomains`（最小域集）**不含** `application` / `oauth-client` 域，
  因此 platform 的 `application-clients`（`POST/GET /iam/v1/platform/application-clients`）**默认未挂载**（404/SPA）。
  需先在该应用 `SetupIAM` 的 `Domains` 里加入 `iam.DomainApplication` + `iam.DomainOAuthClient`（并满足域依赖），
  重启后即可用 `PUT/POST` 更新种子客户端的 `redirectURIs`。
- **SQL 直改**（临时联调）：直接更新种子客户端行的 `redirect_uris` 列（JSON 数组，追加
  `"http://localhost:3000/auth/callback"`），例如：
  ```sql
  UPDATE iam_application_client
     SET redirect_uris = jsonb_set(redirect_uris::jsonb,
           '{0}', '"http://localhost:3000/auth/callback"', true)  -- 或按需 append
   WHERE code = 'platform_admin_web';
  ```
  （仅限本地联调；生产走控制台/接口管理。）

### 方案 B：另建一个 public client 专供 demo-web

在内嵌 IAM 创建 **public client**：`PKCE`（`requirePKCE=enable`）、`tokenEndpointAuthMethod=none`、
`redirect_uri=http://localhost:3000/auth/callback`、`grantTypes=[authorization_code, refresh_token]`、`scopes=[openid, profile, email]`。
建好后把其 `code`（即 client_id）配到以下任一处，让前端拿到正确 clientId：

- `config.yaml` 的 `iam` 段（若宿主扩展了 `clientId` 下发），或
- `/iam/config.json` 的 `clientId`（内嵌默认下发种子客户端，需改为新 client 的 code），或
- 前端环境变量 `VITE_OIDC_CLIENT_ID`（`createAuthProvider` 兜底读取）。

> 无论 A/B：门面包默认回调路径是 **`/auth/callback`**（`redirectPath=/auth/callback`，**不是** `/callback`），
> `/iam/config.json` 的 `callbackPath` 也是 `/auth/callback`，前端登记与客户端白名单务必一致。

---

## 8. 生产安全项（非 dev fail-closed）

`iam.enable=true` 且 `server.env != dev` 时，ark-iam 的 `validateNonDevSecurityConfig` 强制要求（否则 `iam.New` 报错、进程起不来）：

- `iam.signing_key`：OIDC 签名私钥 PEM（RSA/EC），签发/验签 id_token；
- `iam.encryption_key`：会话/令牌加密密钥（≥32 字节随机）；
- `iam.cookie_secure: true`：SSO 会话 cookie 的 Secure 标志，HTTPS 生产必须为 true；
- `iam.issuer`：建议显式设为对外域名（`https://.../iam/oidc` 或反代后的公网地址），须与 `Mount` 前缀一致。

参考 `config.prod.yaml` 已补的 `iam` 段（占位值请用密钥管理/环境变量注入真实密钥，**勿把真实私钥提交入库**）。

---

## 9. 常用操作

**启用（临时，本地联调）**：把 `config.yaml` 的 `iam.enable` 改为 `true`（或用带 `iam.enable: true` 的临时配置），
`BOOTSTRAP_TOKEN=xxx make run APP=demo`，随后走 `/iam/install` 初始化管理员 → 建 client → 前端登录。
联调完**务必还原** `enable: false`。

**/install 初始化**（需引导令牌，头 `X-Bootstrap-Token`，值取环境变量 `BOOTSTRAP_TOKEN` 或配置 `install.bootstrapToken`）：

```bash
# 状态（公开只读）
curl -s http://localhost:8099/iam/install/status
# 初始化平台管理员（口令有强度校验；adminEmail/adminPhone 至少一项）
curl -s -X POST http://localhost:8099/iam/install/initialize \
  -H 'Content-Type: application/json' -H "X-Bootstrap-Token: $BOOTSTRAP_TOKEN" \
  -d '{"adminUsername":"admin","adminPassword":"Adm1n@Strong!Pass","adminEmail":"admin@example.com","tenantName":"Acme"}'
```

> 验证矩阵与端到端实际结果见 [.dsh/docs/notes/2026-11-09-acceptance-report.md](../../.dsh/docs/notes/2026-11-09-acceptance-report.md)。
