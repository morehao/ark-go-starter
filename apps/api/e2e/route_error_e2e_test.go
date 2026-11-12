//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	demo "github.com/morehao/ark-go-starter/apps/api"
	"github.com/morehao/ark-go-starter/apps/api/internal/middleware"
	"github.com/morehao/ark-go-starter/libs/go/code"
	"github.com/morehao/ark-go-starter/libs/go/testsetup"
	"github.com/morehao/golib/gconstant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRouteErrorContract 覆盖「未命中业务路由」三条链路的真实 HTTP 契约：
//
//   - 405：POST 到只注册了 GET 的路由必须返回 405（而不是落到 NoRoute 的 404），
//     且响应是统一 JSON envelope（新错误码 + 中文 msg）、带 Allow 头；
//   - 404：路径不存在返回 JSON envelope（RouteNotFoundError）；
//   - requestID：404/405（以及下面 panic 用例的 500）响应的 envelope 与
//     X-Request-Id 响应头都必须非空；客户端传入时原样透传，没传时生成；
//   - 访问日志：404/405 各产生且只产生一条 http.server.request 事件（access log），
//     命中路由的请求同样只有一条（AccessLog 记录），引擎级兜底不重复记。
//
// 日志断言依赖真实 zap 文件日志：先 shutdown（会 flush 并关闭 logger），再读文件，
// 避免依赖 BufferedWriteSyncer 的 5s 刷新间隔。
func TestRouteErrorContract(t *testing.T) {
	baseURL, shutdown := startServer(t)
	shutdownDone := false
	t.Cleanup(func() {
		if !shutdownDone {
			shutdown()
		}
	})

	// requestID 用 uniqueTag 生成：日志文件是追加的，固定常量会命中历史运行留下的同名事件，
	// 导致「有且仅有一条访问日志」的断言误报。
	notFoundID := "e2e-404-" + uniqueTag()
	methodNotAllowedID := "e2e-405-" + uniqueTag()
	matchedID := "e2e-matched-" + uniqueTag()
	var generatedRequestID string

	t.Run("method_not_allowed", func(t *testing.T) {
		// /v1/demo/health 只注册了 GET：POST 必须命中 405 分支。
		status, header, body := doRaw(t, baseURL, http.MethodPost, "/v1/demo/health", nil,
			map[string]string{gconstant.HeaderRequestID: methodNotAllowedID})
		require.Equal(t, http.StatusMethodNotAllowed, status, "body: %s", string(body))

		var resp apiResp
		require.NoError(t, json.Unmarshal(body, &resp), "405 必须返回 JSON envelope: %s", string(body))
		assert.Equal(t, code.RouteMethodNotAllowedError, resp.Code, "405 应返回专用业务码")
		assert.NotEmpty(t, resp.Msg, "405 envelope 应有 msg")
		assert.Contains(t, resp.Msg, "不允许", "405 msg 应为中文")
		assert.Equal(t, methodNotAllowedID, resp.RequestID, "405 envelope 应透传客户端 requestID")
		assert.Equal(t, methodNotAllowedID, header.Get(gconstant.HeaderRequestID), "405 响应头应带 requestID")
		// RFC 7231 §6.5.5：405 必须带 Allow 头。
		assert.Contains(t, header.Get("Allow"), http.MethodGet, "405 应带 Allow: GET")
	})

	t.Run("not_found", func(t *testing.T) {
		status, header, body := doRaw(t, baseURL, http.MethodGet, "/v1/demo/e2e-no-such-route", nil,
			map[string]string{gconstant.HeaderRequestID: notFoundID})
		require.Equal(t, http.StatusNotFound, status, "body: %s", string(body))

		var resp apiResp
		require.NoError(t, json.Unmarshal(body, &resp), "404 必须返回 JSON envelope: %s", string(body))
		assert.Equal(t, code.RouteNotFoundError, resp.Code, "404 应返回 RouteNotFoundError")
		assert.Equal(t, notFoundID, resp.RequestID, "404 envelope 应透传客户端 requestID")
		assert.Equal(t, notFoundID, header.Get(gconstant.HeaderRequestID), "404 响应头应带 requestID")
	})

	t.Run("request_id_generated_when_absent", func(t *testing.T) {
		// 不传 X-Request-Id 时，引擎级中间件必须生成一个，并同时回填 envelope 与响应头。
		status, header, body := doRaw(t, baseURL, http.MethodGet, "/v1/demo/e2e-no-such-route", nil, nil)
		require.Equal(t, http.StatusNotFound, status, "body: %s", string(body))

		var resp apiResp
		require.NoError(t, json.Unmarshal(body, &resp), "body: %s", string(body))
		require.NotEmpty(t, resp.RequestID, "缺失请求头时应生成 requestID")
		assert.Len(t, resp.RequestID, 36, "生成的 requestID 应为 UUID")
		assert.Equal(t, resp.RequestID, header.Get(gconstant.HeaderRequestID), "响应头与 envelope 的 requestID 应一致")
		generatedRequestID = resp.RequestID
	})

	t.Run("matched_route", func(t *testing.T) {
		// 对照组：命中路由的 requestID 由 v1 分组 AccessLog 写入，同样必须透传。
		status, header, body := doRaw(t, baseURL, http.MethodGet, "/v1/demo/health", nil,
			map[string]string{gconstant.HeaderRequestID: matchedID})
		require.Equal(t, http.StatusOK, status, "body: %s", string(body))

		var resp apiResp
		require.NoError(t, json.Unmarshal(body, &resp), "body: %s", string(body))
		assert.Equal(t, matchedID, resp.RequestID)
		assert.Equal(t, matchedID, header.Get(gconstant.HeaderRequestID))
	})

	t.Run("access_log_events", func(t *testing.T) {
		// 先停服：shutdown 会调用 testsetup.Close → glog.Close，刷出文件缓冲。
		shutdown()
		shutdownDone = true

		logFile := findLatestFullLog(t)
		assertSingleAccessLogEvent(t, logFile, notFoundID, http.MethodGet, http.StatusNotFound)
		assertSingleAccessLogEvent(t, logFile, methodNotAllowedID, http.MethodPost, http.StatusMethodNotAllowed)
		assertSingleAccessLogEvent(t, logFile, generatedRequestID, http.MethodGet, http.StatusNotFound)
		// 命中路由只有 AccessLog 一条：引擎级兜底不得重复记录。
		assertSingleAccessLogEvent(t, logFile, matchedID, http.MethodGet, http.StatusOK)
	})
}

// TestPanicResponseHasRequestID 覆盖 panic 链路：gin 的 panic 必须收敛成 JSON 500
// envelope（SystemErrorErr），并且与 404/405 一样带非空 requestID。
//
// 用 httptest 起真实 HTTP（127.0.0.1 随机端口），但不依赖业务路由/业务库：panic 用例
// 只需要引擎级装配，避免为了造一个 panic 接口去改业务代码。
func TestPanicResponseHasRequestID(t *testing.T) {
	// 初始化 logger，避免 panic 日志落到未初始化/已关闭的 logger 上。
	testsetup.Initialize(testsetup.AppNameDemo)
	t.Cleanup(func() { testsetup.Close(testsetup.AppNameDemo) })

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	middleware.SetupEngine(engine, demo.AppName)
	engine.GET("/e2e-panic", func(*gin.Context) {
		panic("e2e panic")
	})
	srv := httptest.NewServer(engine)
	defer srv.Close()

	const requestID = "e2e-panic-request-id"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/e2e-panic", nil)
	require.NoError(t, err)
	req.Header.Set(gconstant.HeaderRequestID, requestID)

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var body bytes.Buffer
	_, err = body.ReadFrom(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode, "panic 应返回 500，body: %s", body.String())
	var out apiResp
	require.NoError(t, json.Unmarshal(body.Bytes(), &out), "panic 必须返回 JSON envelope: %s", body.String())
	assert.Equal(t, gconstant.SystemErrorErr, out.Code, "panic 应返回系统错误码")
	assert.Equal(t, requestID, out.RequestID, "panic envelope 应有 requestID")
	assert.Equal(t, requestID, resp.Header.Get(gconstant.HeaderRequestID), "panic 响应头应带 requestID")
}

// e2eLogDir 是 config.yaml 里 log.dir（../../../log）相对测试进程工作目录解析出的目录。
//
// go test 把工作目录设为被测包目录（apps/demo/e2e），因此上溯三级即 backend/log，
// 与 testsetup.Initialize → glog.InitLogger 使用的相对路径一致。
func e2eLogDir() string {
	return filepath.Join("..", "..", "..", "log")
}

// findLatestFullLog 找到最近被写入的「全量」日志文件（<dir>/<YYYYMMDD>/<service>_full.log）。
func findLatestFullLog(t *testing.T) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(e2eLogDir(), "*", "*_full.log"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "未找到访问日志文件（log.dir=%q），e2e 依赖真实 zap 文件日志", e2eLogDir())

	latest := ""
	var latestMod time.Time
	for _, match := range matches {
		info, statErr := os.Stat(match)
		if statErr != nil {
			continue
		}
		if latest == "" || info.ModTime().After(latestMod) {
			latest, latestMod = match, info.ModTime()
		}
	}
	require.NotEmpty(t, latest, "未找到可读的访问日志文件")
	return latest
}

// findAccessLogEvents 扫描日志文件，返回 app.request.id 命中且 event.name 为
// http.server.request 的日志行（解析成 map 便于逐字段断言）。
func findAccessLogEvents(t *testing.T, logFile, requestID string) []map[string]any {
	t.Helper()
	//nolint:gosec // 路径由本测试内的日志目录定位（e2eLogDir + glob）得出，不来自外部输入。
	f, err := os.Open(logFile)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	var events []map[string]any
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if !bytes.Contains(line, []byte(requestID)) {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry[gconstant.KeyEventName] != gconstant.ValueEventHTTPServerRequest {
			continue
		}
		if entry[gconstant.KeyAppRequestID] != requestID {
			continue
		}
		events = append(events, entry)
	}
	require.NoError(t, scanner.Err())
	return events
}

// assertSingleAccessLogEvent 断言该 requestID 恰好产生一条访问日志，且关键字段与请求一致。
func assertSingleAccessLogEvent(t *testing.T, logFile, requestID, method string, statusCode int) {
	t.Helper()
	events := findAccessLogEvents(t, logFile, requestID)
	require.Len(t, events, 1, "requestID=%s 应有且仅有一条 http.server.request 事件", requestID)

	event := events[0]
	assert.Equal(t, method, event[gconstant.KeyHttpRequestMethod], "访问日志应记录请求方法")
	assert.EqualValues(t, statusCode, event[gconstant.KeyHttpResponseStatusCode], "访问日志应记录状态码")
	assert.Equal(t, requestID, event[gconstant.KeyAppRequestID], "访问日志应记录 requestID")
	assert.NotEmpty(t, event[gconstant.KeyUrlPath], "访问日志应记录 url.path")
	assert.NotEmpty(t, event[gconstant.KeyClientAddress], "访问日志应记录 client.address")
	// 极快的请求耗时可能四舍五入为 0ms，这里只断言字段存在。
	assert.Contains(t, event, gconstant.KeyAppRequestDurationMs, "访问日志应记录耗时字段")

	// 命中路由的日志带 http.route；未命中的为空串（没有可记录的路由模板）。
	if statusCode == http.StatusOK {
		assert.Equal(t, "/v1/demo/health", event[gconstant.KeyHttpRoute], "命中路由应记录 http.route")
	} else {
		assert.Empty(t, event[gconstant.KeyHttpRoute], "未命中路由不应有 http.route")
	}
	// 4xx/5xx 的访问日志应带 HTTP 层错误分类与业务错误文案（中文，不应被转义掉）。
	if statusCode >= 400 {
		assert.Equal(t, "http", event[gconstant.KeyErrorType], "错误响应应记录 error.type=http")
		appErrMsg, _ := event[gconstant.KeyAppErrorMessage].(string)
		assert.True(t, strings.Contains(appErrMsg, "不允许") || strings.Contains(appErrMsg, "不存在"),
			"访问日志应记录业务错误文案，实际: %q", appErrMsg)
	} else {
		assert.EqualValues(t, 0, event[gconstant.KeyAppErrorCode], "成功请求的 app.error.code 应为 0")
	}
}
