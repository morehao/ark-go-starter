# IAM 基础能力补充设计

## 一、背景

当前 ark-go-starter IAM 服务已具备基础的认证、用户、组织、部门、角色、菜单等管理能力。为支持多租户和单企业混合场景，需要补充以下基础能力：

1. 用户个人中心
2. 密码安全策略
3. 管理员创建用户

## 二、设计方案

### 2.1 用户个人中心模块

#### 2.1.1 新增端点

| HTTP方法 | 路由 | 说明 |
|----------|------|------|
| GET | `/v1/iam/user/getCurrentUserInfo` | 获取当前用户信息 |
| POST | `/v1/iam/user/updateProfile` | 更新个人资料（头像、邮箱、手机） |
| POST | `/v1/iam/user/changePassword` | 修改密码 |
| GET | `/v1/iam/user/loginHistory` | 查看登录历史 |
| POST | `/v1/iam/user/logout` | 使当前 token 和 refreshToken 失效 |

#### 2.1.2 架构设计

- 个人资料走 `PersonEntity`，通过当前用户的 userId 关联
- 登录历史复用现有的 `LoginLogEntity`
- 登出功能通过将当前 token 和 refreshToken 加入黑名单实现

#### 2.1.3 数据模型

**GetCurrentUserInfo 响应：**
```go
type UserInfoResp struct {
    UserID       string   `json:"user_id"`
    Username     string   `json:"username"`
    PersonID     string   `json:"person_id"`
    Email        string   `json:"email"`
    Phone        string   `json:"phone"`
    Avatar       string   `json:"avatar"`
    Nickname     string   `json:"nickname"`
    Status       string   `json:"status"`
    UserType     string   `json:"user_type"`
    TenantID     string   `json:"tenant_id"`
    TenantName   string   `json:"tenant_name"`
    OrgID        string   `json:"org_id"`
    OrgName      string   `json:"org_name"`
    RoleIDs      []string `json:"role_ids"`
    RoleNames    []string `json:"role_names"`
    DeptIDs      []string `json:"dept_ids"`
    DeptNames    []string `json:"dept_names"`
    MenuIDs      []string `json:"menu_ids"`
    MenuCodes    []string `json:"menu_codes"`
}
```

**UpdateProfile 请求：**
```go
type UpdateProfileReq struct {
    Email   string `json:"email"`
    Phone   string `json:"phone"`
    Avatar  string `json:"avatar"`
    Nickname string `json:"nickname"`
}
```

**ChangePassword 请求：**
```go
type ChangePasswordReq struct {
    OldPassword string `json:"old_password" binding:"required"`
    NewPassword string `json:"new_password" binding:"required"`
}
```

**LoginHistory 请求：**
```go
type LoginHistoryReq struct {
    PageRequest
    StartTime string `json:"start_time"`
    EndTime   string `json:"end_time"`
}
```

### 2.2 密码安全策略模块

#### 2.2.1 新增配置项

存储在组织配置（OrganizationConfig）中：

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `password_min_length` | int | 8 | 最小长度 |
| `password_require_uppercase` | bool | true | 必须包含大写字母 |
| `password_require_lowercase` | bool | true | 必须包含小写字母 |
| `password_require_number` | bool | true | 必须包含数字 |
| `password_require_special` | bool | false | 必须包含特殊字符 |
| `login_max_fail_count` | int | 5 | 登录失败最大次数 |
| `login_lock_duration` | int | 300 | 锁定时长/秒 |

#### 2.2.2 用户状态扩展

```go
const (
    UserStatusEnabled  = "enabled"
    UserStatusLocked   = "locked"   // 新增
    UserStatusDisabled = "disabled"
)
```

#### 2.2.3 用户表新增字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `login_fail_count` | int | 连续登录失败次数 |
| `locked_until` | time | 锁定截止时间 |

#### 2.2.4 登录流程调整

1. 验证密码前检查 `locked_until`，若已过期则清零计数
2. 密码验证失败时 `login_fail_count++`，超过阈值则设置 `locked_until`
3. 密码验证成功时清零计数

#### 2.2.5 自助解锁

新增端点：
| HTTP方法 | 路由 | 说明 |
|----------|------|------|
| POST | `/v1/iam/auth/unlockAccount` | 通过验证码自助解锁 |

### 2.3 管理员创建用户模块

#### 2.3.1 端点调整

现有 `/v1/iam/user/create` 扩展：

| 字段 | 类型 | 说明 |
|------|------|------|
| `password` | string | 可选，管理员指定 |
| `send_notify` | bool | 是否发送通知（当前仅记录日志） |

#### 2.3.2 创建响应扩展

返回值增加 `password` 字段（仅创建时返回一次）。

## 三、实现计划

### Phase 1: 用户个人中心
1. 实现 getCurrentUserInfo
2. 实现 updateProfile
3. 实现 changePassword
4. 实现 loginHistory
5. 实现 logout

### Phase 2: 密码安全策略
1. 添加用户表字段
2. 实现密码复杂度校验
3. 调整登录逻辑（失败计数、锁定）
4. 实现 unlockAccount

### Phase 3: 管理员创建用户
1. 扩展 create 接口
2. 添加密码生成逻辑
3. 添加通知日志记录

## 四、明确不包括的功能

- MFA、OAuth2、SSO
- 密码过期策略、密码历史
- 邀请注册流程
- 异常告警、日志保留策略
