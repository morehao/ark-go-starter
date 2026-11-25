package oidcmw

import (
	"github.com/gin-gonic/gin"
)

// Option 配置 OIDC 鉴权中间件。
type Option func(*options)

// UnauthorizedResponder 自定义 401 响应体。
//
// 参数 ctx 是当前 gin 上下文，msg 是失败原因（如 "missing auth token"）。
// 实现方负责终止请求（调用 ctx.AbortWithStatusJSON 等），以便消费方沿用统一错误结构。
type UnauthorizedResponder func(ctx *gin.Context, msg string)

// options 保存中间件的全部可配置项。
type options struct {
	// skipPaths 是免鉴权白名单（按路径前缀匹配），如 health/docs。
	skipPaths []string
	// unauthorized 是 401 响应体钩子；为 nil 时使用默认响应体。
	unauthorized UnauthorizedResponder
}

// WithSkipPaths 注入免鉴权路径前缀白名单（如 health/docs）。
func WithSkipPaths(paths ...string) Option {
	return func(o *options) { o.skipPaths = append(o.skipPaths, paths...) }
}

// WithUnauthorizedResponder 注入 401 响应体钩子，默认返回 {"code":401,"msg":...}。
func WithUnauthorizedResponder(fn UnauthorizedResponder) Option {
	return func(o *options) { o.unauthorized = fn }
}
