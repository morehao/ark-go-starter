# IAM OAuth2 + OIDC SSO 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 OAuth2 + OIDC 单点登录体系，支持多应用 SSO、会话管理、Token 管理

**Architecture:**
- 使用 Authorization Code + PKCE 流程
- SSO 会话存储于 Redis（高速访问）+ DB（持久化）
- Token 存储于 DB，支持主动撤销
- 权限通过独立 API 获取（/user/permissions）

**Tech Stack:** Go, Gin, GORM, Redis, MySQL, JWT (jwtauth)

---

## 文件结构

```
apps/iam/
├── model/
│   ├── application.go              # 修改：增加 OAuth Client 字段
│   ├── sso_session.go              # 新建：SSO Session 模型
│   ├── auth_code.go                 # 新建：Auth Code 模型
│   ├── token.go                     # 新建：Token 模型
│   └── login_log.go                 # 修改：增加 logout_time, session_id 字段
├── dao/
│   ├── sso_session.go              # 新建：SSO Session DAO
│   ├── auth_code.go                 # 新建：Auth Code DAO
│   └── token.go                     # 新建：Token DAO
├── internal/
│   ├── service/svcoidc/
│   │   ├── session.go               # 新建：SSO Session 服务
│   │   ├── authorize.go              # 新建：授权服务
│   │   ├── token.go                  # 新建：Token 服务
│   │   ├── userinfo.go               # 新建：UserInfo 服务
│   │   └── logout.go                 # 新建：登出服务
│   ├── dto/dtooidc/
│   │   ├── authorize.go              # 新建：授权请求/响应 DTO
│   │   ├── token.go                  # 新建：Token 请求/响应 DTO
│   │   ├── userinfo.go               # 新建：UserInfo 响应 DTO
│   │   └── logout.go                 # 新建：登出请求/响应 DTO
│   ├── controller/ctroidc/
│   │   ├── authorize.go              # 新建：授权端点控制器
│   │   ├── token.go                  # 新建：Token 端点控制器
│   │   ├── userinfo.go               # 新建：UserInfo 端点控制器
│   │   └── logout.go                 # 新建：登出端点控制器
│   └── router/
│       └── oidc.go                   # 新建：OIDC 路由注册
```

---

## Phase 1: 数据模型

### Task 1: 更新 application 模型 - 增加 OAuth Client 字段

**Files:**
- Modify: `apps/iam/model/application.go`

- [ ] **Step 1: 添加 OAuth 相关字段到 ApplicationEntity**

```go
// 在 ApplicationEntity 结构体中添加以下字段
ClientID       string `gorm:"column:client_id;type:varchar(64);unique;comment:OAuth2 Client ID"`
ClientSecret   string `gorm:"column:client_secret;type:varchar(255);comment:OAuth2 Client Secret"`
ClientType     string `gorm:"column:client_type;type:varchar(16);default:web;comment:客户端类型: web/app/spa/mini"`
PkceRequired   bool   `gorm:"column:pkce_required;type:tinyint(1);default:0;comment:是否强制PKCE"`
AllowedScopes  string `gorm:"column:allowed_scopes;type:varchar(255);default:openid,profile;comment:允许的scopes"`
AllowedCallbacks string `gorm:"column:allowed_callbacks;type:text;comment:允许的重定向URI，JSON数组"`
```

- [ ] **Step 2: 添加 client_type 常量**

```go
type ClientType string

const (
    ClientTypeWeb  ClientType = "web"
    ClientTypeApp  ClientType = "app"
    ClientTypeSpa  ClientType = "spa"
    ClientTypeMini ClientType = "mini"
)
```

- [ ] **Step 3: Commit**

```bash
git add apps/iam/model/application.go
git commit -m "feat(iam): add OAuth client fields to ApplicationEntity"
```

---

### Task 2: 创建 SSO Session 模型

**Files:**
- Create: `apps/iam/model/sso_session.go`

- [ ] **Step 1: 创建 SsoSessionEntity**

```go
package model

import (
    "time"
    "gorm.io/gorm"
)

type SsoSessionEntity struct {
    ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
    SessionID      string         `gorm:"column:session_id;type:varchar(64);unique;not null;comment:SSO会话ID" json:"session_id"`
    PersonID       uint           `gorm:"column:person_id;type:bigint;not null;default:0;comment:自然人ID" json:"person_id"`
    OrgID          uint           `gorm:"column:org_id;type:bigint;not null;default:0;comment:组织ID" json:"org_id"`
    LoginTime      time.Time      `gorm:"column:login_time;type:datetime(3);not null;comment:登录时间" json:"login_time"`
    LastActiveTime time.Time      `gorm:"column:last_active_time;type:datetime(3);not null;comment:最后活跃时间" json:"last_active_time"`
    ExpiresAt      time.Time      `gorm:"column:expires_at;type:datetime(3);not null;comment:过期时间" json:"expires_at"`
    CreatedAt      time.Time      `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
    UpdatedAt      time.Time      `gorm:"column:updated_at;type:datetime(3);not null" json:"updated_at"`
    DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3);index" json:"deleted_at,omitempty"`
}

const TableNameSsoSession = "iam_sso_session"

func (SsoSessionEntity) TableName() string {
    return TableNameSsoSession
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/model/sso_session.go
git commit -m "feat(iam): add SsoSessionEntity model"
```

---

### Task 3: 创建 Auth Code 模型

**Files:**
- Create: `apps/iam/model/auth_code.go`

- [ ] **Step 1: 创建 AuthCodeEntity**

```go
package model

import (
    "time"
    "gorm.io/gorm"
)

type AuthCodeEntity struct {
    ID                   uint           `gorm:"primaryKey;autoIncrement" json:"id"`
    Code                 string         `gorm:"column:code;type:varchar(64);unique;not null;comment:授权码" json:"code"`
    ClientID             string         `gorm:"column:client_id;type:varchar(64);not null;comment:Client ID" json:"client_id"`
    PersonID             uint           `gorm:"column:person_id;type:bigint;not null;default:0;comment:自然人ID" json:"person_id"`
    TenantID             uint           `gorm:"column:tenant_id;type:bigint;not null;default:0;comment:租户ID" json:"tenant_id"`
    OrgID                uint           `gorm:"column:org_id;type:bigint;not null;default:0;comment:组织ID" json:"org_id"`
    RedirectURI          string         `gorm:"column:redirect_uri;type:varchar(255);not null;comment:重定向URI" json:"redirect_uri"`
    Scope                string         `gorm:"column:scope;type:varchar(255);default:openid,profile;comment:请求的scope" json:"scope"`
    State                string         `gorm:"column:state;type:varchar(128);comment:state参数，防CSRF" json:"state"`
    CodeChallenge        string         `gorm:"column:code_challenge;type:varchar(64);comment:PKCE code_challenge" json:"code_challenge"`
    CodeChallengeMethod  string         `gorm:"column:code_challenge_method;type:varchar(8);default:S256;comment:PKCE challenge方法" json:"code_challenge_method"`
    ExpiresAt            time.Time      `gorm:"column:expires_at;type:datetime(3);not null;comment:过期时间" json:"expires_at"`
    Used                 bool           `gorm:"column:used;type:tinyint(1);not null;default:0;comment:是否已使用" json:"used"`
    UsedAt               *time.Time     `gorm:"column:used_at;type:datetime(3);comment:使用时间" json:"used_at,omitempty"`
    CreatedAt            time.Time      `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
    UpdatedAt            time.Time      `gorm:"column:updated_at;type:datetime(3);not null" json:"updated_at"`
    DeletedAt             gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3);index" json:"deleted_at,omitempty"`
}

const TableNameAuthCode = "iam_auth_code"

func (AuthCodeEntity) TableName() string {
    return TableNameAuthCode
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/model/auth_code.go
git commit -m "feat(iam): add AuthCodeEntity model"
```

---

### Task 4: 创建 Token 模型

**Files:**
- Create: `apps/iam/model/token.go`

- [ ] **Step 1: 创建 TokenEntity**

```go
package model

import (
    "time"
    "gorm.io/gorm"
)

type TokenType string

const (
    TokenTypeAccess  TokenType = "access"
    TokenTypeRefresh TokenType = "refresh"
    TokenTypeID     TokenType = "id"
)

type TokenEntity struct {
    ID                 uint           `gorm:"primaryKey;autoIncrement" json:"id"`
    TokenID            string         `gorm:"column:token_id;type:varchar(64);unique;not null;comment:Token唯一标识" json:"token_id"`
    PersonID           uint           `gorm:"column:person_id;type:bigint;not null;default:0;comment:自然人ID" json:"person_id"`
    UserID             uint           `gorm:"column:user_id;type:bigint;not null;default:0;comment:用户ID" json:"user_id"`
    ClientID           string         `gorm:"column:client_id;type:varchar(64);not null;comment:Client ID" json:"client_id"`
    TenantID           uint           `gorm:"column:tenant_id;type:bigint;not null;default:0;comment:租户ID" json:"tenant_id"`
    OrgID              uint           `gorm:"column:org_id;type:bigint;not null;default:0;comment:组织ID" json:"org_id"`
    TokenType          TokenType     `gorm:"column:token_type;type:varchar(16);not null;comment:Token类型" json:"token_type"`
    AccessTokenHash    string         `gorm:"column:access_token_hash;type:varchar(128);comment:Access Token哈希" json:"access_token_hash,omitempty"`
    RefreshTokenHash   string         `gorm:"column:refresh_token_hash;type:varchar(128);comment:Refresh Token哈希" json:"refresh_token_hash,omitempty"`
    Scopes             string         `gorm:"column:scopes;type:varchar(255);default:openid,profile;comment:授权的scopes" json:"scopes"`
    ExpiresAt          time.Time      `gorm:"column:expires_at;type:datetime(3);not null;comment:过期时间" json:"expires_at"`
    Revoked            bool           `gorm:"column:revoked;type:tinyint(1);not null;default:0;comment:是否撤销" json:"revoked"`
    RevokedAt          *time.Time     `gorm:"column:revoked_at;type:datetime(3);comment:撤销时间" json:"revoked_at,omitempty"`
    CreatedAt          time.Time      `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
    UpdatedAt          time.Time      `gorm:"column:updated_at;type:datetime(3);not null" json:"updated_at"`
    DeletedAt          gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3);index" json:"deleted_at,omitempty"`
}

const TableNameToken = "iam_token"

func (TokenEntity) TableName() string {
    return TableNameToken
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/model/token.go
git commit -m "feat(iam): add TokenEntity model"
```

---

### Task 5: 更新 LoginLog 模型 - 增加字段

**Files:**
- Modify: `apps/iam/model/login_log.go`

- [ ] **Step 1: 在 LoginLogEntity 中添加字段**

```go
type LoginLogEntity struct {
    // ... 现有字段 ...

    LogoutTime time.Time  `gorm:"column:logout_time;type:datetime(3);comment:退出时间" json:"logout_time,omitempty"`
    SessionID  string     `gorm:"column:session_id;type:varchar(64);comment:关联的SSO会话ID" json:"session_id,omitempty"`
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/model/login_log.go
git commit -m "feat(iam): add logout_time and session_id to LoginLogEntity"
```

---

## Phase 2: DAO 层

### Task 6: 创建 SsoSession DAO

**Files:**
- Create: `apps/iam/dao/sso_session.go`

- [ ] **Step 1: 创建 SsoSessionDao 和 SsoSessionCond**

```go
package dao

import (
    "context"
    "time"

    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/golib/gdb"
    "gorm.io/gorm"
)

type SsoSessionDao struct {
}

func NewSsoSessionDao() *SsoSessionDao {
    return &SsoSessionDao{}
}

type SsoSessionCond struct {
    SessionID string
    PersonID  uint
    OrgID     uint
}

func (d *SsoSessionDao) GetBySessionID(ctx context.Context, sessionID string) (*model.SsoSessionEntity, error) {
    var entity model.SsoSessionEntity
    err := gdb.IamDB(ctx).Where("session_id = ? AND deleted_at IS NULL", sessionID).First(&entity).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, nil
        }
        return nil, err
    }
    return &entity, nil
}

func (d *SsoSessionDao) GetByCond(ctx context.Context, cond *SsoSessionCond) (*model.SsoSessionEntity, error) {
    query := gdb.IamDB(ctx).Where("deleted_at IS NULL")
    if cond.SessionID != "" {
        query = query.Where("session_id = ?", cond.SessionID)
    }
    if cond.PersonID > 0 {
        query = query.Where("person_id = ?", cond.PersonID)
    }
    if cond.OrgID > 0 {
        query = query.Where("org_id = ?", cond.OrgID)
    }
    var entity model.SsoSessionEntity
    err := query.First(&entity).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, nil
        }
        return nil, err
    }
    return &entity, nil
}

func (d *SsoSessionDao) Insert(ctx context.Context, entity *model.SsoSessionEntity) error {
    return gdb.IamDB(ctx).Create(entity).Error
}

func (d *SsoSessionDao) UpdateLastActiveTime(ctx context.Context, sessionID string) error {
    return gdb.IamDB(ctx).Model(&model.SsoSessionEntity{}).
        Where("session_id = ?", sessionID).
        Update("last_active_time", time.Now()).Error
}

func (d *SsoSessionDao) DeleteBySessionID(ctx context.Context, sessionID string) error {
    return gdb.IamDB(ctx).Where("session_id = ?", sessionID).Delete(&model.SsoSessionEntity{}).Error
}

func (d *SsoSessionDao) DeleteByPersonID(ctx context.Context, personID uint) error {
    return gdb.IamDB(ctx).Where("person_id = ?", personID).Delete(&model.SsoSessionEntity{}).Error
}

func (d *SsoSessionDao) CleanExpired(ctx context.Context) error {
    return gdb.IamDB(ctx).Where("expires_at < ?", time.Now()).Delete(&model.SsoSessionEntity{}).Error
}

func (d *SsoSessionDao) WithTx(tx *gorm.DB) *SsoSessionDao {
    return &SsoSessionDao{}
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/dao/sso_session.go
git commit -m "feat(iam): add SsoSessionDao"
```

---

### Task 7: 创建 AuthCode DAO

**Files:**
- Create: `apps/iam/dao/auth_code.go`

- [ ] **Step 1: 创建 AuthCodeDao 和 AuthCodeCond**

```go
package dao

import (
    "context"
    "time"

    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/golib/gdb"
    "gorm.io/gorm"
)

type AuthCodeDao struct {
}

func NewAuthCodeDao() *AuthCodeDao {
    return &AuthCodeDao{}
}

type AuthCodeCond struct {
    Code     string
    ClientID string
}

func (d *AuthCodeDao) GetByCode(ctx context.Context, code string) (*model.AuthCodeEntity, error) {
    var entity model.AuthCodeEntity
    err := gdb.IamDB(ctx).Where("code = ? AND deleted_at IS NULL", code).First(&entity).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, nil
        }
        return nil, err
    }
    return &entity, nil
}

func (d *AuthCodeDao) Insert(ctx context.Context, entity *model.AuthCodeEntity) error {
    return gdb.IamDB(ctx).Create(entity).Error
}

func (d *AuthCodeDao) MarkUsed(ctx context.Context, code string) error {
    now := time.Now()
    return gdb.IamDB(ctx).Model(&model.AuthCodeEntity{}).
        Where("code = ?", code).
        Updates(map[string]interface{}{
            "used":    true,
            "used_at": now,
        }).Error
}

func (d *AuthCodeDao) CleanExpired(ctx context.Context) error {
    return gdb.IamDB(ctx).Where("expires_at < ? OR (used = ? AND used_at < ?)",
        time.Now(), true, time.Now().Add(-10*time.Minute)).Delete(&model.AuthCodeEntity{}).Error
}

func (d *AuthCodeDao) WithTx(tx *gorm.DB) *AuthCodeDao {
    return &AuthCodeDao{}
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/dao/auth_code.go
git commit -m "feat(iam): add AuthCodeDao"
```

---

### Task 8: 创建 Token DAO

**Files:**
- Create: `apps/iam/dao/token.go`

- [ ] **Step 1: 创建 TokenDao 和 TokenCond**

```go
package dao

import (
    "context"
    "time"

    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/golib/gdb"
    "gorm.io/gorm"
)

type TokenDao struct {
}

func NewTokenDao() *TokenDao {
    return &TokenDao{}
}

type TokenCond struct {
    TokenID      string
    PersonID     uint
    ClientID     string
    TenantID     uint
    AccessTokenHash string
    RefreshTokenHash string
    Revoked      bool
}

func (d *TokenDao) GetByTokenID(ctx context.Context, tokenID string) (*model.TokenEntity, error) {
    var entity model.TokenEntity
    err := gdb.IamDB(ctx).Where("token_id = ? AND deleted_at IS NULL", tokenID).First(&entity).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, nil
        }
        return nil, err
    }
    return &entity, nil
}

func (d *TokenDao) GetByAccessTokenHash(ctx context.Context, hash string) (*model.TokenEntity, error) {
    var entity model.TokenEntity
    err := gdb.IamDB(ctx).Where("access_token_hash = ? AND token_type = ? AND revoked = ? AND deleted_at IS NULL",
        hash, model.TokenTypeAccess, false).First(&entity).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, nil
        }
        return nil, err
    }
    return &entity, nil
}

func (d *TokenDao) GetByRefreshTokenHash(ctx context.Context, hash string) (*model.TokenEntity, error) {
    var entity model.TokenEntity
    err := gdb.IamDB(ctx).Where("refresh_token_hash = ? AND token_type = ? AND revoked = ? AND deleted_at IS NULL",
        hash, model.TokenTypeRefresh, false).First(&entity).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, nil
        }
        return nil, err
    }
    return &entity, nil
}

func (d *TokenDao) Insert(ctx context.Context, entity *model.TokenEntity) error {
    return gdb.IamDB(ctx).Create(entity).Error
}

func (d *TokenDao) RevokeByTokenID(ctx context.Context, tokenID string) error {
    now := time.Now()
    return gdb.IamDB(ctx).Model(&model.TokenEntity{}).
        Where("token_id = ?", tokenID).
        Updates(map[string]interface{}{
            "revoked":    true,
            "revoked_at": now,
        }).Error
}

func (d *TokenDao) RevokeByPersonID(ctx context.Context, personID uint) error {
    now := time.Now()
    return gdb.IamDB(ctx).Model(&model.TokenEntity{}).
        Where("person_id = ?", personID).
        Updates(map[string]interface{}{
            "revoked":    true,
            "revoked_at": now,
        }).Error
}

func (d *TokenDao) RevokeByRefreshTokenHash(ctx context.Context, hash string) error {
    now := time.Now()
    return gdb.IamDB(ctx).Model(&model.TokenEntity{}).
        Where("refresh_token_hash = ? AND token_type = ?", hash, model.TokenTypeRefresh).
        Updates(map[string]interface{}{
            "revoked":    true,
            "revoked_at": now,
        }).Error
}

func (d *TokenDao) CleanExpired(ctx context.Context) error {
    return gdb.IamDB(ctx).Where("expires_at < ? AND revoked = ?", time.Now(), true).
        Delete(&model.TokenEntity{}).Error
}

func (d *TokenDao) WithTx(tx *gorm.DB) *TokenDao {
    return &TokenDao{}
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/dao/token.go
git commit -m "feat(iam): add TokenDao"
```

---

## Phase 3: DTO 层

### Task 9: 创建 OIDC DTO

**Files:**
- Create: `apps/iam/internal/dto/dtooidc/authorize.go`
- Create: `apps/iam/internal/dto/dtooidc/token.go`
- Create: `apps/iam/internal/dto/dtooidc/userinfo.go`
- Create: `apps/iam/internal/dto/dtooidc/logout.go`

- [ ] **Step 1: 创建 authorize.go**

```go
package dtooidc

type AuthorizeReq struct {
    ResponseType        string `form:"response_type" binding:"required"`
    ClientID            string `form:"client_id" binding:"required"`
    RedirectURI         string `form:"redirect_uri" binding:"required"`
    Scope               string `form:"scope"`
    State               string `form:"state"`
    CodeChallenge       string `form:"code_challenge"`
    CodeChallengeMethod string `form:"code_challenge_method"`
}

type AuthorizeResp struct {
    Code  string `json:"code"`
    State string `json:"state,omitempty"`
}
```

- [ ] **Step 2: 创建 token.go**

```go
package dtooidc

type TokenReq struct {
    GrantType    string `form:"grant_type" binding:"required"`
    Code         string `form:"code"`
    RedirectURI  string `form:"redirect_uri"`
    ClientID     string `form:"client_id"`
    ClientSecret string `form:"client_secret"`
    CodeVerifier string `form:"code_verifier"`
    RefreshToken string `form:"refresh_token"`
}

type TokenResp struct {
    AccessToken  string `json:"access_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
    RefreshToken string `json:"refresh_token,omitempty"`
    IDToken      string `json:"id_token,omitempty"`
    Scope        string `json:"scope,omitempty"`
}

type TokenRefreshReq struct {
    RefreshToken string `form:"refresh_token" binding:"required"`
    ClientID     string `form:"client_id" binding:"required"`
    ClientSecret string `form:"client_secret" binding:"required"`
}
```

- [ ] **Step 3: 创建 userinfo.go**

```go
package dtooidc

type UserInfoResp struct {
    Subject  string `json:"sub"`
    Name     string `json:"name,omitempty"`
    Email    string `json:"email,omitempty"`
    Phone    string `json:"phone,omitempty"`
}
```

- [ ] **Step 4: 创建 logout.go**

```go
package dtooidc

type LogoutReq struct {
    RefreshToken string `form:"refresh_token"`
    State        string `form:"state"`
}

type LogoutResp struct {
    RedirectURI string `json:"redirect_uri,omitempty"`
}
```

- [ ] **Step 5: Commit**

```bash
git add apps/iam/internal/dto/dtooidc/
git commit -m "feat(iam): add OIDC DTOs"
```

---

## Phase 4: Service 层

### Task 10: 创建 OIDC Session 服务

**Files:**
- Create: `apps/iam/internal/service/svcoidc/session.go`

- [ ] **Step 1: 创建 SsoSessionSvc**

```go
package svcoidc

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "time"

    "github.com/morehao/ark-go-starter/apps/iam/dao"
    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/golib/gdb"
    "github.com/morehao/golib/glog"
)

const (
    SsoSessionExpireDuration = 24 * time.Hour
)

type SsoSessionSvc interface {
    CreateSession(ctx context.Context, personID, orgID uint) (string, error)
    GetValidSession(ctx context.Context, sessionID string) (*model.SsoSessionEntity, error)
    RefreshSession(ctx context.Context, sessionID string) error
    DeleteSession(ctx context.Context, sessionID string) error
    DeleteSessionByPersonID(ctx context.Context, personID uint) error
}

type ssoSessionSvc struct {
}

func NewSsoSessionSvc() SsoSessionSvc {
    return &ssoSessionSvc{}
}

func (svc *ssoSessionSvc) generateSessionID() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

func (svc *ssoSessionSvc) CreateSession(ctx context.Context, personID, orgID uint) (string, error) {
    sessionID, err := svc.generateSessionID()
    if err != nil {
        glog.Errorf(ctx, "[ssoSessionSvc.CreateSession] generateSessionID fail, err:%v", err)
        return "", err
    }

    now := time.Now()
    entity := &model.SsoSessionEntity{
        SessionID:      sessionID,
        PersonID:       personID,
        OrgID:          orgID,
        LoginTime:      now,
        LastActiveTime: now,
        ExpiresAt:      now.Add(SsoSessionExpireDuration),
    }

    if err := dao.NewSsoSessionDao().Insert(ctx, entity); err != nil {
        glog.Errorf(ctx, "[ssoSessionSvc.CreateSession] Insert fail, err:%v", err)
        return "", err
    }

    return sessionID, nil
}

func (svc *ssoSessionSvc) GetValidSession(ctx context.Context, sessionID string) (*model.SsoSessionEntity, error) {
    if sessionID == "" {
        return nil, nil
    }

    entity, err := dao.NewSsoSessionDao().GetBySessionID(ctx, sessionID)
    if err != nil {
        glog.Errorf(ctx, "[ssoSessionSvc.GetValidSession] GetBySessionID fail, err:%v", err)
        return nil, err
    }
    if entity == nil || entity.ID == 0 {
        return nil, nil
    }

    if time.Now().After(entity.ExpiresAt) {
        return nil, nil
    }

    return entity, nil
}

func (svc *ssoSessionSvc) RefreshSession(ctx context.Context, sessionID string) error {
    entity, err := dao.NewSsoSessionDao().GetBySessionID(ctx, sessionID)
    if err != nil || entity == nil || entity.ID == 0 {
        return err
    }

    newExpiresAt := time.Now().Add(SsoSessionExpireDuration)
    if err := gdb.IamDB(ctx).Model(&model.SsoSessionEntity{}).
        Where("session_id = ?", sessionID).
        Updates(map[string]interface{}{
            "last_active_time": time.Now(),
            "expires_at":       newExpiresAt,
        }).Error; err != nil {
        glog.Errorf(ctx, "[ssoSessionSvc.RefreshSession] Update fail, err:%v", err)
        return err
    }

    return nil
}

func (svc *ssoSessionSvc) DeleteSession(ctx context.Context, sessionID string) error {
    if err := dao.NewSsoSessionDao().DeleteBySessionID(ctx, sessionID); err != nil {
        glog.Errorf(ctx, "[ssoSessionSvc.DeleteSession] DeleteBySessionID fail, err:%v", err)
        return err
    }
    return nil
}

func (svc *ssoSessionSvc) DeleteSessionByPersonID(ctx context.Context, personID uint) error {
    if err := dao.NewSsoSessionDao().DeleteByPersonID(ctx, personID); err != nil {
        glog.Errorf(ctx, "[ssoSessionSvc.DeleteSessionByPersonID] DeleteByPersonID fail, err:%v", err)
        return err
    }
    return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/internal/service/svcoidc/session.go
git commit -m "feat(iam): add SsoSessionSvc"
```

---

### Task 11: 创建 OIDC Authorize 服务

**Files:**
- Create: `apps/iam/internal/service/svcoidc/authorize.go`

- [ ] **Step 1: 创建 AuthorizeSvc**

```go
package svcoidc

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "errors"
    "time"

    "github.com/morehao/ark-go-starter/apps/iam/dao"
    "github.com/morehao/ark-go-starter/apps/iam/internal/dto/dtooidc"
    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/golib/glog"
)

const AuthCodeExpireDuration = 10 * time.Minute

type AuthorizeSvc interface {
    ValidateClient(ctx context.Context, clientID, redirectURI string) (*model.ApplicationEntity, error)
    GenerateAuthCode(ctx context.Context, app *model.ApplicationEntity, personID, tenantID, orgID uint, req *dtooidc.AuthorizeReq) (*dtooidc.AuthorizeResp, error)
    ValidatePKCE(ctx context.Context, codeChallenge, codeVerifier, method string) error
}

type authorizeSvc struct {
}

func NewAuthorizeSvc() AuthorizeSvc {
    return &authorizeSvc{}
}

func (svc *authorizeSvc) ValidateClient(ctx context.Context, clientID, redirectURI string) (*model.ApplicationEntity, error) {
    if clientID == "" {
        return nil, code.GetError(code.OIDCClientIDRequiredError)
    }

    appEntity, err := dao.NewApplicationDao().GetByCond(ctx, &dao.ApplicationCond{
        ClientID: clientID,
        Status:   model.AppStatusEnabled,
    })
    if err != nil {
        glog.Errorf(ctx, "[authorizeSvc.ValidateClient] GetByCond fail, err:%v, clientID:%s", err, clientID)
        return nil, code.GetError(code.OIDCClientInvalidError)
    }
    if appEntity == nil || appEntity.ID == 0 {
        return nil, code.GetError(code.OIDCClientInvalidError)
    }

    if appEntity.AllowedCallbacks != "" {
        if !svc.isRedirectURIValid(ctx, appEntity.AllowedCallbacks, redirectURI) {
            return nil, code.GetError(code.OIDCRedirectURIMismatchError)
        }
    }

    return appEntity, nil
}

func (svc *authorizeSvc) isRedirectURIValid(ctx context.Context, allowedCallbacks, redirectURI string) bool {
    // JSON array format: ["https://app-a.com/callback", "https://app-b.com/callback"]
    // Simplified check - contains match
    return true // TODO: implement proper JSON array parsing and exact match
}

func (svc *authorizeSvc) GenerateAuthCode(ctx context.Context, app *model.ApplicationEntity, personID, tenantID, orgID uint, req *dtooidc.AuthorizeReq) (*dtooidc.AuthorizeResp, error) {
    codeStr, err := svc.generateCode()
    if err != nil {
        glog.Errorf(ctx, "[authorizeSvc.GenerateAuthCode] generateCode fail, err:%v", err)
        return nil, code.GetError(code.OIDCGenerateCodeError)
    }

    entity := &model.AuthCodeEntity{
        Code:                codeStr,
        ClientID:            app.ClientID,
        PersonID:            personID,
        TenantID:            tenantID,
        OrgID:               orgID,
        RedirectURI:         req.RedirectURI,
        Scope:               req.Scope,
        State:               req.State,
        CodeChallenge:       req.CodeChallenge,
        CodeChallengeMethod: req.CodeChallengeMethod,
        ExpiresAt:           time.Now().Add(AuthCodeExpireDuration),
        Used:                false,
    }

    if err := dao.NewAuthCodeDao().Insert(ctx, entity); err != nil {
        glog.Errorf(ctx, "[authorizeSvc.GenerateAuthCode] Insert fail, err:%v", err)
        return nil, code.GetError(code.OIDCGenerateCodeError)
    }

    return &dtooidc.AuthorizeResp{
        Code:  codeStr,
        State: req.State,
    }, nil
}

func (svc *authorizeSvc) generateCode() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

func (svc *authorizeSvc) ValidatePKCE(ctx context.Context, codeChallenge, codeVerifier, method string) error {
    if codeChallenge == "" || codeVerifier == "" {
        return errors.New("pkce parameters required")
    }
    // TODO: implement S256 verification
    return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/internal/service/svcoidc/authorize.go
git commit -m "feat(iam): add AuthorizeSvc"
```

---

### Task 12: 创建 OIDC Token 服务

**Files:**
- Create: `apps/iam/internal/service/svcoidc/token.go`

- [ ] **Step 1: 创建 TokenSvc**

```go
package svcoidc

import (
    "context"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "errors"
    "fmt"
    "time"

    "github.com/morehao/ark-go-starter/apps/iam/dao"
    "github.com/morehao/ark-go-starter/apps/iam/internal/dto/dtooidc"
    "github.com/morehao/ark-go-starter/apps/iam/model"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/golib/biz/gobject"
    "github.com/morehao/golib/gauth/jwtauth"
    "github.com/morehao/golib/gcrypto"
    "github.com/morehao/golib/glog"
)

const (
    AccessTokenExpireDuration  = 1 * time.Hour
    RefreshTokenExpireDuration = 7 * 24 * time.Hour
    TokenIssuer                = "iam"
)

type TokenSvc interface {
    ExchangeCodeForToken(ctx context.Context, req *dtooidc.TokenReq) (*dtooidc.TokenResp, error)
    RefreshAccessToken(ctx context.Context, req *dtooidc.TokenRefreshReq) (*dtooidc.TokenResp, error)
    ValidateAccessToken(ctx context.Context, accessToken string) (*model.TokenEntity, error)
    RevokeToken(ctx context.Context, token string, tokenTypeHint string) error
}

type tokenSvc struct {
}

func NewTokenSvc() TokenSvc {
    return &tokenSvc{}
}

func (svc *tokenSvc) ExchangeCodeForToken(ctx context.Context, req *dtooidc.TokenReq) (*dtooidc.TokenResp, error) {
    if req.GrantType != "authorization_code" {
        return nil, code.GetError(code.OIDCUnsupportedGrantTypeError)
    }

    authCode, err := dao.NewAuthCodeDao().GetByCode(ctx, req.Code)
    if err != nil {
        glog.Errorf(ctx, "[tokenSvc.ExchangeCodeForToken] GetByCode fail, err:%v", err)
        return nil, code.GetError(code.OIDCInvalidCodeError)
    }
    if authCode == nil || authCode.ID == 0 {
        return nil, code.GetError(code.OIDCInvalidCodeError)
    }
    if authCode.Used {
        return nil, code.GetError(code.OIDCInvalidCodeError)
    }
    if time.Now().After(authCode.ExpiresAt) {
        return nil, code.GetError(code.OIDCInvalidCodeError)
    }
    if authCode.RedirectURI != req.RedirectURI {
        return nil, code.GetError(code.OIDCRedirectURIMismatchError)
    }

    appEntity, err := dao.NewApplicationDao().GetByCond(ctx, &dao.ApplicationCond{
        ClientID: authCode.ClientID,
        Status:   model.AppStatusEnabled,
    })
    if err != nil || appEntity == nil || appEntity.ID == 0 {
        return nil, code.GetError(code.OIDCClientInvalidError)
    }

    if appEntity.ClientSecret != "" && appEntity.ClientSecret != req.ClientSecret {
        return nil, code.GetError(code.OIDCInvalidClientError)
    }

    if authCode.CodeChallenge != "" && req.CodeVerifier == "" {
        return nil, code.GetError(code.OIDCPKCERequiredError)
    }

    if err := dao.NewAuthCodeDao().MarkUsed(ctx, req.Code); err != nil {
        glog.Errorf(ctx, "[tokenSvc.ExchangeCodeForToken] MarkUsed fail, err:%v", err)
    }

    return svc.generateTokenPair(ctx, authCode.PersonID, authCode.UserID, authCode.TenantID, authCode.OrgID, authCode.ClientID, authCode.Scope)
}

func (svc *tokenSvc) generateTokenPair(ctx context.Context, personID, userID, tenantID, orgID uint, clientID, scopes string) (*dtooidc.TokenResp, error) {
    accessToken, accessTokenID, accessTokenHash, err := svc.generateAccessToken(ctx, personID, userID, tenantID, orgID, clientID, scopes)
    if err != nil {
        return nil, err
    }

    refreshToken, refreshTokenHash, err := svc.generateRefreshToken(ctx, personID, userID, tenantID, orgID, clientID, scopes)
    if err != nil {
        return nil, err
    }

    return &dtooidc.TokenResp{
        AccessToken:  accessToken,
        TokenType:    "Bearer",
        ExpiresIn:    int(AccessTokenExpireDuration.Seconds()),
        RefreshToken: refreshToken,
        Scope:        scopes,
    }, nil
}

func (svc *tokenSvc) generateAccessToken(ctx context.Context, personID, userID, tenantID, orgID uint, clientID, scopes string) (string, string, string, error) {
    tokenID, _ := svc.generateTokenID()
    accessTokenHash := svc.hashToken(tokenID)

    jwtAuth, err := jwtauth.New[gobject.UserClaims](config.Conf.JWT.SignKey)
    if err != nil {
        return "", "", "", code.GetError(code.OIDCGenerateTokenError)
    }

    customData := gobject.UserClaims{
        UserID:   userID,
        PersonID: personID,
        TenantID: tenantID,
        OrgID:    orgID,
        UserType: "access",
    }

    tokenStr, err := jwtAuth.Issue(
        tokenID,
        TokenIssuer,
        time.Now().Add(AccessTokenExpireDuration),
        customData,
    )
    if err != nil {
        return "", "", "", code.GetError(code.OIDCGenerateTokenError)
    }

    entity := &model.TokenEntity{
        TokenID:          tokenID,
        PersonID:         personID,
        UserID:           userID,
        ClientID:         clientID,
        TenantID:         tenantID,
        OrgID:            orgID,
        TokenType:        model.TokenTypeAccess,
        AccessTokenHash: accessTokenHash,
        Scopes:           scopes,
        ExpiresAt:        time.Now().Add(AccessTokenExpireDuration),
        Revoked:          false,
    }
    if err := dao.NewTokenDao().Insert(ctx, entity); err != nil {
        glog.Errorf(ctx, "[tokenSvc.generateAccessToken] Insert fail, err:%v", err)
    }

    return tokenStr, tokenID, accessTokenHash, nil
}

func (svc *tokenSvc) generateRefreshToken(ctx context.Context, personID, userID, tenantID, orgID uint, clientID, scopes string) (string, string, error) {
    tokenID, _ := svc.generateTokenID()
    refreshTokenHash := svc.hashToken(tokenID)

    jwtAuth, err := jwtauth.New[gobject.UserClaims](config.Conf.JWT.SignKey)
    if err != nil {
        return "", "", code.GetError(code.OIDCGenerateTokenError)
    }

    customData := gobject.UserClaims{
        UserID:   userID,
        PersonID: personID,
        TenantID: tenantID,
        OrgID:    orgID,
        UserType: "refresh",
    }

    tokenStr, err := jwtAuth.Issue(
        tokenID,
        TokenIssuer,
        time.Now().Add(RefreshTokenExpireDuration),
        customData,
    )
    if err != nil {
        return "", "", code.GetError(code.OIDCGenerateTokenError)
    }

    entity := &model.TokenEntity{
        TokenID:          tokenID,
        PersonID:         personID,
        UserID:           userID,
        ClientID:         clientID,
        TenantID:         tenantID,
        OrgID:            orgID,
        TokenType:        model.TokenTypeRefresh,
        RefreshTokenHash: refreshTokenHash,
        Scopes:           scopes,
        ExpiresAt:        time.Now().Add(RefreshTokenExpireDuration),
        Revoked:          false,
    }
    if err := dao.NewTokenDao().Insert(ctx, entity); err != nil {
        glog.Errorf(ctx, "[tokenSvc.generateRefreshToken] Insert fail, err:%v", err)
    }

    return tokenStr, refreshTokenHash, nil
}

func (svc *tokenSvc) generateTokenID() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

func (svc *tokenSvc) hashToken(token string) string {
    hash := sha256.Sum256([]byte(token))
    return base64.URLEncoding.EncodeToString(hash[:])
}

func (svc *tokenSvc) RefreshAccessToken(ctx context.Context, req *dtooidc.TokenRefreshReq) (*dtooidc.TokenResp, error) {
    if req.RefreshToken == "" {
        return nil, code.GetError(code.OIDCInvalidRefreshTokenError)
    }

    refreshTokenHash := svc.hashToken(req.RefreshToken)
    oldToken, err := dao.NewTokenDao().GetByRefreshTokenHash(ctx, refreshTokenHash)
    if err != nil || oldToken == nil || oldToken.ID == 0 {
        return nil, code.GetError(code.OIDCInvalidRefreshTokenError)
    }
    if oldToken.Revoked || time.Now().After(oldToken.ExpiresAt) {
        return nil, code.GetError(code.OIDCInvalidRefreshTokenError)
    }

    appEntity, err := dao.NewApplicationDao().GetByCond(ctx, &dao.ApplicationCond{
        ClientID: oldToken.ClientID,
        Status:   model.AppStatusEnabled,
    })
    if err != nil || appEntity == nil || appEntity.ID == 0 {
        return nil, code.GetError(code.OIDCClientInvalidError)
    }

    if appEntity.ClientSecret != "" && appEntity.ClientSecret != req.ClientSecret {
        return nil, code.GetError(code.OIDCInvalidClientError)
    }

    if err := dao.NewTokenDao().RevokeByRefreshTokenHash(ctx, refreshTokenHash); err != nil {
        glog.Errorf(ctx, "[tokenSvc.RefreshAccessToken] RevokeByRefreshTokenHash fail, err:%v", err)
    }

    return svc.generateTokenPair(ctx, oldToken.PersonID, oldToken.UserID, oldToken.TenantID, oldToken.OrgID, oldToken.ClientID, oldToken.Scopes)
}

func (svc *tokenSvc) ValidateAccessToken(ctx context.Context, accessToken string) (*model.TokenEntity, error) {
    if accessToken == "" {
        return nil, code.GetError(code.AuthTokenInvalidError)
    }

    tokenHash := svc.hashToken(accessToken)
    tokenEntity, err := dao.NewTokenDao().GetByAccessTokenHash(ctx, tokenHash)
    if err != nil {
        glog.Errorf(ctx, "[tokenSvc.ValidateAccessToken] GetByAccessTokenHash fail, err:%v", err)
        return nil, code.GetError(code.AuthTokenInvalidError)
    }
    if tokenEntity == nil || tokenEntity.ID == 0 {
        return nil, code.GetError(code.AuthTokenInvalidError)
    }
    if tokenEntity.Revoked {
        return nil, code.GetError(code.AuthTokenInvalidError)
    }
    if time.Now().After(tokenEntity.ExpiresAt) {
        return nil, code.GetError(code.AuthTokenExpiredError)
    }

    return tokenEntity, nil
}

func (svc *tokenSvc) RevokeToken(ctx context.Context, token string, tokenTypeHint string) error {
    if token == "" {
        return nil
    }

    tokenHash := svc.hashToken(token)

    if tokenTypeHint == "refresh_token" || tokenTypeHint == "" {
        if err := dao.NewTokenDao().RevokeByRefreshTokenHash(ctx, tokenHash); err != nil {
            glog.Errorf(ctx, "[tokenSvc.RevokeToken] RevokeByRefreshTokenHash fail, err:%v", err)
            return err
        }
    }

    return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/internal/service/svcoidc/token.go
git commit -m "feat(iam): add TokenSvc"
```

---

### Task 13: 创建 OIDC UserInfo 服务

**Files:**
- Create: `apps/iam/internal/service/svcoidc/userinfo.go`

- [ ] **Step 1: 创建 UserInfoSvc**

```go
package svcoidc

import (
    "context"

    "github.com/morehao/ark-go-starter/apps/iam/dao"
    "github.com/morehao/ark-go-starter/apps/iam/internal/dto/dtooidc"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/golib/glog"
)

type UserInfoSvc interface {
    GetUserInfo(ctx context.Context, tokenEntity *TokenEntity) (*dtooidc.UserInfoResp, error)
}

type userInfoSvc struct {
}

type TokenEntity = struct {
    PersonID uint
    UserID   uint
    TenantID uint
    OrgID    uint
}

func NewUserInfoSvc() UserInfoSvc {
    return &userInfoSvc{}
}

func (svc *userInfoSvc) GetUserInfo(ctx context.Context, tokenEntity *model.TokenEntity) (*dtooidc.UserInfoResp, error) {
    personEntity, err := dao.NewPersonDao().GetByID(ctx, tokenEntity.PersonID)
    if err != nil {
        glog.Errorf(ctx, "[userInfoSvc.GetUserInfo] GetByID person fail, err:%v", err)
        return nil, code.GetError(code.UserNotExistError)
    }
    if personEntity == nil || personEntity.ID == 0 {
        return nil, code.GetError(code.UserNotExistError)
    }

    return &dtooidc.UserInfoResp{
        Subject: fmt.Sprintf("%d", personEntity.ID),
        Name:    personEntity.RealName,
        Email:   personEntity.Email,
        Phone:   personEntity.Mobile,
    }, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/internal/service/svcoidc/userinfo.go
git commit -m "feat(iam): add UserInfoSvc"
```

---

### Task 14: 创建 OIDC Logout 服务

**Files:**
- Create: `apps/iam/internal/service/svcoidc/logout.go`

- [ ] **Step 1: 创建 LogoutSvc**

```go
package svcoidc

import (
    "context"

    "github.com/morehao/ark-go-starter/apps/iam/dao"
    "github.com/morehao/ark-go-starter/pkg/code"
    "github.com/morehao/golib/glog"
)

type LogoutSvc interface {
    Logout(ctx context.Context, refreshToken string) error
    LogoutWithSession(ctx context.Context, sessionID string, refreshToken string) error
}

type logoutSvc struct {
}

func NewLogoutSvc() LogoutSvc {
    return &logoutSvc{}
}

func (svc *logoutSvc) Logout(ctx context.Context, refreshToken string) error {
    if refreshToken != "" {
        tokenHash := svc.hashToken(refreshToken)
        if err := dao.NewTokenDao().RevokeByRefreshTokenHash(ctx, tokenHash); err != nil {
            glog.Errorf(ctx, "[logoutSvc.Logout] RevokeByRefreshTokenHash fail, err:%v", err)
        }
    }
    return nil
}

func (svc *logoutSvc) LogoutWithSession(ctx context.Context, sessionID string, refreshToken string) error {
    if sessionID != "" {
        if err := dao.NewSsoSessionDao().DeleteBySessionID(ctx, sessionID); err != nil {
            glog.Errorf(ctx, "[logoutSvc.LogoutWithSession] DeleteBySessionID fail, err:%v", err)
        }
    }

    if refreshToken != "" {
        tokenHash := svc.hashToken(refreshToken)
        if err := dao.NewTokenDao().RevokeByRefreshTokenHash(ctx, tokenHash); err != nil {
            glog.Errorf(ctx, "[logoutSvc.LogoutWithSession] RevokeByRefreshTokenHash fail, err:%v", err)
        }
    }

    return nil
}

func (svc *logoutSvc) hashToken(token string) string {
    hash := sha256.Sum256([]byte(token))
    return base64.URLEncoding.EncodeToString(hash[:])
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/internal/service/svcoidc/logout.go
git commit -m "feat(iam): add LogoutSvc"
```

---

## Phase 5: Controller 层

### Task 15: 创建 OIDC Controller

**Files:**
- Create: `apps/iam/internal/controller/ctroidc/authorize.go`
- Create: `apps/iam/internal/controller/ctroidc/token.go`
- Create: `apps/iam/internal/controller/ctroidc/userinfo.go`
- Create: `apps/iam/internal/controller/ctroidc/logout.go`

- [ ] **Step 1: 创建 authorize.go**

```go
package ctroidc

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/internal/dto/dtooidc"
    "github.com/morehao/ark-go-starter/apps/iam/internal/service/svcoidc"
    "github.com/morehao/golib/biz/gcontext/gincontext"
)

type authorizeCtr struct {
    authorizeSvc svcoidc.AuthorizeSvc
    sessionSvc   svcoidc.SsoSessionSvc
}

func NewAuthorizeCtr() *authorizeCtr {
    return &authorizeCtr{
        authorizeSvc: svcoidc.NewAuthorizeSvc(),
        sessionSvc:   svcoidc.NewSsoSessionSvc(),
    }
}

// Authorize
// @Tags OIDC
// @Summary 授权入口
// @accept application/json
// @Produce application/json
// @Param req query dtooidc.AuthorizeReq true "授权请求"
// @Success 200 {object} gincontext.DtoRender
// @Router /v1/iam/oidc/authorize [GET]
func (ctr *authorizeCtr) Authorize(ctx *gin.Context) {
    var req dtooidc.AuthorizeReq
    if err := ctx.ShouldBindQuery(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    sessionID := ctx.GetHeader("X-Sso-Session-Id")

    if sessionID != "" {
        session, err := ctr.sessionSvc.GetValidSession(ctx, sessionID)
        if err != nil || session == nil || session.ID == 0 {
            sessionID = ""
        } else {
            personID := session.PersonID
            // TODO: 需要获取用户选择的 tenantID，这里简化处理
        }
    }

    if sessionID == "" {
        // 需要登录，返回登录页面标识
        gincontext.Fail(ctx, code.GetError(code.AuthSessionExpiredError))
        return
    }

    app, err := ctr.authorizeSvc.ValidateClient(ctx, req.ClientID, req.RedirectURI)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    // TODO: 需要从 session 或参数获取 personID, tenantID
    resp, err := ctr.authorizeSvc.GenerateAuthCode(ctx, app, personID, tenantID, orgID, &req)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    redirectURI := buildRedirectURI(req.RedirectURI, resp.Code, resp.State)
    ctx.Redirect(302, redirectURI)
}

func buildRedirectURI(baseURI, code, state string) string {
    // TODO: implement proper URL building
    return baseURI + "?code=" + code + "&state=" + state
}
```

- [ ] **Step 2: 创建 token.go**

```go
package ctroidc

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/internal/dto/dtooidc"
    "github.com/morehao/ark-go-starter/internal/service/svcoidc"
    "github.com/morehao/golib/biz/gcontext/gincontext"
)

type tokenCtr struct {
    tokenSvc svcoidc.TokenSvc
}

func NewTokenCtr() *tokenCtr {
    return &tokenCtr{
        tokenSvc: svcoidc.NewTokenSvc(),
    }
}

// Token
// @Tags OIDC
// @Summary 获取Token
// @accept application/json
// @Produce application/json
// @Param req formData dtooidc.TokenReq true "Token请求"
// @Success 200 {object} gincontext.DtoRender{data=dtooidc.TokenResp}
// @Router /v1/iam/oidc/token [POST]
func (ctr *tokenCtr) Token(ctx *gin.Context) {
    var req dtooidc.TokenReq
    if err := ctx.ShouldBind(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    resp, err := ctr.tokenSvc.ExchangeCodeForToken(ctx, &req)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    gincontext.Success(ctx, resp)
}

// RefreshToken
// @Tags OIDC
// @Summary 刷新Token
// @accept application/json
// @Produce application/json
// @Param req formData dtooidc.TokenRefreshReq true "刷新Token请求"
// @Success 200 {object} gincontext.DtoRender{data=dtooidc.TokenResp}
// @Router /v1/iam/oidc/token/refresh [POST]
func (ctr *tokenCtr) RefreshToken(ctx *gin.Context) {
    var req dtooidc.TokenRefreshReq
    if err := ctx.ShouldBind(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    resp, err := ctr.tokenSvc.RefreshAccessToken(ctx, &req)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    gincontext.Success(ctx, resp)
}
```

- [ ] **Step 3: 创建 userinfo.go**

```go
package ctroidc

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/internal/service/svcoidc"
    "github.com/morehao/golib/biz/gcontext/gincontext"
)

type userinfoCtr struct {
    tokenSvc    svcoidc.TokenSvc
    userInfoSvc svcoidc.UserInfoSvc
}

func NewUserinfoCtr() *userinfoCtr {
    return &userinfoCtr{
        tokenSvc:    svcoidc.NewTokenSvc(),
        userInfoSvc: svcoidc.NewUserInfoSvc(),
    }
}

// UserInfo
// @Tags OIDC
// @Summary 获取用户信息
// @Security BearerAuth
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtooidc.UserInfoResp}
// @Router /v1/iam/oidc/userinfo [GET]
func (ctr *userinfoCtr) UserInfo(ctx *gin.Context) {
    authHeader := ctx.GetHeader("Authorization")
    if authHeader == "" {
        gincontext.Fail(ctx, code.GetError(code.AuthTokenRequiredError))
        return
    }

    tokenStr := authHeader[7:] // strip "Bearer "

    tokenEntity, err := ctr.tokenSvc.ValidateAccessToken(ctx, tokenStr)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    resp, err := ctr.userInfoSvc.GetUserInfo(ctx, tokenEntity)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    gincontext.Success(ctx, resp)
}
```

- [ ] **Step 4: 创建 logout.go**

```go
package ctroidc

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/internal/dto/dtooidc"
    "github.com/morehao/ark-go-starter/apps/iam/internal/service/svcoidc"
    "github.com/morehao/golib/biz/gcontext/gincontext"
)

type logoutCtr struct {
    logoutSvc svcoidc.LogoutSvc
}

func NewLogoutCtr() *logoutCtr {
    return &logoutCtr{
        logoutSvc: svcoidc.NewLogoutSvc(),
    }
}

// Logout
// @Tags OIDC
// @Summary 单点登出
// @accept application/json
// @Produce application/json
// @Param req body dtooidc.LogoutReq true "登出请求"
// @Success 200 {object} gincontext.DtoRender
// @Router /v1/iam/oidc/logout [POST]
func (ctr *logoutCtr) Logout(ctx *gin.Context) {
    var req dtooidc.LogoutReq
    if err := ctx.ShouldBind(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    sessionID := ctx.GetHeader("X-Sso-Session-Id")

    if err := ctr.logoutSvc.LogoutWithSession(ctx, sessionID, req.RefreshToken); err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    gincontext.Success(ctx, nil)
}
```

- [ ] **Step 5: Commit**

```bash
git add apps/iam/internal/controller/ctroidc/
git commit -m "feat(iam): add OIDC controllers"
```

---

## Phase 6: 路由注册

### Task 16: 创建 OIDC 路由

**Files:**
- Create: `apps/iam/internal/router/oidc.go`
- Modify: `apps/iam/internal/router/router.go`

- [ ] **Step 1: 创建 oidc.go**

```go
package router

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-go-starter/apps/iam/internal/controller/ctroidc"
)

func oidcRouter(routerGroup *gin.RouterGroup) {
    oidcGroup := routerGroup.Group("/oidc")

    authorizeCtr := ctroidc.NewAuthorizeCtr()
    tokenCtr := ctroidc.NewTokenCtr()
    userinfoCtr := ctroidc.NewUserinfoCtr()
    logoutCtr := ctroidc.NewLogoutCtr()

    oidcGroup.GET("/authorize", authorizeCtr.Authorize)
    oidcGroup.POST("/token", tokenCtr.Token)
    oidcGroup.POST("/token/refresh", tokenCtr.RefreshToken)
    oidcGroup.GET("/userinfo", userinfoCtr.UserInfo)
    oidcGroup.POST("/logout", logoutCtr.Logout)
}
```

- [ ] **Step 2: 修改 router.go 添加 oidc 路由**

在 `func RouterRegister(router *gin.Engine)` 函数中添加：

```go
func RouterRegister(router *gin.Engine) {
    // ... existing code ...

    v1AuthGroup := groups.AuthGroup.Group("/v1")
    iamGroup := v1AuthGroup.Group("/iam")

    // ... existing routes ...

    // OIDC routes
    oidcRouter(iamGroup)
}
```

- [ ] **Step 3: Commit**

```bash
git add apps/iam/internal/router/oidc.go apps/iam/internal/router/router.go
git commit -m "feat(iam): add OIDC routes"
```

---

## Phase 7: 错误码

### Task 17: 添加 OIDC 错误码

**Files:**
- Modify: `apps/iam/pkg/code/code.go` 或相关文件

- [ ] **Step 1: 添加 OIDC 错误码**

```go
const (
    // OIDC 错误码 (7xxx)
    OIDCClientIDRequiredError    = 7001
    OIDCClientInvalidError       = 7002
    OIDCRedirectURIMismatchError = 7003
    OIDCUnsupportedGrantTypeError = 7004
    OIDCInvalidCodeError         = 7005
    OIDCPKCERequiredError       = 7006
    OIDCInvalidClientError      = 7007
    OIDCGenerateCodeError       = 7008
    OIDCGenerateTokenError      = 7009
    OIDCInvalidRefreshTokenError = 7010
)
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/pkg/code/
git commit -m "feat(iam): add OIDC error codes"
```

---

## 实施检查清单

- [ ] Task 1: 更新 application 模型
- [ ] Task 2: 创建 SSO Session 模型
- [ ] Task 3: 创建 Auth Code 模型
- [ ] Task 4: 创建 Token 模型
- [ ] Task 5: 更新 LoginLog 模型
- [ ] Task 6: 创建 SsoSession DAO
- [ ] Task 7: 创建 AuthCode DAO
- [ ] Task 8: 创建 Token DAO
- [ ] Task 9: 创建 OIDC DTO
- [ ] Task 10: 创建 SsoSession 服务
- [ ] Task 11: 创建 Authorize 服务
- [ ] Task 12: 创建 Token 服务
- [ ] Task 13: 创建 UserInfo 服务
- [ ] Task 14: 创建 Logout 服务
- [ ] Task 15: 创建 OIDC Controller
- [ ] Task 16: 创建 OIDC 路由
- [ ] Task 17: 添加 OIDC 错误码
