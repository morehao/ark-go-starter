//go:build integration

// 该用例依赖本地 PostgreSQL/Redis，属集成测试：默认 `go test ./...` 不运行，
// 需 `go test -tags=integration ./...`。

package demo_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	demo "github.com/morehao/ark-go-starter/apps/api"
	"github.com/morehao/ark-go-starter/apps/api/config"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/ark-go-starter/pkg/iamlink"
	"github.com/morehao/ark-go-starter/pkg/testsetup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockJWKS 启动一个返回单条 RSA 公钥 JWKS 的假 OIDC 端点（/keys），并返回其 URL 与清理函数。
func mockJWKS(t *testing.T) string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pub := priv.Public().(*rsa.PublicKey)
	jwks := map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA",
			"kid": "test-kid",
			"alg": "RS256",
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/keys" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/keys"
}

// TestSetupIAMRPMockJWKS rp 形态：用假 JWKS 端点验证 setupRP 成功、KeySource 非空、且不挂任何路由。
func TestSetupIAMRPMockJWKS(t *testing.T) {
	// rp 形态虽不建库/不挂路由，但 setupRP 会经 glog 打印启动日志；必须先初始化日志驱动，
	// 否则 glog.Infof 会在未初始化的 zap logger 上 panic（nil BufferedWriteSyncer）。
	testsetup.Initialize(testsetup.AppNameDemo)
	t.Cleanup(func() { testsetup.Close(testsetup.AppNameDemo) })

	prev := config.Conf
	t.Cleanup(func() { config.Conf = prev })

	jwksURL := mockJWKS(t)
	config.Conf = &config.Config{
		Server: config.Server{Name: "demo", Port: "8099", Env: "dev"},
		IAM:    config.IAMConfig{Mode: config.ModeRP, JWKSURL: jwksURL, Issuer: "http://op.example/oidc"},
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()

	closer, err := demo.SetupIAM()
	require.NoError(t, err, "rp SetupIAM 应成功（JWKS 预取成功）")
	require.NotNil(t, closer, "rp 应返回非 nil closer")
	t.Cleanup(func() { _ = closer.Close() })

	rt := demo.CurrentIAM()
	require.NotNil(t, rt, "rp 应设置 IAM 运行时")
	require.NotNil(t, rt.KeySource(), "rp 形态必须产出非空 KeySource")

	// 能按 kid 取到公钥（验签路径可用）。
	pub, err := rt.KeySource().PublicKey(context.Background(), "test-kid")
	require.NoError(t, err, "应按 kid 取到公钥")
	assert.NotNil(t, pub)

	// rp 不挂任何 /iam/* 路由（不建库、不 Mount）。
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/iam/install/status", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code, "rp 时不应挂载任何 IAM 路由")
}

// ensureMapDDL 幂等地创建 biz_iam_user_map（与 scripts/sql/table_init.sql 一致）。
const ensureMapDDL = `
CREATE SEQUENCE IF NOT EXISTS biz_iam_user_map_local_uid_seq AS BIGINT START WITH 1 INCREMENT BY 1;
CREATE TABLE IF NOT EXISTS biz_iam_user_map (
    issuer      VARCHAR(255) NOT NULL DEFAULT '',
    iam_user_id VARCHAR(128) NOT NULL,
    local_uid   BIGINT NOT NULL DEFAULT nextval('biz_iam_user_map_local_uid_seq'),
    person_id   VARCHAR(128) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (issuer, iam_user_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_biz_iam_user_map_local_uid ON biz_iam_user_map (local_uid);
`

// TestEnsureLocalUserIdempotent EnsureLocalUser 幂等：同一 (issuer, user) 反复调用得到同一 local_uid。
func TestEnsureLocalUserIdempotent(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameDemo)
	t.Cleanup(func() { testsetup.Close(testsetup.AppNameDemo) })

	db := dbclient.DemoDB(context.Background())
	require.NoError(t, db.Exec(ensureMapDDL).Error, "准备 biz_iam_user_map 失败")
	t.Cleanup(func() {
		db.Exec(`DELETE FROM biz_iam_user_map WHERE issuer = ? AND iam_user_id = ?`, "https://idp-a.example", "itest-user-X")
		db.Exec(`DELETE FROM biz_iam_user_map WHERE issuer = ? AND iam_user_id = ?`, "https://idp-a.example", "itest-user-Y")
	})

	svc := iamlink.NewSvc()
	ctx := context.Background()
	const issuer = "https://idp-a.example"

	uid1, err := svc.EnsureLocalUser(ctx, issuer, "itest-user-X", "itest-person-X")
	require.NoError(t, err)
	require.NotZero(t, uid1, "local_uid 不应为 0")

	uid2, err := svc.EnsureLocalUser(ctx, issuer, "itest-user-X", "itest-person-X")
	require.NoError(t, err)
	assert.Equal(t, uid1, uid2, "同一 (issuer, user) 应幂等返回同一 local_uid")

	// 同一 issuer 下不同 IAM user 应得到不同 local_uid。
	uid3, err := svc.EnsureLocalUser(ctx, issuer, "itest-user-Y", "itest-person-Y")
	require.NoError(t, err)
	assert.NotEqual(t, uid1, uid3, "不同 IAM user 应分配不同 local_uid")
}

// TestEnsureLocalUserIssuerIsolation 同一 sub 在不同 issuer 下应映射到不同的 local_uid
// （(issuer, iam_user_id) 复合键防跨 IDP 串号）。
func TestEnsureLocalUserIssuerIsolation(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameDemo)
	t.Cleanup(func() { testsetup.Close(testsetup.AppNameDemo) })

	db := dbclient.DemoDB(context.Background())
	require.NoError(t, db.Exec(ensureMapDDL).Error, "准备 biz_iam_user_map 失败")
	const issuerA = "https://idp-a.example"
	const issuerB = "https://idp-b.example"
	const sub = "itest-shared-sub"
	t.Cleanup(func() {
		db.Exec(`DELETE FROM biz_iam_user_map WHERE iam_user_id = ?`, sub)
	})

	svc := iamlink.NewSvc()
	ctx := context.Background()

	uidA, err := svc.EnsureLocalUser(ctx, issuerA, sub, "person-a")
	require.NoError(t, err)
	require.NotZero(t, uidA)

	uidB, err := svc.EnsureLocalUser(ctx, issuerB, sub, "person-b")
	require.NoError(t, err)
	require.NotZero(t, uidB)

	assert.NotEqual(t, uidA, uidB, "同一 sub 在不同 issuer 下必须映射到不同 local_uid（防串号）")

	// 再次调用各自应幂等命中（缓存/DB 都按 (issuer, sub) 定位）。
	againA, err := svc.EnsureLocalUser(ctx, issuerA, sub, "person-a")
	require.NoError(t, err)
	assert.Equal(t, uidA, againA)
	againB, err := svc.EnsureLocalUser(ctx, issuerB, sub, "person-b")
	require.NoError(t, err)
	assert.Equal(t, uidB, againB)
}
