# 后端代码规范

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

适用于 `backend/` 下所有 Go 代码（`apps/*` 与 `pkg/`）。

## 项目结构（后端）

```
backend/apps/
├── demo/                     # Demo 应用
│   ├── cmd/                  # 入口函数
│   ├── internal/
│   │   ├── controller/ctrxxx/  # 控制器层 (ctr 前缀)
│   │   ├── service/svcxxx/     # 服务层 (svc 前缀)
│   │   ├── dto/dtoxxx/         # DTO 层
│   │   ├── router/             # 路由注册
│   │   └── middleware/         # 中间件
│   ├── model/              # 数据模型
│   ├── dao/                # 数据访问层
│   ├── client/             # 外部客户端
│   └── object/             # 基础对象
backend/pkg/                # 公共包（跨应用共享）
```

## 命名规范

- **包名**: 小写，简短，如 `svcuser`, `dtouser`
- **接口名**: 以 `I` 结尾或使用角色后缀，如 `UserSvc`, `UserCtr`
- **结构体**: 导出使用大驼峰 `UserSvc`，非导出使用小驼峰 `userSvc`
- **文件命名**: 小写下划线，如 `user_service.go`，测试文件 `*_test.go`
- **数据库表**: 下划线命名，如 `user_department`

## 注释规范

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

DDL（`backend/scripts/sql/table_init.sql`）中**每个字段都必须有注释**，且要写两份、保持一致：

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

**本仓库唯一目标库是 PostgreSQL**：建表事实源是 `backend/scripts/sql/table_init.sql`（PG 语法），model 的 gorm tag 只描述列名、Go 类型与注释，**不得携带其它方言的类型字面量**。

| 想要的效果 | 写法 | PG 生成的列 |
|-----------|------|-------------|
| 整型（含无符号语义） | `uint` / `int64`（必要时 `size:16`/`size:32`） | `bigint`、`smallint` … |
| 变长字符串 | `size:255` | `varchar(255)` |
| 长文本 | 什么都不写（string 默认） | `text` |
| PostgreSQL 专属类型 | 直接写 PG 写法，如 `type:jsonb` | `jsonb` |
| 小数 | `precision:10;scale:2` | `numeric(10, 2)` |
| 列注释 | `comment:中文说明` | 建表后自动 `COMMENT ON COLUMN` |

```go
type UserEntity struct {
    gorm.Model
    CompanyID uint   `gorm:"column:company_id;not null;default:0;comment:公司id"`
    Name      string `gorm:"column:name;size:255;not null;default:'';comment:姓名"`
}
```

原因：gorm 只把 `bool/int/uint/float/string/time/bytes` 这类**逻辑类型名**交给方言映射，其它取值会被**原样拼进 DDL**。`type:bigint unsigned` 在 PG 上实测直接失败：

```
ERROR: syntax error at or near "unsigned" (SQLSTATE 42601)
CREATE TABLE "user" (...,"company_id" bigint unsigned NOT NULL DEFAULT 0,...)
```

守护用例：`apps/demo/model/pg_dialect_test.go` 的 `TestModelTagsArePostgresCompatible` 扫描所有模型的 `type:` tag，命中 MySQL 字面量（`unsigned`、`tinyint`、`datetime`、`longtext` …）即失败；**新增 model 要记得加进该用例的 `postgresModels`**（不需要数据库，纯 tag 校验）。

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
// apps/iam/internal/constant/status.go
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
3. 项目内部包 (`github.com/morehao/ark-go-starter/apps/demo/...`, `github.com/morehao/ark-go-starter/pkg/...`)
4. 关联库 (`github.com/morehao/golib/...`)

```go
import (
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify"

    "github.com/morehao/ark-go-starter/apps/demo/internal/dto/dtouser"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/golib/glog"
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

## 事务处理

使用 `dbclient` 封装的事务：

```go
txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
    result, err = user.CreatePersonWithUser(ctx, tx, params)
    if err != nil {
        return err
    }
    return nil
})
if txErr != nil {
    glog.Errorf(ctx, "[svcuser.Create] Transaction fail, err:%v", txErr)
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

## 相关文档

- 路由与 Swagger 注解：[api-conventions.md](./api-conventions.md)
- 测试写法：[testing.md](./testing.md)
- 工具选型：[toolchain.md](./toolchain.md)
