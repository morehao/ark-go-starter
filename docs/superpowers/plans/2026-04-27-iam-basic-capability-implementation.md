# IAM 基础能力补充实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 IAM 服务补充用户个人中心、密码安全策略、管理员创建用户三大基础能力

**Architecture:** 采用现有三层架构（Controller -> Service -> Dao），新增 DTO 层处理请求响应，保持 UserEntity 和 PersonEntity 分离设计

**Tech Stack:** Gin + GORM + Redis + JWT

---

## 文件结构

```
apps/iam/
├── model/user.go                 # 修改：新增 login_fail_count, locked_until 字段
├── model/organization_config.go   # 修改：新增密码策略配置项常量
├── internal/dto/dtouser/
│   ├── request.go                 # 修改：新增个人中心请求 DTO
│   └── response.go                # 修改：新增个人中心响应 DTO
├── internal/dto/dtoauth/
│   └── request.go                 # 修改：新增 unlockAccount 请求 DTO
├── internal/controller/ctruser/user.go   # 修改：新增个人中心接口
├── internal/service/svcuser/user.go      # 修改：新增个人中心服务
├── internal/service/svcauth/auth.go      # 修改：调整登录逻辑、添加 unlockAccount
├── internal/router/user.go        # 修改：注册个人中心路由
└── internal/router/auth.go       # 修改：注册 unlockAccount 路由
```

---

## Phase 1: 用户个人中心

### Task 1: 添加个人中心请求/响应 DTO

**Files:**
- Modify: `apps/iam/internal/dto/dtouser/request.go:1-51`
- Modify: `apps/iam/internal/dto/dtouser/response.go:1-49`

- [ ] **Step 1: 在 request.go 添加新的请求结构体**

在文件末尾添加：

```go
type UserInfoReq struct {
}

type UpdateProfileReq struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required"`
}

type LoginHistoryReq struct {
	gobject.PageQuery
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

type LogoutReq struct {
}
```

- [ ] **Step 2: 在 response.go 添加新的响应结构体**

在文件末尾添加：

```go
type UserInfoResp struct {
	UserID     uint     `json:"userID"`
	Username   string   `json:"username"`
	PersonID   uint     `json:"personID"`
	Email      string   `json:"email"`
	Phone      string   `json:"phone"`
	Avatar     string   `json:"avatar"`
	Nickname   string   `json:"nickname"`
	Status     string   `json:"status"`
	UserType   string   `json:"userType"`
	TenantID   uint     `json:"tenantID"`
	TenantName string   `json:"tenantName"`
	OrgID      uint     `json:"orgID"`
	OrgName    string   `json:"orgName"`
	RoleIDs    []uint   `json:"roleIDs"`
	RoleNames  []string `json:"roleNames"`
	DeptIDs    []uint   `json:"deptIDs"`
	DeptNames  []string `json:"deptNames"`
}

type LoginHistoryItem struct {
	ID          uint   `json:"id"`
	LoginType   string `json:"loginType"`
	LoginStatus string `json:"loginStatus"`
	LoginMessage string `json:"loginMessage"`
	IPAddress   string `json:"ipAddress"`
	Location    string `json:"location"`
	Browser     string `json:"browser"`
	OS          string `json:"os"`
	CreatedAt   string `json:"createdAt"`
}

type LoginHistoryResp struct {
	List  []LoginHistoryItem `json:"list"`
	Total int64             `json:"total"`
}
```

- [ ] **Step 3: Commit**

```bash
git add apps/iam/internal/dto/dtouser/request.go apps/iam/internal/dto/dtouser/response.go
git commit -m "feat(iam): add personal center DTOs"
```

---

### Task 2: 实现 GetCurrentUserInfo 接口

**Files:**
- Modify: `apps/iam/internal/controller/ctruser/user.go:1-232`
- Modify: `apps/iam/internal/service/svcuser/user.go:1-509`
- Modify: `apps/iam/internal/router/user.go:1-22`

- [ ] **Step 1: 在 controller 接口定义中添加方法**

在 `UserCtr` interface 中添加：
```go
GetCurrentUserInfo(ctx *gin.Context)
```

- [ ] **Step 2: 在 service 接口定义中添加方法**

在 `UserSvc` interface 中添加：
```go
GetCurrentUserInfo(ctx *gin.Context) (*dtouser.UserInfoResp, error)
```

- [ ] **Step 3: 在 service 实现中添加方法**

```go
func (svc *userSvc) GetCurrentUserInfo(ctx *gin.Context) (*dtouser.UserInfoResp, error) {
	userID := gincontext.GetUserID(ctx)
	tenantID := gincontext.GetTenantID(ctx)

	userEntity, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil || userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	personEntity, err := dao.NewPersonDao().GetByID(ctx, userEntity.PersonID)
	if err != nil || personEntity == nil {
		return nil, code.GetError(code.UserNotExistError)
	}

	tenantName, orgID, orgName := "", uint(0), ""
	if userEntity.TenantID > 0 {
		tenantEntity, _ := dao.NewTenantDao().GetByID(ctx, userEntity.TenantID)
		if tenantEntity != nil {
			tenantName = tenantEntity.TenantName
			orgID = tenantEntity.OrgID
			orgEntity, _ := dao.NewOrganizationDao().GetByID(ctx, orgID)
			if orgEntity != nil {
				orgName = orgEntity.OrgName
			}
		}
	}

	roleIDs, roleNames := svc.getUserRoles(ctx, userID, tenantID)
	deptIDs, deptNames := svc.getUserDepts(ctx, userID, tenantID)

	return &dtouser.UserInfoResp{
		UserID:     userEntity.ID,
		Username:   userEntity.Username,
		PersonID:   userEntity.PersonID,
		Email:      personEntity.Email,
		Phone:      personEntity.Mobile,
		Avatar:     personEntity.AvatarUrl,
		Nickname:   personEntity.RealName,
		Status:     string(userEntity.Status),
		UserType:   string(userEntity.UserType),
		TenantID:   userEntity.TenantID,
		TenantName: tenantName,
		OrgID:      orgID,
		OrgName:    orgName,
		RoleIDs:    roleIDs,
		RoleNames:  roleNames,
		DeptIDs:    deptIDs,
		DeptNames:  deptNames,
	}, nil
}

func (svc *userSvc) getUserRoles(ctx *gin.Context, userID, tenantID uint) ([]uint, []string) {
	roleIDs := make([]uint, 0)
	roleNames := make([]string, 0)
	userRoleList, _ := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{
		UserID:   userID,
		TenantID: tenantID,
	})
	for _, ur := range userRoleList {
		roleEntity, _ := dao.NewRoleDao().GetByID(ctx, ur.RoleID)
		if roleEntity != nil && roleEntity.ID > 0 {
			roleIDs = append(roleIDs, roleEntity.ID)
			roleNames = append(roleNames, roleEntity.RoleName)
		}
	}
	return roleIDs, roleNames
}

func (svc *userSvc) getUserDepts(ctx *gin.Context, userID, tenantID uint) ([]uint, []string) {
	deptIDs := make([]uint, 0)
	deptNames := make([]string, 0)
	userDeptList, _ := dao.NewUserDepartmentDao().GetListByCond(ctx, &dao.UserDepartmentCond{
		UserID:   userID,
		TenantID: tenantID,
	})
	for _, ud := range userDeptList {
		deptEntity, _ := dao.NewDepartmentDao().GetByID(ctx, ud.DeptID)
		if deptEntity != nil && deptEntity.ID > 0 {
			deptIDs = append(deptIDs, deptEntity.ID)
			deptNames = append(deptNames, deptEntity.DeptName)
		}
	}
	return deptIDs, deptNames
}
```

- [ ] **Step 4: 在 controller 实现中添加处理函数**

```go
func (ctr *userCtr) GetCurrentUserInfo(ctx *gin.Context) {
	res, err := ctr.userSvc.GetCurrentUserInfo(ctx)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
```

- [ ] **Step 5: 在 router 中注册路由**

```go
v1RouterGroup.GET("/user/getCurrentUserInfo", userCtr.GetCurrentUserInfo)
```

- [ ] **Step 6: Commit**

```bash
git add apps/iam/internal/controller/ctruser/user.go apps/iam/internal/service/svcuser/user.go apps/iam/internal/router/user.go
git commit -m "feat(iam): add GetCurrentUserInfo API"
```

---

### Task 3: 实现 UpdateProfile 接口

**Files:**
- Modify: `apps/iam/internal/controller/ctruser/user.go`
- Modify: `apps/iam/internal/service/svcuser/user.go`

- [ ] **Step 1: 在 controller 接口定义中添加方法**

```go
UpdateProfile(ctx *gin.Context)
```

- [ ] **Step 2: 在 service 接口定义中添加方法**

```go
UpdateProfile(ctx *gin.Context, req *dtouser.UpdateProfileReq) error
```

- [ ] **Step 3: 在 service 实现中添加方法**

```go
func (svc *userSvc) UpdateProfile(ctx *gin.Context, req *dtouser.UpdateProfileReq) error {
	userID := gincontext.GetUserID(ctx)

	userEntity, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil || userEntity == nil || userEntity.ID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	personEntity, err := dao.NewPersonDao().GetByID(ctx, userEntity.PersonID)
	if err != nil || personEntity == nil {
		return code.GetError(code.UserNotExistError)
	}

	updateMap := map[string]any{}
	if req.Email != "" {
		updateMap["email"] = req.Email
	}
	if req.Phone != "" {
		updateMap["mobile"] = req.Phone
	}
	if req.Avatar != "" {
		updateMap["avatar_url"] = req.Avatar
	}
	if req.Nickname != "" {
		updateMap["real_name"] = req.Nickname
	}

	if len(updateMap) > 0 {
		if err := dao.NewPersonDao().UpdateMap(ctx, personEntity.ID, updateMap); err != nil {
			glog.Errorf(ctx, "[svcuser.UpdateProfile] UpdateMap fail, err:%v, personID:%d", err, personEntity.ID)
			return code.GetError(code.UserUpdateError)
		}
	}

	return nil
}
```

- [ ] **Step 4: 在 controller 实现中添加处理函数**

```go
func (ctr *userCtr) UpdateProfile(ctx *gin.Context) {
	var req dtouser.UpdateProfileReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.UpdateProfile(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "更新成功")
}
```

- [ ] **Step 5: 在 router 中注册路由**

```go
v1RouterGroup.POST("/user/updateProfile", userCtr.UpdateProfile)
```

- [ ] **Step 6: Commit**

```bash
git add apps/iam/internal/controller/ctruser/user.go apps/iam/internal/service/svcuser/user.go apps/iam/internal/router/user.go
git commit -m "feat(iam): add UpdateProfile API"
```

---

### Task 4: 实现 ChangePassword 接口

**Files:**
- Modify: `apps/iam/internal/controller/ctruser/user.go`
- Modify: `apps/iam/internal/service/svcuser/user.go`

- [ ] **Step 1: 在 controller 接口定义中添加方法**

```go
ChangePassword(ctx *gin.Context)
```

- [ ] **Step 2: 在 service 接口定义中添加方法**

```go
ChangePassword(ctx *gin.Context, req *dtouser.ChangePasswordReq) error
```

- [ ] **Step 3: 在 service 实现中添加方法**

```go
func (svc *userSvc) ChangePassword(ctx *gin.Context, req *dtouser.ChangePasswordReq) error {
	userID := gincontext.GetUserID(ctx)

	userEntity, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil || userEntity == nil || userEntity.ID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	personEntity, err := dao.NewPersonDao().GetByID(ctx, userEntity.PersonID)
	if err != nil || personEntity == nil {
		return code.GetError(code.UserNotExistError)
	}

	if err := gcrypto.ComparePasswordHash(personEntity.PasswordHash, req.OldPassword); err != nil {
		return code.GetError(code.AuthPasswordError)
	}

	newHash, err := gcrypto.GeneratePasswordHash(req.NewPassword)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.ChangePassword] GeneratePasswordHash fail, err:%v", err)
		return code.GetError(code.UserUpdateError)
	}

	updateMap := map[string]any{
		"password_hash": newHash,
	}
	if err := dao.NewPersonDao().UpdateMap(ctx, personEntity.ID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcuser.ChangePassword] UpdateMap fail, err:%v, personID:%d", err, personEntity.ID)
		return code.GetError(code.UserUpdateError)
	}

	return nil
}
```

- [ ] **Step 4: 在 controller 实现中添加处理函数**

```go
func (ctr *userCtr) ChangePassword(ctx *gin.Context) {
	var req dtouser.ChangePasswordReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.ChangePassword(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "密码修改成功")
}
```

- [ ] **Step 5: 在 router 中注册路由**

```go
v1RouterGroup.POST("/user/changePassword", userCtr.ChangePassword)
```

- [ ] **Step 6: Commit**

```bash
git add apps/iam/internal/controller/ctruser/user.go apps/iam/internal/service/svcuser/user.go apps/iam/internal/router/user.go
git commit -m "feat(iam): add ChangePassword API"
```

---

### Task 5: 实现 LoginHistory 接口

**Files:**
- Modify: `apps/iam/internal/controller/ctruser/user.go`
- Modify: `apps/iam/internal/service/svcuser/user.go`

- [ ] **Step 1: 在 controller 接口定义中添加方法**

```go
LoginHistory(ctx *gin.Context)
```

- [ ] **Step 2: 在 service 接口定义中添加方法**

```go
LoginHistory(ctx *gin.Context, req *dtouser.LoginHistoryReq) (*dtouser.LoginHistoryResp, error)
```

- [ ] **Step 3: 在 service 实现中添加方法**

```go
func (svc *userSvc) LoginHistory(ctx *gin.Context, req *dtouser.LoginHistoryReq) (*dtouser.LoginHistoryResp, error) {
	userID := gincontext.GetUserID(ctx)
	tenantID := gincontext.GetTenantID(ctx)

	cond := &dao.LoginLogCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		UserID:   userID,
		TenantID: tenantID,
	}

	loginLogList, total, err := dao.NewLoginLogDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.LoginHistory] GetPageListByCond fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}

	list := make([]dtouser.LoginHistoryItem, 0, len(loginLogList))
	for _, log := range loginLogList {
		list = append(list, dtouser.LoginHistoryItem{
			ID:           log.ID,
			LoginType:    log.LoginType,
			LoginStatus:  log.LoginStatus,
			LoginMessage: log.LoginMessage,
			IPAddress:    log.IPAddress,
			Location:     log.Location,
			Browser:      log.Browser,
			OS:           log.OS,
			CreatedAt:    log.CreatedAt,
		})
	}

	return &dtouser.LoginHistoryResp{
		List:  list,
		Total: total,
	}, nil
}
```

- [ ] **Step 4: 在 controller 实现中添加处理函数**

```go
func (ctr *userCtr) LoginHistory(ctx *gin.Context) {
	var req dtouser.LoginHistoryReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.LoginHistory(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
```

- [ ] **Step 5: 在 router 中注册路由**

```go
v1RouterGroup.GET("/user/loginHistory", userCtr.LoginHistory)
```

- [ ] **Step 6: Commit**

```bash
git add apps/iam/internal/controller/ctruser/user.go apps/iam/internal/service/svcuser/user.go apps/iam/internal/router/user.go
git commit -m "feat(iam): add LoginHistory API"
```

---

### Task 6: 实现 Logout 接口

**Files:**
- Modify: `apps/iam/internal/controller/ctruser/user.go`
- Modify: `apps/iam/internal/service/svcuser/user.go`

- [ ] **Step 1: 在 controller 接口定义中添加方法**

```go
Logout(ctx *gin.Context)
```

- [ ] **Step 2: 在 service 接口定义中添加方法**

```go
Logout(ctx *gin.Context) error
```

- [ ] **Step 3: 在 service 实现中添加方法（复用 svcauth.Logout）**

```go
func (svc *userSvc) Logout(ctx *gin.Context) error {
	return svcauth.NewAuthSvc().Logout(ctx, "")
}
```

- [ ] **Step 4: 在 controller 实现中添加处理函数**

```go
func (ctr *userCtr) Logout(ctx *gin.Context) {
	if err := ctr.userSvc.Logout(ctx); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "登出成功")
}
```

- [ ] **Step 5: 在 router 中注册路由**

```go
v1RouterGroup.POST("/user/logout", userCtr.Logout)
```

- [ ] **Step 6: Commit**

```bash
git add apps/iam/internal/controller/ctruser/user.go apps/iam/internal/service/svcuser/user.go apps/iam/internal/router/user.go
git commit -m "feat(iam): add Logout API for personal center"
```

---

## Phase 2: 密码安全策略

### Task 7: 添加密码策略配置项常量

**Files:**
- Modify: `apps/iam/model/organization_config.go:1-124`

- [ ] **Step 1: 在 model/organization_config.go 添加密码策略配置项常量**

在 `const (...)` 块中添加：

```go
OrgConfigKeyPasswordMinLength          = "auth.password.minLength"
OrgConfigKeyPasswordRequireUppercase   = "auth.password.requireUppercase"
OrgConfigKeyPasswordRequireLowercase   = "auth.password.requireLowercase"
OrgConfigKeyPasswordRequireNumber      = "auth.password.requireNumber"
OrgConfigKeyPasswordRequireSpecial     = "auth.password.requireSpecial"
OrgConfigKeyLoginMaxFailCount          = "auth.login.maxFailCount"
OrgConfigKeyLoginLockDuration          = "auth.login.lockDuration"
```

- [ ] **Step 2: 在 OrgConfigMetaList 中添加密码策略配置项定义**

在 `var OrgConfigMetaList = []OrgConfigMeta{` 中添加：

```go
{
	Group:        OrgConfigGroupAuth,
	Key:          OrgConfigKeyPasswordMinLength,
	Type:         OrgConfigTypeNumber,
	DefaultValue: "8",
	Description:  "密码最小长度",
},
{
	Group:        OrgConfigGroupAuth,
	Key:          OrgConfigKeyPasswordRequireUppercase,
	Type:         OrgConfigTypeBoolean,
	DefaultValue: "true",
	Description:  "密码必须包含大写字母",
},
{
	Group:        OrgConfigGroupAuth,
	Key:          OrgConfigKeyPasswordRequireLowercase,
	Type:         OrgConfigTypeBoolean,
	DefaultValue: "true",
	Description:  "密码必须包含小写字母",
},
{
	Group:        OrgConfigGroupAuth,
	Key:          OrgConfigKeyPasswordRequireNumber,
	Type:         OrgConfigTypeBoolean,
	DefaultValue: "true",
	Description:  "密码必须包含数字",
},
{
	Group:        OrgConfigGroupAuth,
	Key:          OrgConfigKeyPasswordRequireSpecial,
	Type:         OrgConfigTypeBoolean,
	DefaultValue: "false",
	Description:  "密码必须包含特殊字符",
},
{
	Group:        OrgConfigGroupAuth,
	Key:          OrgConfigKeyLoginMaxFailCount,
	Type:         OrgConfigTypeNumber,
	DefaultValue: "5",
	Description:  "登录失败最大次数",
},
{
	Group:        OrgConfigGroupAuth,
	Key:          OrgConfigKeyLoginLockDuration,
	Type:         OrgConfigTypeNumber,
	DefaultValue: "300",
	Description:  "登录锁定时长(秒)",
},
```

- [ ] **Step 3: Commit**

```bash
git add apps/iam/model/organization_config.go
git commit -m "feat(iam): add password policy config keys"
```

---

### Task 8: 添加用户表字段

**Files:**
- Modify: `apps/iam/model/user.go:1-60`

- [ ] **Step 1: 在 UserEntity 中添加字段**

在 `UserEntity` 结构体中添加：

```go
LoginFailCount int       `gorm:"column:login_fail_count;type:int;;default 0;comment: 连续登录失败次数"`
LockedUntil    *time.Time `gorm:"column:locked_until;type:datetime(3);;default NULL;comment: 账户锁定截止时间"`
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/model/user.go
git commit -m "feat(iam): add login_fail_count and locked_until fields to UserEntity"
```

---

### Task 9: 实现密码复杂度校验

**Files:**
- Create: `apps/iam/internal/service/svcuser/password.go`
- Modify: `apps/iam/internal/service/svcauth/auth.go`

- [ ] **Step 1: 创建密码校验服务**

Create `apps/iam/internal/service/svcuser/password.go`:

```go
package svcuser

import (
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/iam/dao"
	"github.com/morehao/ark-go-starter/apps/iam/model"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type PasswordSvc struct {
}

func NewPasswordSvc() *PasswordSvc {
	return &PasswordSvc{}
}

type PasswordPolicy struct {
	MinLength        int
	RequireUppercase bool
	RequireLowercase bool
	RequireNumber    bool
	RequireSpecial   bool
	MaxFailCount     int
	LockDuration     int
}

func (s *PasswordSvc) GetPasswordPolicy(ctx *gin.Context, orgID uint) (*PasswordPolicy, error) {
	policy := &PasswordPolicy{
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumber:    true,
		RequireSpecial:   false,
		MaxFailCount:     5,
		LockDuration:     300,
	}

	policy.MinLength = s.getOrgConfigInt(ctx, orgID, model.OrgConfigKeyPasswordMinLength, 8)
	policy.RequireUppercase = s.getOrgConfigBool(ctx, orgID, model.OrgConfigKeyPasswordRequireUppercase, true)
	policy.RequireLowercase = s.getOrgConfigBool(ctx, orgID, model.OrgConfigKeyPasswordRequireLowercase, true)
	policy.RequireNumber = s.getOrgConfigBool(ctx, orgID, model.OrgConfigKeyPasswordRequireNumber, true)
	policy.RequireSpecial = s.getOrgConfigBool(ctx, orgID, model.OrgConfigKeyPasswordRequireSpecial, false)
	policy.MaxFailCount = s.getOrgConfigInt(ctx, orgID, model.OrgConfigKeyLoginMaxFailCount, 5)
	policy.LockDuration = s.getOrgConfigInt(ctx, orgID, model.OrgConfigKeyLoginLockDuration, 300)

	return policy, nil
}

func (s *PasswordSvc) ValidatePasswordComplexity(ctx *gin.Context, orgID uint, password string) error {
	policy, err := s.GetPasswordPolicy(ctx, orgID)
	if err != nil {
		return err
	}

	if len(password) < policy.MinLength {
		return code.GetError(code.PasswordComplexityError)
	}

	if policy.RequireUppercase && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return code.GetError(code.PasswordComplexityError)
	}

	if policy.RequireLowercase && !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return code.GetError(code.PasswordComplexityError)
	}

	if policy.RequireNumber && !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return code.GetError(code.PasswordComplexityError)
	}

	if policy.RequireSpecial && !regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password) {
		return code.GetError(code.PasswordComplexityError)
	}

	return nil
}

func (s *PasswordSvc) CheckUserLockStatus(ctx *gin.Context, userID uint) error {
	userEntity, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil || userEntity == nil || userEntity.ID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	if userEntity.Status == model.UserStatusLocked {
		if userEntity.LockedUntil != nil && time.Now().After(*userEntity.LockedUntil) {
			dao.NewUserDao().UpdateMap(ctx, userID, map[string]any{
				"status":           model.UserStatusEnabled,
				"login_fail_count": 0,
				"locked_until":     nil,
			})
		} else {
			return code.GetError(code.UserLockedError)
		}
	}

	if userEntity.Status == model.UserStatusDisabled {
		return code.GetError(code.AuthAccountDisabledError)
	}

	return nil
}

func (s *PasswordSvc) RecordLoginFail(ctx *gin.Context, userID uint, orgID uint) error {
	policy, err := s.GetPasswordPolicy(ctx, orgID)
	if err != nil {
		return err
	}

	userEntity, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil || userEntity == nil {
		return err
	}

	newFailCount := userEntity.LoginFailCount + 1
	updateMap := map[string]any{
		"login_fail_count": newFailCount,
	}

	if newFailCount >= policy.MaxFailCount {
		updateMap["status"] = model.UserStatusLocked
		updateMap["locked_until"] = time.Now().Add(time.Duration(policy.LockDuration) * time.Second)
	}

	return dao.NewUserDao().UpdateMap(ctx, userID, updateMap)
}

func (s *PasswordSvc) ClearLoginFail(ctx *gin.Context, userID uint) error {
	return dao.NewUserDao().UpdateMap(ctx, userID, map[string]any{
		"login_fail_count": 0,
	})
}

func (s *PasswordSvc) getOrgConfigInt(ctx *gin.Context, orgID uint, key string, defaultVal int) int {
	orgID = gincontext.GetOrgID(ctx)
	if orgID == 0 {
		return defaultVal
	}
	configEntity, err := dao.NewOrganizationConfigDao().GetByCond(ctx, &dao.OrganizationConfigCond{
		OrgID: orgID,
		ConfigKey: key,
	})
	if err != nil || configEntity == nil || configEntity.ID == 0 {
		return defaultVal
	}
	var val int
	if _, err := fmt.Sscanf(configEntity.ConfigValue, "%d", &val); err != nil {
		return defaultVal
	}
	return val
}

func (s *PasswordSvc) getOrgConfigBool(ctx *gin.Context, orgID uint, key string, defaultVal bool) bool {
	orgID = gincontext.GetOrgID(ctx)
	if orgID == 0 {
		return defaultVal
	}
	configEntity, err := dao.NewOrganizationConfigDao().GetByCond(ctx, &dao.OrganizationConfigCond{
		OrgID: orgID,
		ConfigKey: key,
	})
	if err != nil || configEntity == nil || configEntity.ID == 0 {
		return defaultVal
	}
	return strings.ToLower(configEntity.ConfigValue) == "true"
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/internal/service/svcuser/password.go
git commit -m "feat(iam): add password policy validation service"
```

---

### Task 10: 调整登录逻辑（失败计数、锁定）

**Files:**
- Modify: `apps/iam/internal/service/svcauth/auth.go`

- [ ] **Step 1: 修改 LoginByPassword 方法**

在 `LoginByPassword` 方法中，验证密码前调用 `CheckUserLockStatus`，验证失败后调用 `RecordLoginFail`，成功后调用 `ClearLoginFail`：

```go
func (svc *authSvc) LoginByPassword(ctx *gin.Context, req *dtoauth.LoginByPasswordReq) (*dtoauth.LoginByPasswordResp, error) {
	account := strings.TrimSpace(req.Account)

	orgEntity, err := svc.getCurrentOrg(ctx)
	if err != nil {
		return nil, err
	}

	personEntity, err := svc.findPersonByAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	userEntity, err := dao.NewUserDao().GetByCond(ctx, &dao.UserCond{
		PersonID: personEntity.ID,
		Status:   model.UserStatusEnabled,
	})
	if err != nil || userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	if err := svcuser.NewPasswordSvc().CheckUserLockStatus(ctx, userEntity.ID); err != nil {
		return nil, err
	}

	if err := gcrypto.ComparePasswordHash(personEntity.PasswordHash, req.Password); err != nil {
		svcuser.NewPasswordSvc().RecordLoginFail(ctx, userEntity.ID, orgEntity.ID)
		glog.Errorf(ctx, "[svcauth.LoginByPassword] password mismatch, account:%s", account)
		return nil, code.GetError(code.AuthPasswordError)
	}

	svcuser.NewPasswordSvc().ClearLoginFail(ctx, userEntity.ID)

	userList, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{
		PersonID: personEntity.ID,
		Status:   model.UserStatusEnabled,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcauth.LoginByPassword] GetListByCond fail, err:%v, personID:%d", err, personEntity.ID)
		return nil, code.GetError(code.AuthLoginError)
	}
	userList, err = svc.filterUsersByOrg(ctx, userList, orgEntity.ID)
	if err != nil {
		return nil, err
	}
	if len(userList) == 0 {
		return nil, code.GetError(code.AuthNoTenantError)
	}

	tempToken, err := svc.generateTempToken(personEntity.ID, orgEntity.ID)
	if err != nil {
		return nil, err
	}

	tenantList, err := svc.buildTenantList(ctx, userList)
	if err != nil {
		return nil, err
	}

	return &dtoauth.LoginByPasswordResp{
		TempToken:        tempToken,
		NeedSelectTenant: true,
		TenantList:       tenantList,
		PersonID:         personEntity.ID,
		RealName:         personEntity.RealName,
	}, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/iam/internal/service/svcauth/auth.go
git commit -m "feat(iam): add login lock mechanism to LoginByPassword"
```

---

### Task 11: 实现 UnlockAccount 接口

**Files:**
- Modify: `apps/iam/internal/dto/dtoauth/request.go`
- Modify: `apps/iam/internal/controller/ctrauth/auth.go`
- Modify: `apps/iam/internal/service/svcauth/auth.go`
- Modify: `apps/iam/internal/router/auth.go`

- [ ] **Step 1: 在 dtoauth/request.go 添加 UnlockAccountReq**

```go
type UnlockAccountReq struct {
	Account  string `json:"account" validate:"required" label:"账号"`
	Captcha  string `json:"captcha" validate:"required" label:"验证码"`
	CaptchaID string `json:"captchaId" label:"验证码ID"`
}
```

- [ ] **Step 2: 在 svcauth/auth.go 添加 UnlockAccount 方法**

```go
func (svc *authSvc) UnlockAccount(ctx *gin.Context, req *dtoauth.UnlockAccountReq) error {
	account := strings.TrimSpace(req.Account)

	personEntity, err := svc.findPersonByAccount(ctx, account)
	if err != nil {
		return err
	}

	userEntity, err := dao.NewUserDao().GetByCond(ctx, &dao.UserCond{
		PersonID: personEntity.ID,
		Status:   model.UserStatusLocked,
	})
	if err != nil || userEntity == nil || userEntity.ID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	if err := dao.NewUserDao().UpdateMap(ctx, userEntity.ID, map[string]any{
		"status":           model.UserStatusEnabled,
		"login_fail_count": 0,
		"locked_until":     nil,
	}); err != nil {
		glog.Errorf(ctx, "[svcauth.UnlockAccount] UpdateMap fail, err:%v, userID:%d", err, userEntity.ID)
		return code.GetError(code.UserUpdateError)
	}

	return nil
}
```

- [ ] **Step 3: 在 ctrauth 添加 UnlockAccount controller**

在 controller 接口中添加：
```go
UnlockAccount(ctx *gin.Context)
```

实现：
```go
func (ctr *authCtr) UnlockAccount(ctx *gin.Context) {
	var req dtoauth.UnlockAccountReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.authSvc.UnlockAccount(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "解锁成功")
}
```

- [ ] **Step 4: 在 auth router 注册路由**

```go
v1RouterGroup.POST("/auth/unlockAccount", authCtr.UnlockAccount)
```

- [ ] **Step 5: Commit**

```bash
git add apps/iam/internal/dto/dtoauth/request.go apps/iam/internal/controller/ctrauth/auth.go apps/iam/internal/service/svcauth/auth.go apps/iam/internal/router/auth.go
git commit -m "feat(iam): add UnlockAccount API"
```

---

## Phase 3: 管理员创建用户

### Task 12: 扩展 Create 接口（密码生成）

**Files:**
- Modify: `apps/iam/internal/dto/dtouser/request.go`
- Modify: `apps/iam/internal/dto/dtouser/response.go`
- Modify: `apps/iam/internal/service/svcuser/user.go`

- [ ] **Step 1: 扩展 UserCreateReq**

添加新字段：
```go
type UserCreateReq struct {
	objuser.UserBaseInfo
	Mobile           string `json:"mobile" form:"mobile"`
	Email            string `json:"email" form:"email"`
	RealName         string `json:"realName" form:"realName"`
	PrimaryDeptID    uint   `json:"primaryDeptID" form:"primaryDeptID"`
	SecondaryDeptIDs []uint `json:"secondaryDeptIDs" form:"secondaryDeptIDs"`
	Password         string `json:"password"`
	SendNotify       bool   `json:"sendNotify"`
}
```

- [ ] **Step 2: 扩展 UserCreateResp**

添加新字段：
```go
type UserCreateResp struct {
	UserID   uint   `json:"userID"`
	PersonID uint   `json:"personID"`
	Password string `json:"password,omitempty"`
}
```

- [ ] **Step 3: 修改 Create 方法**

```go
func (svc *userSvc) Create(ctx *gin.Context, req *dtouser.UserCreateReq) (*dtouser.UserCreateResp, error) {
	tenantID := gincontext.GetTenantID(ctx)
	operatorID := gincontext.GetUserID(ctx)

	orgEntity, err := dao.NewOrganizationDao().GetByCond(ctx, &dao.OrganizationCond{
		ID: gincontext.GetOrgID(ctx),
	})
	if err != nil || orgEntity == nil {
		return nil, code.GetError(code.OrganizationNotExistError)
	}

	primaryDeptID, err := svc.getOrCreatePrimaryDeptID(ctx, tenantID, req.PrimaryDeptID)
	if err != nil {
		return nil, err
	}

	if err := svc.checkUsernameUnique(ctx, tenantID, req.Username); err != nil {
		return nil, err
	}

	var passwordHash string
	if req.Password != "" {
		if err := svcuser.NewPasswordSvc().ValidatePasswordComplexity(ctx, orgEntity.ID, req.Password); err != nil {
			return nil, err
		}
		passwordHash, err = gcrypto.GeneratePasswordHash(req.Password)
		if err != nil {
			glog.Errorf(ctx, "[svcuser.Create] GeneratePasswordHash fail, err:%v", err)
			return nil, code.GetError(code.UserCreateError)
		}
	} else {
		generatedPassword := svc.generateRandomPassword()
		passwordHash, err = gcrypto.GeneratePasswordHash(generatedPassword)
		if err != nil {
			glog.Errorf(ctx, "[svcuser.Create] GeneratePasswordHash fail, err:%v", err)
			return nil, code.GetError(code.UserCreateError)
		}
		req.Password = generatedPassword
	}

	params := &user.CreatePersonParams{
		Mobile:      strings.TrimSpace(req.Mobile),
		Email:       strings.TrimSpace(req.Email),
		RealName:    req.RealName,
		OperatorID:  operatorID,
		TenantID:    tenantID,
		DeptID:      primaryDeptID,
		Username:    req.Username,
		UserType:    model.UserType(req.UserType),
		Status:      model.UserStatus(req.Status),
		EmployeeNo:  req.EmployeeNo,
		JobLevel:    req.JobLevel,
		Position:    req.Position,
		LastLoginIp: req.LastLoginIp,
		LoginCount:  req.LoginCount,
		PasswordHash: passwordHash,
	}

	var result *user.CreatePersonResult
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = user.CreatePersonWithUser(ctx, tx, params)
		if err != nil {
			return err
		}
		if err := svc.createUserDeptRelations(ctx, tx, tenantID, result.UserID, primaryDeptID, req.SecondaryDeptIDs, operatorID); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcuser.Create] Transaction fail, err:%v", txErr)
		return nil, code.GetError(code.UserCreateError)
	}

	if req.SendNotify {
		glog.Infof(ctx, "[svcuser.Create] notify user, userID:%d, notify disabled", result.UserID)
	}

	resp := &dtouser.UserCreateResp{
		UserID:   result.UserID,
		PersonID: result.PersonID,
	}
	if req.Password != "" {
		resp.Password = req.Password
	}

	return resp, nil
}

func (svc *userSvc) generateRandomPassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 12)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}
```

- [ ] **Step 4: Commit**

```bash
git add apps/iam/internal/dto/dtouser/request.go apps/iam/internal/dto/dtouser/response.go apps/iam/internal/service/svcuser/user.go
git commit -m "feat(iam): extend Create API to support custom password and notification"
```

---

## 实施检查清单

- [ ] Phase 1 完成
  - [ ] Task 1: DTO 添加完成
  - [ ] Task 2: GetCurrentUserInfo 完成
  - [ ] Task 3: UpdateProfile 完成
  - [ ] Task 4: ChangePassword 完成
  - [ ] Task 5: LoginHistory 完成
  - [ ] Task 6: Logout 完成

- [ ] Phase 2 完成
  - [ ] Task 7: 密码策略配置项完成
  - [ ] Task 8: 用户表字段完成
  - [ ] Task 9: 密码复杂度校验完成
  - [ ] Task 10: 登录锁定逻辑完成
  - [ ] Task 11: UnlockAccount 完成

- [ ] Phase 3 完成
  - [ ] Task 12: Create 接口扩展完成

- [ ] 全局检查
  - [ ] 运行 `make lint APP=iam`
  - [ ] 运行 `go vet ./apps/iam/...`
  - [ ] 运行 `make test APP=iam`
