// Package oidcmw 提供基于 ark-iam sdk/rp 的通用 Gin + OIDC 本地验签中间件。
//
// 它是纯适配层：取 Bearer 令牌 → verifier.Verify 本地验签 → rp.WithIdentity 把身份注入
// request context → 失败一律 fail-closed 401。它不含任何业务语义，也不 import 任何
// ark-go-starter 业务包，只依赖 gin / sdk/rp / contract；验签器（含 issuer/aud 策略）
// 由装配层构造后注入，业务身份映射（string→BIGINT）由独立的下游中间件（如 pkg/iamlink）完成。
package oidcmw

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/contract"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/morehao/golib/glog"
)

// OIDC 鉴权相关常量。
const (
	// authHeaderKey 是承载 Bearer 令牌的请求头。
	authHeaderKey = "Authorization"
	// authBearerPrefix 是 Bearer 令牌前缀。
	authBearerPrefix = "Bearer "
)

// OIDCAuth 构造 OIDC 本地验签中间件。
//
// verifier 由装配层用 rt.KeySource() + rp.NewVerifier(WithIssuer, WithAudiences) 构造一次后注入；
// 传 nil 时中间件 fail-closed（一律 401），便于在密钥来源缺失时拒绝而非放行。
//
// 流程：从 Authorization: Bearer 取 access token → verifier.Verify 本地验签（不访问 OP）→
// 得到 rp.Identity → 经 rp.WithIdentity 注入 request context，供下游中间件用
// rp.IdentityFrom 取回完整身份（UserID/PersonID/TenantID/Scopes 等）。
func OIDCAuth(verifier *rp.Verifier, opts ...Option) gin.HandlerFunc {
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(ctx *gin.Context) {
		if isSkippedPath(ctx.Request.URL.Path, cfg.skipPaths) {
			ctx.Next()
			return
		}
		if verifier == nil {
			glog.Errorf(ctx, "[oidcmw] verifier not ready (key source missing)")
			respondUnauthorized(ctx, cfg, "auth verifier unavailable")
			return
		}

		tokenStr := extractToken(ctx)
		if tokenStr == "" {
			glog.Errorf(ctx, "[oidcmw] missing auth token")
			respondUnauthorized(ctx, cfg, "missing auth token")
			return
		}

		ident, err := verifier.Verify(ctx, tokenStr)
		if err != nil {
			// 密钥来源整体不可用单独打日志，其余按无效令牌处理（均 401，fail-closed）。
			if errors.Is(err, contract.ErrKeySourceUnavailable) {
				glog.Errorf(ctx, "[oidcmw] key source unavailable, err:%v", err)
			} else {
				glog.Errorf(ctx, "[oidcmw] token verify fail, err:%v", err)
			}
			respondUnauthorized(ctx, cfg, "invalid token")
			return
		}

		// 把已验证身份注入 request context（rp/contract 共用同一私有键）。
		ctx.Request = ctx.Request.WithContext(rp.WithIdentity(ctx.Request.Context(), ident))
		ctx.Next()
	}
}

// isSkippedPath 报告 path 是否命中免鉴权前缀白名单。
func isSkippedPath(path string, skipPaths []string) bool {
	for _, p := range skipPaths {
		if p != "" && strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// extractToken 从 Authorization 头提取 access token（兼容带/不带 Bearer 前缀）。
func extractToken(ctx *gin.Context) string {
	auth := ctx.GetHeader(authHeaderKey)
	if auth == "" {
		return ""
	}
	if token, ok := strings.CutPrefix(auth, authBearerPrefix); ok {
		return token
	}
	return auth
}

// respondUnauthorized 以 401 终止请求；优先使用配置的钩子，否则用默认响应体。
func respondUnauthorized(ctx *gin.Context, cfg *options, msg string) {
	if cfg.unauthorized != nil {
		cfg.unauthorized(ctx, msg)
		return
	}
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "msg": msg})
}
