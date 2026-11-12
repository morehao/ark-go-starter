package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morehao/ark-go-starter/pkg/code"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/gconstant"
	"github.com/morehao/golib/glog"

	// 注册 zap 日志驱动：下面用 glog.InitLogger 按驱动名 "zap" 创建带字段钩子的 logger。
	_ "github.com/morehao/golib/glog/driver/zap"
)

// logCapture 通过 glog 的字段钩子在写盘前截获结构化日志字段，用于断言访问日志
// 的内容与条数。比读文件可靠：不依赖日志刷新时机与目录布局。
type logCapture struct {
	mu     sync.Mutex
	fields []map[string]any
}

// hook 是 glog.WithFieldHookFunc 的实现：每条日志的字段先进入这里，再交给 writer。
func (c *logCapture) hook(fields []glog.Field) {
	entry := make(map[string]any, len(fields))
	for _, field := range fields {
		entry[field.Key] = field.Value
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fields = append(c.fields, entry)
}

// serverRequests 返回 app.request.id 命中且 event.name 为 http.server.request 的日志字段。
func (c *logCapture) serverRequests(requestID string) []map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	var matched []map[string]any
	for _, entry := range c.fields {
		if entry[gconstant.KeyEventName] != gconstant.ValueEventHTTPServerRequest {
			continue
		}
		if entry[gconstant.KeyAppRequestID] != requestID {
			continue
		}
		matched = append(matched, entry)
	}
	return matched
}

// captureLogs 初始化一个只写临时目录、并挂上字段钩子的 logger。
func captureLogs(t *testing.T) *logCapture {
	t.Helper()
	capture := &logCapture{}
	cfg := glog.LogConfig{
		LoggerType: glog.LoggerTypeZap,
		Service:    "middleware-test",
		Module:     glog.DefaultModuleName,
		Level:      glog.DebugLevel,
		Writers:    []glog.WriterConfig{{Type: glog.WriterFile, Dir: t.TempDir()}},
	}
	require.NoError(t, glog.InitLogger(&cfg, glog.WithFieldHookFunc(capture.hook)))
	t.Cleanup(func() {
		_ = glog.Close()
		// glog 的全局 logger 是进程级单例，Close 后仍指向已关闭实例：同包其它测试
		// （oidc_auth_test.go）再写日志会空指针 panic。这里恢复一个只记 panic 级别的
		// 兜底 logger，保证关闭钩子不影响其它测试。
		fallback := glog.LogConfig{
			LoggerType: glog.LoggerTypeZap,
			Service:    "middleware-test",
			Module:     glog.DefaultModuleName,
			Level:      glog.PanicLevel,
			Writers:    []glog.WriterConfig{{Type: glog.WriterConsole, Level: glog.PanicLevel}},
		}
		_ = glog.InitLogger(&fallback)
	})
	return capture
}

// routeErrorResp 是引擎级兜底响应的 envelope 契约。
type routeErrorResp struct {
	Code      int    `json:"code"`
	RequestID string `json:"requestID"`
	Msg       string `json:"msg"`
}

// doRequest 发起一次请求并返回状态码、响应头、原始 body 与解析后的 envelope。
func doRequest(t *testing.T, server *httptest.Server, method, path, requestID string) (int, http.Header, []byte, routeErrorResp) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, server.URL+path, nil)
	require.NoError(t, err)
	if requestID != "" {
		req.Header.Set(gconstant.HeaderRequestID, requestID)
	}
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	rawBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var out routeErrorResp
	if len(rawBody) > 0 {
		require.NoError(t, json.Unmarshal(rawBody, &out), "响应应为 JSON envelope: %s", string(rawBody))
	}
	return resp.StatusCode, resp.Header, rawBody, out
}

// TestSetupEngineErrorResponses 覆盖 404/405/panic 三类兜底响应的状态码、envelope、
// requestID 与访问日志（事件字段 + 不重复记录）。
func TestSetupEngineErrorResponses(t *testing.T) {
	capture := captureLogs(t)
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	SetupEngine(engine, "demo")
	engine.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	engine.GET("/panic", func(*gin.Context) { panic("boom") })

	// 复刻 ginserver 在 /v1/{app} 分组上挂载的 AccessLog：命中路由由它记录访问日志，
	// 本用例借此断言引擎级兜底不会给命中路由重复记一条。
	group := engine.Group("/v1/demo")
	group.Use(ginmiddleware.AccessLog())
	group.GET("/ping", func(c *gin.Context) { gincontext.Success(c, gin.H{"ok": true}) })

	server := httptest.NewServer(engine)
	t.Cleanup(server.Close)

	t.Run("not_found", func(t *testing.T) {
		const requestID = "unit-404"
		status, header, body, resp := doRequest(t, server, http.MethodGet, "/v1/demo/none", requestID)
		require.Equal(t, http.StatusNotFound, status, "body: %s", string(body))
		assert.Equal(t, code.RouteNotFoundError, resp.Code)
		assert.Equal(t, "接口不存在", resp.Msg)
		assert.Equal(t, requestID, resp.RequestID, "envelope 应回填 requestID")
		assert.Equal(t, requestID, header.Get(gconstant.HeaderRequestID), "响应头应带 requestID")

		events := capture.serverRequests(requestID)
		require.Len(t, events, 1, "404 应恰好一条访问日志")
		assert.Equal(t, http.MethodGet, events[0][gconstant.KeyHttpRequestMethod])
		assert.EqualValues(t, http.StatusNotFound, events[0][gconstant.KeyHttpResponseStatusCode])
		assert.Equal(t, "/v1/demo/none", events[0][gconstant.KeyUrlPath])
		assert.Empty(t, events[0][gconstant.KeyHttpRoute], "未命中路由不应有 http.route")
		assert.Equal(t, "http", events[0][gconstant.KeyErrorType])
		assert.Equal(t, "接口不存在", events[0][gconstant.KeyAppErrorMessage])
		assert.Contains(t, events[0], gconstant.KeyAppRequestDurationMs)
		assert.NotEmpty(t, events[0][gconstant.KeyClientAddress])
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		const requestID = "unit-405"
		status, header, body, resp := doRequest(t, server, http.MethodPost, "/ping", requestID)
		require.Equal(t, http.StatusMethodNotAllowed, status, "body: %s", string(body))
		assert.Equal(t, code.RouteMethodNotAllowedError, resp.Code)
		assert.Equal(t, "请求方法不允许", resp.Msg)
		assert.Equal(t, requestID, resp.RequestID)
		assert.Contains(t, header.Get("Allow"), http.MethodGet, "405 应带 Allow: GET")

		events := capture.serverRequests(requestID)
		require.Len(t, events, 1, "405 应恰好一条访问日志")
		assert.Equal(t, http.MethodPost, events[0][gconstant.KeyHttpRequestMethod])
		assert.EqualValues(t, http.StatusMethodNotAllowed, events[0][gconstant.KeyHttpResponseStatusCode])
		assert.Equal(t, "/ping", events[0][gconstant.KeyUrlPath])
		assert.Equal(t, "请求方法不允许", events[0][gconstant.KeyAppErrorMessage])
	})

	t.Run("matched_route_logged_once", func(t *testing.T) {
		const requestID = "unit-ok"
		status, _, body, resp := doRequest(t, server, http.MethodGet, "/v1/demo/ping", requestID)
		require.Equal(t, http.StatusOK, status, "body: %s", string(body))
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, requestID, resp.RequestID)

		// 命中路由只有 AccessLog 一条：引擎级兜底不得重复记录。
		events := capture.serverRequests(requestID)
		require.Len(t, events, 1, "命中路由应恰好一条访问日志（AccessLog），不得重复")
		assert.EqualValues(t, http.StatusOK, events[0][gconstant.KeyHttpResponseStatusCode])
		assert.Equal(t, "/v1/demo/ping", events[0][gconstant.KeyHttpRoute])
	})

	t.Run("panic_returns_500_envelope", func(t *testing.T) {
		const requestID = "unit-panic"
		status, header, body, resp := doRequest(t, server, http.MethodGet, "/panic", requestID)
		require.Equal(t, http.StatusInternalServerError, status, "body: %s", string(body))
		assert.Equal(t, gconstant.SystemErrorErr, resp.Code)
		assert.Equal(t, requestID, resp.RequestID, "panic envelope 应有 requestID")
		assert.Equal(t, requestID, header.Get(gconstant.HeaderRequestID))

		events := capture.serverRequests(requestID)
		require.Len(t, events, 1, "panic 应恰好一条访问日志")
		assert.EqualValues(t, http.StatusInternalServerError, events[0][gconstant.KeyHttpResponseStatusCode])
		panicErr, _ := events[0][gconstant.KeyAppRequestError].(string)
		assert.Contains(t, panicErr, "boom", "访问日志应带 panic 原因")
	})
}

// TestRequestIDMiddleware 覆盖 requestID 的三种来源：请求头透传、缺失时生成、已有值不覆盖。
func TestRequestIDMiddleware(t *testing.T) {
	captureLogs(t)
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	// 模拟更外层中间件已经写入 requestID（例如网关/测试注入），引擎级中间件不得覆盖它。
	engine.Use(func(c *gin.Context) {
		c.Set(gcontext.KeyRequestID, "preset-request-id")
		c.Next()
	})
	SetupEngine(engine, "demo")
	engine.GET("/ping", func(c *gin.Context) { gincontext.Success(c, nil) })

	server := httptest.NewServer(engine)
	t.Cleanup(server.Close)

	t.Run("keep_existing_value", func(t *testing.T) {
		_, header, _, resp := doRequest(t, server, http.MethodGet, "/ping", "header-request-id")
		assert.Equal(t, "preset-request-id", resp.RequestID, "已存在的 requestID 不应被覆盖")
		assert.Equal(t, "preset-request-id", header.Get(gconstant.HeaderRequestID))
	})

	t.Run("generate_when_absent", func(t *testing.T) {
		// 去掉外层预置中间件的影响：直接验证「无请求头、无预置值」时生成 UUID。
		freshEngine := gin.New()
		SetupEngine(freshEngine, "demo")
		freshEngine.GET("/ping", func(c *gin.Context) { gincontext.Success(c, nil) })
		freshServer := httptest.NewServer(freshEngine)
		t.Cleanup(freshServer.Close)

		_, header1, _, resp1 := doRequest(t, freshServer, http.MethodGet, "/ping", "")
		require.NotEmpty(t, resp1.RequestID, "缺失请求头时应生成 requestID")
		assert.Len(t, resp1.RequestID, 36, "生成的 requestID 应为 UUID")
		assert.Equal(t, resp1.RequestID, header1.Get(gconstant.HeaderRequestID))

		_, _, _, resp2 := doRequest(t, freshServer, http.MethodGet, "/ping", "")
		assert.NotEqual(t, resp1.RequestID, resp2.RequestID, "每个请求应生成新的 requestID")
	})

	t.Run("echo_request_header", func(t *testing.T) {
		freshEngine := gin.New()
		SetupEngine(freshEngine, "demo")
		freshEngine.GET("/ping", func(c *gin.Context) { gincontext.Success(c, nil) })
		freshServer := httptest.NewServer(freshEngine)
		t.Cleanup(freshServer.Close)

		_, header, _, resp := doRequest(t, freshServer, http.MethodGet, "/ping", "client-request-id")
		assert.Equal(t, "client-request-id", resp.RequestID, "应透传客户端 requestID")
		assert.Equal(t, "client-request-id", header.Get(gconstant.HeaderRequestID))
	})
}
