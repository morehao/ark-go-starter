# 可选账号认证（ark-iam 三态：disabled / op / rp）

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)
>
> 反映 ark-iam（`sdk/server` 门面 + 独立 `contract` module）的**三态**接入方式；
> 依赖版本：`sdk v0.5.0` / `contract v0.1.0` / `pkg v0.4.0`（均已发布到 Go proxy）。

本模板支持**可选接入** [ark-iam](https://github.com/morehao/ark-iam) 账号认证体系。是否接入、以什么形态接入，
完全由 `config.yaml` 的 `iam.mode` 决定（默认 `disabled` 不接入，对业务零副作用）。接入形态有三种：

- **disabled**：不接入，dev 演示身份（默认）。
- **op**：本进程内嵌 OIDC Provider——**本项目自己就是 IAM**。
- **rp**：对接外部 ark-iam 服务，本进程只做本地验签（Relying Party）。

---

## 0. 术语表（先读这一段）

采用 ark-iam 官方术语 **OP / RP**，而不是 `embedded` / `remote`——因为 `embedded` 有歧义（既可指「内嵌了 OP 服务端」，也可被误读成「内嵌了某个 SDK 依赖」），而 rp 模式**同样引 SDK 依赖、只是不跑 OP**，用 `embedded` 点不出「跑不跑 OP」这个本质区别。

| 术语 | 全称 | 含义 |
|------|------|------|
| **OP** | OpenID Provider | OIDC 服务端（签发/验签 token 的那一方），ark-iam 服务本身 |
| **RP** | Relying Party | 依赖方（业务应用），用 OP 的公钥做本地验签 |

| `iam.mode` | 谁跑 OP | 是否引 SDK | 是否建库 / 挂 `/iam/*` | 前端登录页来自 | 一句话 |
|---|---|---|---|---|---|
| `disabled` | 无 | 否（可完全不引） | 否 | 无（dev 演示身份） | 不接入认证 |
| `op` | **本项目进程（内嵌）** | 是（`sdk/server` + `sdk/rp`） | **是**（建 `iam_` 表、挂 `/iam/*`） | 本项目前端（npm 门面包） | 本项目自身就是 IAM |
| `rp` | **外部 ark-iam 服务** | 是（仅 `sdk/rp`） | **否**（不建库、不挂路由） | 外部 ark-iam 控制台 | 只对接别人的 IAM，本地验签 |

> op 与 rp 都「引 SDK、都 OIDC 登录、都本地验签、鉴权中间件完全相同」，唯一区别是 **OP 在谁那里**（本进程 vs 外部服务）以及**是否建表挂路由**。这也是后端装配只有两套 `setupOP` / `setupRP` 的原因。

---

## 1. 相关源码（均在 `backend/apps/demo`）

| 文件 | 职责 |
|------|------|
| `config/config.go`（`IAMConfig`） | iam 配置结构、`EffectiveMode()` 归一、`GetPrefix()` 前缀归一 |
| `iam_setup.go`（`SetupIAM`） | 单一装配入口：按 `mode` 分派 disabled / op / rp，返回 `io.Closer` |
| `app.go`（`Routers` / `buildAuthMiddleware`） | 按 `mode` 装配鉴权中间件（op/rp 共用 `OIDCAuth`） |
| `internal/middleware/oidc_auth.go` | 业务接口 OIDC 本地验签中间件（依赖 `contract.KeySource`） |
| `backend/pkg/iamlink/*` | IAM string 身份 ↔ 本地 BIGINT 用户 ID 的映射层（与形态无关，三态通用） |
| `backend/scripts/sql/table_init.sql` | 业务表 DDL（含 `biz_iam_user_map`）；不含 `iam_` 表 |

---

## 2. 三态配置（`config.yaml` 的 `iam` 段）

```yaml
iam:
  mode: disabled          # disabled | op | rp（默认 disabled，等价旧 enable:false）
  # —— op（本进程内嵌 OP）专有 ——
  prefix: /iam            # 挂载前缀（op 必填，须与前端 basePath 一致）
  dsn: ""                 # 空则复用 db_configs 第一个库
  redis_addr: ""          # 空则复用 redis_config
  redis_password: ""      # 空则复用 redis_config
  signing_key: ""         # OIDC 签名私钥 PEM（非 dev 必填）
  encryption_key: ""      # 会话/令牌加密密钥（非 dev 必填）
  cookie_secure: false    # SSO 会话 cookie Secure 标志（生产 HTTPS 必须 true）
  # —— 通用 / rp 专有 ——
  issuer: ""              # op: 空则按端口推导；rp: 远端 OP issuer（必填）
  jwks_url: ""            # rp: 空则由 issuer 走 discovery 推导
  audience: ""            # rp: 本应用 client_id，用于 aud 校验（建议必填）
  # —— 兼容字段 ——
  # enable: false         # 旧字段，保留解析：enable:true 且 mode 为空 → 视为 op
```

- **字段归一**：`IAMConfig.EffectiveMode()` 决定实际形态——
  `mode` 非空则以之归一（只接受 `disabled`/`op`/`rp`，非法值报错）；
  `mode` 空且 `enable:true` → `op`；否则 `disabled`。
- **兼容旧配置**：`enable:true` 自动映射到 `op`，平滑升级、无需立即改配置。
- **切换形态只改 `config.yaml` + 前端环境变量**，不改任何 `app.go` 分支逻辑（分支收敛在 `SetupIAM` / `buildAuthMiddleware` 两处装配点）。

### 各态要点

| | `disabled` | `op` | `rp` |
|---|---|---|---|
| `SetupIAM` 返回 | `(nil, nil)`，零副作用 | `io.Closer`（内嵌 OP 实例） | `io.Closer`（远端 keys 句柄） |
| 是否建库/迁移 | 否 | 是（`RunMigrations` 建 `iam_` 表） | **否** |
| 是否挂 `/iam/*` | 否（404） | 是（`Mount` + `MountConfig`） | **否** |
| KeySource 来源 | 无 | `app.KeySource()`（进程内公钥） | `rp.NewKeys(...)`（远端 JWKS） |
| 业务鉴权 | dev 演示身份 / 非 dev 无身份 | OIDC 本地验签 | OIDC 本地验签（与 op 相同） |
| 必填项 | 无 | `prefix`、非 dev 安全项 | `issuer`（必填）、`audience`（建议） |

---

## 3. 后端装配：单一入口 `SetupIAM`（`iam_setup.go`）

`SetupIAM` 收敛为唯一装配点，按 `EffectiveMode()` 分派：

```go
type iamRuntime struct {
    mode      string              // op | rp
    app       *server.App         // op 时非 nil
    keySource contract.KeySource  // op: app.KeySource()；rp: rp.NewKeys(远端)
    issuer    string
    closer    io.Closer
}

// disabled 返回 (nil, nil)，零副作用；未知 mode 报错。
func SetupIAM(engine *gin.Engine) (io.Closer, error) {
    switch cfg.EffectiveMode() {
    case "disabled": currentIAM = nil; return nil, nil
    case "op":       return setupOP(engine)   // 见下
    case "rp":       return setupRP()         // 见下
    default:         return nil, fmt.Errorf("unknown iam.mode: %q", cfg.Mode)
    }
}
```

**`setupOP`（本进程内嵌 OP）**——顺序为 `server.New` → `RunMigrations` → `Mount` → `MountConfig`：

```go
app, err := server.New(server.Config{
    Env:          config.Conf.Server.Env,
    DB:           server.DBConfig{DSN: dsn, Service: "iam"},
    Redis:        resolveIAMRedis(cfg),
    Domains:      server.DefaultConsoleDomains, // core-auth/install/person/user/department/role/menu
    SameDatabase: true,
    Raw:          buildRawConfig(issuer, cfg),  // issuer/signing_key/encryption_key/cookie_secure 等
})
app.RunMigrations(ctx)                        // 建 iam_ 前缀表
app.Mount(engine.Group(prefix))               // 挂 /oidc、/install、/v1/{platform,tenant,auth}
app.MountConfig(engine, prefix)               // 仅提供 {prefix}/config.json（不再托管前端 SPA）
rt.keySource = app.KeySource()                // 类型即 contract.KeySource
```

**`setupRP`（对接外部 ark-iam，仅验签）**——不建库、不 `Mount`、不 `RunMigrations`：

```go
ks, err := rp.NewKeys(ctx, resolveJWKSURL(cfg))  // jwks_url 为空则由 issuer 走 discovery
rt.keySource = ks
rt.issuer    = cfg.Issuer
```

**鉴权中间件装配（`app.go` 的 `buildAuthMiddleware`）**：op / rp 共用同一套 `OIDCAuth`，差异只在 KeySource 来源：

```go
switch config.Conf.IAM.EffectiveMode() {
case "op", "rp":
    rt := CurrentIAM()
    if rt == nil { panic("iam enabled but runtime nil") }   // fail-closed
    return middleware.OIDCAuth(
        middleware.WithOIDCKeySource(rt.keySource),
        middleware.WithOIDCIssuer(rt.issuer),
        middleware.WithLocalUserMapper(iamlink.NewSvc()),
        middleware.WithOIDCSkipPaths("/"+AppName+"/docs", "/v1/"+AppName+"/health"),
    )
default: // disabled
    if config.Conf.Server.Env == "dev" { return middleware.DemoIdentity() }
    return nil
}
```

> 这是本设计的核心收敛点：op 与 rp 的业务鉴权**完全相同**（都本地验签 + `iamlink` 映射 uint），**差异只在 KeySource 来源**。

---

## 4. 依赖：直接 `go get` 已发布的 ark-iam 模块

ark-iam 的三态能力已发布到 Go proxy，本项目**直接 `go get` 真实版本，无 `replace`、无本地 `go.work` 罩**：

```go
require (
    github.com/morehao/ark-iam/sdk      v0.5.0   // 门面 sdk/server + RP sdk/rp
    github.com/morehao/ark-iam/contract v0.1.0   // 契约 module
    github.com/morehao/ark-iam/pkg      v0.4.0   // 公共层（随 sdk 传递引入，通常无需直接 require）
)
```

- `backend/go.work` 只 `use ./apps/demo` 与 `./pkg`（本仓自己的模块），不涉及外部源码。
- 下游模板用户：`go mod download` 即可，无需任何本地源码准备。
- `sdk` 会传递引入 `backend/apps/{auth,platformadmin,tenantadmin}` 作为 indirect 依赖——这是 ark-iam 内部分层，消费者无需感知。

> 版本号如有更新，以后续 ark-iam 发布为准；`make test` 与 `make lint` 均基于已发布版本，CI 无需特殊处理。

---

## 5. 身份映射表 `biz_iam_user_map`（三态通用）

**为什么需要**：ark-iam 的 `rp.Identity.UserID` 是 **string**（如 `"user-2222"`），而本站 `operator.Operator.UserID` 与业务表审计列（`created_by/updated_by/deleted_by`）均为 **BIGINT（uint）**，golib 的 `gincontext.GetUserID()` 按 uint 解析（库函数不可改）。直接把 string 写进 `gcontext.KeyUserID` 会让审计列全解析成 0。

**解决**：一张映射表把「外部 string 身份」与「本地 uint 主体」解耦：

```
biz_iam_user_map
  iam_user_id VARCHAR(128) PRIMARY KEY   -- IAM 侧 rp.Identity.UserID
  local_uid   BIGINT NOT NULL DEFAULT nextval('biz_iam_user_map_local_uid_seq')
  person_id   VARCHAR(128) NOT NULL DEFAULT ''
  created_at / updated_at TIMESTAMPTZ
  UNIQUE INDEX uk_biz_iam_user_map_local_uid(local_uid)
```

- 写入走幂等 upsert（`ON CONFLICT (iam_user_id) DO UPDATE ... RETURNING local_uid`），单语句原子、并发安全。
- 首次登录自动分配 `local_uid`，已存在则原样返回；`pkg/iamlink.Svc.EnsureLocalUser` 是唯一入口，带 30 分钟 Redis 缓存。
- 该映射层**与 iam.mode 无关**（op/rp 都用它），op/rp 切形态不触碰 `iamlink`、业务表、审计列。

> 边界：本地 `local_uid` 与业务表主键是**两套 ID 空间**——映射表只负责把「谁」投影成本地 uint，业务侧对 IAM 零感知。

---

## 6. 表前缀与迁移边界（仅 op 相关）

- **IAM 域表**（`iam_` 前缀，如 `iam_person`/`iam_user`/`iam_tenant`/`iam_application_client`）：由 op 模式的 `RunMigrations`（AutoMigrate）在启用时**自动创建**——宿主项目**不要**把它们写进 `table_init.sql`。
- **业务表 + 映射表**（无 `iam_` 前缀）：由 `table_init.sql` 管。`biz_iam_user_map` 特意**不设为 AutoMigrate**（含独立序列与唯一索引，交由 DDL 精确控制）。
- **rp 模式**：**不建任何表**（`iam_` 表在外部 ark-iam 那边），本项目只需已存在的业务表 + `biz_iam_user_map`。
- 同库共存：op 用 `SameDatabase: true`，IAM 表与业务表在同一 PG 库；ark-iam 有同库撞名检测，届时按提示显式配置。

> op 首次在空库启用：先执行 `table_init.sql`（建业务表 + `biz_iam_user_map`），再以 `mode: op` 启动（`RunMigrations` 建 `iam_` 表）。

---

## 7. OIDC 鉴权中间件如何保护业务接口（op / rp 相同）

`mode` 为 `op` 或 `rp` 时，`app.go` 的 `buildAuthMiddleware` 返回 `middleware.OIDCAuth(...)`，挂在 `/{AppName}/v1` 路由组上（即所有 `/v1/demo/*` 业务接口）。流程：

1. 取 `Authorization: Bearer <access_token>`；缺失 → **401**（fail-closed）。
2. 用 `contract.KeySource` **本地验签**（op 是进程内公钥；rp 是远端 JWKS 缓存），并校验 `iss == issuer`；失败/过期/`aud` 不匹配 → 401。
3. 得到 `rp.Identity`（string UserID）→ 经 `iamlink.EnsureLocalUser` 映射出 `local_uid`（uint）→ 写 `gcontext.KeyUserID`（uint 可解析）、`KeyPersonID`、`KeyAuthToken`；映射失败 → 401。
4. 业务侧 `ginhelper.FillOperator` 照旧读这些 gin Keys 并按 uint 解析，**审计列非 0**。

**白名单**（免鉴权）：`/demo/docs`（swag）、`/v1/demo/health`（健康检查）。`/iam/*`（仅 op 有）不在业务路由组内，天然放行。

**验收口径**：无 token 调 `/v1/demo/products` → 401；有效 token → 200 且审计列 = 该用户 `local_uid`。

---

## 8. 前端接入（demo-web）

前端 `@morehao/ark-iam-react`（门面包）`createAuthProvider` / `useAuthGuard` 契约不变，op/rp 都走 OIDC 登录，差异只在**配置来源**。`src/auth/config.ts` 的 `detectIAMMode()`：

```ts
export type IAMMode = 'disabled' | 'op' | 'rp'
// 判定顺序：VITE_IAM_MODE 显式优先 → op 探测 /iam/config.json 可达 → rp 直连 VITE_OIDC_ISSUER → disabled
```

### op 路径（本进程内嵌 OP）

- 后端 `MountConfig` 挂出 `{prefix}/config.json`（默认 `/iam/config.json`），下发 `issuer` / `clientId` / `callbackPath` 等。
- 前端 `createAuthProvider` 用 **`runtimeConfigBase=/iam`** 探测并读取 `config.json`。
- `main.tsx` 用 `IAMAuthProvider` + `AuthGuard` 包裹；令牌经 `AuthGuard` 注入 `@ark-go-starter/api`（`setTokenProvider`），401 时 `removeUser` + `signinRedirect`。
- 登录页 / 控制台 UI 由 npm 包 `@morehao/ark-iam-react` 提供，**Go 不再托管前端静态资源**。

### rp 路径（对接外部 ark-iam）

- 不再从本进程探测 `config.json`；前端用 **`VITE_OIDC_ISSUER`** + **`VITE_OIDC_CLIENT_ID`** 直连**远端 ark-iam** 的 issuer。
- 登录页来自**外部 ark-iam 控制台**（不是本仓库前端），本仓库前端只是 OIDC client 端。
- `vite.config.ts` 里 `/iam` 代理**仅 op 需要**；rp 不需要（前端直连远端 issuer），保留也无害。

### 通用

- **redirect_uri 白名单**：门面包默认回调路径是 **`/auth/callback`**（`redirectPath=/auth/callback`，**不是** `/callback`）。demo-web 即 `http://localhost:3000/auth/callback`，必须登记到所用 OIDC 客户端 `redirect_uris` 白名单。
- **clientID**：op 下建议用门面包 `useRuntimeConfig: true` 从 `/iam/config.json` 下发，`VITE_OIDC_CLIENT_ID` 兜底；rp 下由 `VITE_OIDC_CLIENT_ID` 指定。
- op 内嵌客户端建议建 **public client（PKCE，`tokenEndpointAuthMethod=none`）**，把其 `code` 配到 clientId。

---

## 9. 生产安全项（非 dev fail-closed）

`mode` 为 `op` 且 `server.env != dev` 时，ark-iam SDK 在 `server.New` 内自动 `validateNonDevSecurityConfig`（**fail-closed**，宿主不再重复实现），必须显式配置，否则进程起不来：

| 配置 | 要求 |
|------|------|
| `iam.signing_key` | OIDC 签名私钥 PEM（RSA/EC），签发/验签 id_token |
| `iam.encryption_key` | 会话/令牌加密密钥（≥32 字节随机） |
| `iam.cookie_secure` | 必须 `true`（HTTPS 生产） |
| `iam.issuer` | 显式设为对外域名（`https://.../iam/oidc` 或反代后公网地址），须与 `Mount` 前缀一致 |

- **dev**：`signing_key` / `encryption_key` 可留空（IAM 自动生成/放行），`cookie_secure` 可为 false。
- **rp**：不跑 OP，无需 `signing_key` / `encryption_key`；但必须配 `issuer`（远端 OP 的 issuer，用于 `iss` 校验），建议配 `audience`（本应用 client_id，用于 `aud` 校验）。
- `config.prod.yaml` 的 `iam` 段用**占位值**，请用密钥管理 / 环境变量注入真实密钥，**勿把真实私钥提交入库**。

---

## 10. 常见坑（踩坑必读）

1. **`MountFrontend` 已删除**：v0.4 起 Go 侧不再托管任何前端静态资源，只保留 `MountConfig`（出 `/config.json`）。登录/控制台 UI 由 npm 包 `@morehao/ark-iam-react` 提供。不要再找 `MountFrontend`，也不要指望 `{prefix}/login` 由 Go 提供 SPA。
2. **`contract` 是独立 module**：import 路径从 `github.com/morehao/ark-iam/sdk/contract` 改为 **`github.com/morehao/ark-iam/contract`**。中间件、KeySource 类型都依赖这个独立路径。
3. **`iam` 模块已删除**：入口从 `iam.New` 改为 **`sdk/server.New`**。`New`/`Mount`/`RunMigrations`/`KeySource`/`Close` 名字基本 1:1 保留，只是包名变了。
4. **redirect_uri 白名单**：回调必须是 `/auth/callback`（不是 `/callback`），且登记在客户端 `redirect_uris`。内置种子客户端 `platform_admin_web` 默认只登记了控制台回调（如 `http://localhost:4001/auth/callback`），拿它跑 demo-web 会卡在回调对齐——需补登 `http://localhost:3000/auth/callback` 或另建 public client。
5. **issuer 必须与挂载前缀一致**：op 下 issuer 形如 `{scheme}://{host}{prefix}/oidc`，否则 OP 公布的 issuer 与端点不符，验签因 `iss` 不匹配失败。rp 下 `issuer` 必须等于远端 OP 的 issuer。
6. **依赖**：`sdk`/`contract`/`pkg` 已发布到 proxy，直接 `go get`（`sdk v0.5.0` / `contract v0.1.0` / `pkg v0.4.0`）；无需 `replace`，`go.work` 只 `use` 本仓模块。

---

## 11. 从 v0.3.0（`iam` 模块）迁移的 old → new 映射表

| 维度 | 旧（v0.3.0，`iam` 模块） | 新（v0.4，三态 `sdk`） |
|------|--------------------------|------------------------|
| 开关 | `iam.enable: bool`（内嵌/关闭二态） | `iam.mode: disabled\|op\|rp`（三态；`enable:true` 兼容映射 `op`） |
| 内嵌服务端入口 | `import ".../ark-iam/iam"`；`iam.New(iam.Config{...})` | `import ".../ark-iam/sdk/server"`；`server.New(server.Config{...})` |
| 建表 | `app.RunMigrations(ctx)` | 不变（同名方法） |
| 挂路由 | `app.Mount(engine.Group(prefix))` | 不变（同名方法） |
| 前端托管 | `app.MountFrontend(engine, prefix)`（Go embed 托管 SPA） | **删除**；改 `app.MountConfig(engine, prefix)`（仅 `/config.json`），登录 UI 由 npm `@morehao/ark-iam-react` 提供 |
| 默认域集 | `iam.DefaultConsoleDomains` | `server.DefaultConsoleDomains`（值相同） |
| KeySource | `app.KeySource()`（类型 `rp.KeySource`） | `app.KeySource()`（类型 `contract.KeySource`） |
| 契约类型 import | `.../ark-iam/sdk/contract` | `.../ark-iam/contract`（独立 module） |
| 中间件 KeySource 字段 | `rp.KeySource` | `contract.KeySource`（中间件只依赖契约） |
| rp 验签 | `sdk/rp.NewKeys`/`NewVerifier` | 不变（`sdk/rp` 接口稳定） |
| rp 形态 | 不支持（只有内嵌/关闭） | 新增 `setupRP()`：仅 `rp.NewKeys`，不建库不挂路由 |
| 生产安全校验 | 宿主/`iam.New` 内 `validateNonDevSecurityConfig` | SDK 在 `server.New` 内自动 fail-closed，宿主不再重复 |
| Go 依赖模块 | `ark-iam/iam` + `ark-iam/pkg` + `ark-iam/sdk` v0.3.0 | `ark-iam/sdk` v0.5.0 + `ark-iam/contract` v0.1.0 + `ark-iam/pkg` v0.4.0（均已发布，直接 go get） |
| 鉴权中间件 import | `.../ark-iam/sdk/contract` | `.../ark-iam/contract` |

**后端调用面 80% 可平移**（`New/Mount/RunMigrations/KeySource/Close` 名字不变，仅包名 `iam` → `sdk/server`）；真正的破坏点是：① `MountFrontend` 删除、② `contract` 独立 module 的 import 路径、③ `go.mod` 模块集合变化。

---

## 12. 常用操作

**op 本地联调**：`config.yaml` 设 `iam.mode: op`（或旧 `iam.enable: true`），`BOOTSTRAP_TOKEN=xxx make run APP=demo`，走 `/iam/install` 初始化管理员 → 建 client → 前端登录。联调完还原 `mode: disabled`。

**/install 初始化**（op，需引导令牌，头 `X-Bootstrap-Token`，值取环境变量 `BOOTSTRAP_TOKEN` 或配置 `install.bootstrapToken`）：

```bash
# 状态（公开只读）
curl -s http://localhost:8099/iam/install/status
# 初始化平台管理员（口令有强度校验；adminEmail/adminPhone 至少一项）
curl -s -X POST http://localhost:8099/iam/install/initialize \
  -H 'Content-Type: application/json' -H "X-Bootstrap-Token: $BOOTSTRAP_TOKEN" \
  -d '{"adminUsername":"admin","adminPassword":"Adm1n@Strong!Pass","adminEmail":"admin@example.com","tenantName":"Acme"}'
```

**rp 联调**：`config.yaml` 设 `iam.mode: rp` + `issuer`（远端 ark-iam），前端设 `VITE_IAM_MODE=rp` + `VITE_OIDC_ISSUER` + `VITE_OIDC_CLIENT_ID`；不建库、不挂 `/iam/*`，登录跳远端 ark-iam。

**校验三态装配**（单测覆盖）：disabled 无中间件/零副作用；op 有 KeySource；rp 用 `rp.NewKeys`（mock JWKS）；非法 `mode` 报错。

> 验证矩阵与端到端实际结果见 [.dsh/docs/notes/2026-11-09-acceptance-report.md](../../.dsh/docs/notes/2026-11-09-acceptance-report.md)。
