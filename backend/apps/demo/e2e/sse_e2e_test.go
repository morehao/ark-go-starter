//go:build e2e

package e2e

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sseReading 是一次 SSE 读取的采集结果。
type sseReading struct {
	contentType string
	events      []string
	dataLines   []string
}

// readSSE 连接 SSE 端点，读到 want 条 data 事件后主动断开（不读满、不等待流自然结束）。
//
// 用 request context 控制整体读超时：SSE 是无限流，靠 http.Client.Timeout 容易把
// 「流正常但耗时长」误判成失败；这里只约束读完 N 条事件所需的时间。
func readSSE(t *testing.T, baseURL, path string, want int) sseReading {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	require.NoError(t, err)
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "SSE 连接失败: %s", path)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode, "SSE 应返回 200")
	contentType := resp.Header.Get("Content-Type")
	require.True(t, strings.HasPrefix(contentType, "text/event-stream"),
		"SSE Content-Type 应为 text/event-stream, actual: %q", contentType)

	out := sseReading{contentType: contentType}
	reader := bufio.NewReader(resp.Body)
	for len(out.dataLines) < want {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			t.Fatalf("读取 SSE 第 %d 条 data 事件失败（已读 %d 条）: %v",
				len(out.dataLines)+1, len(out.dataLines), readErr)
		}
		line = strings.TrimRight(line, "\r\n")
		// 兼容两种写法：手写流是 "data: xxx"（带空格），gin 的 ctx.SSEvent 生成
		// "event:xxx" / "data:xxx"（冒号后无空格）。SSE 规范两者都合法。
		switch {
		case strings.HasPrefix(line, "event:"):
			out.events = append(out.events, strings.TrimSpace(strings.TrimPrefix(line, "event:")))
		case strings.HasPrefix(line, "data:"):
			out.dataLines = append(out.dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	// 读到目标条数后立即关闭连接，模拟浏览器/客户端主动断开。
	return out
}

// TestSSEEndpoints 覆盖三个 SSE 示例端点：各读 3 条事件后断开，
// 断言响应头、事件内容格式与「流可连续推送」（不阻塞、不需要读满到结束）。
func TestSSEEndpoints(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	t.Run("sse_time", func(t *testing.T) {
		reading := readSSE(t, baseURL, "/v1/demo/sse/time", 3)
		require.Len(t, reading.dataLines, 3)
		for _, line := range reading.dataLines {
			_, err := time.Parse(time.DateTime, line)
			assert.NoError(t, err, "data 应是 time.DateTime 格式的时间串, actual: %q", line)
		}
	})

	t.Run("sse_time_raw", func(t *testing.T) {
		reading := readSSE(t, baseURL, "/v1/demo/sse/timeRaw", 3)
		require.Len(t, reading.dataLines, 3)
		for _, line := range reading.dataLines {
			_, err := time.Parse(time.DateTime, line)
			assert.NoError(t, err, "timeRaw 的 data 应是时间串, actual: %q", line)
		}
		assert.Contains(t, reading.events, "time", "timeRaw 应带 event: time")
	})

	t.Run("sse_process", func(t *testing.T) {
		reading := readSSE(t, baseURL, "/v1/demo/sse/process", 3)
		require.Len(t, reading.dataLines, 3)
		assert.Contains(t, reading.events, "progress", "process 应以 progress 事件推进度")
		assert.Contains(t, reading.dataLines[0], "progress", "data 应是进度 JSON, actual: %q", reading.dataLines[0])
	})
}
