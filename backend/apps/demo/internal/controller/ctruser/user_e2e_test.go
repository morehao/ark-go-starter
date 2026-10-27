package ctruser_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/demo"
	"github.com/morehao/ark-go-starter/demo/config"
	"github.com/morehao/ark-go-starter/demo/internal/middleware"
	"github.com/morehao/ark-go-starter/demo/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/ark-go-starter/pkg/testsetup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 依赖本地 PostgreSQL（同 pkg/dbclient 的集成测试），配置读取 apps/demo/config/config.yaml。
//
// 覆盖目标：验证 gin 中间件 → controller → 请求 DTO（operator.Operator）→ service → dao 的完整链路上，
// 操作人身份确实由 controller 从登录态取出并流入 DTO，最终落到 deleted_by 列。

// configPath 定位 app 配置文件，与 testsetup.findConfigPath 保持同一约定（从包目录回溯到 apps/demo）。
func configPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// filename: .../apps/demo/internal/controller/ctruser/user_e2e_test.go
	// 依次上溯 ctruser → controller → internal → demo，得到 .../apps/demo
	appDir := filepath.Dir(filename) // .../ctruser
	for i := 0; i < 3; i++ {
		appDir = filepath.Dir(appDir) // controller → internal → demo
	}
	return filepath.Join(appDir, "config", "config.yaml")
}

// apiResp 通用响应包装。
type apiResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// newTestEngine 构造与 cmd 启动一致的引擎：真实 router + 真实 DTO/service/dao。
//
// config.Conf 由 testsetup 之外的 businessInit 负责，这里直接在初始化前写入，
// 让 demo.Routers 走 dev 分支、挂上 DemoIdentity（与手工验证时同一个中间件）。
func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	// testsetup 负责 DB/Redis/日志（dbclient 单例由它初始化）；
	// 它不写 config.Conf，而 router 依赖它，所以这里单独加载同一份配置文件。
	testsetup.Initialize(testsetup.AppNameDemo)
	config.LoadConfig(configPath(t))
	t.Cleanup(func() { testsetup.Close(testsetup.AppNameDemo) })

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	demo.Routers(engine)
	return engine
}

func doJSON(t *testing.T, engine *gin.Engine, method, path string, headers map[string]string, body any) apiResp {
	t.Helper()
	payload := []byte{}
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		payload = raw
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "unexpected http status, body: %s", rec.Body.String())
	var out apiResp
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), "body: %s", rec.Body.String())
	return out
}

func createUser(t *testing.T, engine *gin.Engine) uint {
	t.Helper()
	return createUserWithHeaders(t, engine, nil)
}

func createUserWithHeaders(t *testing.T, engine *gin.Engine, headers map[string]string) uint {
	t.Helper()
	resp := doJSON(t, engine, http.MethodPost, "/v1/demo/users", headers, map[string]any{
		"companyID":    11,
		"departmentID": 22,
		"name":         "svc-operator-e2e",
	})
	require.Equal(t, 0, resp.Code, "create failed: %s", resp.Msg)
	var created struct {
		UserID uint `json:"userID"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &created))
	require.NotZero(t, created.UserID)
	return created.UserID
}

// auditRow 直接读库核对审计列，绕开接口层，确认值真的落库。
type auditRow struct {
	CreatedBy int64
	UpdatedBy int64
	DeletedBy int64
}

func queryAudit(t *testing.T, userID uint) auditRow {
	t.Helper()
	var row auditRow
	err := dbclient.DemoDB(context.Background()).Raw(
		`SELECT COALESCE(created_by,0) AS created_by, COALESCE(updated_by,0) AS updated_by,
		        COALESCE(deleted_by,0) AS deleted_by
		   FROM "user" WHERE id = ?`, userID,
	).Scan(&row).Error
	require.NoError(t, err)
	return row
}

func cleanupUser(t *testing.T, userID uint) {
	t.Helper()
	err := dbclient.DemoDB(context.Background()).Unscoped().
		Where("id = ?", userID).Delete(&model.UserEntity{}).Error
	require.NoError(t, err)
}

func userPath(userID uint) string {
	return "/v1/demo/users/" + strconv.FormatUint(uint64(userID), 10)
}

func TestUserCRUDWithOperator(t *testing.T) {
	engine := newTestEngine(t)
	// 创建人、修改人、删除人各用不同身份，证明三列分别来自对应请求
	creatorHeaders := map[string]string{
		middleware.HeaderUserID:   "9001",
		middleware.HeaderTenantID: "501",
		middleware.HeaderDeptID:   "601",
	}
	const updaterID uint64 = 9002
	const deleterID uint64 = 9003

	// 创建：created_by/updated_by 应记创建人
	userID := createUserWithHeaders(t, engine, creatorHeaders)
	t.Cleanup(func() { cleanupUser(t, userID) })
	afterCreate := queryAudit(t, userID)
	assert.EqualValues(t, 9001, afterCreate.CreatedBy, "created_by 应等于创建请求的操作人")
	assert.EqualValues(t, 9001, afterCreate.UpdatedBy, "创建时 updated_by 应初始化为创建人")

	// 详情
	detail := doJSON(t, engine, http.MethodGet, userPath(userID), nil, nil)
	assert.Equal(t, 0, detail.Code, detail.Msg)

	// 列表
	list := doJSON(t, engine, http.MethodGet, "/v1/demo/users?page=1&pageSize=10", nil, nil)
	assert.Equal(t, 0, list.Code, list.Msg)

	// 修改：updated_by 应换成修改人，created_by 保持不变
	update := doJSON(t, engine, http.MethodPut, userPath(userID), map[string]string{
		middleware.HeaderUserID: strconv.FormatUint(updaterID, 10),
	}, map[string]any{
		"companyID":    11,
		"departmentID": 22,
		"name":         "svc-operator-e2e-updated",
	})
	assert.Equal(t, 0, update.Code, update.Msg)
	afterUpdate := queryAudit(t, userID)
	assert.EqualValues(t, 9001, afterUpdate.CreatedBy, "修改不应改动 created_by")

	// 详情响应里的审计字段也要带上人员 ID，而不只是时间
	detailAfter := doJSON(t, engine, http.MethodGet, userPath(userID), nil, nil)
	assert.Equal(t, 0, detailAfter.Code, detailAfter.Msg)
	var detailData struct {
		CreatedBy uint `json:"createdBy"`
		UpdatedBy uint `json:"updatedBy"`
	}
	require.NoError(t, json.Unmarshal(detailAfter.Data, &detailData))
	assert.EqualValues(t, 9001, detailData.CreatedBy, "详情响应 createdBy 应来自数据库")
	assert.EqualValues(t, updaterID, detailData.UpdatedBy, "详情响应 updatedBy 应来自数据库")
	assert.EqualValues(t, updaterID, afterUpdate.UpdatedBy, "updated_by 应等于修改请求的操作人")

	// 删除：deleted_by 应等于删除人
	del := doJSON(t, engine, http.MethodDelete, userPath(userID),
		map[string]string{middleware.HeaderUserID: strconv.FormatUint(deleterID, 10)}, nil)
	assert.Equal(t, 0, del.Code, del.Msg)
	afterDelete := queryAudit(t, userID)
	assert.EqualValues(t, deleterID, afterDelete.DeletedBy, "deleted_by 应等于删除请求的操作人")

	// 软删除后再查详情：不应查到
	gone := doJSON(t, engine, http.MethodGet, userPath(userID), nil, nil)
	assert.NotEqual(t, 0, gone.Code, "软删除后不应再查到")
}

// TestUserDeleteWithoutIdentity 无身份时删除仍成功，操作人记为 0。
func TestUserDeleteWithoutIdentity(t *testing.T) {
	engine := newTestEngine(t)
	userID := createUser(t, engine)
	t.Cleanup(func() { cleanupUser(t, userID) })

	del := doJSON(t, engine, http.MethodDelete, userPath(userID), nil, nil)
	assert.Equal(t, 0, del.Code, del.Msg)
	assert.Zero(t, queryAudit(t, userID).DeletedBy)
}

// TestUserDeleteForgedOperatorIgnored 客户端无法通过请求体伪造操作人（Operator 字段 json:"-"）。
func TestUserDeleteForgedOperatorIgnored(t *testing.T) {
	engine := newTestEngine(t)
	userID := createUser(t, engine)
	t.Cleanup(func() { cleanupUser(t, userID) })

	del := doJSON(t, engine, http.MethodDelete, userPath(userID), nil, map[string]any{
		"operator": map[string]any{"userID": 12345},
		"Operator": map[string]any{"userID": 12345},
	})
	assert.Equal(t, 0, del.Code, del.Msg)
	assert.Zero(t, queryAudit(t, userID).DeletedBy, "请求体不能伪造操作人")
}
