package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-go-starter/libs/go/code"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/gconstant"
	"github.com/morehao/golib/gerror"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

// ctxKeyRequestStart 是引擎级中间件内部使用的 gin Keys，保存请求开始时间（time.Time）。
//
// 用 "engine." 前缀与 golib gcontext 的 Key 命名空间区分开，避免与业务 identity Key 冲突。
const ctxKeyRequestStart = "engine.request.start_time"

// SetupEngine 装配引擎级中间件与兜底处理器，补齐「未命中路由」链路的错误语义与可观测性。
//
// 装配顺序与职责：
//  1. RequestID：最外层，为每个请求确定 requestID（请求头 X-Request-Id 优先，缺失则生成
//     UUID），写入 gin Keys（由 gincontext 回填 envelope 的 requestID 字段）与响应头。
//     它必须挂在 engine 级：v1 分组的 AccessLog 只在一行命中路由时运行，NoRoute/NoMethod
//     的请求链路上没有它，不在这里补就会出现「404/405/panic 响应 requestID 为空」。
//  2. Recovery：把 panic 收敛成 JSON 500 envelope，并按 AccessLog 同款字段补一条访问日志。
//  3. 405：gin 默认把「路径存在、方法不匹配」落到 NoRoute 返回 404，丢掉 405 语义；
//     HandleMethodNotAllowed = true 后才会走 NoMethod，由它返回统一 JSON 405（并带 Allow 头）。
//  4. 404：路径不存在时返回统一 JSON 404，而不是 gin 默认的 text/plain。
//
// 这里只负责「未命中路由」与「panic」两条链路：命中路由的访问日志由 ginserver 挂在
// /v1/{app} 分组上的 ginmiddleware.AccessLog 记录，本函数不重复挂载，避免一次请求
// 产生两条 http.server.request 事件。
func SetupEngine(engine *gin.Engine, appName string) {
	// 先挂中间件，再注册 NoRoute/NoMethod：gin 的 NoRoute/NoMethod 会把当前
	// engine.Handlers 拼接到兜底链路前面，顺序反了兜底处理器就取不到 requestID。
	engine.Use(requestIDMiddleware())
	engine.Use(recoveryMiddleware(appName))

	engine.HandleMethodNotAllowed = true
	engine.NoMethod(noMethodHandler())
	engine.NoRoute(noRouteHandler())
}

// requestIDMiddleware 为每个请求确定唯一 requestID，写入 gin Keys 与响应头。
//
// 幂等：gin Keys 里已有 requestID（例如更外层中间件已写入）时沿用原值，不覆盖；
// 否则优先复用请求头 X-Request-Id（便于跨服务串联链路），都没有才生成 UUID。
// 这样与 AccessLog 的取值顺序一致（AccessLog 也是「请求头优先，否则沿用 context，
// 最后才生成」），命中路由时两边拿到的是同一个 ID。
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := gincontext.GetRequestID(c)
		if requestID == "" {
			requestID = strings.TrimSpace(c.GetHeader(gconstant.HeaderRequestID))
		}
		if requestID == "" {
			requestID = gutil.GenUUID()
		}
		c.Set(gcontext.KeyRequestID, requestID)
		// 响应头在进入业务链前就写好，panic 500、404、405 的响应都带得上。
		c.Writer.Header().Set(gconstant.HeaderRequestID, requestID)
		c.Set(ctxKeyRequestStart, time.Now())
		c.Next()
	}
}

// recoveryMiddleware 把 panic 收敛为 JSON 500 envelope，并补一条访问日志。
//
// 复用 gin.CustomRecovery 而不是自己写 recover：它保留 gin 对 broken pipe 等场景的
// 判定与栈打印行为，本仓库只需要在回调里补日志与 envelope。
func recoveryMiddleware(appName string) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		panicErr := fmt.Sprint(recovered)
		appErr := code.GetError(gconstant.SystemErrorErr)
		glog.Errorf(c, "[%s.recovery] panic recovered, method:%s path:%s panic:%s",
			appName, c.Request.Method, c.Request.URL.Path, panicErr)
		logUnmatchedRequest(c, http.StatusInternalServerError, appErr, panicErr)
		gincontext.FailWithStatus(c, http.StatusInternalServerError, appErr)
	})
}

// noRouteHandler 处理未匹配到任何路由的请求：返回统一 JSON 404 并补访问日志。
func noRouteHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		appErr := code.GetError(code.RouteNotFoundError)
		logUnmatchedRequest(c, http.StatusNotFound, appErr, "")
		gincontext.FailWithStatus(c, http.StatusNotFound, appErr)
	}
}

// noMethodHandler 处理「路径存在但 HTTP 方法不允许」的请求：返回统一 JSON 405 并补访问日志。
//
// gin 在进入本处理器前已按 RFC 7231 写入 Allow 响应头，这里只补 envelope 与日志。
func noMethodHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		appErr := code.GetError(code.RouteMethodNotAllowedError)
		logUnmatchedRequest(c, http.StatusMethodNotAllowed, appErr, "")
		gincontext.FailWithStatus(c, http.StatusMethodNotAllowed, appErr)
	}
}

// logUnmatchedRequest 为「未命中业务路由」类响应（404/405/panic 500）补一条访问日志。
//
// 字段与 golib ginmiddleware.AccessLog 对齐（event.name、HTTP 方法/状态码、url.path、
// client.address、app.request.id、耗时等），让同一套日志查询同时覆盖命中与未命中的请求；
// 日志级别也与 AccessLog 一致：5xx 记 error、4xx 记 warn。
//
// 有意不记录 url.full / url.query：AccessLog 写日志前会按敏感字段脱敏，这里没有可复用的
// 脱敏器，记录原始查询串可能把 token 一类凭据落盘；404/405 的排查靠 url.path 已足够。
func logUnmatchedRequest(c *gin.Context, statusCode int, appErr gerror.Error, requestErr string) {
	end := time.Now()
	start := requestStart(c)
	// 与 AccessLog 一致：优先用链路错误，缺失时回落到业务错误文案。
	errorMessage := requestErr
	if errorMessage == "" {
		errorMessage = appErr.Msg
	}
	kvs := []any{
		gconstant.KeyEventName, gconstant.ValueEventHTTPServerRequest,
		gconstant.KeyHttpRequestMethod, c.Request.Method,
		gconstant.KeyHttpResponseStatusCode, statusCode,
		// 未命中路由时 FullPath 为空串，语义即「没有匹配到 http.route」。
		gconstant.KeyHttpRoute, c.FullPath(),
		gconstant.KeyUrlPath, c.Request.URL.Path,
		gconstant.KeyServerAddress, c.Request.Host,
		gconstant.KeyClientAddress, gincontext.GetClientIP(c),
		gconstant.KeyErrorType, "http",
		gconstant.KeyErrorMessage, errorMessage,
		gconstant.KeyAppErrorCode, appErr.Code,
		gconstant.KeyAppErrorMessage, appErr.Msg,
		gconstant.KeyAppRequestID, gincontext.GetRequestID(c),
		gconstant.KeyAppHandler, c.HandlerName(),
		gconstant.KeyNetworkProtocolName, c.Request.Proto,
		gconstant.KeyAppRequestStartTime, gutil.FormatRequestTime(start),
		gconstant.KeyAppRequestEndTime, gutil.FormatRequestTime(end),
		gconstant.KeyAppRequestDurationMs, gutil.GetRequestCost(start, end),
		gconstant.KeyAppRequestError, requestErr,
	}
	switch {
	case statusCode >= 500:
		glog.Errorw(c, gconstant.MsgEventNotice, kvs...)
	case statusCode >= 400:
		glog.Warnw(c, gconstant.MsgEventNotice, kvs...)
	default:
		glog.Infow(c, gconstant.MsgEventNotice, kvs...)
	}
}

// requestStart 取 requestIDMiddleware 记下的请求开始时间；缺失时退化为当前时间（耗时记 0）。
func requestStart(c *gin.Context) time.Time {
	if value, ok := c.Get(ctxKeyRequestStart); ok {
		if start, ok := value.(time.Time); ok {
			return start
		}
	}
	return time.Now()
}
