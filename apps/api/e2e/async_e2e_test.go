//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/morehao/golib/task/gasync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 任务中心（golib v1.36.3 task/gasync）对外契约。
//
// v1.36.3 起 gasync 从「asynq 任务类型注册表」改为「有状态任务中心」：任务是个体实例
// （有 ID），由 manager.Submit 提交落库，管理面 ginasync 暴露 /asyncTasks 资源式端点
// （列表/详情/尝试记录/取消/重放/统计）。因此这里不再有「按类型注册、启停、mark-stale」
// 这类旧语义，改为围绕具体任务实例的提交 → 查询 → 取消 → 重放 → 统计。
//
// e2e 的 harness 只装配组件、不启动 worker（initGolibComponents 不调用 Start），
// 因此直接 Submit 的任务会稳定停留在 pending，便于对 status 做确定性断言。

// asyncTaskResp 是任务详情/列表项契约（对齐 golib ginasync.asyncTaskResponse）。
type asyncTaskResp struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	BizType     string `json:"biz_type"`
	BizID       string `json:"biz_id"`
	UniqKey     string `json:"uniq_key"`
	Generation  int    `json:"generation"`
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"max_attempts"`
	NextRunAt   int64  `json:"next_run_at"`
	WorkerID    string `json:"worker_id"`
	StartedAt   *int64 `json:"started_at"`
	FinishedAt  *int64 `json:"finished_at"`
	DurationMS  int64  `json:"duration_ms"`
	LastError   string `json:"last_error"`
	RequestID   string `json:"request_id"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// asyncTaskList 是任务分页响应。
type asyncTaskList struct {
	List  []asyncTaskResp `json:"list"`
	Total int64           `json:"total"`
}

// asyncRunResp 是一次尝试记录契约（对齐 golib ginasync.runResponse）。
type asyncRunResp struct {
	ID          string `json:"id"`
	AsyncTaskID string `json:"async_task_id"`
	Generation  int    `json:"generation"`
	Attempt     int    `json:"attempt"`
	Status      string `json:"status"`
	Failure     string `json:"failure"`
	WorkerID    string `json:"worker_id"`
	StartedAt   int64  `json:"started_at"`
	FinishedAt  *int64 `json:"finished_at"`
	DurationMS  int64  `json:"duration_ms"`
	Error       string `json:"error"`
	RequestID   string `json:"request_id"`
	CreatedAt   int64  `json:"created_at"`
}

// asyncRunList 是尝试记录分页响应。
type asyncRunList struct {
	List  []asyncRunResp `json:"list"`
	Total int64          `json:"total"`
}

// asyncCancelResp 是取消结果。
type asyncCancelResp struct {
	Canceled bool `json:"canceled"`
}

// asyncStatsResp 是任务统计。
type asyncStatsResp struct {
	Pending   int64 `json:"pending"`
	Running   int64 `json:"running"`
	Succeeded int64 `json:"succeeded"`
	Failed    int64 `json:"failed"`
	Canceled  int64 `json:"canceled"`
	Skipped   int64 `json:"skipped"`
	Total     int64 `json:"total"`
}

// submitAsyncTask 通过任务中心门面（manager.Submit）提交一条真实任务，返回任务 ID。
//
// 与旧 ginasync「按类型注册处理器」不同，任务中心的任务是提交出来的个体实例；
// 这里用带唯一 type 的 spec 提交，便于列表按 type 精确筛选命中本次任务。
func submitAsyncTask(t *testing.T, taskType string) string {
	t.Helper()
	task, _, err := asyncManager(t).Submit(context.Background(), &gasync.AsyncTaskSpec{
		Type:    taskType,
		Payload: []byte(`{"hello":"world"}`),
		BizType: "e2e",
	})
	require.NoError(t, err, "submit async task failed")
	require.NotEmpty(t, task.ID)
	t.Cleanup(func() {
		dbExec(t, `DELETE FROM core_async_task_run WHERE async_task_id = ?`, task.ID)
		dbExec(t, `DELETE FROM core_async_task WHERE id = ?`, task.ID)
	})
	return task.ID
}

// seedAsyncRun 直连库插入一条 running 状态的尝试记录。
//
// 执行记录由 worker 在真实执行时写入；e2e 需要一个确定可控的尝试行来覆盖「尝试记录查询」。
// 列名对齐 v1.36.3 core_async_task_run（async_task_id/generation/attempt/status/started_at）。
func seedAsyncRun(t *testing.T, taskID string) string {
	t.Helper()
	runID := uniqueTag() + "-async-run"
	dbExec(t, `INSERT INTO core_async_task_run (id, created_at, async_task_id, generation, attempt, status, failure_kind, duration_ms, started_at)
		VALUES (?, ?, ?, 1, 1, 'running', '', 0, ?)`,
		runID, time.Now(), taskID, time.Now().Add(-time.Hour))
	t.Cleanup(func() {
		dbExec(t, `DELETE FROM core_async_task_run WHERE id = ?`, runID)
	})
	return runID
}

// TestAsyncFlowOverHTTP 覆盖任务中心管理面：提交 → 列表/详情 → 尝试记录 →
// 取消（pending → canceled）→ 重放（终态 → pending，generation+1）→ 统计。
func TestAsyncFlowOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	taskType := tag + "-async-type"
	taskID := submitAsyncTask(t, taskType)

	// 1) 列表：按 type 精确筛选，只应命中本次提交的任务，且为 pending。
	listResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/asyncTasks?type="+taskType, nil, nil)
	require.Equal(t, 0, listResp.Code, listResp.Msg)
	var tasks asyncTaskList
	require.NoError(t, json.Unmarshal(listResp.Data, &tasks))
	require.EqualValues(t, 1, tasks.Total, "按唯一 type 筛选应只命中 1 条")
	require.Len(t, tasks.List, 1)
	assert.Equal(t, taskID, tasks.List[0].ID)
	assert.Equal(t, "pending", tasks.List[0].Status)

	// 2) 详情。
	detail := doJSON(t, baseURL, http.MethodGet, "/v1/demo/asyncTasks/"+taskID, nil, nil)
	require.Equal(t, 0, detail.Code, detail.Msg)
	var task asyncTaskResp
	require.NoError(t, json.Unmarshal(detail.Data, &task))
	assert.Equal(t, taskID, task.ID)
	assert.Equal(t, taskType, task.Type)
	assert.Equal(t, "pending", task.Status)

	// 3) 尝试记录：初始为空。
	runsResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/asyncTasks/"+taskID+"/runs", nil, nil)
	require.Equal(t, 0, runsResp.Code, runsResp.Msg)
	var runs asyncRunList
	require.NoError(t, json.Unmarshal(runsResp.Data, &runs))
	assert.EqualValues(t, 0, runs.Total, "新提交的任务应无尝试记录")

	// 4) 造一条 running 尝试记录后，尝试记录列表应能查到它。
	runID := seedAsyncRun(t, taskID)
	runs = asyncRunList{}
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/asyncTasks/"+taskID+"/runs", nil, nil).Data, &runs))
	require.EqualValues(t, 1, runs.Total)
	require.Len(t, runs.List, 1)
	assert.Equal(t, runID, runs.List[0].ID)
	assert.Equal(t, taskID, runs.List[0].AsyncTaskID)
	assert.Equal(t, "running", runs.List[0].Status)

	// 5) 取消：pending → canceled。
	cancelResp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/asyncTasks/"+taskID+"/cancel", nil, nil)
	require.Equal(t, 0, cancelResp.Code, cancelResp.Msg)
	var canceled asyncCancelResp
	require.NoError(t, json.Unmarshal(cancelResp.Data, &canceled))
	assert.True(t, canceled.Canceled, "pending 任务应可取消")
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/asyncTasks/"+taskID, nil, nil).Data, &task))
	assert.Equal(t, "canceled", task.Status, "取消后状态应为 canceled")

	// 6) 重放：canceled（终态）→ pending，generation+1。
	replayResp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/asyncTasks/"+taskID+"/replay", nil, nil)
	require.Equal(t, 0, replayResp.Code, replayResp.Msg)
	var replayed asyncTaskResp
	require.NoError(t, json.Unmarshal(replayResp.Data, &replayed))
	assert.Equal(t, "pending", replayed.Status, "重放后应回到 pending")
	assert.GreaterOrEqual(t, replayed.Generation, 2, "重放应使 generation 递增")

	// 7) 统计：应包含本次任务的计数（至少 total ≥ 1）。
	statsResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/asyncTasks/-/stats", nil, nil)
	require.Equal(t, 0, statsResp.Code, statsResp.Msg)
	var stats asyncStatsResp
	require.NoError(t, json.Unmarshal(statsResp.Data, &stats))
	assert.GreaterOrEqual(t, stats.Total, int64(1), "统计 total 至少包含本次任务")
}

// TestAsyncNegativeOverHTTP 覆盖任务中心的校验/不存在反例。
func TestAsyncNegativeOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	taskID := submitAsyncTask(t, tag+"-async-neg-type")

	t.Run("detail_missing_id_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodGet, "/v1/demo/asyncTasks/"+uniqueTag(), nil, nil)
		assert.NotEqual(t, 0, resp.Code, "不存在的任务 ID 应返回业务错误")
	})

	t.Run("runs_on_missing_task_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodGet, "/v1/demo/asyncTasks/"+uniqueTag()+"/runs", nil, nil)
		assert.NotEqual(t, 0, resp.Code, "不存在任务的尝试记录应返回业务错误")
	})

	t.Run("replay_non_terminal_rejected", func(t *testing.T) {
		// 仍处于 pending（非终态）的任务不允许重放。
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/asyncTasks/"+taskID+"/replay", nil, nil)
		assert.NotEqual(t, 0, resp.Code, "非终态任务重放应返回业务错误")
	})

	t.Run("cancel_missing_id_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/asyncTasks/"+uniqueTag()+"/cancel", nil, nil)
		assert.NotEqual(t, 0, resp.Code, "取消不存在的任务应返回业务错误")
	})
}

// TestAsyncDetailMissingIDContract 断言「不存在 ID 的详情端点」的契约：
// 必须返回非 5xx 的规范错误响应（业务错误 JSON，code 非 0），不得 panic。
//
// v1.36.3 ginasync 的 handleGet 把 (nil, nil) 收口为 ErrAsyncTaskNotFound 并以 404 +
// envelope 返回，本用例随之严格校验。
func TestAsyncDetailMissingIDContract(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	path := "/v1/demo/asyncTasks/" + uniqueTag()
	status, resp, rawBody := doJSONStatus(t, baseURL, http.MethodGet, path, nil, nil)
	require.Less(t, status, 500, "详情端点对不存在 ID 不得 panic/5xx，实际 %d, body: %q", status, string(rawBody))
	require.True(t, json.Valid(rawBody), "应返回业务错误 JSON，实际 body: %q", string(rawBody))
	assert.NotEqual(t, 0, resp.Code, "不存在的 ID 必须返回非 0 业务错误码")
	assert.NotEmpty(t, resp.Msg, "业务错误应带可读 msg")
}
