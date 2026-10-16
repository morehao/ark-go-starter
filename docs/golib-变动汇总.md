# golib 变动汇总

本文件记录 `morehao/golib` 重构后的破坏性变更，供其他依赖 golib 的项目（ark-go-starter 及其后续应用）升级参考。

## 一、包路径迁移

| 旧路径 | 新路径 | 说明 |
|--------|--------|------|
| `golib/biz/genericdao` | `golib/dbaccess/gormdao` | DAO 层，含 `BaseCond` / `Dao` |
| `golib/biz/gconstant` | `golib/gconstant` | 顶层，含错误码 Map 与常量 |
| — | `golib/glog/driver/slog`、`golib/glog/driver/zap` | glog 注册驱动子包 |
| — | `golib/dbaccess/dbgorm/driver/mysql`、`/postgres`、`/sqlite` | dbgorm 注册驱动子包 |

> 注：`golib/biz/gserver`、`golib/biz/gmiddleware`、`golib/biz/gcontext`、`golib/biz/gobject`、`golib/biz/testkit` 路径未变。

## 二、API 变更

### 1. genericdao → gormdao（API 改名）

| 旧 | 新 |
|----|----|
| `genericdao.GenericDao[T, L]` | `gormdao.Dao[T, L]` |
| `genericdao.NewGenericDao[T,L](tableName, daoName, getDB)` | `gormdao.NewDao[T,L](tableName, daoName, getDB DBGetter, opts...)` |
| `genericdao.GenericDao:`（内嵌字段名） | `gormdao.Dao:` |
| `genericdao.BaseCond` | `gormdao.BaseCond` |

- `DBGetter` 类型：`func(context.Context) *gorm.DB`，与 `dbclient.DemoDB` 等签名一致。
- 新增可选参 `gormdao.WithoutSoftDelete()`。
- **重要**：`genericdao.DBErrorMsgMap` 已移到顶层 **`golib/gconstant`**，代码中需改用 `gconstant.DBErrorMsgMap`。

### 2. ginserver 路由分组

| 旧 | 新 |
|----|----|
| `ginserver.Version{Name: ...}` | `ginserver.VersionGroup{Version: ...}` |

`NewRouterGroups(engine, appName, versions ...VersionGroup)` 参数类型已变更，字段 `Name` 改为 `Version`。

### 3. glog 配置格式（破坏性变更）

`glog.LogConfig` 结构体重构，`Writer` 单字段改为 `Writers` 切片，日志轮转字段下沉到 `WriterConfig`。

#### LogConfig 顶层字段变更

| 旧字段 | 变更 |
|--------|------|
| `Writer WriterType` | **删除**，改为 `Writers []WriterConfig` |
| `Dir string` | **删除**，移入 `WriterConfig.Dir` |
| `MaxSize int` | **删除**，移入 `WriterConfig.MaxSize` |
| `MaxBackups int` | **删除**，移入 `WriterConfig.MaxBackups` |
| `MaxAge int` | **删除**，移入 `WriterConfig.MaxAge` |
| `Compress bool` | **删除**，移入 `WriterConfig.Compress` |
| `LoggerType LoggerType` | **新增**，可选 `"zap"`（默认）/ `"slog"` |
| `Service` / `Module` | 新增 `json` / `yaml` tag |

#### WriterConfig 结构体（新增）

```go
type WriterConfig struct {
    Type       WriterType `json:"type" yaml:"type"`            // console / file
    Level      Level      `json:"level" yaml:"level"`          // 单个 writer 级别（为空则继承全局）
    FileName   string     `json:"file_name" yaml:"file_name"`  // 日志文件名
    Dir        string     `json:"dir" yaml:"dir"`              // 目录（默认 "./logs"）
    MaxSize    int        `json:"max_size" yaml:"max_size"`    // MB（默认 100）
    MaxBackups int        `json:"max_backups" yaml:"max_backups"` // 默认 10
    MaxAge     int        `json:"max_age" yaml:"max_age"`      // 天（默认 7）
    Compress   bool       `json:"compress" yaml:"compress"`
    WfOnly     bool       `json:"wf_only" yaml:"wf_only"`      // 只输出 warn/fatal
}
```

#### YAML 配置迁移示例

**旧格式：**
```yaml
log:
  default:
    service: demo
    module: default
    level: info
    writer: file
    dir: ../../../log
    extra_keys:
      - requestId
```

**新格式：**
```yaml
log:
  default:
    service: demo
    module: default
    level: info
    writers:
      - type: file
        dir: ../../../log
    extra_keys:
      - requestId
```

支持多个 writer，例如同时输出控制台和文件：
```yaml
    writers:
      - type: console
        level: debug
      - type: file
        dir: ../../../log
        level: info
```

### 4. 常量位置

> `golib/gconstant` 中**没有** `ApiVersionV1` 等版本常量，它们位于 `golib/biz/gserver/ginserver`（如 `ginserver.ApiVersionV1 = "v1"`）。

## 三、初始化注册机制（重点）

### 1. glog 注册机制

glog 改为通过 driver 子包的 `init()` 注册 LoggerType 实现：

```go
// 顶层 glog
func RegisterLoggerType(t LoggerType, factory LoggerFactory)

// 内部依据 cfg.LoggerType 从 registeredFactories 匹配
// 空值默认为 LoggerTypeZap（注意：曾为 slog，已改为 zap）
// 未注册对应 driver 会报错：
// "unknown LoggerType X, import glog/driver/slog or glog/driver/zap to register"
```

升级时**必须** blank import 对应驱动，否则 `glog.InitLogger` / `glog.GetDefaultLogger` 报错：

```go
import _ "github.com/morehao/golib/glog/driver/zap"   // 默认 zap（推荐）
// 或
import _ "github.com/morehao/golib/glog/driver/slog"  // slog（配置需显式设 logger_type: slog）
```

- `InitLogger(cfg *LogConfig, ...)` 接口不变，`LogConfig` 新增 `LoggerType` 字段（yaml 可配 `logger_type`）。
- **陷阱**：`cmd` 入口 blank import 只覆盖应用进程；**测试初始化的路径（如 testsetup / testkit）也必须 blank import**，否则测试跑 `glog.InitLogger` 时 panic。

### 2. glog caller 定位语义重构（重点）

glog 的 `WithCallerSkip` 语义已彻底重写，以保证不同 driver（zap / slog）与不同调用方式（包级函数 or 直接调 Logger 方法）下，相同的 skip 都定位到同一处业务代码：

- **删除** `glog.CallerFrame(...)`（此前基于运行时查帧定位调用点的 API）。
- **新增** `glog.CallerOffsetLogger` 接口，内置驱动实现之：
  ```go
  type CallerOffsetLogger interface {
      LogDepth(ctx context.Context, level Level, msg string, kvs []any, extra int)
  }
  ```
- **包级日志函数**（`glog.Debugf` / `glog.Infow` / ...）不再调用 `Logger` 接口方法，而是改调 `LogDepth(..., pkgEntryFrame=1)`（`pkgEntryFrame` 为包级函数相对"直接调 Logger 方法"多出的固定帧数）。**第三方自定义 driver 未实现 `CallerOffsetLogger` 时**，包级调用退回 `logEntryFallback`，只影响 caller 偏一帧。
- **`WithCallerSkip` 新语义**：skip 表示相对"调用 glog API 的那一帧"（包级函数或 Logger 方法）再向上额外跳过的帧数。驱动内部封装深度与包级入口帧已由 `glog` 固定常量抵消（driver base：zap=2、slog=3），因此该值**与 driver 无关**。
- **`glog.GetDefaultLogger` 不再传入默认 skip**（常量 `DefaultLogCallerSkip` 已删除），默认 getter 直接 `newLogger(GetDefaultLogConfig())`。
- **默认 `LoggerType` 为 zap**：`GetDefaultLogConfig()` 与 `newLogger` 兜底均返回 `LoggerTypeZap`。

#### db 各层 callerSkip 默认值校准

新语义下，各数据库封装层在业务直接调用 db 方法时的 default callerSkip 已重校准：

| 包 | 旧默认 | 新默认 |
|----|--------|--------|
| `dbgorm` | 8 | **3** |
| `dbredis` | 8 | **4** |
| `dbes` | 9 | **6** |

若业务在 db 调用前又套了额外封装层，才需用对应包的 `WithCallerSkip` 微调；业务直接调用时**建议不传**，直接使用校准默认。

- 新增 `glog.CloneLogConfig(cfg)`：返回配置的浅拷贝（`Writers`、`ExtraKeys` 独立切片）。`dbgorm`/`dbredis`/`dbes` 的 `New` 已改用 `CloneLogConfig(glog.GetLoggerConfig())` 跟随全局 logger 配置。
- `glog.Logger` 接口本身**未变**（含 `Debug/Debugf/.../With/Close/GetConfig`）。

### 3. dbgorm 注册机制

dbgorm 改为通过 driver 子包的 `init()` 注册 `DialectorFactory`，`New` 依据 URL 前缀匹配已注册的 dialector：

```go
func Register(name string, factory DialectorFactory)  // DialectorFactory: Name / MatchURL / Dialector / ParseURL

// 未注册匹配驱动报错：
// "no registered dialector matches url, make sure to import the driver (e.g. _ \".../dbaccess/dbgorm/driver/mysql\")"
```

升级时**必须** blank import 对应数据库驱动（例如 MySQL）：

```go
import _ "github.com/morehao/golib/dbaccess/dbgorm/driver/mysql"
```

## 四、本仓库升级记录（ark-go-starter）

ark-go-starter 本次迁移涉及的文件分类：

- **路径迁移**：`biz/genericdao` → `dbaccess/gormdao`、`biz/gconstant` → `golib/gconstant`。
- **类型改名**：`dbgorm.GormConfig` → `dbgorm.Config`；`ginserver.Version{Name}` → `ginserver.VersionGroup{Version}`。
- **符号迁移**：`genericdao.DBErrorMsgMap` → `gconstant.DBErrorMsgMap`；`gconstant.ApiVersionV1` → `ginserver.ApiVersionV1`。
- **新增 blank import**：`glog/driver/zap`（cmd 两处与 testsetup）、`dbgorm/driver/mysql`（dbclient）。
- **glog 默认驱动切换**：默认 `LoggerType` 由 slog 改为 zap，cmd / testsetup 的 blank import 由 `driver/slog` 改为 `driver/zap`，config.yaml / config.prod.yaml 各日志块补齐 `logger_type: zap`。
- **callerSkip 化繁为简**：移除 `pkg/dbclient` 中 `dbgorm.WithCallerSkip(3)` 与 `dbredis.WithCallerSkip(9)` 的显式覆盖，改为依赖 golib 校准默认（gorm=3 / redis=4 / es=6）。
- **glog 配置格式迁移**：`Writer` 单字段 → `Writers` 切片，`Dir` 等字段移入 `WriterConfig`，涉及 `apps/demo/config/config.yaml`、`apps/demo/config/config.prod.yaml`、`apps/ragforge/config/config.yaml`。

### 附带修复（非迁移引起）

- `pkg/code/ragforge.go` 与 `apikey.go` 错误码在 `101101-101108` 段重叠导致的 code 包 init panic，已把 ragforge 段整体平移至 `101300-101352`。
- `httpbingo` 测试依赖全局 `config.Conf` 未初始化，已在测试内 `config.LoadConfig` 修复。
- 清理了一批 pre-existing 的 golangci-lint 问题（unused 死代码、errcheck、staticcheck、grpc.Dial 弃用等）。

## 五、v1.32.13 → v1.32.16 升级记录

升级目标：`github.com/morehao/golib` v1.32.13 → **v1.32.16**（当时最新）。`gocli` 仍为 v1.32.6（已是最新），golib 自身 `go.mod` 未变，**未引入新的传递依赖**。

### 5.1 破坏性 / 契约变更

#### storage / filestore

- `storage.Config`：删除 `MaxRetries`、`Timeout`、`ExtraOptions`（全仓零消费方），新增 `Retry RetryConfig`（`MaxAttempts`，语义是"尝试次数"而非"重试次数"）与 `MultipartTTL`（local driver 分片会话存活时间，0=默认 24h，负值=关闭回收）。
- `storage.Storage` 方法改名：`CreateMultipartUpload`→`CreateMultipart`、`AbortMultipartUpload`→`AbortMultipart`、`CompleteMultipartUpload`→`CompleteMultipart`。
- `storage.CompletedPart` → `storage.PartInfo`（`PartNumber` 由 `int` 改为 `int32`，`ETag` 不再带引号）。
- `storage.PathBuilder` 不再负责对外 URL：`PublicURL` / `ParsePublicURL` 删除，新增 `URI()`；`WithBucket`、`WithIfNotExistsS3Opt`、`URLStyle` 一并删除。
- 新增能力：`Caps` / `Limits` / `ProviderProfile`、`OpError` + `KindOf` / `IsRetryable`、预签名 token（`EncodePresignToken` / `DecodePresignToken`）、`ListParts`、`ValidateParts` / `ValidatePartCount`、`DeleteObjectsChunked`、`MultipartCleaner`（`CleanupExpiredMultipart`）。
- `filestore.RecordUpload` / `RecordUploadRequest` 删除，统一走 `UploadAndRecord`；新增 `WithMaxUploadBytes`（默认 5GiB）、`ListParts`、`HandlePresignedUploadPart`、`PathBuilder`。

#### ginupload（HTTP 契约）

- `POST /files` 现在**必须**带 `content_hash`；当其形如 SHA256 时服务端会用实测哈希校验，不一致返回 `filestore: content hash mismatch`。
- 对象 key 完全由服务端生成，客户端不能再通过 `storage_path` 指定落点。
- 分片预签名响应新增 `method` 与 `headers`：客户端必须按 `Method + URL + Headers` 原样发起请求，漏发被签名覆盖的头会得到 `SignatureDoesNotMatch`。
- `CompleteMultipartUpload` 会校验合并后 size 与声明一致（`filestore: size mismatch`）。
- 新增 `GET /files/:id/parts`、`GET /files/multipart/:fileID/parts`；上传体积上限由 `filestore.WithMaxUploadBytes` 控制。

#### 其它包

- `gincontext`：新增 `SetAppError` / `GetAppError` / `FailWithStatus`；`Fail` 现在把应用错误码/消息写入 context，access log 的 `app.error.code` 不再恒为 0。
- `glog`：`GetLoggerConfig()` 未初始化时回退默认配置（不再返回 nil）；`AppendExtraKeys(nil, ...)` 不再 panic。
- `gconstant`：新增 `KeyHttpRequestBodyTruncated` / `KeyHttpResponseBodyTruncated`。
- `dbgorm`：新增 `dbgorm/driver/sqlite` dialector 子包。
- `dbgorm` / `dbredis` / `dbes` / `gormdao` / `ginserver` / `gobject` / `testkit` / `gtrace` / `ghttp` / `gutil` / `gerror`：**无 API 变更**。

### 5.2 本仓库适配点

- **Go 代码零改动**：`pkg` 与 `apps/demo` 均无需修改即可编译通过（`go build` / `go vet` 干净）。仅 `go.mod` + `go.sum` + `go.work.sum` 同步版本。
- **`base_url` 必须补 `/objects` 段**（`config.yaml`、`config.prod.yaml`）：ginupload 的对象直传路由是 `/v1/{app}/objects/:bucket/*key`，而 local driver 按 `base_url + "/" + bucket + "/" + key` 拼预签名 URL。原先的 `http://127.0.0.1:8099/v1/demo` 签出的是 404 地址，分片预签名 PUT 必然失败；正确值为 `http://127.0.0.1:8099/v1/demo/objects`。
- **`config.prod.yaml` 补齐 `file_storage`**：`Routers()` 无条件调用 `initFileStore()`，缺该段时 `storage.New("")` 报 `Driver is required` 并 panic。Dockerfile 用 `config.prod.yaml` 作为 `/app/config.yaml`，故 `make docker-run APP=demo` 此前启动即崩（pre-existing，与本次升级无关）。

### 5.3 环境侧遗留（需人工确认，非代码问题）

- 本地 MySQL `demo` 库的 `core_file` / `core_file_upload` 的 `id` 列仍是旧版 `bigint unsigned AUTO_INCREMENT`，而模型为 `varchar(36)`（`gormdao.StringID`，UUID v7）。**GORM AutoMigrate 不会修改已存在的主键列类型**，因此上传报 `Error 1265: Data truncated for column 'id'`。处理方式：重命名旧表让 AutoMigrate 重建（本次已把两张表重命名为 `core_file_bak_20260915` / `core_file_upload_bak_20260915`），或在新建库上直接启动。
  - 全新建库验证：AutoMigrate 会正确建出 `id varchar(36)`，完整链路（直传、去重、详情、serve、预签名 GET、分片预签名 PUT + list parts + complete）实测通过。
  - 另有不属于本应用的遗留表 `demo.core_file_verify`（同样 `bigint` 主键），未处理。
- 本地 Elasticsearch（127.0.0.1:9200）未启动，`pkg/dbclient` 的 `TestDbcheckVerify/Elasticsearch` 与 `/v1/demo/health` 的 ES 项会失败，属既有环境依赖。

## 六、v1.32.16 → v1.32.23 升级记录

升级目标：`github.com/morehao/golib` v1.32.16 → **v1.32.23**（当前最新）。`gocli` 不涉及。**本仓库 Go 代码零改动**：`go build` / `go vet` / `go test -run '^$' ./...`（测试二进制编译+链接）/ `go mod verify` 全部通过。

### 6.1 依赖图变化（golib 自身 go.mod）

| 变化 | 说明 |
|------|------|
| 移除 `gorm.io/driver/sqlite` + `github.com/mattn/go-sqlite3` | sqlite 驱动由 cgo 实现换为纯 Go 实现 |
| 新增 `github.com/ncruces/go-sqlite3`、`.../gormlite`（及 `-wasm/v2`、`julianday`） | 纯 Go（wasm）sqlite 驱动，见 golib `docs/sqlite-pure-go-migration.md` |
| 新增 `github.com/cloudwego/eino`、`eino-ext/components/model/openai` | 支撑新包 `golib/gllm` |
| 新增若干 eino 传递依赖（gonja / go-openai / json-patch / logrus 等） | 只体现在 go.sum；本仓库未引用 gllm，不进 go.mod |

> 本仓库两个模块的 `go.mod` **indirect 列表无需新增**（已用「临时 require+replace 后跑 tidy」的方式核对：tidy 不会为 golib 新依赖补充任何 indirect 项）；`go.work.sum` 未变化；`apps/demo`、`pkg` 的 `go.sum` 仅新增上述模块的校验和。

### 6.2 契约变更

#### protocol/ghttp：CallOption 化 + 响应体缓冲上限

| 变更 | 影响 |
|------|------|
| 所有整包方法（`Get`/`Post`/`Put`/`Patch`/`Delete` 及 `*JSON`、`GetStream`/`PostStream`）末尾新增可变参数 `opts ...CallOption` | **向后兼容**，本仓库 `client/httpbingo` 的 4 参调用原样编译 |
| 新增 `CallOption` 与 `WithStream()`、`WithMaxResponseBytes(n)` | 单次调用可选流式返回、或覆盖缓冲上限 |
| 新增 `Client.MaxResponseBytes` 与配置项 `max_response_bytes` | 默认 0 = 不限制，与旧行为一致（**本仓库无需改配置**） |
| 新增 `ErrResponseTooLarge`、`ErrNotBuffered`，`Result` 新增 `Read`/`Close`/`Buffer`/`Streaming` | 流式消费的新入口；缓冲路径超限时报错而非静默截断 |
| `GetStream`/`PostStream`/`StreamResult`/`StreamResult.ToResult` 标记 Deprecated | 仍可用；`StreamResult` 内嵌 `*Result`，`IsSuccess`/`IsError` 由方法提升保留 |
| `StreamResult.ToResult()` 语义 = `Result.Buffer()` | 同样受 `MaxResponseBytes` 约束（默认不限制时与旧行为等价） |

#### filestore

- `New` 新增 `db == nil` 校验：返回 error，而不是继续走到 `AutoMigrate`。
- 新增 `Migrate(db)`、`SchemaModels()`、`WithoutAutoMigrate()`：可关闭 `New` 的隐式建表，改为外部 DDL 流程（共库 / 运行账号无 DDL 权限场景）。
- `Init(db, st, bucket, opts...)` 签名与幂等语义不变（内部失败仍 panic），本仓库 `app.go` 调用点无需改动。

#### gormplugin（本仓库未直接使用）

- 新增 `ScopeResolver`（`(value, inject, declared)` 三元组）、`MissingScope` 回调、`ErrEmptyScopeValue`。
- `Resolver` 与 `ExtractFunc` **互斥**（同时提供时 `New` 返回 error）；`inject=true` 但值为空时 fail-closed 拒绝本次 SQL，不再静默生成不成立的过滤条件。
- 本仓库未 import `dbaccess/gormplugin`（`pkg/dbclient` 只用 `dbgorm`），不受影响。

#### gcontext / gincontext（纯新增，无破坏）

- `gcontext`：`TenantScope` / `TenantScopeKind`、`WithTenantScope`、`TenantScopeFrom`、`TenantScopeFilter`、`CurrentScope` / `ExplicitScope` / `AllScope`。
- `gincontext`：`SetTenantScope`、`WithRequestValue`、`AsyncContext` / `AsyncContextWithTimeout`。
- 既有 `Success` / `Fail` / `FailWithStatus` / `SetAppError` 未变，本仓库无需适配。

#### 其它包

- `glog`：修复 `Infow`/`Errorw` 等传入 `glog.KV(...)` 时的字段展开（zap 侧输出 `!badKeyN`、slog 侧输出 `!BADKEY` 且步进错误），字段名不再丢失、字段脱敏 Hook 也能看到它。**纯 bugfix，无 API 变更**。
- `gconstant`：新增 LLM 错误码 `120000-120008` 与 `LLMErrorMsgMap`；本仓库 `pkg/code` 只注册 `DBErrorMsgMap` / `SystemErrorMsgMap` / `AuthErrorMsgMap`（未注册 LLM Map），且自有错误码位于 `100100-101216` 段，无冲突。
- `storage`：新增 `DriverKodo`（`kodo`）与 `storage/driver/kodo`；`storage.Config` 字段未变。
- `protocol`：`HttpClientConfig` 新增 `MaxResponseBytes`（可选）。
- 新增包：`gllm`、`dict`、`internal/ddlgen`、`configkv/migrate.go` + `option.go`；本仓库均未引用。
- `dbgorm` / `dbredis` / `dbes` / `gormdao` / `ginserver` / `ginupload` / `gobject` / `testkit` / `gtrace` / `gutil` / `gerror`：**无 API 变更**（ginupload、dbredis、gormdao 等仅有测试文件变化）。

### 6.3 本仓库适配点

- **Go 代码零改动**：`pkg` 与 `apps/demo` 均无需修改即可编译通过（`go build` / `go vet` 干净）。
- 仅同步 `backend/apps/demo/go.mod`、`backend/pkg/go.mod` 的 golib 版本行，以及两个 `go.sum`；`go.work.sum` 无变化。
- 未启用本轮新增的可选能力（`max_response_bytes`、`WithoutAutoMigrate`、流式 `CallOption`），运行行为与 v1.32.16 一致。
- `apps/demo/go.sum` 因下述 tidy 限制未做收敛，`go get` 补入的若干 `模块/go.mod` 校验和均为依赖图内模块，合法且已通过 `go mod verify`。

### 6.4 验证方式与实测结果

本地无 MySQL，故改用本机 Docker 里的 PostgreSQL 18（127.0.0.1:5432）实测：新建隔离库 `demo_golib_upgrade_check`（不触碰既有 `demo` 库），临时把 `apps/demo/config/config.yaml` 的 `db_configs[0].url` 换成 `postgres://postgres:***@127.0.0.1:5432/demo_golib_upgrade_check?sslmode=disable&TimeZone=Asia/Shanghai`，跑完测试后已 `git checkout` 还原该文件。

| 用例 | 结果 |
|------|------|
| `demo/client/httpbingo` `TestGet` | **通过**（真实走通 `ghttp.GetJSON`，验证 CallOption 化后调用点无回归） |
| `pkg/dbclient` `TestDbcheckVerify/{MySQL(实际连 PG),Redis}` | **通过**（`SELECT 1`、`SET/GET/DEL` 均正常） |
| `pkg/dbclient` `TestDbcheckVerify/Elasticsearch` | 失败：`dial tcp 127.0.0.1:9200: connect: connection refused`（ES 未启动，既有环境依赖） |
| `demo/internal/service/svcuser` `TestUserList` | 失败：PG 语法错误，见下方 pre-existing 缺陷 |

另：`go build` / `go vet` / `go test -run '^$' ./...`（测试二进制编译+链接）/ `go mod verify` 全部通过。

### 6.5 环境侧遗留（pre-existing，与本次升级无关）

- **demo 在 PostgreSQL 上不可用（保留字表名）**：`user` 是 PostgreSQL 保留字，而 golib `gormdao` 会把表名**不加引号**地拼进条件字符串——`dao.go:85`/`170` 的 `fmt.Sprintf("%s.deleted_at IS NULL", d.TableName)`、`cond.go:64-76` 的 `%s.id` / `%s.created_at`、`cond.go:102` 的 `%s.%s`——生成的 `user.deleted_at IS NULL` 在**解析阶段**即失败：`ERROR: syntax error at or near "." (SQLSTATE 42601)`。仓库侧 `apps/demo/dao/user.go` 的 `UserCond` 用 `tableName+".company_id = ?"` 拼条件，是同一问题的另一处。**已用 v1.32.16（HEAD）+ 同一 PG 库复现完全相同的报错**。处理方式见第七节。
- 本地 **MySQL（127.0.0.1:3306）未启动**，MySQL 路径本轮未实测（自第七节起本项目以 PostgreSQL 为样板库，该限制不再影响默认配置）。
- **pre-existing**：workspace 模式下 `apps/demo` 的 `go mod tidy` 无法解析 go.work 中的 `pkg` 模块，报 `module github.com/morehao/ark-go-starter@latest found (...), but does not contain package .../pkg/dbclient`，因此 `make deps` / `make tidy` 会停在该模块（`backend/pkg` 的 tidy 正常）。该问题在 HEAD 上同样复现；若后续需要，可在 `apps/demo/go.mod` 中补 `require` + `replace` 指向 `../../pkg` 以恢复 tidy。

## 七、样板库切换为 PostgreSQL（v1.32.23 之后的配套调整）

本项目自本次起以 **PostgreSQL 为样板库**：默认配置、DDL、健康检查、文档均以 PG 为准；MySQL 驱动保留注册，改 `db_configs.url` 前缀即可切回。

### 7.1 变更点

| 位置 | 变更 |
|------|------|
| `backend/apps/demo/config/config.yaml`、`config.prod.yaml` | `db_configs[0].url` → `postgres://postgres:...@127.0.0.1:5432/demo?sslmode=disable&TimeZone=Asia/Shanghai` |
| `backend/apps/demo/config/code_gen.yaml` | `database_dsn` → PostgreSQL（gocli 代码生成用），MySQL 示例降为注释 |
| `backend/scripts/sql/table_init.sql` | 由 MySQL DDL 改写为 PostgreSQL DDL（`"user"`、`user_login_log`、索引、表注释） |
| `backend/apps/demo/dao/user.go` | 曾把传给 `gormdao.NewDao` 的表名换成预引号常量（`BuildCondition` 保持模板原样），见 7.2 与 8.2；**该权宜做法已在第十节（v1.32.26）撤销** |
| `backend/pkg/dbclient/gorm.go` | 错误文案去 MySQL 硬编码（`db config is empty` / `init db failed`）；`driver/mysql` 的 blank import 保留 |
| `dtohealth.DbCheckResp` / `svchealth` | 健康检查字段 `mysql` → `postgres`；swagger 已用 `make swag APP=demo` 重新生成（diff 仅此一行） |
| `README.md` / `README.zh.md` / `AGENTS.md` / `apps/demo/README.md` / `pkg/testsetup/README.md` | MySQL 描述改为 PostgreSQL；健康检查响应示例改为 `postgres` |

### 7.2 必须知道的上游限制：gormdao 的裸表名拼接

golib `dbaccess/gormdao` 共有 8 处把表名直接字符串拼接进条件：

| 位置 | 拼接 |
|------|------|
| `gormdao/dao.go:85`、`170` | `fmt.Sprintf("%s.deleted_at IS NULL", d.TableName)` |
| `gormdao/cond.go:64`、`68` | `%s.id = ?`、`%s.id IN (?)` |
| `gormdao/cond.go:72`、`76` | `%s.created_at >= ?`、`%s.created_at <= ?` |
| `gormdao/cond.go:102` | `%s.%s`（OrConditions） |

表名落在 PostgreSQL 保留字上时（demo 的 `user`），生成的 `user.deleted_at IS NULL` 会在解析阶段直接失败（`SQLSTATE 42601`）。这与 golib 版本无关（v1.32.16 / v1.32.23 表现相同），且仓库侧 gocli 风格的条件（`tableName+".company_id = ?"`）有同样问题。

**当时的做法（保留至今）**：给 `gormdao.NewDao` 传已加双引号的表名。

```go
// apps/demo/dao/user.go
const tableNameUserQuoted = `"` + model.TableNameUser + `"`

gormdao.NewDao[model.UserEntity, model.UserEntityList, uint](tableNameUserQuoted, "UserDao", dbclient.DemoDB)
```

之所以一处即可覆盖：上述所有拼接都复用同一个 `tableName` 字符串（仓库侧同理），因此引号会出现在每条 `WHERE` 上；`BuildCondition` 的模板写法因此可以原样保留。
**代价**：引号与方言绑定（PostgreSQL 双引号 / MySQL 反引号）。

**上游修复**：**v1.32.24** 起 golib 内部（软删过滤、主键/时间条件、排序、OrConditions）已全部改用 `clause.Column` 引用标识符，不再拼裸表名；但仓库侧若沿用模板的拼接写法，表名本身仍需是已引用的形式，故本仓库继续使用该常量，理由与备选方案见 8.2。

### 7.3 验证结果

- **DDL**：在新建空库上以 `psql -v ON_ERROR_STOP=1 -f backend/scripts/sql/table_init.sql` 执行通过，`"user"`、`user_login_log` 两表及索引、注释创建正确。
- **Go 测试**（对由该 DDL 建表的新库）：`demo/client/httpbingo TestGet`、`demo/internal/service/svcuser TestUserList`（原 PG 失败用例）均通过；`pkg/dbclient TestDbcheckVerify` 的 Postgres/Redis 子用例通过，`Elasticsearch` 子用例仍失败（本地 9200 未启动，环境依赖）。
- **HTTP 全链路冒烟**（`cd backend/apps/demo/cmd && go run .`）：
  - `GET /v1/demo/health` → `{"postgres":"ok","redis":"ok","es":"<connection refused>"}`
  - `POST /v1/demo/users` → `userID:1`；`GET /v1/demo/users?companyID=7` 命中 `"user".company_id = ?`；`GET /v1/demo/users/1`、`PUT /v1/demo/users/1`、`DELETE /v1/demo/users/1` 均成功；删除后详情返回 `[100505] 用户管理不存在`（软删过滤生效）。
- **Swagger**：`make swag APP=demo` 重新生成后仅 `"mysql"` → `"postgres"` 一行差异。

> 本地遗留：`demo` 库当前只有历史遗留空表（`core_config`、`core_file_verify`、`pg_audit_log`、`pg_int64_user`），尚无 `"user"` 表；按上面命令执行一次 DDL 即可（`core_file` / `core_file_upload` 由 filestore 启动时 AutoMigrate 创建）。

## 八、v1.32.23 → v1.32.24 升级记录（保留字问题的上游修复，**已在 v1.32.25 回滚**）

> ⚠️ 本节记录的 gormdao 引用改动已在 **v1.32.25 被回滚**（`clause.Column`、`OrderByFields` 一并撤销）。
> 当前仓库的实际做法与结论以**第九节**为准，本节仅作历史留档。

升级目标：v1.32.23 → **v1.32.24**（2026-10-16 发布）。该版本**非测试代码只动了 gormdao**，正是第七节所述保留字问题的上游根治。

### 8.1 golib 侧变更

| 文件 | 变更 |
|------|------|
| `dbaccess/gormdao/dao.go` | `fmt.Sprintf("%s.deleted_at IS NULL", ...)` → 新增 `deletedAtIsNull()`：`clause.Eq{Column: clause.Column{Table: tableName, Name: "deleted_at"}, Value: nil}`（`deletedScope` 与 `GetByID` 共用）；文档注明 `user`、`order` 等保留字表名已安全 |
| `dbaccess/gormdao/cond.go` | `%s.id = ?` / `%s.id IN (?)` / `%s.created_at >= ?` / `%s.created_at <= ?` → `clause.Eq` / `clause.IN` / `clause.Gte` / `clause.Lte` + `clause.Column`；`OrConditions` 的表名前缀改为 `db.Statement.Quote(tableName)` 按方言引用；新增结构化排序 `OrderByField` 与 `BaseCond.OrderByFields` |
| `internal/testutil/dsn.go` | 新增默认 DSN 常量（内部测试用） |
| 其余 | 仅 `_test.go`（含新增 `gormdao/reserved_test.go`：保留字表名的 dry-run 与 PG 实测） |

**兼容性**：无删除、无签名变更，`OrderByFields` 为新增字段。一个契约变化：`OrCondGroup.Query` 现在是「**表名前缀由 golib 引用，`Query` 内部按原样拼入**」，所以列名本身是保留字时要自己写成 `"order" = ?`；`OrderField` 保持原始片段语义（不做引用），需要引用时用 `OrderByFields`。

### 8.2 本仓库调整

1. `backend/apps/demo/dao/user.go`：**`BuildCondition` 保持 gocli 模板原样不动**（仍是 `db.Where(tableName+".company_id = ?", ...)`），只在 `NewUserDao` 处把传给 `gormdao.NewDao` 的表名换成**预引号常量**（v1.32.25 之后该常量统一放在 `model/table.go`，见第九节）：

   ```go
   gormdao.NewDao[model.UserEntity, model.UserEntityList, uint](model.TableNameUserQuoted, "UserDao", dbclient.DemoDB)
   ```

   原理：golib 把 `NewDao` 收到的表名原样交给 `BuildCondition`（也用于 `Table()`），所以模板拼接出来就是 `"user".company_id`，PG 合法。这样**调用点写法零改动**，gocli 模板也无需改（只有表名恰好是 PG 保留字时才需要这个常量）。
2. `backend/go.work.sum` 随升级由 `go work sync` 重算（补入 golib 新依赖的校验和，移除已下沉到各模块 `go.sum` 的条目）。

#### 8.2.1 为什么不改条件写法（以及备选方案）

改条件写法的确有几种写法能让调用层不依赖 `gorm.io/gorm/clause`。实测在 postgres 方言下渲染的 SQL（`db.Table("user")`，`1` 为示例值）：

| 写法 | 渲染结果 | 结论 |
|------|----------|------|
| `Where(tableName+".company_id = ?", 1)`（**本仓库采用**，配预引号表名） | `WHERE "user".company_id = 1` | ✅ 调用点零改动 |
| `Where("company_id = ?", 1)`（裸列名，golib 测试里的建议） | `WHERE company_id = 1` | 可用，但列名为保留字（`order`）时报 `42601`，且丢表名前缀 |
| `Where(map[string]any{"company_id": 1})` | `WHERE "user"."company_id" = 1` | ✅ 表名+列名都受引用保护（备选） |
| `Where(&model.UserEntity{CompanyID: 1})` | `WHERE "user"."company_id" = 1` | ✅ 但会跳过零值字段，表达能力受限 |
| `Where("order = ?", 1)` | `WHERE order = 1` | ❌ `SQLSTATE 42601` |
| `Where(map[string]any{"order": 1})` | `WHERE "user"."order" = 1` | ✅ 列名同样受保护 |

**结论**：想「调用点一字不改」就用预引号表名；想「跨方言同一份代码」就改条件写法（map）或把表名改成非保留字。当前选择前者。

预引号表名的代价与边界：

- 引号与方言绑定，**MySQL 下需换成反引号**（`"user"` 在 MySQL 会被当字符串，`FROM "user"` 报错）。若要在保留调用点写法的同时兼容双方言，需要在 `dbclient` 增加一个「按已注册 dialect 引用标识符」的小工具，再把结果传给 `NewDao`。
- 预引号不会与 golib 内部的 `clause.Column` 二次引用——golib v1.32.24 用的是同一个表名字符串，实测渲染为：

  ```
  SELECT count(*) FROM "user" WHERE "user"."deleted_at" IS NULL AND "user".company_id = 7 AND "user"."deleted_at" IS NULL
  ```

  即：`FROM "user"`（GORM 自己引用）、`"user"."deleted_at"`（golib 的 clause 条件）、`"user".company_id`（模板拼接）三者一致，无 `"""user"""` 之类的重复引用。

### 8.3 验证（PostgreSQL，表名仍是保留字 `user`）

- **Go 测试**：`demo/client/httpbingo TestGet`、`demo/internal/service/svcuser TestUserList` 通过；`pkg/dbclient` 仅 `Elasticsearch` 子用例失败（本地 9200 未启动）。
- **HTTP 全链路**：`POST /v1/demo/users`、列表、详情、`PUT`、`DELETE` 全部成功，删除后详情返回 `[100505] 用户管理不存在`。
- **直连 DAO 遍历全部条件**（临时用例，跑完已删除）：`company_id`、`department_id`、`name`、`created_by`、`updated_by`、`deleted_by`（模板拼接条件 + 预引号表名）与主键、`created_at` 范围、`OrderByFields`、`OrConditions`（golib 侧 clause 条件）以及 `GetByID`、不匹配用例、软删后查空，全部符合预期。
- **实测 SQL**：所有标识符均按方言引用，不再出现裸表名：
  ```
  SELECT count(*) FROM "user" WHERE "user"."deleted_at" IS NULL AND "user".company_id = 7 AND "user"."deleted_at" IS NULL
  SELECT * FROM "user" WHERE "user"."deleted_at" IS NULL AND "user"."deleted_at" IS NULL ORDER BY "user"."id" DESC
  ```

### 8.4 gocli 模板：无需改动（仅保留字表名才需要额外一行）

gocli（v1.32.6）的 DAO 模板仍是 `db.Where(tableName+".xxx = ?", ...)` 这一条，本仓库**保持了它的原样**，因此：

- 表名不含 PostgreSQL 保留字（绝大多数情况）：模板产物直接可用，**不需要任何改动**；
- 表名恰好是保留字（`user`、`order`、`group`…）：在 `NewXxxDao` 处把表名换成预引号常量即可（见 8.2），`BuildCondition` 依旧不用改。

若希望「生成的代码在任意表名下开箱可用」，可选做法（按代价从低到高）：

1. 模板里给出注释提示：保留字表名需传预引号表名；
2. 模板改为 map 形式 `db.Where(map[string]any{"{{.ColumnName}}": c.{{.FieldName}}})` —— 列名与表名都受引用保护，但改变了既有调用写法；
3. `NewXxxDao` 的表名参数改为按 dialect 引用后再传入 —— 需在 `dbclient` 增加一个小工具函数。

## 九、v1.32.24 → v1.32.25：gormdao 回滚 + 保留字表名的最终处理方式

> ⚠️ 本节的「预引号表名」做法**已被 v1.32.26 取代**（引用能力重新落地，仓库不再需要自己加引号）。
> 第十节是当前做法与结论，本节仅作历史留档。

升级目标：v1.32.24 → **v1.32.25**（2026-10-16 发布）。第八节的引用改动被撤销，改回字符串拼接，并在注释里**明确要求调用方在 `NewDao` 传入已按方言引用的表名**。

### 9.1 golib 侧变更（v1.32.25）

| 文件 | 变更 |
|------|------|
| `dbaccess/gormdao/dao.go` | `deletedAtIsNull()` 删除，恢复 `db.Where(fmt.Sprintf("%s.deleted_at IS NULL", d.TableName))`；注释改为：「追加的过滤条件与 BaseCond 生成的条件都**不做标识符引用**（只做字符串拼接）：表名撞保留字时需由调用方在 `NewDao` 传入已按方言引用的表名（PostgreSQL `"user"`、MySQL `` `user` ``）或 schema 限定名」 |
| `dbaccess/gormdao/cond.go` | `clause.Eq/IN/Gte/Lte` 恢复为 `fmt.Sprintf("%s.id = ?")` 等；`quoteIdent()` 与 `buildOrClause` 的引用逻辑撤销；**`OrderByField` / `BaseCond.OrderByFields` 一并移除**（本仓库未使用）；`OrCondGroup.Query` 恢复为「表名与片段都原样拼入，列名为保留字时调用方自己写 `"order" = ?`」 |

**兼容性**：本仓库 Go 代码不受影响（未使用 `OrderByFields`），编译即通过。

### 9.2 本仓库做法：表名常量拆到 `model/table.go`

按「表名相关的事在声明表名的地方处理」的原则，`backend/apps/demo/model/table.go` 声明两个常量：

```go
const (
	// 物理表名，Entity.TableName() 返回它 —— 必须不带引号（理由见 9.3）
	TableNameUser = "user"

	// 预先按方言引用后的表名，专供 gormdao.NewDao 使用
	TableNameUserQuoted = `"` + TableNameUser + `"`
)
```

`dao/user.go` 只有 `NewDao` 的入参变化，`BuildCondition` 与调用点保持 gocli 模板原样：

```go
gormdao.NewDao[model.UserEntity, model.UserEntityList, uint](model.TableNameUserQuoted, "UserDao", dbclient.DemoDB)
```

原理：gormdao 把 `NewDao` 收到的表名原样用于 `Table()` 与条件拼接，所以传「已引用」的名字后，模板拼出来就是 `"user".company_id`，在 PG 下合法。

### 9.3 为什么引号不能直接写进 `TableNameUser`

`TableName()` 的返回值会被 GORM 的 Migrator 系列 API 当作**字符串值**去比对表名。实测（PG，`demo_ddl_check` 库，`"user"` 表已存在）：

| 调用 | 结果 |
|------|------|
| `Migrator().HasTable("user")` | **true** |
| ``Migrator().HasTable(`"user"`)`` | **false** |
| `db.Table(`"user"`).Count(&n)` | 正常（查询路径不受影响，`FROM "user"`） |

即：查询路径能容忍带引号的表名，但 `HasTable` 会误判「表不存在」，进而让 `AutoMigrate` 重复建表。所以模型侧保持裸表名，只在 DAO 侧使用引用形式。

### 9.4 验证（PostgreSQL，表名 `user` 为保留字）

- **直连 DAO 全条件遍历**（临时用例，跑完已删除）：模板拼接的 6 个字段条件 + golib 的 `%s.id` / `%s.created_at` / `%s.%s`（OrConditions）+ `GetByID` + 软删后查空，全部符合预期。
- **实测 SQL**：表名各处均为已引用形式，无裸表名：
  ```
  SELECT count(*) FROM "user" WHERE "user".deleted_at IS NULL AND "user".company_id = 7 AND "user"."deleted_at" IS NULL
  SELECT count(*) FROM "user" WHERE "user".deleted_at IS NULL AND ("user".name = 'v13225') AND "user"."deleted_at" IS NULL
  ```
- **Go 测试**：`httpbingo TestGet`、`svcuser TestUserList` 通过；`pkg/dbclient` 仅 ES 子用例失败（本地 9200 未启动）。

### 9.5 代价与备选

- **代价**：引号与方言绑定 —— 当前值 `"user"` 只适用于 PostgreSQL，MySQL 需改成反引号 `` `user` ``。这与「本项目以 PG 为样板库」一致。
- **备选一（schema 限定名）**：golib 注释提到的另一条路。实测 PG 下 `SELECT count(*) FROM public.user WHERE public.user.deleted_at IS NULL` **可以正常解析**（裸 `user.deleted_at` 报 42601，但带 schema 前缀后不再被当成 `USER` 关键字）。即传 `public.user` 也能免引号解决，代价是硬编码 schema（search_path 变化或换库时不成立）。
- **备选二（跨方言）**：把表名改成非保留字（如 `demo_user`），代码零 hack、PG/MySQL 通吃，代价是改 DDL 与模型常量。
- **备选三（运行时按方言引用）**：在 `dbclient` 里保存已初始化连接的方言，用 `(&gorm.Statement{DB: db}).Quote(name)`（实测 PG 得 `"user"`、MySQL 得 `` `user` ``）在构造 DAO 时生成表名。调用点不变且跨方言，代价是引入全局状态与初始化时序约束（未初始化时会静默退化为裸表名）。

## 十、v1.32.25 → v1.32.26 升级记录：gormdao 重新落地标识符引用

升级目标：v1.32.25 → **v1.32.26**（2026-10-16 发布）。该版本非测试代码只动了 `dbaccess/gormdao`，外加 `configkv` / `dict` / `filestore` / `task/*` 内置 Cond 的同步改写。

### 10.1 golib 侧变更

| 文件 | 变更 |
|------|------|
| `dbaccess/gormdao/doc.go` | 新增包文档：表名/标识符引用契约、`Cond.BuildCondition` 各写法边界（由 `caller_style_test.go` 逐条钉住） |
| `dbaccess/gormdao/cond.go` | 新增 `quoteTable(db, tableName)`（= `db.Statement.Quote(tableName)`，与 FROM 子句逐字节一致，对逻辑名/已引用名/schema 限定名幂等）与 `colRef`；`%s.id = ?` / `%s.id IN (?)` / `%s.created_at >=/<=` 恢复为 `clause.Eq/IN/Gte/Lte` + `clause.Column`；`OrCondGroup` 的**表名前缀**改由库引用（`"user".name = ?`），片段内部仍原样拼入 |
| `dbaccess/gormdao/dao.go` | 软删过滤改 `notDeletedExpr`（`clause.Eq{Column: {Table: tableName, Name: "deleted_at"}, Value: nil}`）；`GetByCond`/`GetListByCond`/`GetPageListByCond`/`CountByCond` 调 `cond.BuildCondition(db, quoteTable(db, d.TableName))` |
| `configkv` / `dict` / `filestore` / `task/gasync` / `task/gcron` | 内置 Cond 的 `tableName+".x = ?"`（或 `fmt.Sprintf`）统一改为 `clause.Eq{Column: clause.Column{Table: tableName, Name: "x"}, ...}` |

**契约**：`NewDao` 传**逻辑表名**即可，不要自己加引号；已引用名（`"user"`）与 schema 限定名（`public.user`）幂等可用。`BuildCondition` 收到的 `tableName` 已经是当前方言下的引用形式：

```go
// 推荐：结构化条件，表名/列名都交给 Dialector 引用
db.Where(clause.Eq{Column: clause.Column{Table: tableName, Name: "status"}, Value: v})
// 也可：tableName 已是引用形式，直接拼前缀；**不要**再手动引用一次（会得到 ""user""）
db.Where(tableName+".company_id = ?", v)
```

**兼容性**：无删除、无签名变更（v1.32.24 引入的 `OrderByFields` 仍不存在）。一处内部收敛：`BuildBaseCondition` 里的 `if cond.IsDelete { db.Unscoped() }` 被移除，统一由 `Dao.deletedScope` 依据 `IncludeDeleted()` 处理。

仍由调用方负责的自由片段只有两处：`BaseCond.OrderField` 与 `OrCondGroup.Query` 的**片段内部**（其中的列名是保留字时要自己写成 `"order" DESC`）。

### 10.2 本仓库适配点

按同级 gocli（分支 `fix/dao-cond-clause-quoting`）的模板改法对齐，**Go 侧改动集中在 DAO 一处**：

1. `backend/apps/demo/dao/user.go`：`BuildCondition` 的 6 处 `tableName+".xxx = ?"` 改为 `clause.Eq{Column: clause.Column{Table: tableName, Name: "xxx"}, Value: ...}`，补 `gorm.io/gorm/clause` import；`NewDao` 的表名由 `model.TableNameUserQuoted` 换回逻辑表名 `model.TableNameUser`。与 gocli 的 `template/ark/apps/demo/dao/user.go.tmpl`、`template/generate/{module,model}/dao.go.tpl` 一致（生成器配套 `addDaoCondImport`：只在确有基本类型条件字段时才补 clause import，避免无字段时生成物 import 未使用）。
2. `backend/apps/demo/model/table.go`：删除 `TableNameUserQuoted`，只留逻辑表名常量；保留「`TableName()` 不能带引号（`Migrator().HasTable` 按字符串值比对，带引号会误判表不存在）」的说明。
3. `backend/scripts/sql/table_init.sql`：头部注释由「Go 侧必须传带引号表名」改为「Go 侧传逻辑表名，由 gormdao 按方言引用」。

依赖侧只动 `apps/demo/go.mod`、`pkg/go.mod` 的版本行与两个 `go.sum`（`go get` + `go work sync`）；`go.work.sum` 不含 golib 条目（两模块 `go.sum` 已覆盖），故无 golib 相关变化。`go build` / `go vet` 干净。

### 10.3 验证（PostgreSQL 18，隔离库）

在新建隔离库 `demo_golib_13226_check` 上实测（用 `backend/scripts/sql/table_init.sql` 建表，含保留字表 `"user"`；跑完已 drop，未触碰既有 `demo` 库），随后又按同一脚本给本地 `demo` 库补建了 `"user"` / `user_login_log`（`core_file` / `core_file_upload` 由 filestore 启动时 AutoMigrate 创建），复跑全套：

| 用例 | 结果 |
|------|------|
| `demo/dao` 临时全条件遍历（模板 6 字段 + `BaseCond.ID/IDs/时间范围/OrConditions` + `GetByID`/`GetByCond` + 软删过滤 + `IsDelete` 查已删） | **通过**（临时用例，跑完已删） |
| `demo/internal/service/svcuser` `TestUserList` | **通过**（上一次升级时该用例因无 `"user"` 表而失败） |
| `demo/client/httpbingo` `TestGet` | **通过** |
| `pkg/dbclient` `TestDbcheckVerify/{Postgres,Redis}` | **通过** |
| `demo` HTTP 全链路冒烟（`go run .`）：`GET /v1/demo/health`、`POST/GET/PUT/DELETE /v1/demo/users` | **通过**（health：`postgres:ok, redis:ok, es:<connection refused>`；列表带 `companyID=7` 条件命中；删除后详情返回 `[100505] 用户管理不存在`） |
| `pkg/dbclient` `TestDbcheckVerify/Elasticsearch` | 失败：`dial tcp 127.0.0.1:9200: connect: connection refused`（ES 未启动，既有环境依赖） |

实测 SQL（标识符全部按方言引用，无裸表名）：

```
SELECT count(*) FROM "user" WHERE "user"."deleted_at" IS NULL AND "user"."company_id" = 7 AND "user"."department_id" = 9 AND "user"."name" = 'v13226' AND "user"."created_by" = 11 AND "user"."updated_by" = 12 AND "user"."deleted_at" IS NULL
SELECT count(*) FROM "user" WHERE "user"."deleted_at" IS NULL AND "user"."id" = 1 AND "user"."id" IN (1,999999) AND "user"."created_at" >= '...' AND "user"."created_at" <= '...' AND ("user".name = 'v13226') AND "user"."company_id" = 7 AND "user"."deleted_at" IS NULL
```

### 10.4 与前几节的关系

第七～九节记录的「预引号表名」是 v1.32.25 及以前的上游限制下的权宜做法，**自 v1.32.26 起不再需要**：`model.TableNameUserQuoted` 已删除，`NewDao` 直接传 `model.TableNameUser`。9.5 的备选方案（schema 限定名、改非保留字表名、运行时按方言引用）随之作废。

> 环境侧现状：本地 `demo` 库已补建 `"user"` / `user_login_log`（`"user"` 是 PG 保留字，脚本内引用均带双引号），因此 `make test APP=demo` 现在可全绿（仅 `pkg/dbclient` 的 Elasticsearch 子用例依赖本机 9200）。重建库时执行 `psql -d demo -f backend/scripts/sql/table_init.sql` 即可。

