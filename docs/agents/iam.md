# 可选账号认证（ark-iam · RP 接入方，两态 disabled / rp）

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)
>
> 本模板接入 [ark-iam](https://github.com/morehao/ark-iam) 的**唯一可选形态是 rp（Relying Party）**：
> 对接**外部** ark-iam 服务做 OIDC 本地验签。默认 `disabled`（开箱免登录）。

---

## 1. 概述

本模板支持**可选接入** [ark-iam](https://github.com/morehao/ark-iam) 账号认证体系。是否接入由
`config.yaml` 的 `iam.mode` 决定（留空回落 `disabled`，对业务零副作用）。接入形态只有两种：

- **disabled**（默认）：不接入，dev 下用固定演示身份，开箱免登录。
- **rp**（可选）：作为**依赖方（Relying Party）**对接**外部 ark-iam 服务**，仅在本地用其公钥做 OIDC 验签。

> **op（内嵌 OIDC Provider）已下线**：早期版本支持在本进程内嵌整台 ark-iam 服务端（建 `iam_` 表、
> 挂 `/iam/*` 路由、`/iam/install` 初始化管理员、前端跳 `/iam/login`）。该形态已整体移除，本模板
> **不再作为 IAM 服务端**。需要本地自举一套 IAM 请移步 ark-iam 仓库：
> <https://github.com/morehao/ark-iam>。

术语：**OP** = OpenID Provider（签发/验签 token 的服务端，即外部 ark-iam）；
**RP** = Relying Party（业务应用，用 OP 的公钥验签）。本模板只扮演 RP。

| `iam.mode` | 是否引 SDK | 是否建库 / 挂 `/iam/*` | 登录页来自 | 一句话 |
|---|---|---|---|---|
| `disabled` | 否（可完全不引） | 否 | 无（dev 演示身份） | 不接入认证，免登录 |
| `rp` | 是（仅 `sdk/rp` + `contract`） | **否** | 外部 ark-iam 控制台 | 对接外部 IAM，本地验签 |

---

## 2. 两态配置（`config.yaml` 的 `iam` 段）

```yaml
iam:
  mode: disabled     # disabled | rp（默认 disabled；留空回落到 disabled）
  issuer: ""         # rp：外部 OP 的 issuer（必填，用于 iss 校验）
  jwks_url: ""       # rp：JWKS 端点；留空则由 issuer 走 discovery/`/keys`/`.well-known/jwks.json` 推导
  audience: ""       # rp：本应用 client_id，用于 aud 校验（建议必填）
```

字段说明（仅 rp 相关四个字段）：

| 字段 | 必填 | 说明 |
|------|------|------|
| `mode` | 否 | `disabled`（默认）/ `rp`。留空回落 `disabled`；非法值 `Validate()` 报错。 |
| `issuer` | rp 必填 | 外部 OP 的 issuer，用于 `iss` 校验；`jwks_url` 留空时也用于推导 JWKS 地址。 |
| `jwks_url` | 否 | JWKS 端点。留空则由外部 OP 按标准 discovery 推导。 |
| `audience` | 建议必填 | 本应用 client_id，用于 `aud` 校验。 |

> 旧的 op 专有字段（`prefix` / `dsn` / `redis_addr` / `redis_password` / `signing_key` /
> `encryption_key` / `cookie_secure` / `redirect_uris` / `post_logout_redirect_uris` /
> `frontend_url` / `install.bootstrapToken` 等）已随 op 下线，**不再使用**。切换形态只改
> `config.yaml` + 前端环境变量，不改任何后端分支逻辑。

---

## 3. 后端接入（复用的是验签引擎，不是框架）

相关源码（均在 `apps/api`）：

| 文件 | 职责 |
|------|------|
| `config/config.go`（`IAMConfig`） | iam 配置结构、`Validate()`、`EffectiveMode()` 归一（`disabled`/`rp`） |
| `iam_setup.go`（`SetupIAM`） | 单一装配入口：按 `mode` 分派 disabled / rp，返回 `io.Closer` |
| `app.go`（`buildAuthMiddlewares`） | rp 时返回两层鉴权链 `oidcmw.OIDCAuth` + `iamlink.Middleware`；disabled 回落 dev 演示身份 |
| `pkg/oidcmw/*` | 通用 gin+OIDC 薄适配（取 Bearer → `rp.Verifier.Verify` → `rp.WithIdentity` 注入身份），只依赖 `sdk/rp`+`contract`，不含业务语义 |
| `pkg/iamlink/middleware.go` | 从 `rp.IdentityFrom` 取 identity → `EnsureLocalUser` → 写 `gcontext.Key*`（IAM string → 本地 BIGINT 映射） |
| `pkg/iamlink/*` | IAM string 身份 ↔ 本地 BIGINT 用户 ID 的映射层 |

**`SetupIAM`（`iam_setup.go`）两态分派**：

```go
switch cfg.EffectiveMode() {
case config.ModeDisabled:
    currentIAM = nil
    return nil, nil                       // 零副作用
case config.ModeRP:
    return setupRP()                      // 仅构造远端 JWKS 密钥来源
default:
    return nil, fmt.Errorf(...)           // Validate 已拦截，兜底
}
```

**`setupRP`**：不建库、不 `Mount`、不 `RunMigrations`，只 `rp.NewKeys` 预取一次 JWKS（fail-fast）：

```go
keys, err := rp.NewKeys(ctx, jwksURL)      // jwksURL = jwks_url 或 issuer（discovery 推导）
rt := &iamRuntime{keySource: keys, issuer: cfg.Issuer, closer: rpKeysCloser{keys}}
currentIAM = rt
```

> **重点**：本模板复用的是 ark-iam 的**验签引擎**（`sdk/rp` 的 `rp.NewKeys` + `rp.NewVerifier`、
> `contract.KeySource` 契约接口），**而不是 ark-iam 的框架/路由**。鉴权拆成两层、均为**本仓自研**：
> `pkg/oidcmw`（通用 gin 适配，取 token → `rp.Verifier.Verify` → `rp.WithIdentity` 注入身份 → fail-closed 401）
> 与 `pkg/iamlink/middleware.go`（从 `rp.IdentityFrom` 取 identity → `EnsureLocalUser` → 写 `gcontext.Key*`，
> 完成 string→BIGINT 映射）。前者只依赖框架无关的 `contract.KeySource` 接口，从而斩断对 `sdk/server` 的依赖。

**中间件装配（`app.go` 的 `buildAuthMiddlewares`）**：返回 `[]gin.HandlerFunc`，rp 时拼接「验签 + 映射」两层：

```go
func buildAuthMiddlewares() []gin.HandlerFunc {
    if config.Conf.IAM.EffectiveMode() == config.ModeRP {
        rt := CurrentIAM()
        if rt == nil { panic("iam enabled but runtime is nil") }   // fail-closed

        // 验签器在装配层构造一次（RS256 + iss + aud）；中间件只负责取 token + 调 SDK。
        verifierOpts := []rp.VerifierOption{rp.WithAlgorithms("RS256")}
        if rt.issuer != "" {
            verifierOpts = append(verifierOpts, rp.WithIssuer(rt.issuer))
        }
        if config.Conf.IAM.Audience != "" {
            verifierOpts = append(verifierOpts, rp.WithAudiences(config.Conf.IAM.Audience))
        }
        verifier := rp.NewVerifier(rt.KeySource(), verifierOpts...)

        oidcMW := oidcmw.OIDCAuth(verifier,
            oidcmw.WithSkipPaths("/"+AppName+"/docs", "/v1/"+AppName+"/health"),
        )
        iamMW := iamlink.Middleware(rt.issuer)                     // 从 rp.IdentityFrom 取 identity 映射并写 gcontext.Key*
        return []gin.HandlerFunc{oidcMW, iamMW}
    }
    // disabled：dev 回落演示身份；非 dev 返回 nil（无身份）
    if config.Conf.Server.Env == "dev" {
        return []gin.HandlerFunc{middleware.DemoIdentity()}
    }
    return nil
}
```

**验签流程**（挂在 `/{AppName}/v1` 业务路由组）：

1. 取 `Authorization: Bearer <access_token>`；缺失 → **401**（fail-closed）。（`pkg/oidcmw`）
2. 用 `contract.KeySource`（远端 JWKS 缓存）**本地验签**，校验 `iss == issuer`；失败/过期/`aud` 不匹配 → 401，成功后 `rp.WithIdentity` 把 `rp.Identity` 注入 request context。（`pkg/oidcmw`）
3. 从 `rp.IdentityFrom(ctx)` 取回 `rp.Identity`（string UserID）→ 经 `iamlink.EnsureLocalUser` 映射出 `local_uid`（uint）→ 写
   `gcontext.KeyUserID`、`KeyPersonID`、`KeyAuthToken`；映射失败 → 401。（`pkg/iamlink/middleware.go`）
4. 业务侧 `ginhelper.FillOperator` 照旧按 uint 解析，**审计列非 0**。

---

## 4. 身份映射 `biz_iam_user_map`

**为什么需要**：ark-iam 的 `rp.Identity.UserID` 是 **string**，而本站 `operator.UserID` 与业务审计列
（`created_by/updated_by/deleted_by`）均为 **uint（BIGINT）**，`gincontext.GetUserID()` 按 uint 解析（库函数不可改）。
直接把 string 写进 `gcontext.KeyUserID` 会让审计列全解析成 0。

**解决**：一张映射表把「外部 string 身份」与「本地 uint 主体」解耦（DDL 见 `scripts/sql/table_init.sql`）。

> **复合键（另一任务升级）**：映射表主键正由单一 `iam_user_id` 升级为 **(issuer, iam_user_id) 复合键**，
> 以支持**多 IDP**（同一 string user id 在不同 issuer 下是不同主体）。届时 upsert 冲突键改为
> `(issuer, iam_user_id)`，`pkg/iamlink` 的缓存 key 也需带上 issuer。本文档描述以升级后口径为准。

- 写入走**幂等 JIT upsert**（首次登录即插即映射，`ON CONFLICT ... DO UPDATE ... RETURNING local_uid`），单语句原子、并发安全。
- 首次登录自动分配 `local_uid`，已存在则原样返回；`pkg/iamlink.Svc.EnsureLocalUser` 是唯一入口（带**可选** Redis 缓存，无 Redis 时自动回源 DB，见 §6「Redis 依赖边界」）。
- **边界**：本地 `local_uid` 与业务表主键是两套 ID 空间——映射表只负责把「谁」投影成本地 uint，业务侧对 IAM 零感知。
- **不建任何 `iam_` 表**：rp 形态下 IAM 域表在外部 ark-iam 那边；本仓 `table_init.sql` 只管业务表与 `biz_iam_user_map`。

---

## 5. 前端接入（rp）

前端 `@morehao/ark-iam-react`（门面包）只作为 OIDC client 端，直连**外部 ark-iam** 的 issuer：

- 用 **`VITE_OIDC_ISSUER`** + **`VITE_OIDC_CLIENT_ID`** 指定远端 issuer 与 client_id。
- `createAuthProvider(useRuntimeConfig: false)`：**不从本进程探测 `/iam/config.json`**（op 已下线，无该端点），
  配置全部来自上面的环境变量。
- 登录页来自**外部 ark-iam 控制台**（不是本仓库前端）。
- **个人中心 / 登出**复用门面包组件（`ProfileCenter` / `useLogout`），`main.tsx` 用 `IAMAuthProvider` +
  `AuthedApp`（内含 `AuthGuard`）包裹，令牌经 `setTokenProvider` 注入 `@ark-go-starter/api`，401 时
  `removeUser` + `signinRedirect`。

---

## 6. 依赖

```go
require (
    github.com/morehao/ark-iam/sdk/rp   // 常驻（rp 验签引擎）
    github.com/morehao/ark-iam/contract // 常驻（KeySource / Identity 契约，独立 module）
)
```

- **不再依赖 `sdk/server`**（op 服务端随下线移除）。
- 版本号以后续 ark-iam 发布为准；`make test` 与 `make lint` 均基于已发布版本，CI 无需特殊处理。

### Redis 依赖边界（重要）

- **rp 接入 ark-iam 不需要 ark-iam 的 Redis**：`rp.NewKeys` / `rp.NewVerifier` 是**无状态本地验签**
  （JWKS 缓存在进程内存，按 TTL 后台刷新），源码不引用任何 Redis。ark-iam 的 Redis 只属于已下线的
  op（服务端）本身——那是另一套、与 rp 无关的基础设施。
- **本仓 `iamlink` 的 Redis 缓存是可选的、fail-open**：它缓存 `(issuer, iam_user_id) → local_uid` 映射
  以提速，`dbclient.RedisCli == nil`（未配置 Redis）时自动**回源 DB**，功能不受影响，仅每次鉴权多一次
  查询。属于本仓库自己的 Redis，非 ark-iam 的。
- 一句话：**作为 RP，两个 Redis 都不是硬依赖**——ark-iam 的 Redis 完全不涉及，本仓 iamlink 缓存缺省
  也能正常工作。

---

## 7. 常见坑 / 验收口径

**验收口径**（rp 形态）：
- 无 token 调 `/v1/demo/products` → **401**（fail-closed）。
- 有效 token → **200**，且审计列 = 该用户映射出的 `local_uid`（非 0）。

**坑**：
1. **白名单**：`/demo/docs`（swag）与 `/v1/demo/health`（健康检查）免鉴权；rp 无 `/iam/*` 路由，不要再找它。
2. **`issuer` 必须等于外部 OP 公布的 issuer**，否则验签因 `iss` 不匹配失败。
3. **`audience` 建议必填**：不配则 `aud` 校验放空，有安全风险。
4. **`jwks_url` 留空**依赖外部 OP 的标准 discovery / `/keys` / `.well-known/jwks.json` 端点可达。
5. **redirect_uri 白名单**：在**外部 ark-iam 的该 client**上登记本应用回调（门面包默认 `/auth/callback`）。
6. **本地验签、不回调外部**：每个请求都只用本地缓存的 JWKS 验签，外部 OP 短暂不可用不影响已签发的 token。

> **rp 联调**：`config.yaml` 设 `iam.mode: rp` + `issuer`（远端 ark-iam）+ `audience`；前端设
> `VITE_OIDC_ISSUER` + `VITE_OIDC_CLIENT_ID`。不建库、不挂 `/iam/*`，登录跳外部 ark-iam。
> 需要本地自举 IAM（op 能力）请移步 ark-iam 仓库：<https://github.com/morehao/ark-iam>。
