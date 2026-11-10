//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/morehao/golib/task/gcron"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cronSpecFarFuture 是测试用 cron 表达式：每年 1 月 1 日 00:00，
// 保证 e2e 运行窗口内不会真的触发执行（注册本身就是被测行为）。
const cronSpecFarFuture = "0 0 1 1 *"

// cronTaskResp 是定时任务对外契约（对齐 golib gincron.taskResponse）。
type cronTaskResp struct {
	ID          string `json:"id"`
	BizID       string `json:"biz_id"`
	BizType     string `json:"biz_type"`
	Name        string `json:"name"`
	TaskType    string `json:"task_type"`
	Spec        string `json:"spec"`
	Description string `json:"description"`
	Status      string `json:"status"`
	LastRunAt   *int64 `json:"last_run_at"`
	NextRunAt   *int64 `json:"next_run_at"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// cronTaskList 是定时任务分页响应。
type cronTaskList struct {
	List  []cronTaskResp `json:"list"`
	Total int64          `json:"total"`
}

// cronRunResp 是定时任务运行记录契约。
type cronRunResp struct {
	ID         string `json:"id"`
	TaskID     string `json:"task_id"`
	StartAt    int64  `json:"start_at"`
	EndAt      *int64 `json:"end_at"`
	DurationMS int64  `json:"duration_ms"`
	Status     string `json:"status"`
	ErrorMsg   string `json:"error_msg"`
	RequestID  string `json:"request_id"`
	CreatedAt  int64  `json:"created_at"`
}

// cronRunList 是运行记录分页响应。
type cronRunList struct {
	List  []cronRunResp `json:"list"`
	Total int64         `json:"total"`
}

// affectedResp 是 mark-stale / cleanup 的影响行数响应。
type affectedResp struct {
	Affected int64 `json:"affected"`
}

// registerCronTask 通过业务侧接口（Scheduler.Register）注册一个真实定时任务。
//
// gincron 不提供 POST 建任务（任务定义由业务代码注册），因此 e2e 必须走同一入口
// 才能让「启停/移除」这类与调度器内存态绑定的动作可用。
func registerCronTask(t *testing.T, taskID, taskType, name string) {
	t.Helper()
	err := cronScheduler(t).Register(gcron.Task{
		ID:       taskID,
		TaskType: taskType,
		Name:     name,
		Spec:     cronSpecFarFuture,
		Handler:  func(context.Context) error { return nil },
	})
	require.NoError(t, err, "register cron task failed")
	t.Cleanup(func() {
		dbExec(t, `DELETE FROM core_cron_task_run WHERE task_id = ?`, taskID)
		dbExec(t, `DELETE FROM core_cron_task WHERE id = ?`, taskID)
	})
}

// seedCronRun 直连库插入一条 running 状态的运行记录（store 不暴露插入入口，
// 且真实执行窗口由 cron 表达式决定，测试需要一个确定可控的陈旧 running 行）。
func seedCronRun(t *testing.T, taskID string) string {
	t.Helper()
	runID := uniqueTag() + "-cron-run"
	// 主键 varchar(36)：uniqueTag 约 25 字符 + 后缀 9 字符，控制在列宽内。
	dbExec(t, `INSERT INTO core_cron_task_run (id, created_at, task_id, start_at, status, duration_ms)
		VALUES (?, ?, ?, ?, 'running', 0)`,
		runID, time.Now(), taskID, time.Now().Add(-time.Hour))
	t.Cleanup(func() {
		dbExec(t, `DELETE FROM core_cron_task_run WHERE id = ?`, runID)
	})
	return runID
}

// TestCronFlowOverHTTP 覆盖定时任务管理面：注册后列表/详情/启停 → 运行记录查询 →
// mark-stale 修正陈旧 running → cleanup 清理历史 → remove 移除。
func TestCronFlowOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	taskID := tag + "-cron-task"
	taskType := tag + "-cron-type"
	registerCronTask(t, taskID, taskType, tag+"-任务")

	// 1) 列表：按 task_type 精确筛选，只应命中本次注册的任务。
	listResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/cron-tasks?task_type="+taskType, nil, nil)
	require.Equal(t, 0, listResp.Code, listResp.Msg)
	var tasks cronTaskList
	require.NoError(t, json.Unmarshal(listResp.Data, &tasks))
	require.EqualValues(t, 1, tasks.Total, "按唯一 task_type 筛选应只命中 1 条")
	require.Len(t, tasks.List, 1)
	assert.Equal(t, taskID, tasks.List[0].ID)
	assert.Equal(t, "enabled", tasks.List[0].Status)

	// 2) 详情。
	detail := doJSON(t, baseURL, http.MethodGet, "/v1/demo/cron-tasks/"+taskID, nil, nil)
	require.Equal(t, 0, detail.Code, detail.Msg)
	var task cronTaskResp
	require.NoError(t, json.Unmarshal(detail.Data, &task))
	assert.Equal(t, taskID, task.ID)
	assert.Equal(t, cronSpecFarFuture, task.Spec)

	// 3) 启停：disabled → 详情同步；再 enabled 恢复。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPatch,
		"/v1/demo/cron-tasks/"+taskID+"/status", nil, map[string]any{"status": "disabled"}).Code)
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/cron-tasks/"+taskID, nil, nil).Data, &task))
	assert.Equal(t, "disabled", task.Status, "停用后详情应为 disabled")

	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPatch,
		"/v1/demo/cron-tasks/"+taskID+"/status", nil, map[string]any{"status": "enabled"}).Code)
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/cron-tasks/"+taskID, nil, nil).Data, &task))
	assert.Equal(t, "enabled", task.Status, "启用后详情应为 enabled")

	// 4) 运行记录：先造一条 running（start_at 早于 1 小时），再查列表/详情。
	runID := seedCronRun(t, taskID)
	runsResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/cron-tasks/"+taskID+"/runs", nil, nil)
	require.Equal(t, 0, runsResp.Code, runsResp.Msg)
	var runs cronRunList
	require.NoError(t, json.Unmarshal(runsResp.Data, &runs))
	require.EqualValues(t, 1, runs.Total)
	require.Len(t, runs.List, 1)
	assert.Equal(t, runID, runs.List[0].ID)
	assert.Equal(t, "running", runs.List[0].Status)

	runDetail := doJSON(t, baseURL, http.MethodGet, "/v1/demo/cron-runs/"+runID, nil, nil)
	require.Equal(t, 0, runDetail.Code, runDetail.Msg)
	var run cronRunResp
	require.NoError(t, json.Unmarshal(runDetail.Data, &run))
	assert.Equal(t, taskID, run.TaskID)
	assert.Equal(t, "running", run.Status)

	// 5) mark-stale：cutoff 60s 应把 1 小时前的 running 标记成 failed 并补 end_at。
	markResp := doJSON(t, baseURL, http.MethodPost,
		"/v1/demo/cron-tasks/"+taskID+"/runs/mark-stale", nil, map[string]any{"cutoff_seconds": 60})
	require.Equal(t, 0, markResp.Code, markResp.Msg)
	var marked affectedResp
	require.NoError(t, json.Unmarshal(markResp.Data, &marked))
	assert.EqualValues(t, 1, marked.Affected, "应标记 1 条陈旧 running")

	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/cron-runs/"+runID, nil, nil).Data, &run))
	assert.Equal(t, "failed", run.Status, "陈旧 running 应被标记为 failed")
	assert.NotNil(t, run.EndAt, "标记后应补齐 end_at")
	assert.Contains(t, run.ErrorMsg, "stale")

	// 6) cleanup：before=未来时间点，应把该任务的历史运行记录清空。
	cleanResp := doJSON(t, baseURL, http.MethodDelete,
		"/v1/demo/cron-tasks/"+taskID+"/runs", nil, map[string]any{"before_unix": time.Now().Add(time.Hour).Unix()})
	require.Equal(t, 0, cleanResp.Code, cleanResp.Msg)
	var cleaned affectedResp
	require.NoError(t, json.Unmarshal(cleanResp.Data, &cleaned))
	assert.EqualValues(t, 1, cleaned.Affected, "应清理 1 条历史运行记录")

	runs = cronRunList{}
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/cron-tasks/"+taskID+"/runs", nil, nil).Data, &runs))
	assert.EqualValues(t, 0, runs.Total, "清理后运行记录应为空")

	// 7) remove：移除后列表查不到；重复移除应返回业务错误（调度器内已不存在）。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodDelete, "/v1/demo/cron-tasks/"+taskID, nil, nil).Code)
	tasks = cronTaskList{}
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/cron-tasks?task_type="+taskType, nil, nil).Data, &tasks))
	assert.EqualValues(t, 0, tasks.Total, "移除后列表不应再出现该任务")
	assert.NotEqual(t, 0, doError(t, baseURL, http.MethodDelete,
		"/v1/demo/cron-tasks/"+taskID, nil, nil).Code, "重复移除不存在的任务应返回业务错误")
}

// TestCronNegativeOverHTTP 覆盖定时任务的校验/不存在反例（这些路径当前已是规范错误，
// 不依赖 golib 的详情端点修复）。
func TestCronNegativeOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	taskID := tag + "-cron-neg-task"
	registerCronTask(t, taskID, tag+"-cron-neg-type", tag+"-负例任务")

	t.Run("invalid_status_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPatch,
			"/v1/demo/cron-tasks/"+taskID+"/status", nil, map[string]any{"status": "paused"})
		assert.NotEqual(t, 0, resp.Code, "非法状态值应返回业务错误")
	})

	t.Run("status_on_missing_task_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPatch,
			"/v1/demo/cron-tasks/"+uniqueTag()+"/status", nil, map[string]any{"status": "enabled"})
		assert.NotEqual(t, 0, resp.Code, "对不存在的任务启停应返回业务错误")
	})

	t.Run("mark_stale_requires_positive_cutoff", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost,
			"/v1/demo/cron-tasks/"+taskID+"/runs/mark-stale", nil, map[string]any{"cutoff_seconds": 0})
		assert.NotEqual(t, 0, resp.Code, "cutoff_seconds 必须为正数，绑定阶段应拒绝")
	})

	t.Run("cleanup_requires_positive_before", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodDelete,
			"/v1/demo/cron-tasks/"+taskID+"/runs", nil, map[string]any{"before_unix": 0})
		assert.NotEqual(t, 0, resp.Code, "before_unix 必须为正数，绑定阶段应拒绝")
	})
}

// TestCronDetailMissingIDContract 断言「不存在 ID 的详情端点」的契约：
// 必须返回非 5xx 的规范错误响应（业务错误 JSON，code 非 0），不得 panic。
//
// 背景：golib v1.34.0 在这两条路径上会对 nil 实体取字段 panic → gin.Recovery 返回
// 500 空 body；v1.34.1 起 handleGetTask/handleGetRun 把 (nil, nil) 收口为
// ErrTaskNotFound 并以 404 + envelope 返回，本用例随之转为严格校验。
func TestCronDetailMissingIDContract(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	cases := []struct {
		name string
		path string
	}{
		{name: "cron_task_detail_missing_id", path: "/v1/demo/cron-tasks/" + uniqueTag()},
		{name: "cron_run_detail_missing_id", path: "/v1/demo/cron-runs/" + uniqueTag()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, resp, rawBody := doJSONStatus(t, baseURL, http.MethodGet, tc.path, nil, nil)
			require.Less(t, status, 500, "详情端点对不存在 ID 不得 panic/5xx，实际 %d, body: %q", status, string(rawBody))
			require.True(t, json.Valid(rawBody), "应返回业务错误 JSON，实际 body: %q", string(rawBody))
			assert.NotEqual(t, 0, resp.Code, "不存在的 ID 必须返回非 0 业务错误码")
			assert.NotEmpty(t, resp.Msg, "业务错误应带可读 msg")
		})
	}
}
