//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// asyncTaskResp 是异步任务对外契约（对齐 golib ginasync.taskResponse）。
type asyncTaskResp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// asyncTaskList 是异步任务分页响应。
type asyncTaskList struct {
	List  []asyncTaskResp `json:"list"`
	Total int64           `json:"total"`
}

// asyncRunResp 是异步任务运行记录契约。
type asyncRunResp struct {
	ID         string `json:"id"`
	TaskType   string `json:"task_type"`
	Queue      string `json:"queue"`
	Status     string `json:"status"`
	Retried    int    `json:"retried"`
	MaxRetry   int    `json:"max_retry"`
	StartAt    *int64 `json:"start_at"`
	EndAt      *int64 `json:"end_at"`
	DurationMS int64  `json:"duration_ms"`
	ErrorMsg   string `json:"error_msg"`
	RequestID  string `json:"request_id"`
	CreatedAt  int64  `json:"created_at"`
}

// asyncRunList 是异步运行记录分页响应。
type asyncRunList struct {
	List  []asyncRunResp `json:"list"`
	Total int64          `json:"total"`
}

// registerAsyncTask 通过业务侧接口注册异步任务类型（ginasync 不提供 POST 建任务）。
func registerAsyncTask(t *testing.T, taskType, name string) {
	t.Helper()
	err := asyncServer(t).Register(taskType, func(context.Context, []byte) error { return nil })
	require.NoError(t, err, "register async task failed")
	t.Cleanup(func() {
		dbExec(t, `DELETE FROM core_async_task_run WHERE task_type = ?`, taskType)
		dbExec(t, `DELETE FROM core_async_task WHERE id = ?`, taskType)
	})
}

// seedAsyncRun 直连库插入一条 processing 状态的运行记录（与 cron 同理：执行记录由
// 消费端在真实执行时写入，e2e 需要一个确定可控的陈旧 processing 行）。
func seedAsyncRun(t *testing.T, taskType string) string {
	t.Helper()
	runID := uniqueTag() + "-async-run"
	dbExec(t, `INSERT INTO core_async_task_run (id, created_at, task_type, queue, status, retried, max_retry, start_at)
		VALUES (?, ?, ?, ?, 'processing', 0, 3, ?)`,
		runID, time.Now(), taskType, "default", time.Now().Add(-time.Hour))
	t.Cleanup(func() {
		dbExec(t, `DELETE FROM core_async_task_run WHERE id = ?`, runID)
	})
	return runID
}

// TestAsyncFlowOverHTTP 覆盖异步任务管理面：注册后列表/详情/启停 → 运行记录查询 →
// mark-stale 修正陈旧 processing → cleanup 清理历史。
func TestAsyncFlowOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	taskType := tag + "-async-type"
	registerAsyncTask(t, taskType, tag+"-异步类型")

	// 1) 列表：全量分页里应能找到刚注册的类型（接口不支持按类型筛选）。
	listResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/async-tasks?page=1&page_size=100", nil, nil)
	require.Equal(t, 0, listResp.Code, listResp.Msg)
	var tasks asyncTaskList
	require.NoError(t, json.Unmarshal(listResp.Data, &tasks))
	var found *asyncTaskResp
	for i := range tasks.List {
		if tasks.List[i].ID == taskType {
			found = &tasks.List[i]
			break
		}
	}
	require.NotNil(t, found, "注册的任务类型应出现在列表中")
	assert.Equal(t, "enabled", found.Status)

	// 2) 详情（按类型）。
	detail := doJSON(t, baseURL, http.MethodGet, "/v1/demo/async-tasks/"+taskType, nil, nil)
	require.Equal(t, 0, detail.Code, detail.Msg)
	var task asyncTaskResp
	require.NoError(t, json.Unmarshal(detail.Data, &task))
	assert.Equal(t, taskType, task.ID)
	assert.Equal(t, "enabled", task.Status)

	// 3) 启停。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPatch,
		"/v1/demo/async-tasks/"+taskType+"/status", nil, map[string]any{"status": "disabled"}).Code)
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/async-tasks/"+taskType, nil, nil).Data, &task))
	assert.Equal(t, "disabled", task.Status, "下线后详情应为 disabled")

	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPatch,
		"/v1/demo/async-tasks/"+taskType+"/status", nil, map[string]any{"status": "enabled"}).Code)
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/async-tasks/"+taskType, nil, nil).Data, &task))
	assert.Equal(t, "enabled", task.Status, "恢复后详情应为 enabled")

	// 4) 运行记录：造一条 1 小时前的 processing。
	runID := seedAsyncRun(t, taskType)
	runsResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/async-tasks/"+taskType+"/runs", nil, nil)
	require.Equal(t, 0, runsResp.Code, runsResp.Msg)
	var runs asyncRunList
	require.NoError(t, json.Unmarshal(runsResp.Data, &runs))
	require.EqualValues(t, 1, runs.Total)
	require.Len(t, runs.List, 1)
	assert.Equal(t, runID, runs.List[0].ID)
	assert.Equal(t, "processing", runs.List[0].Status)

	runDetail := doJSON(t, baseURL, http.MethodGet, "/v1/demo/async-runs/"+runID, nil, nil)
	require.Equal(t, 0, runDetail.Code, runDetail.Msg)
	var run asyncRunResp
	require.NoError(t, json.Unmarshal(runDetail.Data, &run))
	assert.Equal(t, taskType, run.TaskType)
	assert.Equal(t, "processing", run.Status)

	// 5) mark-stale：陈旧 processing → failed。
	markResp := doJSON(t, baseURL, http.MethodPost,
		"/v1/demo/async-tasks/"+taskType+"/runs/mark-stale", nil, map[string]any{"cutoff_seconds": 60})
	require.Equal(t, 0, markResp.Code, markResp.Msg)
	var marked affectedResp
	require.NoError(t, json.Unmarshal(markResp.Data, &marked))
	assert.EqualValues(t, 1, marked.Affected, "应标记 1 条陈旧 processing")

	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/async-runs/"+runID, nil, nil).Data, &run))
	assert.Equal(t, "failed", run.Status, "陈旧 processing 应被标记为 failed")
	assert.NotNil(t, run.EndAt, "标记后应补齐 end_at")
	assert.Contains(t, run.ErrorMsg, "stale")

	// 6) cleanup：清空该类型的历史运行记录。
	cleanResp := doJSON(t, baseURL, http.MethodDelete,
		"/v1/demo/async-tasks/"+taskType+"/runs", nil, map[string]any{"before_unix": time.Now().Add(time.Hour).Unix()})
	require.Equal(t, 0, cleanResp.Code, cleanResp.Msg)
	var cleaned affectedResp
	require.NoError(t, json.Unmarshal(cleanResp.Data, &cleaned))
	assert.EqualValues(t, 1, cleaned.Affected, "应清理 1 条历史运行记录")

	runs = asyncRunList{}
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/async-tasks/"+taskType+"/runs", nil, nil).Data, &runs))
	assert.EqualValues(t, 0, runs.Total, "清理后运行记录应为空")
}

// TestAsyncNegativeOverHTTP 覆盖异步任务的校验/不存在反例。
func TestAsyncNegativeOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	taskType := tag + "-async-neg-type"
	registerAsyncTask(t, taskType, tag+"-异步负例")

	t.Run("detail_missing_type_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodGet, "/v1/demo/async-tasks/"+uniqueTag(), nil, nil)
		assert.NotEqual(t, 0, resp.Code, "不存在的任务类型应返回业务错误")
	})

	t.Run("invalid_status_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPatch,
			"/v1/demo/async-tasks/"+taskType+"/status", nil, map[string]any{"status": "paused"})
		assert.NotEqual(t, 0, resp.Code, "非法状态值应返回业务错误")
	})

	t.Run("status_on_missing_type_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPatch,
			"/v1/demo/async-tasks/"+uniqueTag()+"/status", nil, map[string]any{"status": "enabled"})
		assert.NotEqual(t, 0, resp.Code, "对不存在的类型启停应返回业务错误")
	})

	t.Run("mark_stale_requires_positive_cutoff", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost,
			"/v1/demo/async-tasks/"+taskType+"/runs/mark-stale", nil, map[string]any{"cutoff_seconds": 0})
		assert.NotEqual(t, 0, resp.Code, "cutoff_seconds 必须为正数，绑定阶段应拒绝")
	})

	t.Run("cleanup_requires_positive_before", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodDelete,
			"/v1/demo/async-tasks/"+taskType+"/runs", nil, map[string]any{"before_unix": 0})
		assert.NotEqual(t, 0, resp.Code, "before_unix 必须为正数，绑定阶段应拒绝")
	})
}

// TestAsyncRunDetailMissingIDContract 断言 async-runs/:id 对不存在 ID 的契约：
// 必须返回非 5xx 的规范错误响应（业务错误 JSON，code 非 0），不得 panic。
//
// 背景：golib v1.34.0 该端点对 nil 实体取字段 panic → 500 空 body；
// v1.34.1 起 handleGetRun 把 (nil, nil) 收口为 ErrRunNotFound 并以 404 + envelope 返回。
func TestAsyncRunDetailMissingIDContract(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	path := "/v1/demo/async-runs/" + uniqueTag()
	status, resp, rawBody := doJSONStatus(t, baseURL, http.MethodGet, path, nil, nil)
	require.Less(t, status, 500, "详情端点对不存在 ID 不得 panic/5xx，实际 %d, body: %q", status, string(rawBody))
	require.True(t, json.Valid(rawBody), "应返回业务错误 JSON，实际 body: %q", string(rawBody))
	assert.NotEqual(t, 0, resp.Code, "不存在的运行记录必须返回非 0 业务错误码")
	assert.NotEmpty(t, resp.Msg, "业务错误应带可读 msg")
}
