# golib 公共组件库能力说明

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

本项目的通用基础能力（日志、DB 客户端、错误码、上下文、路由分组等）统一复用 [`github.com/morehao/golib`](https://github.com/morehao/golib)，**写新功能前先查本页，能复用的不要自己造轮子**。当前版本以 `backend/pkg/go.mod` 中声明为准（现为 `v1.34.0`）。

破坏性变更与升级记录见 [docs/golib-变动汇总.md](../golib-变动汇总.md)（升级 golib 版本时必读，并追加记录）。

## 项目内已封装的公共层（backend/pkg）

优先使用项目自己的封装，它们是对 golib 的薄封装（多租户/多应用初始化、连接管理）：

| 包 | 能力 | 关键 API |
|----|------|----------|
| `pkg/code` | 业务错误码（基于 golib `gerror`） | `code.GetError(code.XXXError)`；每类资源一个文件（`user.go`、`org.go`…） |
| `pkg/dbclient` | PostgreSQL / Redis 连接管理（基于 `dbgorm`/`dbredis`） | `dbclient.DemoDB(ctx)`、`dbclient.GetDB(ctx, name)`、`dbclient.RedisCli`、`dbclient.DemoDB(ctx).Transaction(func(tx *gorm.DB) error)` |
| `pkg/token` | Token 处理 | 见包内实现 |
| `pkg/testsetup` | 测试环境初始化（基于 golib `testkit`） | `Initialize/Close`、`NewContext(WithUserID…)`，详见 [pkg/testsetup/README.md](../../backend/pkg/testsetup/README.md) |

## golib 常用组件速查（本项目已使用 ✅ / 可直接引入 ⬜）

| 组件 | 状态 | 用途 | 备注 |
|------|------|------|------|
| `biz/gcontext/gincontext` | ✅ | Controller 统一返回 | `gincontext.Success/Fail(ctx, ...)`，见硬性规则 6 |
| `biz/gserver/ginserver` | ✅ | 路由分组工厂 `/v1/{app}`，自动挂 otel + 访问日志中间件 | `NewRouterGroups(engine, appName, versions...)` |
| `biz/gserver/gindocs` | ✅ | Swagger 文档挂载 | |
| `biz/gobject` | ✅ | UserClaims / OperatorBaseInfo / PageQuery 等通用对象 | DTO 中分页参数直接用 `gobject.PageQuery` |
| `dbaccess/dbgorm` | ✅ | GORM MySQL/PostgreSQL 客户端 | 驱动子包需 blank import：`dbgorm/driver/postgres` |
| `dbaccess/gormdao` | ✅ | 泛型 DAO（原 `genericdao`） | `gormdao.NewDao[T,L](table, name, getDB)`，见 [变动汇总](../golib-变动汇总.md) |
| `dbaccess/dbredis` | ✅ | Redis 客户端 | |
| `dbaccess/dbes` | ⬜ | Elasticsearch 客户端 | 本项目已移除 ES 集成（无 DDL、无健康检查项），需要时再引入 |
| `glog` | ✅ | 日志（zap 默认 / slog 双驱动） | 驱动需 blank import `glog/driver/zap`；配置格式见变动汇总 |
| `gtrace/otel` | ✅ | OpenTelemetry 链路初始化 | OTLP gRPC 导出，`otlptracegrpc` 子包 |
| `gerror` + `gconstant` | ✅ | 业务错误码封装 | 错误码 Map 已迁至顶层 `gconstant` |
| `gutil` | ✅ | 常用工具函数（随机、字符串、时间、slice/map） | |
| `protocol/ghttp` | ✅ | HTTP 客户端（结构体映射、智能重试：4xx 不重试 5xx 重试） | 参考 `apps/demo/client/httpbingo` |
| `protocol/gresty` | ⬜ | Resty 封装，支持 SSE | 服务端 SSE 示例见 `ctrexample/sse.go` |
| `biz/gserver/ginupload` | ✅ | 文件上传中间件 | |
| `biz/gserver/gindict` | ✅ | 数据字典（`dict`）管理面路由：类型 / 码值 / 树 | demo 已在 v1 分组注册，组件实例于启动期初始化 |
| `biz/gserver/ginsetting` | ✅ | 配置中心（`gsetting`）管理面路由 | demo 已在 v1 分组注册，组件实例于启动期初始化 |
| `biz/gserver/gincron` | ✅ | 定时任务（`task/gcron`）查询 + 运维动作路由 | demo 已在 v1 分组注册，组件实例于启动期初始化 |
| `biz/gserver/ginasync` | ✅ | 异步任务（`task/gasync`）查询 + 运维动作路由 | demo 已在 v1 分组注册，组件实例于启动期初始化 |
| `filestore` + `storage` | ✅ | 文件业务层：元数据落库、内容哈希去重、分片上传、预签名 URL | `storage/driver/local` 已注册；另有 S3 适配 |
| `biz/gmiddleware` | ⬜ | JWT 认证、CORS、访问日志、Token 黑名单中间件 | |
| `gauth/jwtauth` | ⬜ | 泛型 JWT 签发/解析/续签 | |
| `gcrypto` | ⬜ | AES(GCM/CBC)、RSA、bcrypt | 测试中已有使用 |
| `biz/testkit` | ✅ | 测试初始化与上下文构建 | 由 `pkg/testsetup` 封装 |
| `dict` | ✅ | 数据字典（类型+码值+树形），替代硬编码枚举 | 经 `biz/gserver/gindict` 接入；共库时 `code` 全表唯一，需约定命名；默认自动建表，可用 `WithoutAutoMigrate()` |
| `gsetting` | ✅ | 数据库键值配置存储（支持加密，原 `configkv`） | 经 `biz/gserver/ginsetting` 接入；v1.34.0 起由 `configkv` 改名，表 `core_config`→`core_setting` |
| `distlock` | ⬜ | Redis 分布式锁（redsync，自动续期，不可重入） | |
| `ratelimit` | ⬜ | 分布式限流（GCRA），Redis 故障自动降级本地令牌桶 | |
| `gconc` | ⬜ | 固定 worker 并发任务池（panic 安全、优雅关闭） | |
| `task/gcron` | ✅ | 定时任务：秒级 cron、多实例分布式锁互斥、执行记录落库 | 经 `biz/gserver/gincron` 接入 |
| `task/gasync` | ✅ | 异步任务：基于 asynq，重试/延迟/优先级队列/trace 传递 | 经 `biz/gserver/ginasync` 接入 |
| `gvalidate` | ⬜ | 参数校验封装 | |
| `gtree` | ⬜ | 树构建（实现 GetKey/GetParentKey/IsRoot 即可） | |
| `gexcel` | ⬜ | Excel 读写（结构体 tag 映射 + validator 校验） | |
| `gllm` | ⬜ | LLM 接入配置层（provider/model 配置 → eino ChatModel） | 需要 AI 能力时使用 |
| `codegen` | ✅（经 `make codegen`） | 读表结构生成 CRUD 代码 | 见 [commands.md](./commands.md#代码生成) |

## 使用约定

1. **先查再写**：需要日志、HTTP 调用、加解密、分布式锁、限流、定时/异步任务、Excel、树构建等通用能力时，先在本页和 [golib README](https://github.com/morehao/golib) 确认是否已有组件；各组件详细用法见 golib 仓库内对应子包 README。
2. **Import 分组**：golib 统一放 Import 的最后一组，规则见 [backend-conventions.md](./backend-conventions.md#import-排序)。
3. **驱动注册**：`glog/driver/*`、`dbgorm/driver/*` 等驱动子包必须 blank import（`_ "..."`）才会生效，参照 `apps/demo/cmd/init.go` 与 `pkg/dbclient/gorm.go`。
4. **内置表与 DDL**：golib 的 `dict`/`gsetting`/`filestore`/`task/*` 自带内置表，默认 `New` 时幂等 AutoMigrate；本模板接入的 `gindict`/`ginsetting`/`gincron`/`ginasync` 四个 gserver 路由模块，其组件实例在启动期用 demo 主库（PostgreSQL）初始化，**内置表默认 `AutoMigrate` 建表**。共库或无 DDL 权限时传 `WithoutAutoMigrate()` 并走发布流程显式迁移。本项目唯一目标库是 PostgreSQL，接入前确认其 DDL 支持 PostgreSQL。
5. **升级**：改 golib 版本用 `make update-dep ARGS="-u github.com/morehao/golib@vX.Y.Z"`，升级后跑 `go build ./... && go vet ./...`，并在 [docs/golib-变动汇总.md](../golib-变动汇总.md) 追加变更记录。
