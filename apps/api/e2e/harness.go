//go:build e2e

// Package e2e 是 demo 应用的真端口黑盒端到端测试。
//
// 与 internal/controller 下基于 httptest 的集成级 e2e 不同，这里通过真实 TCP 端口
// 发起 HTTP 请求，覆盖「配置加载 → 中间件 → 路由 → controller → service → dao → 真实库」
// 的完整运行时链路，用于验证构建产物可起、端口可通、核心接口落库正确。
//
// 运行方式：
//
//	go test -tags=e2e ./apps/demo/e2e/... -v
//
// 依赖本地 PostgreSQL(5432)/Redis(6379)（配置见 apps/demo/config/config.yaml），
// 未加 e2e build tag 时默认不编译、不执行，不污染 `go test ./...` 的单元信号。
package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	demo "github.com/morehao/ark-go-starter/apps/api"
	"github.com/morehao/ark-go-starter/apps/api/config"
	"github.com/morehao/ark-go-starter/apps/api/internal/middleware"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/ark-go-starter/pkg/testsetup"
	"github.com/morehao/golib/dict"
	"github.com/morehao/golib/distlock"
	"github.com/morehao/golib/gsetting"
	"github.com/morehao/golib/task/gasync"
	"github.com/morehao/golib/task/gcron"
	"github.com/stretchr/testify/require"
)

// configPath 定位 app 配置文件：优先 config.test.yaml（测试专用，iam.mode=disabled），
// 使真端口 e2e 走 off/DemoIdentity 路径；无测试配置时回退生产 config.yaml。
func configPath(t *testing.T) string {
	t.Helper()
	return testsetup.TestConfigPath(testsetup.AppNameDemo)
}

// apiResp 是统一的接口响应包装。
type apiResp struct {
	Code      int             `json:"code"`
	RequestID string          `json:"requestID"`
	Msg       string          `json:"msg"`
	Data      json.RawMessage `json:"data"`
}

// httpClient 是所有 e2e 请求共用的客户端：统一超时，避免服务端异常时测试挂死。
//
// 注意 SSE 用例不用它（流式响应需要更细粒度的读超时控制），见 sse_e2e_test.go。
var httpClient = &http.Client{Timeout: 15 * time.Second}

// noRedirectClient 不跟随 3xx：用于断言 redirect 端点的状态码与 Location 头本身，
// 而不是被客户端自动跟到（本地存储的预签名 URL 指向配置里的 8099，e2e 端口是随机的）。
var noRedirectClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// startServer 在随机空闲端口上启动真实引擎，返回 baseURL 与关闭函数。
//
// 通过 127.0.0.1:0 让内核分配空闲端口，避免与本地 `make run`（8099）冲突；
// 轮询 TCP 就绪后才返回，保证后续请求不会因端口未监听而失败。
func startServer(t *testing.T) (baseURL string, shutdown func()) {
	t.Helper()
	testsetup.Initialize(testsetup.AppNameDemo)
	config.LoadConfig(configPath(t))
	initGolibComponents(t)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	// 与 cmd/main.go 共用同一份引擎级装配（requestID / panic 500 / 404 / 405 JSON envelope），
	// 否则 404/405/requestID 这类引擎级行为只有真实进程覆盖，e2e 测不到。
	middleware.SetupEngine(engine, demo.AppName)
	demo.Routers(engine)

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err, "listen on random port failed")
	addr := ln.Addr().String()

	srv := &http.Server{
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- serveErr
		}
	}()

	// 轮询端口就绪，最多等 10s。
	dialer := &net.Dialer{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(10 * time.Second)
	for {
		select {
		case serveErr := <-errCh:
			require.NoError(t, serveErr, "server exited unexpectedly")
		default:
		}
		conn, dialErr := dialer.DialContext(context.Background(), "tcp", addr)
		if dialErr == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			require.NoError(t, dialErr, "server not ready within 10s")
		}
		time.Sleep(50 * time.Millisecond)
	}

	shutdown = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		testsetup.Close(testsetup.AppNameDemo)
	}
	return "http://" + addr, shutdown
}

// initGolibComponents 初始化 golib 的字典/配置/定时任务/异步任务组件实例。
//
// 与 cmd/init.go 的 initComponents 保持同一装配顺序与参数：demo 的入口在 package main，
// 无法被 e2e 包 import，因此这里复刻装配。实例写入 config 的进程级单例后，
// demo.Routers → router.registerGolibModules 才会挂上对应路由（否则整组路由 404）。
func initGolibComponents(t *testing.T) {
	t.Helper()
	db := dbclient.DemoDB(context.Background())

	dictInstance, err := dict.New(db)
	require.NoError(t, err, "init dict failed")
	config.SetDict(dictInstance)

	settingInstance, err := gsetting.New(db)
	require.NoError(t, err, "init setting failed")
	config.SetSetting(settingInstance)

	cronInstance, err := gcron.New(db, &config.Conf.Cron, distlock.NewRedisStorage(dbclient.RedisCli))
	require.NoError(t, err, "init cron failed")
	config.SetCron(cronInstance)

	asyncCfg := config.Conf.Async.ToGasync()
	asyncInstance, err := gasync.NewServer(&asyncCfg, db)
	require.NoError(t, err, "init async failed")
	config.SetAsync(asyncInstance)
}

// doJSON 发起 JSON 请求并解析统一响应；要求 HTTP 200。
func doJSON(t *testing.T, baseURL, method, path string, headers map[string]string, body any) apiResp {
	t.Helper()
	status, resp, rawBody := doJSONStatus(t, baseURL, method, path, headers, body)
	require.Equal(t, http.StatusOK, status, "unexpected http status, body: %s", string(rawBody))
	return resp
}

// doJSONStatus 与 doJSON 相同，但把 HTTP 状态码交给调用方断言（业务错误走 200 + code≠0，
// 少数接口用状态码表达语义，如 files 去重冲突 409）。
func doJSONStatus(t *testing.T, baseURL, method, path string, headers map[string]string, body any) (int, apiResp, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, baseURL+path, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	rawBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var out apiResp
	if len(rawBody) > 0 {
		// 非 JSON body（如 panic 后的 500 空 body、storage 端点的纯文本错误）不在此断言，
		// 交给调用方按场景判断，这里只尽力解析。
		_ = json.Unmarshal(rawBody, &out)
	}
	return resp.StatusCode, out, rawBody
}

// doError 发起请求并断言返回「规范业务错误」：HTTP 非 5xx、响应体是统一 envelope、code != 0。
//
// 业务错误码统一走 HTTP 200 + code≠0；但少数错误用状态码本身表达语义（如不存在资源
// 返回 404、files 去重冲突返回 409，见 golib gincontext.FailWithStatus 注释）。
// 因此反例断言不能写死 200，只约束「不是 5xx 且 envelope 里 code 非 0」。
func doError(t *testing.T, baseURL, method, path string, headers map[string]string, body any) apiResp {
	t.Helper()
	status, resp, rawBody := doJSONStatus(t, baseURL, method, path, headers, body)
	require.Less(t, status, 500, "业务错误不应返回 5xx，body: %q", string(rawBody))
	require.True(t, json.Valid(rawBody), "业务错误应返回 JSON envelope，body: %q", string(rawBody))
	require.NotEqual(t, 0, resp.Code, "业务错误 code 不能为 0，body: %s", string(rawBody))
	return resp
}

// doRaw 发起任意请求并返回状态码、响应头与原始 body，供非 JSON envelope 的接口使用
// （files serve/redirect、SSE、swagger 文档）。
func doRaw(t *testing.T, baseURL, method, path string, body io.Reader, headers map[string]string) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, baseURL+path, body)
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	rawBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, resp.Header, rawBody
}

// doRawNoRedirect 与 doRaw 相同，但不跟随 3xx 重定向（用于 redirect 端点断言 Location）。
func doRawNoRedirect(t *testing.T, baseURL, method, path string, body io.Reader, headers map[string]string) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, baseURL+path, body)
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := noRedirectClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	rawBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, resp.Header, rawBody
}

// doMultipartUpload 以 multipart/form-data 上传文件，返回统一 JSON 响应。
func doMultipartUpload(t *testing.T, baseURL, path string, fields map[string]string, fieldName, fileName string, content []byte) apiResp {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	fw, err := mw.CreateFormFile(fieldName, fileName)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+path, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	rawBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var out apiResp
	require.NoError(t, json.Unmarshal(rawBody, &out), "body: %s", string(rawBody))
	return out
}

// doMultipartFieldsOnly 发送只含普通字段（不含文件 part）的 multipart 请求，
// 用于覆盖「缺 file part」的校验分支。
func doMultipartFieldsOnly(t *testing.T, baseURL, path string, fields map[string]string) apiResp {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	require.NoError(t, mw.Close())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+path, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	rawBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var out apiResp
	require.NoError(t, json.Unmarshal(rawBody, &out), "body: %s", string(rawBody))
	return out
}

// suiteSeq 给同一次测试进程内的造数提供单调序号，配合纳秒时间戳保证唯一。
var suiteSeq atomic.Int64

// uniqueTag 生成 e2e 专用唯一标识（e2e- 前缀 + 时间戳 + 序号）。
//
// 用于所有写入真实库的造数：多 agent/多次运行共用同一个 demo 库时，
// 唯一前缀既能让 use 例只命中自己的数据，也让清理不会误删他人数据。
func uniqueTag() string {
	return fmt.Sprintf("e2e-%d-%d", time.Now().UnixNano(), suiteSeq.Add(1))
}

// uniqueDictToken 生成符合 dict 码值字符集（[A-Za-z0-9_.:-]）的唯一标识。
//
// dict 的 code/value 有字符集校验，不允许 '-'，因此把 e2e- 前缀的下划线形式用于码值，
// 展示字段（name/label）仍保留标准 e2e- 前缀。
func uniqueDictToken(prefix string) string {
	return strings.ReplaceAll(prefix+"-"+uniqueTag(), "-", "_")
}

// sha256Hex 计算内容哈希，files 上传/去重都以此为准。
func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// --- 直连库的造数与清理辅助 ---

// dbExec 直连 demo 库执行一条写语句（造数/清理），失败即终止用例。
func dbExec(t *testing.T, sql string, args ...any) {
	t.Helper()
	require.NoError(t, dbclient.DemoDB(context.Background()).Exec(sql, args...).Error, "db exec failed: %s", sql)
}

// dbCount 直连库统计行数，绕开接口层断言落库真实状态。
func dbCount(t *testing.T, sql string, args ...any) int64 {
	t.Helper()
	var count int64
	require.NoError(t, dbclient.DemoDB(context.Background()).Raw(sql, args...).Scan(&count).Error,
		"db count failed: %s", sql)
	return count
}

// cronScheduler 返回当前服务实例使用的调度器（由 initGolibComponents 装配）。
func cronScheduler(t *testing.T) *gcron.Scheduler {
	t.Helper()
	instance := config.GetCron()
	require.NotNil(t, instance, "cron instance is nil, initGolibComponents 未生效")
	return instance
}

// asyncServer 返回当前服务实例使用的异步任务服务端。
func asyncServer(t *testing.T) *gasync.Server {
	t.Helper()
	instance := config.GetAsync()
	require.NotNil(t, instance, "async instance is nil, initGolibComponents 未生效")
	return instance
}

// cleanupProduct 物理删除商品，避免 e2e 数据残留。
func cleanupProduct(t *testing.T, productID uint) {
	t.Helper()
	dbExec(t, `DELETE FROM "product" WHERE id = ?`, productID)
}

// cleanupProductByName 按商品名物理删除（事务用例拿不到回滚后的 ID，只能按名清理）。
func cleanupProductByName(t *testing.T, name string) {
	t.Helper()
	dbExec(t, `DELETE FROM "product" WHERE name = ?`, name)
}

// queryAudit 直连库读取审计列，绕开接口层核对落库值。
type auditRow struct {
	CreatedBy int64
	UpdatedBy int64
	DeletedBy int64
}

func queryAudit(t *testing.T, productID uint) auditRow {
	t.Helper()
	var row auditRow
	err := dbclient.DemoDB(context.Background()).Raw(
		`SELECT COALESCE(created_by,0) AS created_by, COALESCE(updated_by,0) AS updated_by,
		        COALESCE(deleted_by,0) AS deleted_by
		   FROM "product" WHERE id = ?`, productID,
	).Scan(&row).Error
	require.NoError(t, err)
	return row
}

// fmtProductPath 拼接商品资源路径。
func fmtProductPath(productID uint) string {
	return fmt.Sprintf("/v1/demo/products/%d", productID)
}
