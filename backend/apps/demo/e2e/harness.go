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
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/demo"
	"github.com/morehao/ark-go-starter/demo/config"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/ark-go-starter/pkg/testsetup"
	"github.com/stretchr/testify/require"
)

// configPath 定位 app 配置文件，约定与 testsetup.findConfigPath 一致（从包目录回溯到 apps/demo）。
func configPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// filename: .../apps/demo/e2e/harness.go，上溯 e2e → demo 即得 .../apps/demo
	appDir := filepath.Dir(filepath.Dir(filename))
	return filepath.Join(appDir, "config", "config.yaml")
}

// apiResp 是统一的接口响应包装。
type apiResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// startServer 在随机空闲端口上启动真实引擎，返回 baseURL 与关闭函数。
//
// 通过 127.0.0.1:0 让内核分配空闲端口，避免与本地 `make run`（8099）冲突；
// 轮询 TCP 就绪后才返回，保证后续请求不会因端口未监听而失败。
func startServer(t *testing.T) (baseURL string, shutdown func()) {
	t.Helper()
	testsetup.Initialize(testsetup.AppNameDemo)
	config.LoadConfig(configPath(t))

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
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

// doJSON 发起 JSON 请求并解析统一响应；要求 HTTP 200。
func doJSON(t *testing.T, baseURL, method, path string, headers map[string]string, body any) apiResp {
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

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	rawBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected http status, body: %s", string(rawBody))

	var out apiResp
	require.NoError(t, json.Unmarshal(rawBody, &out), "body: %s", string(rawBody))
	return out
}

// cleanupProduct 物理删除商品，避免 e2e 数据残留。
func cleanupProduct(t *testing.T, productID uint) {
	t.Helper()
	err := dbclient.DemoDB(context.Background()).Unscoped().
		Exec(`DELETE FROM "product" WHERE id = ?`, productID).Error
	require.NoError(t, err)
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
