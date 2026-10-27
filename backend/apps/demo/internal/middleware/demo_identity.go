package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/biz/gcontext"
)

// 演示用身份请求头。
const (
	HeaderUserID   = "X-User-ID"
	HeaderPersonID = "X-Person-ID"
	HeaderTenantID = "X-Tenant-ID"
	HeaderOrgID    = "X-Org-ID"
	HeaderDeptID   = "X-Dept-ID"
	HeaderUserType = "X-User-Type"
)

// DemoIdentity 演示用的身份中间件：从请求头读取身份写入 gin Keys。
//
// 这是接入真实鉴权之前的**占位实现**，只应在 dev 环境挂载（见 demo.Routers）：
// 它信任客户端传入的请求头，任何客户端都能伪造身份，绝不可用于生产。
// 接入真实鉴权后应删除本中间件，改由鉴权中间件解析令牌 claims 后写入同名 gin Keys。
//
// 只负责“身份”，不负责租户隔离：数据隔离由 gcontext.TenantScope 在 DAO 侧强制。
func DemoIdentity() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		setUintKey(ctx, gcontext.KeyUserID, HeaderUserID)
		setUintKey(ctx, gcontext.KeyPersonID, HeaderPersonID)
		setUintKey(ctx, gcontext.KeyTenantID, HeaderTenantID)
		setUintKey(ctx, gcontext.KeyOrgID, HeaderOrgID)
		setUintKey(ctx, gcontext.KeyDeptID, HeaderDeptID)
		if userType := ctx.GetHeader(HeaderUserType); userType != "" {
			ctx.Set(gcontext.KeyUserType, userType)
		}
		ctx.Next()
	}
}

// setUintKey 把 header 里的无符号整数写入 gin Keys；缺失或非法时保留原值（不覆盖）。
func setUintKey(ctx *gin.Context, key, header string) {
	raw := ctx.GetHeader(header)
	if raw == "" {
		return
	}
	if value, err := strconv.ParseUint(raw, 10, 64); err == nil {
		ctx.Set(key, strconv.FormatUint(value, 10))
	}
}
