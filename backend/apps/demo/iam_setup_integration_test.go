//go:build integration

// 该用例依赖本地 PostgreSQL/Redis，属集成测试：默认 `go test ./...` 不运行，
// 需 `go test -tags=integration ./...`。

package demo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/demo"
	"github.com/morehao/ark-go-starter/demo/config"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/ark-go-starter/pkg/iamlink"
	"github.com/morehao/ark-go-starter/pkg/testsetup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupIAMIntegrationEnv 初始化 DB/Redis 并写入一份启用内嵌 IAM 的配置。
func setupIAMIntegrationEnv(t *testing.T) *gin.Engine {
	t.Helper()
	testsetup.Initialize(testsetup.AppNameDemo)
	t.Cleanup(func() { testsetup.Close(testsetup.AppNameDemo) })

	prev := config.Conf
	t.Cleanup(func() { config.Conf = prev })
	// 加载真实配置（拿到 DBConfigs/RedisConfig），再打开 iam.enable。
	config.LoadConfig(integrationConfigPath(t))
	config.Conf.IAM = config.IAMConfig{Enable: true, Prefix: "/iam"}

	gin.SetMode(gin.TestMode)
	return gin.New()
}

// integrationConfigPath 定位 app 配置文件（从包目录回溯到 apps/demo）。
func integrationConfigPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	appDir := filepath.Dir(filename) // .../demo
	return filepath.Join(appDir, "config", "config.yaml")
}

// TestSetupIAMEnabledMountsRoutes Enable=true：/iam/install 可达、/iam/config.json 可达。
func TestSetupIAMEnabledMountsRoutes(t *testing.T) {
	engine := setupIAMIntegrationEnv(t)

	closer, err := demo.SetupIAM(engine)
	require.NoError(t, err, "Enable=true SetupIAM 应成功")
	require.NotNil(t, closer, "Enable=true 应返回非 nil closer")
	t.Cleanup(func() { _ = closer.Close() })
	require.NotNil(t, demo.CurrentIAM(), "Enable=true 应设置 IAM 运行时")

	for _, path := range []string{"/iam/install", "/iam/config.json"} {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		assert.NotEqual(t, http.StatusNotFound, rec.Code, "%s 应可达（非 404）", path)
	}

	// 建表校验：iam_ 前缀表应存在。
	var cnt int64
	err = dbclient.DemoDB(context.Background()).
		Raw(`SELECT count(*) FROM information_schema.tables WHERE table_name LIKE 'iam_%'`).
		Scan(&cnt).Error
	require.NoError(t, err)
	assert.Positive(t, cnt, "RunMigrations 应建出 iam_ 前缀表")
}

// ensureMapDDL 幂等地创建 biz_iam_user_map（与 scripts/sql/table_init.sql 一致）。
const ensureMapDDL = `
CREATE SEQUENCE IF NOT EXISTS biz_iam_user_map_local_uid_seq AS BIGINT START WITH 1 INCREMENT BY 1;
CREATE TABLE IF NOT EXISTS biz_iam_user_map (
    iam_user_id VARCHAR(128) PRIMARY KEY,
    local_uid   BIGINT NOT NULL DEFAULT nextval('biz_iam_user_map_local_uid_seq'),
    person_id   VARCHAR(128) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_biz_iam_user_map_local_uid ON biz_iam_user_map (local_uid);
`

// TestEnsureLocalUserIdempotent EnsureLocalUser 幂等：同一 IAM user 反复调用得到同一 local_uid。
func TestEnsureLocalUserIdempotent(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameDemo)
	t.Cleanup(func() { testsetup.Close(testsetup.AppNameDemo) })

	db := dbclient.DemoDB(context.Background())
	require.NoError(t, db.Exec(ensureMapDDL).Error, "准备 biz_iam_user_map 失败")
	t.Cleanup(func() {
		db.Exec(`DELETE FROM biz_iam_user_map WHERE iam_user_id = ?`, "itest-user-X")
	})

	svc := iamlink.NewSvc()
	ctx := context.Background()

	uid1, err := svc.EnsureLocalUser(ctx, "itest-user-X", "itest-person-X")
	require.NoError(t, err)
	require.NotZero(t, uid1, "local_uid 不应为 0")

	uid2, err := svc.EnsureLocalUser(ctx, "itest-user-X", "itest-person-X")
	require.NoError(t, err)
	assert.Equal(t, uid1, uid2, "同一 IAM user 应幂等返回同一 local_uid")

	// 不同 IAM user 应得到不同 local_uid。
	uid3, err := svc.EnsureLocalUser(ctx, "itest-user-Y", "itest-person-Y")
	require.NoError(t, err)
	assert.NotEqual(t, uid1, uid3, "不同 IAM user 应分配不同 local_uid")
	t.Cleanup(func() {
		db.Exec(`DELETE FROM biz_iam_user_map WHERE iam_user_id = ?`, "itest-user-Y")
	})
}
