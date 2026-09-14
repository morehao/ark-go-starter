# 用户注册策略设计

## 1. 概述

本文档描述 ark-go-starter 项目中用户注册策略的设计与实现方案。

### 1.1 背景

现有注册功能逻辑耦合在 `authSvc.Register` 方法中，仅支持域名驱动的租户选择和简单的审核配置。为支持多种注册场景（开放型、域名驱动、邀请制、仅管理员添加、SSO），需要对注册流程进行重构，采用策略模式实现。

### 1.2 目标

- 支持 5 种注册策略：开放型、域名驱动、邀请制、SSO（管理员添加走后台创建，不走注册流程）
- 策略可扩展，新增策略只需实现接口
- 注册验证码支持（注册前验证、登录时验证）
- 注册后自动分配默认角色/部门
- 工号自动生成

---

## 2. 注册策略

### 2.1 策略类型

| 策略 | 类型标识 | 租户确定方式 |
|------|----------|--------------|
| 开放型 | `open` | 从组织配置读取指定租户 ID |
| 域名驱动 | `domain` | 从访问域名匹配租户 |
| 邀请制 | `invite` | 从邀请码关联获取租户 |
| SSO | `sso` | 从 SSO 绑定或配置默认租户 |
| 管理员添加 | - | 不走注册流程，由管理员后台创建 |

### 2.2 策略选择

通过组织配置项 `register_way` 指定注册策略类型。

---

## 3. 策略接口设计

### 3.1 接口定义

```go
type RegisterStrategy interface {
    PreRegister(ctx *gin.Context, req *dtouser.RegisterReq) (*RegisterResult, error)
    PostRegister(ctx *gin.Context, req *dtouser.RegisterReq, userID uint) error
    GetStrategyType() RegisterStrategyType
}
```

**PreRegister 职责**：
1. 校验注册条件（如开关、租户状态、域名匹配等）
2. 确定目标租户
3. 校验用户信息（如邮箱、手机号唯一性）
4. 生成密码哈希

**PostRegister 职责**：
1. 分配默认角色/部门
2. 其他后置操作（如发送通知、记录邀请关系）

**GetStrategyType**：
- 返回策略类型标识，用于日志、审计

### 3.2 注册结果

```go
type RegisterResult struct {
    TenantID     uint              // 目标租户ID
    PersonID     uint              // 人员ID
    UserID       uint              // 用户ID
    Status       model.UserStatus  // 用户状态（enabled/pending）
    PersonExists bool              // 人员是否已存在
    PasswordHash string            // 密码哈希
    Message      string            // 提示信息
}
```

---

## 4. 策略实现

### 4.1 策略选择器

```go
type strategySelector struct {
    configKeyClient *configkv.Client
}

func (s *strategySelector) SelectStrategy(ctx *gin.Context, req *dtouser.RegisterReq) (RegisterStrategy, error) {
    orgEntity, err := getCurrentOrg(ctx)
    if err != nil {
        return nil, err
    }

    registerWay, err := s.configKeyClient.GetString(ctx, orgEntity.ID, model.OrgConfigKeyRegisterWay)
    if err != nil {
        return nil, code.GetError(code.AuthRegisterDisabled)
    }

    switch RegisterStrategyType(registerWay) {
    case RegisterStrategyOpen:
        return NewOpenStrategy(s.configKeyClient), nil
    case RegisterStrategyDomain:
        return NewDomainStrategy(s.configKeyClient), nil
    case RegisterStrategyInvite:
        return NewInviteStrategy(s.configKeyClient), nil
    case RegisterStrategySSO:
        return NewSSOStrategy(s.configKeyClient), nil
    default:
        return nil, code.GetError(code.AuthRegisterDisabled)
    }
}
```

### 4.2 公共方法

各策略共用以下公共方法：

```go
type strategyCommon struct {
    configKeyClient *configkv.Client
}
```

#### 4.2.1 createRegisterResult

```go
func (sc *strategyCommon) createRegisterResult(ctx *gin.Context, orgID, tenantID uint, req *dtouser.RegisterReq) (*RegisterResult, error)
```

1. 校验注册开关 `register_enabled`
2. 校验身份信息（邮箱/手机号唯一性）
3. 生成密码哈希
4. 根据 `register_require_approval` 配置决定用户状态

#### 4.2.2 validateIdentity

```go
func (sc *strategyCommon) validateIdentity(ctx *gin.Context, req *dtouser.RegisterReq, identityType model.RegisterIdentityType) error
```

根据 `register_identity_type` 配置校验身份信息唯一性。

#### 4.2.3 createUser

```go
func (sc *strategyCommon) createUser(ctx *gin.Context, tx *gorm.DB, result *RegisterResult, req *dtouser.RegisterReq) (uint, error)
```

1. 生成工号（调用 `generateEmployeeNo`）
2. 创建用户实体

#### 4.2.4 generateEmployeeNo

```go
func (sc *strategyCommon) generateEmployeeNo(ctx *gin.Context, tenantCode string) (string, error)
```

工号格式：`租户编码前2位 + 年月日(8位) + 4位序号`

示例：`GO202601010001`

序号使用 Redis INCR 生成，每日重置，过期时间次日零点。

#### 4.2.5 assignDefaultRolesAndDepts

```go
func (sc *strategyCommon) assignDefaultRolesAndDepts(ctx *gin.Context, userID uint) error
```

从配置 `register_default_roles` 和 `register_default_depts` 读取默认角色/部门ID列表，为用户分配。

### 4.3 OpenStrategy（开放型）

```go
type openStrategy struct {
    *strategyCommon
}
```

**PreRegister**：
1. 从配置 `register_open_tenant_id` 获取指定租户 ID
2. 校验租户状态
3. 调用 `createRegisterResult`

### 4.4 DomainStrategy（域名驱动）

```go
type domainStrategy struct {
    *strategyCommon
}
```

**PreRegister**：
1. 从请求中解析域名
2. 根据域名和 orgID 查询租户（`TenantCond{Domain: domain}`)
3. 校验租户状态
4. 调用 `createRegisterResult`

### 4.5 InviteStrategy（邀请制）

```go
type inviteStrategy struct {
    *strategyCommon
}
```

**PreRegister**：
1. 校验邀请码必填
2. 查询邀请码（`InviteCodeCond{Code: inviteCode}`)
3. 校验邀请码状态（active）
4. 校验过期时间（使用 `register_code_expire_hours` 配置）
5. 校验使用次数（使用 `register_code_max_use` 配置）
6. 获取邀请码关联的租户
7. 校验租户状态
8. 使用次数 +1
9. 调用 `createRegisterResult`

**PostRegister**：
1. 调用 `assignDefaultRolesAndDepts`
2. 记录邀请关系（被邀请人 -> 邀请码 -> 邀请人）

### 4.6 SSOStrategy（SSO）

```go
type ssoStrategy struct {
    *strategyCommon
}
```

**PreRegister**：
1. 从请求获取 SSO 类型（`sso_type`）和 openID
2. 查询 SSO 绑定表（`SSOBindCond{SSOType: ssoType, OpenID: openID}`）
3. 如已绑定租户，返回对应租户
4. 如未绑定，使用配置 `register_sso_default_tenant_id` 的默认租户
5. 调用 `createRegisterResult`

**PostRegister**：
1. 调用 `assignDefaultRolesAndDepts`
2. SSO 用户可能需要补充信息（手机号等）

---

## 5. Service 层改造

### 5.1 Register 入口

```go
func (svc *authSvc) Register(ctx *gin.Context, req *dtouser.RegisterReq) (*dtouser.RegisterResp, error)
```

1. 选择注册策略
2. 执行 PreRegister
3. 创建或获取人员（事务）
4. 创建用户（事务）
5. 执行 PostRegister
6. 返回注册结果

### 5.2 createOrGetPerson

```go
func (svc *authSvc) createOrGetPerson(ctx *gin.Context, tx *gorm.DB, result *RegisterResult, req *dtouser.RegisterReq) (uint, error)
```

1. 根据邮箱查询人员
2. 如存在且 `PersonExists=true`，返回已有人员 ID
3. 如不存在，创建新人员（包含密码哈希）

---

## 6. 验证码支持

### 6.1 验证码类型

| 类型 | 用途 | 配置项 |
|------|------|--------|
| `register` | 注册前验证邮箱/手机 | `register_verify_enabled` |
| `login` | 登录时防暴力破解 | `login_captcha_enabled` |

### 6.2 验证码场景

#### 6.2.1 注册前验证

- 用户注册时需先完成验证码校验
- 验证码发送到请求的邮箱/手机
- 校验通过后才能调用注册接口
- 配置项：`register_verify_enabled`

#### 6.2.2 登录时验证

- 连续登录失败 N 次后触发验证码
- 配置项：`login_captcha_enabled`、`login_max_fail_count`

### 6.3 验证码生成与校验

使用 Redis 存储验证码：
- Key 格式：`captcha:{type}:{identifier}`（identifier 为邮箱或手机号）
- Value：验证码
- 过期时间：5 分钟

---

## 7. 数据模型

### 7.1 邀请码表（新增）

```go
type InviteCodeEntity struct {
    gorm.Model
    OrgID       uint       `gorm:"column:org_id;type:bigint;not null;default 0"`
    TenantID    uint       `gorm:"column:tenant_id;type:bigint;not null;default 0"`
    Code        string     `gorm:"column:code;type:varchar(32);not null;default ''"`
    Status      string     `gorm:"column:status;type:varchar(16);default 'active'"`
    ExpiredAt   *time.Time `gorm:"column:expired_at;type:datetime;"`
    MaxUseCount int        `gorm:"column:max_use_count;type:int;default 0"`
    UseCount    int        `gorm:"column:use_count;type:int;default 0"`
    CreatedBy   uint       `gorm:"column:created_by;type:bigint;not null;default 0"`
}
```

表名：`iam_invite_code`

### 7.2 SSO 绑定表（新增）

```go
type SSOBindEntity struct {
    gorm.Model
    OrgID    uint   `gorm:"column:org_id;type:bigint;not null;default 0"`
    TenantID uint   `gorm:"column:tenant_id;type:bigint;not null;default 0"`
    UserID   uint   `gorm:"column:user_id;type:bigint;not null;default 0"`
    SSOType  string `gorm:"column:sso_type;type:varchar(32);not null;default ''"`
    OpenID   string `gorm:"column:open_id;type:varchar(128);not null;default ''"`
}
```

表名：`iam_sso_bind`

### 7.3 组织配置项（新增）

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `register_way` | string | 注册策略：open/domain/invite/sso |
| `register_open_tenant_id` | string | 开放型注册指定租户 ID |
| `register_sso_default_tenant_id` | string | SSO 默认租户 ID |
| `register_code_expire_hours` | int | 邀请码过期小时数 |
| `register_code_max_use` | int | 邀请码最大使用次数 |
| `register_verify_enabled` | bool | 注册验证码开关 |
| `register_default_roles` | JSON | 注册后默认角色 ID 列表 |
| `register_default_depts` | JSON | 注册后默认部门 ID 列表 |
| `login_captcha_enabled` | bool | 登录验证码开关 |
| `login_max_fail_count` | int | 登录失败次数阈值 |

---

## 8. DTO 改造

### 8.1 RegisterReq 新增字段

```go
type RegisterReq struct {
    Username   string `json:"username" validate:"required" label:"用户名"`
    Password   string `json:"password" validate:"required" label:"密码"`
    Mobile     string `json:"mobile" label:"手机号"`
    Email      string `json:"email" validate:"required" label:"邮箱"`
    RealName   string `json:"realName" validate:"required" label:"真实姓名"`
    InviteCode string `json:"inviteCode" label:"邀请码"`      // 邀请制
    SSOType    string `json:"ssoType" label:"SSO类型"`       // SSO
    OpenID     string `json:"openID" label:"OpenID"`         // SSO
}
```

### 8.2 CaptchaReq（验证码请求）

```go
type CaptchaReq struct {
    Type       string `json:"type" validate:"required" label:"类型"`       // register/login
    Identifier string `json:"identifier" validate:"required" label:"标识"` // 邮箱/手机号
}
```

### 8.3 CaptchaResp（验证码响应）

```go
type CaptchaResp struct {
    CaptchaID  string `json:"captchaId"`  // 验证码ID，用于前端关联
    ExpiresAt  int64  `json:"expiresAt"` // 过期时间戳
    // 图片验证码时返回 base64
    ImageData  string `json:"imageData,omitempty"`
}
```

---

## 9. API 设计

### 9.1 注册相关 API

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 注册 | POST | `/v1/iam/user/register` | 根据策略注册 |
| 发送验证码 | POST | `/v1/iam/user/sendCaptcha` | 发送注册验证码 |
| 校验验证码 | POST | `/v1/iam/user/verifyCaptcha` | 校验验证码 |
| 登录 | POST | `/v1/iam/user/loginByPassword` | 登录（可触发验证码） |

### 9.2 邀请码管理 API（管理员）

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 创建邀请码 | POST | `/v1/iam/inviteCode/create` | 创建邀请码 |
| 邀请码列表 | POST | `/v1/iam/inviteCode/pageList` | 分页查询 |
| 删除邀请码 | POST | `/v1/iam/inviteCode/delete` | 删除邀请码 |

---

## 10. 文件结构

```
apps/iam/
├── internal/
│   ├── service/svcuser/
│   │   ├── auth.go              # 保留登录相关
│   │   ├── register.go          # 新增：注册主入口
│   │   └── strategy/
│   │       ├── selector.go      # 策略选择器
│   │       ├── common.go        # 公共方法
│   │       ├── open.go          # 开放型策略
│   │       ├── domain.go        # 域名驱动策略
│   │       ├── invite.go        # 邀请制策略
│   │       └── sso.go           # SSO策略
│   ├── controller/ctruser/
│   │   ├── user.go              # 保留原有
│   │   └── captcha.go           # 新增：验证码控制器
│   └── router/
│       └── user.go              # 路由注册
├── model/
│   ├── invite_code.go           # 新增：邀请码模型
│   └── sso_bind.go              # 新增：SSO绑定模型
└── internal/dto/dtouser/
    ├── auth.go                  # 更新：新增字段
    └── captcha.go               # 新增：验证码DTO
```

---

## 11. 实现顺序

1. 数据模型：邀请码表、SSO绑定表
2. 策略接口和选择器
3. OpenStrategy 实现
4. DomainStrategy 实现
5. InviteStrategy 实现
6. SSOStrategy 实现
7. Service 层改造（注册主入口）
8. 验证码支持
9. DTO 改造
10. API 路由和控制器
11. 测试

---

## 12. 注意事项

1. 策略实现需遵循错误处理规范，使用 `glog.Errorf` 记录错误日志
2. 数据库操作使用事务保证数据一致性
3. 验证码需设置合理的过期时间和使用限制
4. SSO 用户无需密码，但需要补充手机号等基本信息
5. 邀请码使用次数递增操作需考虑并发场景