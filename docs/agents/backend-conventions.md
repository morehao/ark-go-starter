# 后端代码规范

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

适用于仓库内所有 Go 代码：应用在 `apps/api`（逻辑标识 `demo`），公共层在 `pkg`。

## 项目结构（后端）

> 迁移后布局（P2）：Go workspace 在仓库根 `go.work`，`use ./apps/api ./pkg`。

```
apps/api/                     # Demo 应用（逻辑标识 demo），Go 服务 :8099
├── cmd/                      # 入口函数
├── internal/
│   ├── controller/ctrxxx/    # 控制器层 (ctr 前缀)
│   ├── service/svcxxx/       # 服务层 (svc 前缀)
│   ├── dto/dtoxxx/           # DTO 层
│   ├── router/               # 路由注册
│   └── middleware/           # 中间件
├── model/                    # 数据模型
├── dao/                      # 数据访问层
├── client/                   # 外部客户端
├── object/                   # 基础对象
└── config/                   # 配置（config.yaml / config.prod.yaml / config.test.yaml；code_gen.yaml 为 gocli 代码生成配置）
pkg/                      # 公共包（跨应用共享）
```

### 部署目录与基础设施层的命名边界

> 2026-11 起：仓库根的编排目录由 `infra/` 更名为 `deploy/`（原因：与代码侧「基础设施层」的通用叫法撞名）。以下约定自此生效。

**名字预算（固定，不再改）**

| 位置 | 名字 | 装什么 | 参与编译 |
|------|------|--------|----------|
| 仓库根 | `deploy/` | 构建与部署资产：Dockerfile、compose、k8s/Helm 清单、发布脚本 | 否（不进 `go.work` / `pnpm-workspace` / Nx project graph） |
| 代码内 | `internal/infra/`（预留，当前不建） | 基础设施层 / 适配器实现：持久化、缓存、消息队列、外部 HTTP、对象存储 | 是 |

**`deploy/` 不含建表 SQL**

DDL（`scripts/sql/table_init.sql`）是数据库初始化资产，归 `scripts/`，不进 `deploy/`。判断口径：

- 由**容器 / 编排工具**读取的（Dockerfile、compose、k8s/Helm 清单）→ `deploy/`
- 由 **`psql -f`、CI、quickstart 直接执行**的（建库建表、数据修复脚本）→ `scripts/`

`deploy/` 只回答「怎么构建、怎么编排」，不回答「库里有什么表」；反过来 `scripts/` 不放编排描述。

**1. 唯一住所，禁止并存**

引入 `internal/infra/` 时，现有按技术命名的适配器必须在**同一个 PR 内**迁入并删除，不允许并存超过一个提交：

| 现状 | 迁入后 |
|------|--------|
| `apps/api/dao/` | `apps/api/internal/infra/persistence` |
| `apps/api/client/` | `apps/api/internal/infra/external` |
| `apps/api/storage/` | `apps/api/internal/infra/storage` |
| 将来 Redis / MQ | `apps/api/internal/infra/cache`、`apps/api/internal/infra/mq` |

`model/` 不迁：它是 service 与 dao 共用的表结构实体，不属于 infra。

**2. 依赖方向**

`infra` 只能 import `model`、`pkg/*`、第三方 SDK；**禁止** import `controller`、`service`、`dto`（防反向依赖）。接口沿用 Go 惯例的**消费方定义**（service 内定义 `ProductRepo interface`，infra 提供实现），不为了「正统」单独造 `domain/` 或 `port/` 大目录。

**3. 触发条件（满足其一才建）**

- 单测需要 mock 外部依赖（DB / HTTP / 存储）；
- 需要替换实现（换库、多存储共存、灰度双写）；
- 两个以上团队要并行改适配器。

不满足就维持现状：`dao` / `client` / `storage` 本身已是基础设施层，只是按技术名而非层名组织。

**4. 事务边界不随搬家**

当前事务写在 service（`dbclient.DemoDB(ctx).Transaction(...)`，见 `svcproduct`）。事务下移属于重写写入路径，必须单独出方案，不在「搬目录」的 PR 里顺带做。

**5. 表达兜底**

文档、PR、issue 里出现 `infra` 必须带限定：`deploy/` 指部署资产，`internal/infra/` 指代码层。裸说「infra 改一下」视为不合格表述，评审里容易产生歧义。

## 命名规范

- **包名**: 小写，简短，如 `svcuser`, `dtouser`
- **接口名**: 以 `I` 结尾或使用角色后缀，如 `UserSvc`, `UserCtr`
- **结构体**: 导出使用大驼峰 `UserSvc`，非导出使用小驼峰 `userSvc`
- **文件命名**: 小写下划线，如 `user_service.go`，测试文件 `*_test.go`
- **数据库表**: 下划线命名，如 `user_department`
- **缩略词全部大写或全部小写**：`APIKey`、`OIDC`、`URL`、`HTTPClient`，
  **禁止** `ApiKey`、`Oidc`、`Url`、`HttpClient`（`revive` 的 `var-naming` 会报错）

```go
// 正确
const APIKeyCreateError = 101101
type OIDCConfig struct{}

// 错误 —— 缩略词按单词首字母大写
const ApiKeyCreateError = 101101
type OidcConfig struct{}
```

## 注释规范

> 由 `.golangci.yml` 的 `revive`（`package-comments` / `exported`）强制，**lint 不通过即视为不合规**。

### 包注释（硬性要求）

每个包必须有且仅有一个文件带 `// Package xxx ...` 注释，紧贴 `package` 子句：

```go
// Package svcuser 是用户管理服务层：承载业务规则、事务与错误转换。
package svcuser
```

- 包注释放在该包**最主要的文件**中（通常是与包同名的文件或入口文件）；
- 文件顶部有 `//go:build` 时，包注释放在 build tag 之后；
- 文件头有版权/作者块注释时，与包注释之间**留一个空行**，否则会被当成包注释的一部分。

### 导出符号注释（硬性要求）

所有导出标识符（类型、函数、方法、常量、变量）都必须有 godoc 注释，且**以符号名开头**：

```go
// UserSvc 是用户管理服务接口。
type UserSvc interface{}

// NewUserSvc 构造 UserSvc。
func NewUserSvc() UserSvc { return &userSvc{} }

// Create 创建用户管理。
func (svc *userSvc) Create(...) (...) {}
```

- `const` / `var` 成组声明时，可给**整个块**加一行注释，无需逐个注释；
- `type (...)` 块内的多个别名需要**拆成独立声明**，各自带注释（块注释只能匹配第一个名字）；
- 中文注释结尾**不加 ASCII 句号**（`.golangci.yml` 已关闭 `godot`，避免把「示例」改成「示例.」）。

### DTO 字段注释（硬性要求）

`internal/dto/` 下所有请求/响应结构体的**每个字段都必须有中文行内注释**，作为字段语义的事实说明（Swagger 与前端联调都依赖它）：

```go
type UserCreateResp struct {
    UserID uint `json:"userID"` // 自增 ID
}

type UserPageListResp struct {
    List  []UserPageListItem `json:"list"`  // 数据列表
    Total int64              `json:"total"` // 数据总条数
}
```

- 注释写在 tag 之后，`// ` 开头、紧跟字段行，由 `gofmt` 统一对齐；
- 内嵌的基础结构（`object/objxxx`、`golib/biz/gobject`）同样适用：被复用结构体的字段也要逐个注释；
- 字段含义与库表列注释保持一致，列注释改了这里同步改。

### SQL 字段注释（硬性要求）

DDL（`scripts/sql/table_init.sql`）中**每个字段都必须有注释**，且要写两份、保持一致：

1. 字段行尾的 `-- 中文说明`：保证脚本本身可读；
2. 建表后的 `COMMENT ON COLUMN`：把同一份说明写进库元数据，`psql` 里 `\d+ 表名`、以及各类客户端都能看到。

```sql
CREATE TABLE IF NOT EXISTS "user"
(
    id         BIGSERIAL   PRIMARY KEY,            -- 自增ID
    name       VARCHAR(255) NOT NULL DEFAULT '',   -- 姓名
    deleted_at TIMESTAMPTZ          DEFAULT NULL   -- 删除时间（软删除）
);

COMMENT ON TABLE "user" IS '用户表';
COMMENT ON COLUMN "user".id IS '自增ID';
COMMENT ON COLUMN "user".name IS '姓名';
COMMENT ON COLUMN "user".deleted_at IS '删除时间（软删除）';
```

> `user` 等保留字表名在 `COMMENT ON COLUMN` 中必须写成 `"user"`（双引号），列名一般无需引号。
> 同一份说明还出现在 `model/*.go` 的 gorm tag 里（如 `gorm:"...;comment:公司id"`），共三处：model tag、SQL 行内注释、`COMMENT ON COLUMN`。
> 新增字段、改名字段或调整列语义时，三处 + 对应 DTO 字段注释一起改。
> PostgreSQL 下模型的 `comment:` tag 会被 gorm 转成 `COMMENT ON COLUMN` 写进库元数据，所以 model tag 就是列注释的事实源。

## 模型字段类型规范（PostgreSQL）

**本仓库唯一目标库是 PostgreSQL**：建表事实源是 `scripts/sql/table_init.sql`（PG 语法），model 的 gorm tag **必须显式写 `type:`，且取值只能是 PostgreSQL 合法类型**（与脚本同款，结构体自描述、读模型即知列类型），不得写其它方言的字面量。

| 想要的效果 | 写法 | PG 生成的列 |
|-----------|------|-------------|
| 大整型主键 / 外键 | `uint` + `type:bigint` | `bigint`（主键带 autoIncrement 时为 `bigserial`） |
| 变长字符串 | `type:varchar(255)` | `varchar(255)` |
| 长文本 | `type:text` | `text` |
| 时间 | `time.Time`（不写 type，由方言推断） | `timestamptz` |
| 二进制 | `[]byte`（不写 type） | `bytea` |
| JSON / 集合 / 结构对象 | `type:jsonb` + `serializer:json`（推荐对象内含数组，见下） | `jsonb` |
| 小数 | `type:numeric` + `precision:10;scale:2` | `numeric(10, 2)` |
| 列注释 | `comment:中文说明` | 建表后自动 `COMMENT ON COLUMN` |

```go
type UserEntity struct {
    gorm.Model
    CompanyID uint   `gorm:"column:company_id;type:bigint;not null;default:0;comment:公司id"`
    Name      string `gorm:"column:name;type:varchar(255);not null;default:'';comment:姓名"`
}
```

原因：gorm 只把 `bool/int/uint/float/string/time/bytes` 这类**逻辑类型名**交给方言映射，其它取值会被**原样拼进 DDL**。`type:bigint unsigned` 在 PG 上实测直接失败：

```
ERROR: syntax error at or near "unsigned" (SQLSTATE 42601)
CREATE TABLE "user" (...,"company_id" bigint unsigned NOT NULL DEFAULT 0,...)
```

### 集合 / 结构体字段：`serializer:json` 优先，对象内含数组便于扩展

gorm 对「元素为非 byte 的切片」没有默认映射（`[]string` 连 `schema.Parse` 都过不去），读写 PG 数组需要 `pq.StringArray` 这类实现 `driver.Valuer`/`sql.Scanner` 的类型。**推荐改用 gorm 的 `serializer:json`**：不需要 Valuer/Scanner，也不引入 `lib/pq`，且 PG 上用 `jsonb` 列，后续加字段不用改表：

```go
// ✅ 推荐：直接存 []string
Tags []string `gorm:"column:tags;type:jsonb;serializer:json;comment:标签列表"`

// ✅ 更推荐：对象内包含数组，后续扩展字段不动表结构
type UserMeta struct {
    Tags    []string `json:"tags"`
    Level   int      `json:"level"`
    Remarks string   `json:"remarks"`
}
Meta UserMeta `gorm:"column:meta;type:jsonb;serializer:json;comment:扩展元信息"`
```

```go
// ❌ 建表能过但读写报错（42804 expression is of type record）
Tags []string `gorm:"column:tags;type:text[]"`
// ❌ serializer 不写 type 时列是 text 而不是 jsonb（没法走 GIN 索引、不能 jsonb 查询）
Meta UserMeta `gorm:"column:meta;serializer:json"`
```

两者都已在本机 PG 实测：`[]string`/结构体 + `type:jsonb;serializer:json` 建表读写全通；`type:text[]` 配裸 `[]string` 写入必炸；`serializer:json` 不写 `type` 推断成 `text` 列。

守护用例：`apps/api/model/pg_dialect_test.go` 的 `TestModelTagsArePostgresCompatible` 扫描所有模型的 `type:` tag，命中 MySQL 字面量（`unsigned`、`tinyint`、`datetime`、`longtext` …）即失败；**新增 model 要记得加进该用例的 `postgresModels`**（不需要数据库，纯 tag 校验）。

> model 目前不参与 AutoMigrate，建表只走 `table_init.sql`。若将来改成 AutoMigrate 建表，先对齐两处差异：
> 脚本里 `created_at/updated_at` 是 `NOT NULL DEFAULT now()`，而 `gorm.Model` 生成的是可空 `timestamptz`（无默认值）。

## 常量定义规范

**凡字典字符串，禁止硬编码，均需定义成常量。**

| 常量类型 | 存放位置 | 说明 |
|---------|---------|------|
| 数据表名常量 | `model/*.go` | 定义在对应结构体文件 |
| 数据库存储的枚举值 | `model/*.go` | 定义在对应结构体文件 |
| 字典值常量 | `model/*.go` | 所有字典字符串定义成常量 |
| 应用层常量（前端专用） | `internal/constant/` | 状态映射等前端专用常量 |

### 数据表常量（model 层）

```go
// model/department.go

// 数据表名
const TableNameDepartment = "iam_department"

// 业务枚举类型
type DeptStatus string

// 字典值常量（禁止硬编码）
const (
    DeptStatusActive   DeptStatus = "active"
    DeptStatusInactive DeptStatus = "inactive"
)

func (DepartmentEntity) TableName() string {
    return TableNameDepartment
}
```

### 应用层常量（internal/constant/）

前端专用的状态映射等常量，放在应用的 `internal/constant/` 目录下：

```go
// apps/api/internal/constant/status.go（internal/constant/ 目录按需创建）
package constant

const (
    StatusEnabled  = "enable"
    StatusDisabled = "disable"
)

var StatusTextMap = map[string]string{
    StatusEnabled:  "启用",
    StatusDisabled: "停用",
}
```

## Import 排序

按以下顺序分组，无空行分隔：

1. 标准库 (`fmt`, `strings`, `time`...)
2. 第三方库 (`github.com/gin-gonic/gin`, `github.com/stretchr/testify`...)
3. 项目内部包 (`github.com/morehao/ark-go-starter/apps/api/...`, `github.com/morehao/ark-go-starter/pkg/...`)
4. 关联库 (`github.com/morehao/golib/...`)

```go
import (
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify"

    "github.com/morehao/ark-go-starter/apps/api/internal/dto/dtouser"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/golib/glog"
)
```

### blank import 必须写明用途

驱动注册类 `_ "..."` 导入必须在**上一行**写注释说明注册了什么、谁在用（`revive` 的 `blank-imports` 会报错）：

```go
import (
    "github.com/morehao/golib/dbaccess/dbgorm"
    // 注册 postgres 驱动：gormdao 按驱动名 "postgres" 解析连接，必须 blank import 触发 init。
    _ "github.com/morehao/golib/dbaccess/dbgorm/driver/postgres"
)
```

## 接口定义与依赖注入

使用接口定义服务层，通过构造函数注入：

```go
type UserSvc interface {
    Create(ctx *gin.Context, req *dtouser.UserCreateReq) (*dtouser.UserCreateResp, error)
}

type userSvc struct {
}

var _ UserSvc = (*userSvc)(nil)  // 编译时接口检查

func NewUserSvc() UserSvc {
    return &userSvc{}
}
```

## 错误处理

- 使用统一的错误码包 `github.com/morehao/ark-go-starter/pkg/code`
- 业务错误通过 `code.GetError(code.XXXError)` 返回
- 错误日志使用 `glog.Errorf(ctx, "[module.Method] msg, err:%v", err)`

```go
if err != nil {
    glog.Errorf(ctx, "[svcuser.Create] daoUser GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
    return nil, code.GetError(code.UserCreateError)
}
```

### 包装错误必须用 `%w`

需要保留错误链（供上层 `errors.Is` / `errors.As` 判断）时，用 `%w` 而非 `%v`；
仅用于日志展示的地方才用 `%v`（`errorlint` 会报错）：

```go
// 正确 —— 保留错误链
return fmt.Errorf("init db failed: %w", err)

// 错误 —— 错误链断裂，上层无法 errors.Is
return fmt.Errorf("init db failed: %v", err)

// 无格式化参数时用 errors.New，不要 fmt.Errorf（perfsprint 会报错）
return errors.New("db config is empty")
```

## 事务处理

使用 `dbclient` 封装的连接开启事务（本项目库名固定为 `demo`，用 `dbclient.DemoDB`）：

```go
txErr := dbclient.DemoDB(ctx).Transaction(func(tx *gorm.DB) error {
    // 事务内的 DAO 必须 WithTx 绑定到同一个 tx，否则写入不参与本事务
    userDao := dao.NewUserDao().WithTx(tx)
    if err := userDao.Insert(ctx, first); err != nil {
        return err   // 原样上抛，由事务层回滚
    }
    return nil
})
if txErr != nil {
    glog.Errorf(ctx, "[svcuser.CreateInTx] Transaction fail, err:%v", txErr)
    return nil, code.GetError(code.UserCreateError)
}
```

## Controller 返回模式

统一使用 `gincontext` 封装响应：

```go
func (ctr *userCtr) Create(ctx *gin.Context) {
    var req dtouser.UserCreateReq
    if err := ctx.ShouldBindJSON(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    res, err := ctr.userSvc.Create(ctx, &req)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    gincontext.Success(ctx, res)
}
```

## 数据库连接池（硬性要求）

`config.yaml` 的 `db_configs` 必须显式配置连接池三项，不依赖默认零值（golib `dbgorm` 仅在 `>0` 时才下发到 `sql.DB`）：

```yaml
db_configs:
  - url: "postgres://..."
    service: demo
    max_idle_conns: 10
    max_open_conns: 100
    conn_max_lifetime: 1h
```

## 提交前自检（硬性要求）

改完 Go 代码、提交前必须全部通过，否则 CI 会拦：

```bash
gofmt -w .          # 或 golangci-lint fmt ./...
make lint           # golangci-lint run，必须 0 issues
make test APP=demo  # 单元测试（无需 DB/Redis）
```

- `make test` 覆盖 `pkg` + 指定应用；依赖 DB/Redis 的集成测试单独用 `make test-integration APP=demo`；
- 在仓库根按模块分别检查（`pkg` 与 `apps/api`），见 [commands.md](./commands.md#lint-和代码检查)。

## 相关文档

- 路由与 Swagger 注解：[api-conventions.md](./api-conventions.md)
- 测试写法：[testing.md](./testing.md)
- 工具选型：[toolchain.md](./toolchain.md)
