// Package iamlink 负责把外部 IAM 的字符串身份映射为本地 BIGINT 用户身份。
package iamlink

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/glog"
)

// LocalUserEnsurer 把 IAM string 身份映射为本地 uint 用户ID。
//
// 抽象为接口便于在无 DB 环境下单测中间件；生产实现为 *Svc。
type LocalUserEnsurer interface {
	EnsureLocalUser(ctx context.Context, issuer, iamUserID, personID string) (uint, error)
}

// MiddlewareOption 配置身份映射中间件。
type MiddlewareOption func(*middlewareConfig)

// middlewareConfig 保存映射中间件的可配置项。
type middlewareConfig struct {
	// issuer 是装配层已知的 OP issuer，作为映射联合键的一部分（不读 token）。
	issuer string
	// localUsers 是 string→uint 映射器；默认使用 NewSvc()。
	localUsers LocalUserEnsurer
}

// WithLocalUserEnsurer 注入身份映射器（便于单测替换）；默认 NewSvc()。
func WithLocalUserEnsurer(svc LocalUserEnsurer) MiddlewareOption {
	return func(c *middlewareConfig) { c.localUsers = svc }
}

// Middleware 构造身份映射中间件。
//
// 前置依赖 oidcmw.OIDCAuth 已把 rp.Identity 注入 request context；本中间件从
// rp.IdentityFrom(ctx) 取回 identity，经 EnsureLocalUser（(issuer, iamUserID, personID)
// 复合键）映射为本地 uint 用户ID，再写入 gcontext.KeyUserID/KeyPersonID/KeyAuthToken/
// KeyTenantID，供业务侧 ginhelper.FillOperator 按 uint 解析。
//
// issuer 由装配层注入（已用于精确校验 token 的 iss），不从 token 或 identity 读取：
// 不同 IDP 下相同 sub 必须映射到不同的本地用户。
//
// fail-closed：identity 缺失、user_id 为空、映射失败一律 401。
func Middleware(issuer string, opts ...MiddlewareOption) gin.HandlerFunc {
	cfg := &middlewareConfig{issuer: issuer, localUsers: NewSvc()}
	for _, opt := range opts {
		opt(cfg)
	}
	return func(ctx *gin.Context) {
		ident := rp.IdentityFrom(ctx.Request.Context())
		if ident == nil {
			glog.Errorf(ctx, "[iamlink] identity missing from context (oidcmw not applied?)")
			abortUnauthorized(ctx, "identity missing")
			return
		}
		if ident.UserID == "" {
			glog.Errorf(ctx, "[iamlink] token missing user_id")
			abortUnauthorized(ctx, "invalid token")
			return
		}

		localUID, err := cfg.localUsers.EnsureLocalUser(ctx.Request.Context(), cfg.issuer, ident.UserID, ident.PersonID)
		if err != nil {
			glog.Errorf(ctx, "[iamlink] ensure local user fail, err:%v, iamUserID:%s", err, ident.UserID)
			abortUnauthorized(ctx, "resolve local user failed")
			return
		}

		uidStr := strconv.FormatUint(uint64(localUID), 10)
		ctx.Set(gcontext.KeyUserID, uidStr)
		// 最小控制台域下自然人/用户为 1:1，personID 亦按同一 local_uid 投影，保证
		// operator.PersonID 可按 uint 解析（审计主用 UserID，PersonID 为次要归属）。
		if ident.PersonID != "" {
			ctx.Set(gcontext.KeyPersonID, uidStr)
		}
		ctx.Set(gcontext.KeyAuthToken, extractToken(ctx))
		if ident.TenantID != "" {
			ctx.Set(gcontext.KeyTenantID, ident.TenantID)
		}
		ctx.Next()
	}
}

// abortUnauthorized 以 401 终止请求，响应体沿用本站统一错误结构。
func abortUnauthorized(ctx *gin.Context, msg string) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "msg": msg})
}

// extractToken 从 Authorization 头取原始 access token（兼容带/不带 Bearer 前缀）。
func extractToken(ctx *gin.Context) string {
	auth := ctx.GetHeader("Authorization")
	if token, ok := strings.CutPrefix(auth, "Bearer "); ok {
		return token
	}
	return auth
}
