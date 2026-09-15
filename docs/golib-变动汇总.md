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
