# 用户注册策略实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现用户注册策略模块，支持开放型、域名驱动、邀请制、SSO 四种注册策略，以及验证码支持。

**Architecture:** 采用策略模式，通过 `RegisterStrategy` 接口定义注册流程，各策略独立实现。Service 层通过策略选择器根据配置选择对应策略执行。

**Tech Stack:** Go, GORM, Gin, Redis

---

## 文件结构

```
apps/iam/
├── model/
│   ├── invite_code.go           # 新增：邀请码模型
│   ├── sso_bind.go              # 新增：SSO绑定模型
│   └── organization_config.go    # 修改：新增配置项常量
├── dao/
│   ├── invite_code.go           # 新增：邀请码DAO
│   ├── sso_bind.go              # 新增：SSO绑定DAO
│   └── organization_config.go   # 修改：新增配置查询方法
├── internal/
│   ├── service/svcuser/
│   │   ├── auth.go              # 修改：移除注册逻辑
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
│   └── dto/dtouser/
│       ├── auth.go              # 修改：新增字段
│       └── captcha.go           # 新增：验证码DTO
```

---

## Task 1: 新增配置项常量

**Files:**
- Modify: `apps/iam/model/organization_config.go`

- [ ] **Step 1: 添加新配置项常量**

在 `organization_config.go` 的配置项常量区添加以下常量：

```go
const (
    OrgConfigKeyRegisterWay              = "auth.register.way"
    OrgConfigKeyRegisterOpenTenantID     = "auth.register.openTenantId"
    OrgConfigKeyRegisterSSODefaultTenantID = "auth.register.ssoDefaultTenantId"
    OrgConfigKeyRegisterCodeExpireHours  = "auth.register.codeExpireHours"
    OrgConfigKeyRegisterCodeMaxUse        = "auth.register.codeMaxUse"
    OrgConfigKeyRegisterVerifyEnabled   = "auth.register.verifyEnabled"
    OrgConfigKeyRegisterDefaultRoles     = "auth.register.defaultRoles"
    OrgConfigKeyRegisterDefaultDepts     = "auth.register.defaultDepts"
    OrgConfigKeyLoginCaptchaEnabled      = "auth.login.captchaEnabled"
)
```

在 `OrgConfigMetaList` 中添加新配置项的元数据定义。

---

## Task 2: 创建邀请码模型和DAO

**Files:**
- Create: `apps/iam/model/invite_code.go`
- Create: `apps/iam/dao/invite_code.go`

- [ ] **Step 1: 创建邀请码模型**

```go
package model

import (
    "time"

    "gorm.io/gorm"
)

type InviteCodeStatus string

const (
    InviteCodeStatusActive   InviteCodeStatus = "active"
    InviteCodeStatusDisabled InviteCodeStatus = "disabled"
    InviteCodeStatusExpired  InviteCodeStatus = "expired"
)

type InviteCodeEntity struct {
    gorm.Model
    OrgID       uint             `gorm:"column:org_id;type:bigint;not null;default 0;comment: 组织ID"`
    TenantID    uint             `gorm:"column:tenant_id;type:bigint;not null;default 0;comment: 租户ID"`
    Code        string           `gorm:"column:code;type:varchar(32);not null;default '';comment: 邀请码"`
    Status      InviteCodeStatus `gorm:"column:status;type:varchar(16);default 'active';comment: 状态"`
    ExpiredAt   *time.Time      `gorm:"column:expired_at;type:datetime;"`
    MaxUseCount int             `gorm:"column:max_use_count;type:int;default 0;comment: 最大使用次数，0表示不限制"`
    UseCount    int             `gorm:"column:use_count;type:int;default 0;comment: 已使用次数"`
    CreatedBy   uint            `gorm:"column:created_by;type:bigint;not null;default 0;comment: 创建人ID"`
}

type InviteCodeEntityList []InviteCodeEntity

const TableNameInviteCode = "iam_invite_code"

func (InviteCodeEntity) TableName() string {
    return TableNameInviteCode
}

func (l InviteCodeEntityList) ToMap() map[uint]InviteCodeEntity {
    m := make(map[uint]InviteCodeEntity)
    for _, v := range l {
        m[v.ID] = v
    }
    return m
}
```

- [ ] **Step 2: 创建邀请码DAO**

```go
package dao

import (
    "context"

    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/ark-go-starter/pkg/dbclient"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

type InviteCodeCond struct {
    *genericdao.BaseCond
    OrgID   uint
    TenantID uint
    Code    string
    Status  model.InviteCodeStatus
}

func (c *InviteCodeCond) BuildCondition(db *gorm.DB, tableName string) {
    if c.BaseCond != nil {
        c.BaseCond.BuildCondition(db, tableName)
    }
    if c.OrgID > 0 {
        db.Where(tableName+".org_id = ?", c.OrgID)
    }
    if c.TenantID > 0 {
        db.Where(tableName+".tenant_id = ?", c.TenantID)
    }
    if c.Code != "" {
        db.Where(tableName+".code = ?", c.Code)
    }
    if c.Status != "" {
        db.Where(tableName+".status = ?", c.Status)
    }
}

type InviteCodeDao struct {
    *genericdao.GenericDao[model.InviteCodeEntity, model.InviteCodeEntityList]
}

func NewInviteCodeDao() *InviteCodeDao {
    return &InviteCodeDao{
        GenericDao: genericdao.NewGenericDao[model.InviteCodeEntity, model.InviteCodeEntityList](
            model.TableNameInviteCode, "InviteCodeDao",
            dbclient.IamDB,
        ),
    }
}

func (dao *InviteCodeDao) IncrUseCount(ctx context.Context, id uint) (int, error) {
    return dao.DB().WithContext(ctx).Model(&model.InviteCodeEntity{}).Where("id = ?", id).
        Update("use_count", gorm.Expr("use_count + 1")).RowsAffected, nil
}
```

---

## Task 3: 创建SSO绑定模型和DAO

**Files:**
- Create: `apps/iam/model/sso_bind.go`
- Create: `apps/iam/dao/sso_bind.go`

- [ ] **Step 1: 创建SSO绑定模型**

```go
package model

import (
    "gorm.io/gorm"
)

type SSOBindEntity struct {
    gorm.Model
    OrgID    uint   `gorm:"column:org_id;type:bigint;not null;default 0;comment: 组织ID"`
    TenantID uint   `gorm:"column:tenant_id;type:bigint;not null;default 0;comment: 租户ID"`
    UserID   uint   `gorm:"column:user_id;type:bigint;not null;default 0;comment: 用户ID"`
    SSOType  string `gorm:"column:sso_type;type:varchar(32);not null;default '';comment: SSO类型，如wechat/oidc"`
    OpenID   string `gorm:"column:open_id;type:varchar(128);not null;default '';comment: OpenID"`
}

type SSOBindEntityList []SSOBindEntity

const TableNameSSOBind = "iam_sso_bind"

func (SSOBindEntity) TableName() string {
    return TableNameSSOBind
}

func (l SSOBindEntityList) ToMap() map[uint]SSOBindEntity {
    m := make(map[uint]SSOBindEntity)
    for _, v := range l {
        m[v.ID] = v
    }
    return m
}
```

- [ ] **Step 2: 创建SSO绑定DAO**

```go
package dao

import (
    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/ark-go-starter/pkg/dbclient"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

type SSOBindCond struct {
    *genericdao.BaseCond
    OrgID   uint
    TenantID uint
    UserID  uint
    SSOType string
    OpenID  string
}

func (c *SSOBindCond) BuildCondition(db *gorm.DB, tableName string) {
    if c.BaseCond != nil {
        c.BaseCond.BuildCondition(db, tableName)
    }
    if c.OrgID > 0 {
        db.Where(tableName+".org_id = ?", c.OrgID)
    }
    if c.TenantID > 0 {
        db.Where(tableName+".tenant_id = ?", c.TenantID)
    }
    if c.UserID > 0 {
        db.Where(tableName+".user_id = ?", c.UserID)
    }
    if c.SSOType != "" {
        db.Where(tableName+".sso_type = ?", c.SSOType)
    }
    if c.OpenID != "" {
        db.Where(tableName+".open_id = ?", c.OpenID)
    }
}

type SSOBindDao struct {
    *genericdao.GenericDao[model.SSOBindEntity, model.SSOBindEntityList]
}

func NewSSOBindDao() *SSOBindDao {
    return &SSOBindDao{
        GenericDao: genericdao.NewGenericDao[model.SSOBindEntity, model.SSOBindEntityList](
            model.TableNameSSOBind, "SSOBindDao",
            dbclient.IamDB,
        ),
    }
}
```

---

## Task 4: 创建策略接口和选择器

**Files:**
- Create: `apps/iam/internal/service/svcuser/strategy/selector.go`

- [ ] **Step 1: 创建策略接口和类型**

在 `selector.go` 中定义：

```go
package strategy

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/model"
)

type RegisterStrategyType string

const (
    RegisterStrategyOpen   RegisterStrategyType = "open"
    RegisterStrategyDomain RegisterStrategyType = "domain"
    RegisterStrategyInvite RegisterStrategyType = "invite"
    RegisterStrategySSO    RegisterStrategyType = "sso"
)

type RegisterStrategy interface {
    PreRegister(ctx *gin.Context, req *RegisterRequest) (*RegisterResult, error)
    PostRegister(ctx *gin.Context, req *RegisterRequest, userID uint) error
    GetStrategyType() RegisterStrategyType
}

type RegisterRequest struct {
    Username   string
    Password   string
    Mobile     string
    Email      string
    RealName   string
    InviteCode string
    SSOType    string
    OpenID     string
}

type RegisterResult struct {
    TenantID     uint
    PersonID     uint
    PasswordHash string
    Status       model.UserStatus
    PersonExists bool
    Message      string
}
```

- [ ] **Step 2: 创建策略选择器**

```go
type strategySelector struct {
}

func NewStrategySelector() *strategySelector {
    return &strategySelector{}
}

func (s *strategySelector) SelectStrategy(ctx *gin.Context, req *RegisterRequest) (RegisterStrategy, error) {
    orgEntity, err := s.getCurrentOrg(ctx)
    if err != nil {
        return nil, err
    }

    registerWay, err := s.getOrgConfigString(ctx, orgEntity.ID, model.OrgConfigKeyRegisterWay)
    if err != nil {
        return nil, code.GetError(code.AuthRegisterDisabled)
    }

    switch RegisterStrategyType(registerWay) {
    case RegisterStrategyOpen:
        return NewOpenStrategy(), nil
    case RegisterStrategyDomain:
        return NewDomainStrategy(), nil
    case RegisterStrategyInvite:
        return NewInviteStrategy(), nil
    case RegisterStrategySSO:
        return NewSSOStrategy(), nil
    default:
        return nil, code.GetError(code.AuthRegisterDisabled)
    }
}

func (s *strategySelector) getCurrentOrg(ctx *gin.Context) (*model.OrganizationEntity, error) {
    // 复用 authSvc 中的 getCurrentOrg 逻辑
    return getCurrentOrg(ctx)
}

func (s *strategySelector) getOrgConfigString(ctx *gin.Context, orgID uint, configKey string) (string, error) {
    // 复用 authSvc 中的 getOrgConfigString 逻辑
    return getOrgConfigString(ctx, orgID, configKey)
}
```

---

## Task 5: 创建公共方法

**Files:**
- Create: `apps/iam/internal/service/svcuser/strategy/common.go`

- [ ] **Step 1: 创建公共方法**

```go
package strategy

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/dao"
    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/ark-go-starter/pkg/dbclient"
    "github.com/morehao/golib/gcrypto"
    "github.com/morehao/golib/glog"
    "gorm.io/gorm"
)

type strategyCommon struct {
}

func newStrategyCommon() *strategyCommon {
    return &strategyCommon{}
}

func (sc *strategyCommon) getOrgConfigBool(ctx *gin.Context, orgID uint, configKey string) (bool, error) {
    configEntity, err := dao.NewOrganizationConfigDao().GetByCond(ctx, &dao.OrganizationConfigCond{
        OrgID:     orgID,
        ConfigKey: configKey,
    })
    if err != nil {
        glog.Errorf(ctx, "[strategyCommon.getOrgConfigBool] GetByCond fail, err:%v, orgID:%d, key:%s", err, orgID, configKey)
        return false, err
    }
    if configEntity == nil || configEntity.ID == 0 {
        return false, nil
    }
    return strings.ToLower(configEntity.ConfigValue) == "true", nil
}

func (sc *strategyCommon) getOrgConfigString(ctx *gin.Context, orgID uint, configKey string) (string, error) {
    configEntity, err := dao.NewOrganizationConfigDao().GetByCond(ctx, &dao.OrganizationConfigCond{
        OrgID:     orgID,
        ConfigKey: configKey,
    })
    if err != nil {
        glog.Errorf(ctx, "[strategyCommon.getOrgConfigString] GetByCond fail, err:%v, orgID:%d, key:%s", err, orgID, configKey)
        return "", err
    }
    if configEntity == nil || configEntity.ID == 0 {
        return "", nil
    }
    return configEntity.ConfigValue, nil
}

func (sc *strategyCommon) getOrgConfigInt(ctx *gin.Context, orgID uint, configKey string) (int, error) {
    configEntity, err := dao.NewOrganizationConfigDao().GetByCond(ctx, &dao.OrganizationConfigCond{
        OrgID:     orgID,
        ConfigKey: configKey,
    })
    if err != nil {
        glog.Errorf(ctx, "[strategyCommon.getOrgConfigInt] GetByCond fail, err:%v, orgID:%d, key:%s", err, orgID, configKey)
        return 0, err
    }
    if configEntity == nil || configEntity.ID == 0 {
        return 0, nil
    }
    var val int
    _, err = fmt.Sscanf(configEntity.ConfigValue, "%d", &val)
    return val, err
}

func (sc *strategyCommon) createRegisterResult(ctx *gin.Context, orgID, tenantID uint, req *RegisterRequest) (*RegisterResult, error) {
    registerEnabled, err := sc.getOrgConfigBool(ctx, orgID, model.OrgConfigKeyRegisterEnabled)
    if err != nil {
        glog.Errorf(ctx, "[createRegisterResult] GetBool registerEnabled fail, err:%v", err)
        return nil, code.GetError(code.AuthRegisterDisabled)
    }
    if !registerEnabled {
        return nil, code.GetError(code.AuthRegisterDisabled)
    }

    identityType, err := sc.getOrgConfigString(ctx, orgID, model.OrgConfigKeyRegisterIdentityType)
    if err != nil {
        glog.Errorf(ctx, "[createRegisterResult] GetString identityType fail, err:%v", err)
        return nil, code.GetError(code.AuthRegisterError)
    }
    if identityType == "" {
        identityType = string(model.RegisterIdentityTypeEmail)
    }
    if err := sc.validateIdentity(ctx, req, model.RegisterIdentityType(identityType)); err != nil {
        return nil, err
    }

    passwordHash, err := gcrypto.GeneratePasswordHash(req.Password)
    if err != nil {
        glog.Errorf(ctx, "[createRegisterResult] GeneratePasswordHash fail, err:%v", err)
        return nil, code.GetError(code.AuthRegisterError)
    }

    requireApproval, err := sc.getOrgConfigBool(ctx, orgID, model.OrgConfigKeyRegisterRequireApproval)
    if err != nil {
        glog.Errorf(ctx, "[createRegisterResult] GetBool requireApproval fail, err:%v", err)
        return nil, code.GetError(code.AuthRegisterError)
    }
    userStatus := model.UserStatusEnabled
    message := "注册成功"
    if requireApproval {
        userStatus = model.UserStatusPending
        message = "注册成功，等待管理员审核"
    }

    return &RegisterResult{
        TenantID:     tenantID,
        PasswordHash: passwordHash,
        Status:       userStatus,
        Message:      message,
    }, nil
}

func (sc *strategyCommon) validateIdentity(ctx *gin.Context, req *RegisterRequest, identityType model.RegisterIdentityType) error {
    switch identityType {
    case model.RegisterIdentityTypeEmail:
        email := strings.TrimSpace(req.Email)
        if email == "" {
            return code.GetError(code.UserEmailRequiredError)
        }
        person, err := dao.NewPersonDao().GetByCond(ctx, &dao.PersonCond{Email: email})
        if err != nil {
            glog.Errorf(ctx, "[validateIdentity] GetByCond email fail, err:%v", err)
            return code.GetError(code.UserEmailExistsError)
        }
        if person != nil && person.ID > 0 {
            return code.GetError(code.UserEmailExistsError)
        }
    case model.RegisterIdentityTypeMobile:
        mobile := strings.TrimSpace(req.Mobile)
        if mobile == "" {
            return code.GetError(code.UserMobileRequiredError)
        }
        person, err := dao.NewPersonDao().GetByCond(ctx, &dao.PersonCond{Mobile: mobile})
        if err != nil {
            glog.Errorf(ctx, "[validateIdentity] GetByCond mobile fail, err:%v", err)
            return code.GetError(code.UserMobileExistsError)
        }
        if person != nil && person.ID > 0 {
            return code.GetError(code.UserMobileExistsError)
        }
    case model.RegisterIdentityTypeBoth:
        email := strings.TrimSpace(req.Email)
        mobile := strings.TrimSpace(req.Mobile)
        if email == "" || mobile == "" {
            return code.GetError(code.UserIdentityRequiredError)
        }
        person, err := dao.NewPersonDao().GetByCond(ctx, &dao.PersonCond{Email: email, Mobile: mobile})
        if err != nil {
            glog.Errorf(ctx, "[validateIdentity] GetByCond both fail, err:%v", err)
            return code.GetError(code.UserIdentityExistsError)
        }
        if person != nil && person.ID > 0 {
            return code.GetError(code.UserIdentityExistsError)
        }
    }
    return nil
}

func (sc *strategyCommon) createUser(ctx *gin.Context, tx *gorm.DB, result *RegisterResult, req *RegisterRequest) (uint, error) {
    tenant, err := dao.NewTenantDao().GetByID(ctx, result.TenantID)
    if err != nil {
        glog.Errorf(ctx, "[createUser] GetByID tenant fail, err:%v", err)
        return 0, code.GetError(code.AuthRegisterError)
    }
    if tenant == nil {
        return 0, code.GetError(code.TenantNotExistError)
    }

    employeeNo, err := sc.generateEmployeeNo(ctx, tenant.TenantCode)
    if err != nil {
        return 0, err
    }

    userEntity := &model.UserEntity{
        TenantID:   result.TenantID,
        PersonID:   result.PersonID,
        EmployeeNo: employeeNo,
        Username:   req.Username,
        Status:     result.Status,
        UserType:   model.UserTypeNormal,
        CreatedBy:  0,
        UpdatedBy:  0,
    }
    if err := dao.NewUserDao().WithTx(tx).Insert(ctx, userEntity); err != nil {
        glog.Errorf(ctx, "[createUser] Insert fail, err:%v", err)
        return 0, code.GetError(code.UserCreateError)
    }
    return userEntity.ID, nil
}

func (sc *strategyCommon) generateEmployeeNo(ctx *gin.Context, tenantCode string) (string, error) {
    if len(tenantCode) < 2 {
        tenantCode = fmt.Sprintf("%-2s", tenantCode)
    }
    today := time.Now().Format("20060102")
    key := fmt.Sprintf("employee_no:%s:%s", tenantCode, today)

    seq, err := dbclient.RedisCli.Incr(ctx, key).Result()
    if err != nil {
        glog.Errorf(ctx, "[generateEmployeeNo] Redis Incr fail, err:%v", err)
        return "", code.GetError(code.InternalError)
    }

    expiry := time.Now().AddDate(0, 0, 1)
    if _, err := dbclient.RedisCli.ExpireAt(ctx, key, expiry).Result(); err != nil {
        glog.Errorf(ctx, "[generateEmployeeNo] Redis ExpireAt fail, err:%v", err)
    }

    return fmt.Sprintf("%s%s%04d", tenantCode[:2], today, seq), nil
}

func (sc *strategyCommon) assignDefaultRolesAndDepts(ctx *gin.Context, orgID uint, userID uint) error {
    defaultRoleIDsStr, err := sc.getOrgConfigString(ctx, orgID, model.OrgConfigKeyRegisterDefaultRoles)
    if err != nil && err.Error() != "record not found" {
        glog.Errorf(ctx, "[assignDefaultRolesAndDepts] GetConfig roles fail, err:%v", err)
        return code.GetError(code.InternalError)
    }
    if defaultRoleIDsStr != "" {
        var roleIDs []uint
        if err := json.Unmarshal([]byte(defaultRoleIDsStr), &roleIDs); err == nil {
            for _, roleID := range roleIDs {
                userRole := &model.UserRoleEntity{
                    UserID: userID,
                    RoleID: roleID,
                }
                if err := dao.NewUserRoleDao().Insert(ctx, userRole); err != nil {
                    glog.Errorf(ctx, "[assignDefaultRolesAndDepts] Insert role fail, userID:%d, roleID:%d, err:%v", userID, roleID, err)
                }
            }
        }
    }

    defaultDeptIDsStr, err := sc.getOrgConfigString(ctx, orgID, model.OrgConfigKeyRegisterDefaultDepts)
    if err != nil && err.Error() != "record not found" {
        glog.Errorf(ctx, "[assignDefaultRolesAndDepts] GetConfig depts fail, err:%v", err)
        return code.GetError(code.InternalError)
    }
    if defaultDeptIDsStr != "" {
        var deptIDs []uint
        if err := json.Unmarshal([]byte(defaultDeptIDsStr), &deptIDs); err == nil {
            for _, deptID := range deptIDs {
                userDept := &model.UserDepartmentEntity{
                    UserID: userID,
                    DeptID: deptID,
                }
                if err := dao.NewUserDepartmentDao().Insert(ctx, userDept); err != nil {
                    glog.Errorf(ctx, "[assignDefaultRolesAndDepts] Insert dept fail, userID:%d, deptID:%d, err:%v", userID, deptID, err)
                }
            }
        }
    }
    return nil
}
```

---

## Task 6: 实现OpenStrategy

**Files:**
- Create: `apps/iam/internal/service/svcuser/strategy/open.go`

- [ ] **Step 1: 实现开放型策略**

```go
package strategy

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/dao"
    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/golib/glog"
)

type openStrategy struct {
    *strategyCommon
}

func NewOpenStrategy() RegisterStrategy {
    return &openStrategy{
        strategyCommon: newStrategyCommon(),
    }
}

func (s *openStrategy) PreRegister(ctx *gin.Context, req *RegisterRequest) (*RegisterResult, error) {
    orgEntity, err := getCurrentOrg(ctx)
    if err != nil {
        return nil, err
    }

    tenantIDStr, err := s.getOrgConfigString(ctx, orgEntity.ID, model.OrgConfigKeyRegisterOpenTenantID)
    if err != nil {
        glog.Errorf(ctx, "[openStrategy.PreRegister] GetString tenantID fail, err:%v", err)
        return nil, code.GetError(code.AuthRegisterError)
    }
    if tenantIDStr == "" {
        glog.Errorf(ctx, "[openStrategy.PreRegister] tenantID not configured")
        return nil, code.GetError(code.AuthRegisterError)
    }

    var tenantID uint
    if _, err := fmt.Sscanf(tenantIDStr, "%d", &tenantID); err != nil {
        glog.Errorf(ctx, "[openStrategy.PreRegister] parse tenantID fail, err:%v", err)
        return nil, code.GetError(code.AuthRegisterError)
    }

    tenant, err := dao.NewTenantDao().GetByID(ctx, tenantID)
    if err != nil {
        glog.Errorf(ctx, "[openStrategy.PreRegister] GetByID tenant fail, err:%v, tenantID:%d", err, tenantID)
        return nil, code.GetError(code.TenantNotExistError)
    }
    if tenant == nil {
        return nil, code.GetError(code.TenantNotExistError)
    }
    if tenant.Status != model.TenantStatusEnabled {
        return nil, code.GetError(code.AuthRegisterError)
    }

    return s.createRegisterResult(ctx, orgEntity.ID, tenant.ID, req)
}

func (s *openStrategy) PostRegister(ctx *gin.Context, req *RegisterRequest, userID uint) error {
    orgEntity, err := getCurrentOrg(ctx)
    if err != nil {
        return err
    }
    return s.assignDefaultRolesAndDepts(ctx, orgEntity.ID, userID)
}

func (s *openStrategy) GetStrategyType() RegisterStrategyType {
    return RegisterStrategyOpen
}
```

---

## Task 7: 实现DomainStrategy

**Files:**
- Create: `apps/iam/internal/service/svcuser/strategy/domain.go`

- [ ] **Step 1: 实现域名驱动策略**

```go
package strategy

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/dao"
    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/golib/glog"
    "strings"
)

type domainStrategy struct {
    *strategyCommon
}

func NewDomainStrategy() RegisterStrategy {
    return &domainStrategy{
        strategyCommon: newStrategyCommon(),
    }
}

func (s *domainStrategy) PreRegister(ctx *gin.Context, req *RegisterRequest) (*RegisterResult, error) {
    orgEntity, err := getCurrentOrg(ctx)
    if err != nil {
        return nil, err
    }

    domain := resolveDomainFromHost(ctx)
    if domain == "" {
        return nil, code.GetError(code.TenantNotExistError)
    }

    tenant, err := dao.NewTenantDao().GetByCond(ctx, &dao.TenantCond{
        OrgID:  orgEntity.ID,
        Domain: domain,
    })
    if err != nil {
        glog.Errorf(ctx, "[domainStrategy.PreRegister] GetByCond fail, domain:%s, err:%v", domain, err)
        return nil, code.GetError(code.TenantNotExistError)
    }
    if tenant == nil {
        return nil, code.GetError(code.TenantNotExistError)
    }
    if tenant.Status != model.TenantStatusEnabled {
        return nil, code.GetError(code.AuthRegisterError)
    }

    return s.createRegisterResult(ctx, orgEntity.ID, tenant.ID, req)
}

func (s *domainStrategy) PostRegister(ctx *gin.Context, req *RegisterRequest, userID uint) error {
    orgEntity, err := getCurrentOrg(ctx)
    if err != nil {
        return err
    }
    return s.assignDefaultRolesAndDepts(ctx, orgEntity.ID, userID)
}

func (s *domainStrategy) GetStrategyType() RegisterStrategyType {
    return RegisterStrategyDomain
}

func resolveDomainFromHost(ctx *gin.Context) string {
    host := ctx.Request.Host
    if idx := strings.Index(host, ":"); idx != -1 {
        host = host[:idx]
    }
    return strings.TrimSpace(host)
}
```

---

## Task 8: 实现InviteStrategy

**Files:**
- Create: `apps/iam/internal/service/svcuser/strategy/invite.go`

- [ ] **Step 1: 实现邀请制策略**

```go
package strategy

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/dao"
    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/golib/glog"
    "strings"
    "time"
)

type inviteStrategy struct {
    *strategyCommon
}

func NewInviteStrategy() RegisterStrategy {
    return &inviteStrategy{
        strategyCommon: newStrategyCommon(),
    }
}

func (s *inviteStrategy) PreRegister(ctx *gin.Context, req *RegisterRequest) (*RegisterResult, error) {
    orgEntity, err := getCurrentOrg(ctx)
    if err != nil {
        return nil, err
    }

    inviteCode := strings.TrimSpace(req.InviteCode)
    if inviteCode == "" {
        return nil, code.GetError(code.InviteCodeRequiredError)
    }

    invite, err := dao.NewInviteCodeDao().GetByCond(ctx, &dao.InviteCodeCond{
        OrgID: orgEntity.ID,
        Code:  inviteCode,
    })
    if err != nil {
        glog.Errorf(ctx, "[inviteStrategy.PreRegister] GetByCond fail, code:%s, err:%v", inviteCode, err)
        return nil, code.GetError(code.InviteCodeInvalidError)
    }
    if invite == nil {
        return nil, code.GetError(code.InviteCodeInvalidError)
    }

    if invite.Status != model.InviteCodeStatusActive {
        return nil, code.GetError(code.InviteCodeInvalidError)
    }
    if invite.ExpiredAt != nil && invite.ExpiredAt.Before(time.Now()) {
        return nil, code.GetError(code.InviteCodeExpiredError)
    }
    maxUse, _ := s.getOrgConfigInt(ctx, orgEntity.ID, model.OrgConfigKeyRegisterCodeMaxUse)
    if maxUse > 0 && invite.UseCount >= maxUse {
        return nil, code.GetError(code.InviteCodeUsedUpError)
    }

    tenant, err := dao.NewTenantDao().GetByID(ctx, invite.TenantID)
    if err != nil {
        glog.Errorf(ctx, "[inviteStrategy.PreRegister] GetByID tenant fail, tenantID:%d, err:%v", invite.TenantID, err)
        return nil, code.GetError(code.TenantNotExistError)
    }
    if tenant == nil {
        return nil, code.GetError(code.TenantNotExistError)
    }
    if tenant.Status != model.TenantStatusEnabled {
        return nil, code.GetError(code.AuthRegisterError)
    }

    if _, err := dao.NewInviteCodeDao().IncrUseCount(ctx, invite.ID); err != nil {
        glog.Errorf(ctx, "[inviteStrategy.PreRegister] IncrUseCount fail, inviteID:%d, err:%v", invite.ID, err)
    }

    return s.createRegisterResult(ctx, orgEntity.ID, tenant.ID, req)
}

func (s *inviteStrategy) PostRegister(ctx *gin.Context, req *RegisterRequest, userID uint) error {
    orgEntity, err := getCurrentOrg(ctx)
    if err != nil {
        return err
    }
    if err := s.assignDefaultRolesAndDepts(ctx, orgEntity.ID, userID); err != nil {
        return err
    }
    return nil
}

func (s *inviteStrategy) GetStrategyType() RegisterStrategyType {
    return RegisterStrategyInvite
}
```

---

## Task 9: 实现SSOStrategy

**Files:**
- Create: `apps/iam/internal/service/svcuser/strategy/sso.go`

- [ ] **Step 1: 实现SSO策略**

```go
package strategy

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/dao"
    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/golib/glog"
)

type ssoStrategy struct {
    *strategyCommon
}

func NewSSOStrategy() RegisterStrategy {
    return &ssoStrategy{
        strategyCommon: newStrategyCommon(),
    }
}

func (s *ssoStrategy) PreRegister(ctx *gin.Context, req *RegisterRequest) (*RegisterResult, error) {
    orgEntity, err := getCurrentOrg(ctx)
    if err != nil {
        return nil, err
    }

    ssoType := req.SSOType
    openID := req.OpenID
    if openID == "" {
        return nil, code.GetError(code.AuthRegisterError)
    }

    tenant, err := s.findTenantBySSO(ctx, orgEntity.ID, ssoType, openID)
    if err != nil {
        return nil, err
    }

    return s.createRegisterResult(ctx, orgEntity.ID, tenant.ID, req)
}

func (s *ssoStrategy) PostRegister(ctx *gin.Context, req *RegisterRequest, userID uint) error {
    orgEntity, err := getCurrentOrg(ctx)
    if err != nil {
        return err
    }
    return s.assignDefaultRolesAndDepts(ctx, orgEntity.ID, userID)
}

func (s *ssoStrategy) GetStrategyType() RegisterStrategyType {
    return RegisterStrategySSO
}

func (s *ssoStrategy) findTenantBySSO(ctx *gin.Context, orgID uint, ssoType, openID string) (*model.TenantEntity, error) {
    ssoBind, err := dao.NewSSOBindDao().GetByCond(ctx, &dao.SSOBindCond{
        OrgID:   orgID,
        SSOType: ssoType,
        OpenID:  openID,
    })
    if err != nil {
        glog.Errorf(ctx, "[findTenantBySSO] GetByCond fail, ssoType:%s, err:%v", ssoType, err)
        return nil, code.GetError(code.AuthRegisterError)
    }
    if ssoBind != nil && ssoBind.TenantID > 0 {
        return dao.NewTenantDao().GetByID(ctx, ssoBind.TenantID)
    }

    tenantIDStr, err := s.getOrgConfigString(ctx, orgID, model.OrgConfigKeyRegisterSSODefaultTenantID)
    if err != nil {
        glog.Errorf(ctx, "[findTenantBySSO] GetString defaultTenantID fail, err:%v", err)
        return nil, code.GetError(code.AuthRegisterError)
    }
    if tenantIDStr == "" {
        return nil, code.GetError(code.TenantNotExistError)
    }

    var tenantID uint
    if _, err := fmt.Sscanf(tenantIDStr, "%d", &tenantID); err != nil {
        return nil, code.GetError(code.AuthRegisterError)
    }
    return dao.NewTenantDao().GetByID(ctx, tenantID)
}
```

---

## Task 10: 创建注册主入口

**Files:**
- Create: `apps/iam/internal/service/svcuser/register.go`

- [ ] **Step 1: 创建注册主入口Service**

```go
package svcuser

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/internal/dto/dtouser"
    "github.com/morehao/ark-go-starter/apps/iam/internal/service/svcuser/strategy"
    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/ark-go-starter/pkg/dbclient"
    "github.com/morehao/golib/glog"
    "gorm.io/gorm"
)

type registerSvc struct {
    selector *strategy.StrategySelector
}

func NewRegisterSvc() *registerSvc {
    return &registerSvc{
        selector: strategy.NewStrategySelector(),
    }
}

func (svc *registerSvc) Register(ctx *gin.Context, req *dtouser.RegisterReq) (*dtouser.RegisterResp, error) {
    registerReq := &strategy.RegisterRequest{
        Username:   req.Username,
        Password:   req.Password,
        Mobile:     req.Mobile,
        Email:      req.Email,
        RealName:   req.RealName,
        InviteCode: req.InviteCode,
        SSOType:    req.SSOType,
        OpenID:     req.OpenID,
    }

    regStrategy, err := svc.selector.SelectStrategy(ctx, registerReq)
    if err != nil {
        return nil, err
    }

    preResult, err := regStrategy.PreRegister(ctx, registerReq)
    if err != nil {
        return nil, err
    }

    var userID uint
    txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
        personID, err := svc.createOrGetPerson(ctx, tx, preResult, registerReq)
        if err != nil {
            return err
        }
        preResult.PersonID = personID

        userID, err = svc.createUser(ctx, tx, preResult, registerReq)
        if err != nil {
            return err
        }
        return nil
    })
    if txErr != nil {
        glog.Errorf(ctx, "[registerSvc.Register] Transaction fail, err:%v", txErr)
        return nil, code.GetError(code.UserCreateError)
    }

    if err := regStrategy.PostRegister(ctx, registerReq, userID); err != nil {
        glog.Errorf(ctx, "[registerSvc.Register] PostRegister fail, userID:%d, err:%v", userID, err)
    }

    return &dtouser.RegisterResp{
        UserID:       userID,
        PersonID:     preResult.PersonID,
        Status:       string(preResult.Status),
        PersonExists: preResult.PersonExists,
        Message:      preResult.Message,
    }, nil
}

func (svc *registerSvc) createOrGetPerson(ctx *gin.Context, tx *gorm.DB, result *strategy.RegisterResult, req *strategy.RegisterRequest) (uint, error) {
    email := ""
    if req.Email != "" {
        email = strings.TrimSpace(req.Email)
    }

    personEntity, _ := dao.NewPersonDao().WithTx(tx).GetByCond(ctx, &dao.PersonCond{Email: email})
    if personEntity != nil && personEntity.ID > 0 {
        result.PersonExists = true
        return personEntity.ID, nil
    }

    newPerson := &model.PersonEntity{
        Mobile:       strings.TrimSpace(req.Mobile),
        Email:        email,
        RealName:     req.RealName,
        PasswordHash: result.PasswordHash,
        CreatedBy:    0,
        UpdatedBy:    0,
    }
    if err := dao.NewPersonDao().WithTx(tx).Insert(ctx, newPerson); err != nil {
        glog.Errorf(ctx, "[createOrGetPerson] Insert fail, err:%v", err)
        return 0, code.GetError(code.UserCreateError)
    }
    return newPerson.ID, nil
}

func (svc *registerSvc) createUser(ctx *gin.Context, tx *gorm.DB, result *strategy.RegisterResult, req *strategy.RegisterRequest) (uint, error) {
    tenant, err := dao.NewTenantDao().GetByID(ctx, result.TenantID)
    if err != nil {
        glog.Errorf(ctx, "[createUser] GetByID tenant fail, err:%v", err)
        return 0, code.GetError(code.AuthRegisterError)
    }
    if tenant == nil {
        return 0, code.GetError(code.TenantNotExistError)
    }

    employeeNo, err := svc.generateEmployeeNo(ctx, tenant.TenantCode)
    if err != nil {
        return 0, err
    }

    userEntity := &model.UserEntity{
        TenantID:   result.TenantID,
        PersonID:   result.PersonID,
        EmployeeNo: employeeNo,
        Username:   req.Username,
        Status:     result.Status,
        UserType:   model.UserTypeNormal,
        CreatedBy:  0,
        UpdatedBy:  0,
    }
    if err := dao.NewUserDao().WithTx(tx).Insert(ctx, userEntity); err != nil {
        glog.Errorf(ctx, "[createUser] Insert fail, err:%v", err)
        return 0, code.GetError(code.UserCreateError)
    }
    return userEntity.ID, nil
}

func (svc *registerSvc) generateEmployeeNo(ctx *gin.Context, tenantCode string) (string, error) {
    if len(tenantCode) < 2 {
        tenantCode = fmt.Sprintf("%-2s", tenantCode)
    }
    today := time.Now().Format("20060102")
    key := fmt.Sprintf("employee_no:%s:%s", tenantCode, today)

    seq, err := dbclient.RedisCli.Incr(ctx, key).Result()
    if err != nil {
        glog.Errorf(ctx, "[generateEmployeeNo] Redis Incr fail, err:%v", err)
        return "", code.GetError(code.InternalError)
    }

    expiry := time.Now().AddDate(0, 0, 1)
    if _, err := dbclient.RedisCli.ExpireAt(ctx, key, expiry).Result(); err != nil {
        glog.Errorf(ctx, "[generateEmployeeNo] Redis ExpireAt fail, err:%v", err)
    }

    return fmt.Sprintf("%s%s%04d", tenantCode[:2], today, seq), nil
}
```

---

## Task 11: 更新DTO

**Files:**
- Modify: `apps/iam/internal/dto/dtouser/auth.go`
- Create: `apps/iam/internal/dto/dtouser/captcha.go`

- [ ] **Step 1: 更新RegisterReq**

```go
type RegisterReq struct {
    Username   string `json:"username" validate:"required" label:"用户名"`
    Password   string `json:"password" validate:"required" label:"密码"`
    Mobile     string `json:"mobile" label:"手机号"`
    Email      string `json:"email" validate:"required" label:"邮箱"`
    RealName   string `json:"realName" validate:"required" label:"真实姓名"`
    InviteCode string `json:"inviteCode" label:"邀请码"`
    SSOType    string `json:"ssoType" label:"SSO类型"`
    OpenID     string `json:"openID" label:"OpenID"`
}
```

- [ ] **Step 2: 创建验证码DTO**

```go
package dtouser

type CaptchaReq struct {
    Type       string `json:"type" validate:"required" label:"类型"`
    Identifier string `json:"identifier" validate:"required" label:"标识"`
}

type CaptchaResp struct {
    CaptchaID string `json:"captchaId"`
    ExpiresAt int64 `json:"expiresAt"`
}

type VerifyCaptchaReq struct {
    Type       string `json:"type" validate:"required" label:"类型"`
    Identifier string `json:"identifier" validate:"required" label:"标识"`
    CaptchaID  string `json:"captchaId" validate:"required" label:"验证码ID"`
    Code       string `json:"code" validate:"required" label:"验证码"`
}
```

---

## Task 12: 更新Controller和路由

**Files:**
- Modify: `apps/iam/internal/controller/ctruser/user.go`
- Modify: `apps/iam/internal/router/user.go`

- [ ] **Step 1: 更新UserController.Register**

将 Register 方法改为调用 `registerSvc.Register`。

- [ ] **Step 2: 更新路由注册**

在 `user.go` 路由中确保 Register 路由已注册。

---

## Task 13: 更新错误码

**Files:**
- Modify: `pkg/code/error.go` 或相关错误码文件

- [ ] **Step 1: 添加错误码**

```go
const (
    InviteCodeRequiredError   = 10001
    InviteCodeInvalidError   = 10002
    InviteCodeExpiredError   = 10003
    InviteCodeUsedUpError    = 10004
)
```

---

## Task 14: 测试验证

**Files:**
- Create: `apps/iam/internal/service/svcuser/strategy/*_test.go`

- [ ] **Step 1: 编写策略单元测试**

为各策略编写基础单元测试，验证策略选择和基本逻辑。

- [ ] **Step 2: 运行测试**

```bash
make test APP=iam
```

---

## 注意事项

1. 策略实现中复用了 `auth.go` 中的辅助方法（如 `getCurrentOrg`），需要确保这些方法在包级别可访问
2. Redis 操作使用 `dbclient.RedisCli`，需确认 Redis 客户端配置
3. 邀请码使用次数递增需考虑并发场景，可使用 Redis INCR 替代数据库更新
4. SSO 用户注册暂不实现完整 SSO 登录流程，只支持基础字段
5. 验证码功能（Task 15）暂未列入，可后续迭代实现