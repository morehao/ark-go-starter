// Package middleware 提供 Gin 中间件，含演示用身份注入与 OIDC 鉴权。
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/libs/go/iamlink"
	"github.com/morehao/ark-iam/contract"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/glog"
)

// OIDC 鉴权相关常量。
const (
	// authHeaderKey 是承载 Bearer 令牌的请求头。
	authHeaderKey = "Authorization"
	// authBearerPrefix 是 Bearer 令牌前缀。
	authBearerPrefix = "Bearer "
)

// LocalUserEnsurer 把 IAM string 身份映射为本地 uint 用户ID。
//
// 抽象为接口便于在无 DB 环境下单测中间件；生产实现为 *iamlink.Svc。
type LocalUserEnsurer interface {
	EnsureLocalUser(ctx context.Context, iamUserID, personID string) (uint, error)
}

// OIDCAuthOption 配置 OIDC 鉴权中间件。
type OIDCAuthOption func(*oidcAuthConfig)

type oidcAuthConfig struct {
	// keySource 是验签公钥来源；用 contract.KeySource（框架无关接口）斩断对 sdk 的依赖。
	// 具体实现（op 形态的 sdk/server、rp 形态的 sdk/rp *Keys）由装配层注入。
	keySource contract.KeySource
	issuer    string
	// audiences 是本应用认可的 aud 集合；非空时验签强制校验 aud（rp 形态安全必需）。
	audiences []string
	// skipPaths 是免鉴权白名单（按路径前缀匹配），如 health/docs。
	skipPaths []string
	// localUsers 把 IAM string 身份映射为本地 uint 用户ID。
	localUsers LocalUserEnsurer
}

// WithOIDCKeySource 注入验签公钥来源（op 形态取 server.App.KeySource()；rp 形态取 rp.*Keys）。
//
// 参数类型为 contract.KeySource，op 与 rp 两种实现均满足该接口，调用方直接传入即可。
func WithOIDCKeySource(source contract.KeySource) OIDCAuthOption {
	return func(c *oidcAuthConfig) { c.keySource = source }
}

// WithOIDCAudiences 注入本应用认可的 aud 集合。
//
// rp 形态对接外部 ark-iam 时必须设置：不校验 aud 会误接受签发给别的应用的令牌。
// 设置后验签要求令牌 aud 命中其中之一，不匹配则 401。op 形态（内嵌同进程）可不设。
func WithOIDCAudiences(audiences ...string) OIDCAuthOption {
	return func(c *oidcAuthConfig) { c.audiences = append(c.audiences, audiences...) }
}

// WithOIDCIssuer 注入期望的 OP issuer（配置后强制校验令牌 iss）。
func WithOIDCIssuer(issuer string) OIDCAuthOption {
	return func(c *oidcAuthConfig) { c.issuer = issuer }
}

// WithOIDCSkipPaths 注入免鉴权路径前缀白名单（如 health/docs）。
func WithOIDCSkipPaths(paths ...string) OIDCAuthOption {
	return func(c *oidcAuthConfig) { c.skipPaths = append(c.skipPaths, paths...) }
}

// WithLocalUserMapper 注入 IAM string 身份 → 本地 uint 用户ID 的映射器。
func WithLocalUserMapper(svc LocalUserEnsurer) OIDCAuthOption {
	return func(c *oidcAuthConfig) { c.localUsers = svc }
}

// OIDCAuth 构造基于 ark-iam sdk 本地验签的鉴权中间件。
//
// 流程：从 Authorization: Bearer 取 access token → 用 OP 公钥本地验签（不访问 OP）→
// 得到 rp.Identity（UserID string）→ 经 iamlink 映射为本地 uint 用户ID → 写入
// gcontext.KeyUserID（uint 可解析字符串）/KeyPersonID/KeyAuthToken。业务侧
// ginhelper.FillOperator 读取这些 gin Keys 时按 uint 解析，无需改动。
//
// fail-closed：密钥来源不可用、令牌缺失/无效/过期一律 401，绝不静默放行。
func OIDCAuth(opts ...OIDCAuthOption) gin.HandlerFunc {
	cfg := &oidcAuthConfig{localUsers: iamlink.NewSvc()}
	for _, opt := range opts {
		opt(cfg)
	}

	verifier := buildVerifier(cfg)

	return func(ctx *gin.Context) {
		if isSkippedPath(ctx.Request.URL.Path, cfg.skipPaths) {
			ctx.Next()
			return
		}
		if verifier == nil {
			glog.Errorf(ctx, "[oidcauth] verifier not ready (key source missing)")
			abortUnauthorized(ctx, "auth verifier unavailable")
			return
		}

		tokenStr := extractToken(ctx)
		if tokenStr == "" {
			glog.Errorf(ctx, "[oidcauth] missing auth token")
			abortUnauthorized(ctx, "missing auth token")
			return
		}

		ident, err := verifier.Verify(ctx, tokenStr)
		if err != nil {
			// 密钥来源整体不可用单独打日志，其余按无效令牌处理（均 401，fail-closed）。
			if errors.Is(err, contract.ErrKeySourceUnavailable) {
				glog.Errorf(ctx, "[oidcauth] key source unavailable, err:%v", err)
			} else {
				glog.Errorf(ctx, "[oidcauth] token verify fail, err:%v", err)
			}
			abortUnauthorized(ctx, "invalid token")
			return
		}

		if err := injectIdentity(ctx, ident, tokenStr, cfg.localUsers); err != nil {
			glog.Errorf(ctx, "[oidcauth] inject identity fail, err:%v, iamUserID:%s", err, ident.UserID)
			abortUnauthorized(ctx, "resolve local user failed")
			return
		}
		ctx.Next()
	}
}

// buildVerifier 依据配置构造验签器；缺少 KeySource 时返回 nil（中间件据此 fail-closed）。
func buildVerifier(cfg *oidcAuthConfig) *rp.Verifier {
	if cfg.keySource == nil {
		return nil
	}
	opts := []rp.VerifierOption{}
	if cfg.issuer != "" {
		opts = append(opts, rp.WithIssuer(cfg.issuer))
	}
	// rp 形态对接外部 IAM 必须校验 aud，否则别的应用的令牌会被误接受（安全必需）。
	if len(cfg.audiences) > 0 {
		opts = append(opts, rp.WithAudiences(cfg.audiences...))
	}
	return rp.NewVerifier(cfg.keySource, opts...)
}

// injectIdentity 把验签通过的 IAM 身份映射为本地 uint 身份并写入 gin 上下文。
//
// 关键：IAM 的 UserID 是 string，而业务侧 operator.Operator.UserID 与业务表审计列均为
// BIGINT（uint）。这里经 iamlink.EnsureLocalUser 映射出 local_uid，写入可被
// gincontext.GetUserID 按 uint 解析的字符串，避免审计列全 0。
func injectIdentity(ctx *gin.Context, ident *rp.Identity, tokenStr string, localUsers LocalUserEnsurer) error {
	if ident.UserID == "" {
		return errors.New("token missing user_id")
	}
	localUID, err := localUsers.EnsureLocalUser(ctx.Request.Context(), ident.UserID, ident.PersonID)
	if err != nil {
		return err
	}
	uidStr := strconv.FormatUint(uint64(localUID), 10)
	ctx.Set(gcontext.KeyUserID, uidStr)
	// 最小控制台域下自然人/用户为 1:1，personID 亦按同一 local_uid 投影，保证
	// operator.PersonID 可按 uint 解析（审计主用 UserID，PersonID 为次要归属）。
	if ident.PersonID != "" {
		ctx.Set(gcontext.KeyPersonID, uidStr)
	}
	ctx.Set(gcontext.KeyAuthToken, tokenStr)
	if ident.TenantID != "" {
		ctx.Set(gcontext.KeyTenantID, ident.TenantID)
	}
	return nil
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

// abortUnauthorized 以 401 终止请求，响应体沿用本站统一错误结构。
func abortUnauthorized(ctx *gin.Context, msg string) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "msg": msg})
}
