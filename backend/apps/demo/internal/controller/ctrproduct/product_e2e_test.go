//go:build integration

// 该 e2e 用例依赖本地 PostgreSQL/Redis 与真实 HTTP 栈，属集成测试：
// 默认 `go test ./...` 不会运行，需显式 `go test -tags=integration ./...`。
// 未加此 build tag 时，无数据库环境会直接失败，污染单元测试信号。

package ctrproduct_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/morehao/ark-go-starter/demo"
	"github.com/morehao/ark-go-starter/demo/config"
	"github.com/morehao/ark-go-starter/demo/internal/controller/ctrproduct"
	"github.com/morehao/ark-go-starter/demo/internal/middleware"
	"github.com/morehao/ark-go-starter/demo/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/ark-go-starter/pkg/iamlink"
	"github.com/morehao/ark-go-starter/pkg/testsetup"
	"github.com/morehao/ark-iam/sdk/contract"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 依赖本地 PostgreSQL（同 pkg/dbclient 的集成测试），配置读取 apps/demo/config/config.yaml。

// configPath 定位 app 配置文件，与 testsetup.findConfigPath 保持同一约定（从包目录回溯到 apps/demo）。
func configPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	appDir := filepath.Dir(filename) // .../ctrproduct
	for range 3 {
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

// newTestEngine 构造与 cmd 启动一致的引擎（走 demo.Routers 的默认 off 路径）。
//
// config.Conf 由 testsetup 之外的 businessInit 负责，这里直接在初始化前写入，
// 让 demo.Routers 走 dev 分支、挂上 DemoIdentity（Enable=false 时唯一的身份来源）。
func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	testsetup.Initialize(testsetup.AppNameDemo)
	config.LoadConfig(configPath(t))
	t.Cleanup(func() { testsetup.Close(testsetup.AppNameDemo) })

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	demo.Routers(engine)
	return engine
}

// newTokenEngine 构造一个「真鉴权」引擎：业务 v1/demo 分组挂 OIDCAuth（OIDC 本地验签 +
// iamlink 身份映射），从而以真实 token 驱动 controller/service/dao，验证审计列来自令牌身份。
//
// 返回引擎与签发器，供各用例按身份签发 access token。
func newTokenEngine(t *testing.T) (*gin.Engine, *tokenIssuer) {
	t.Helper()
	testsetup.Initialize(testsetup.AppNameDemo)
	t.Cleanup(func() { testsetup.Close(testsetup.AppNameDemo) })

	// 准备 biz_iam_user_map（幂等）。
	require.NoError(t, dbclient.DemoDB(context.Background()).Exec(ensureMapDDL).Error)

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	sum := sha256.Sum256(priv.PublicKey.N.Bytes())
	kid := base64.RawURLEncoding.EncodeToString(sum[:16])
	ks := rp.NewKeysFromSet(map[string]*rsa.PublicKey{kid: &priv.PublicKey})
	t.Cleanup(ks.Close)

	const issuer = "http://localhost:8099/iam/oidc"
	authMW := middleware.OIDCAuth(
		middleware.WithOIDCKeySource(ks),
		middleware.WithOIDCIssuer(issuer),
		middleware.WithOIDCSkipPaths("/v1/demo/health"),
		middleware.WithLocalUserMapper(iamlink.NewSvc()),
	)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	grp := engine.Group("/v1/demo")
	grp.Use(authMW)
	productCtr := ctrproduct.NewProductCtr()
	grp.POST("/products", productCtr.Create)
	grp.POST("/products/tx", productCtr.CreateInTx)
	grp.GET("/products", productCtr.PageList)
	grp.GET("/products/:productID", productCtr.Detail)
	grp.PUT("/products/:productID", productCtr.Update)
	grp.DELETE("/products/:productID", productCtr.Delete)

	return engine, &tokenIssuer{priv: priv, kid: kid, issuer: issuer}
}

// tokenIssuer 用固定私钥签发测试 access token。
type tokenIssuer struct {
	priv   *rsa.PrivateKey
	kid    string
	issuer string
}

// sign 签发一枚自然人 access token，userID/personID 为 IAM string 身份。
func (ti *tokenIssuer) sign(t *testing.T, userID, personID string) string {
	t.Helper()
	claims := contract.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ti.issuer,
			Subject:   contract.BuildPersonSubject(personID),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:        userID,
		PersonIDClaim: personID,
		ClientID:      "demo_web",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = ti.kid
	raw, err := tok.SignedString(ti.priv)
	require.NoError(t, err)
	return raw
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

// bearer 把 token 包装为 Authorization 头。
func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func createProductWithHeaders(t *testing.T, engine *gin.Engine, headers map[string]string) uint {
	t.Helper()
	resp := doJSON(t, engine, http.MethodPost, "/v1/demo/products", headers, map[string]any{
		"name":       "svc-operator-e2e",
		"categoryID": 22,
		"price":      19.90,
	})
	require.Equal(t, 0, resp.Code, "create failed: %s", resp.Msg)
	var created struct {
		ProductID uint `json:"productID"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &created))
	require.NotZero(t, created.ProductID)
	return created.ProductID
}

// auditRow 直接读库核对审计列，绕开接口层，确认值真的落库。
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

func cleanupProduct(t *testing.T, productID uint) {
	t.Helper()
	err := dbclient.DemoDB(context.Background()).Unscoped().
		Where("id = ?", productID).Delete(&model.ProductEntity{}).Error
	require.NoError(t, err)
}

func productPath(productID uint) string {
	return "/v1/demo/products/" + strconv.FormatUint(uint64(productID), 10)
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

// TestProductCRUDWithOIDCToken 真鉴权链路：真实 token → OIDC 验签 → 身份映射 → 审计列。
//
// 三个动作（建/改/删）用三枚不同身份的 token，断言 created_by/updated_by/deleted_by 分别
// 落为对应 IAM 身份映射出的本地 uint（而非字面 userID），证明"真鉴权 + ID 映射"端到端成立。
func TestProductCRUDWithOIDCToken(t *testing.T) {
	engine, issuer := newTokenEngine(t)

	// 三个 IAM 身份 → 预先映射出各自 local_uid（与中间件同一 EnsureLocalUser）。
	svc := iamlink.NewSvc()
	ctx := context.Background()
	creatorUID, err := svc.EnsureLocalUser(ctx, "itest-prod-creator", "itest-person-c")
	require.NoError(t, err)
	updaterUID, err := svc.EnsureLocalUser(ctx, "itest-prod-updater", "itest-person-u")
	require.NoError(t, err)
	deleterUID, err := svc.EnsureLocalUser(ctx, "itest-prod-deleter", "itest-person-d")
	require.NoError(t, err)
	t.Cleanup(func() {
		db := dbclient.DemoDB(ctx)
		db.Exec(`DELETE FROM biz_iam_user_map WHERE iam_user_id IN (?,?,?)`,
			"itest-prod-creator", "itest-prod-updater", "itest-prod-deleter")
	})

	creatorTok := issuer.sign(t, "itest-prod-creator", "itest-person-c")
	updaterTok := issuer.sign(t, "itest-prod-updater", "itest-person-u")
	deleterTok := issuer.sign(t, "itest-prod-deleter", "itest-person-d")

	// 创建：created_by/updated_by 应记创建人映射出的 local_uid。
	productID := createProductWithHeaders(t, engine, bearer(creatorTok))
	t.Cleanup(func() { cleanupProduct(t, productID) })
	afterCreate := queryAudit(t, productID)
	assert.EqualValues(t, creatorUID, afterCreate.CreatedBy, "created_by 应等于创建人映射的 local_uid")
	assert.EqualValues(t, creatorUID, afterCreate.UpdatedBy, "创建时 updated_by 应初始化为创建人")

	// 详情/列表
	assert.Equal(t, 0, doJSON(t, engine, http.MethodGet, productPath(productID), bearer(creatorTok), nil).Code)
	assert.Equal(t, 0, doJSON(t, engine, http.MethodGet, "/v1/demo/products?page=1&pageSize=10", bearer(creatorTok), nil).Code)

	// 修改：updated_by 换成修改人映射的 local_uid，created_by 不变。
	update := doJSON(t, engine, http.MethodPut, productPath(productID), bearer(updaterTok), map[string]any{
		"name":       "svc-operator-e2e-updated",
		"categoryID": 33,
		"price":      29.90,
	})
	assert.Equal(t, 0, update.Code, update.Msg)
	afterUpdate := queryAudit(t, productID)
	assert.EqualValues(t, creatorUID, afterUpdate.CreatedBy, "修改不应改动 created_by")
	assert.EqualValues(t, updaterUID, afterUpdate.UpdatedBy, "updated_by 应等于修改人映射的 local_uid")

	// 删除：deleted_by 等于删除人映射的 local_uid。
	del := doJSON(t, engine, http.MethodDelete, productPath(productID), bearer(deleterTok), nil)
	assert.Equal(t, 0, del.Code, del.Msg)
	afterDelete := queryAudit(t, productID)
	assert.EqualValues(t, deleterUID, afterDelete.DeletedBy, "deleted_by 应等于删除人映射的 local_uid")

	// 软删除后再查详情：不应查到。
	gone := doJSON(t, engine, http.MethodGet, productPath(productID), bearer(creatorTok), nil)
	assert.NotEqual(t, 0, gone.Code, "软删除后不应再查到")
}

// TestProductNoTokenUnauthorized 无 token 访问业务接口应 401（fail-closed）。
func TestProductNoTokenUnauthorized(t *testing.T) {
	engine, _ := newTokenEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/demo/products?page=1&pageSize=10", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "无 token 应 401")
}

// TestProductCRUDWithDemoIdentity off 路径回归：Enable=false 时 DemoIdentity（请求头）仍生效。
func TestProductCRUDWithDemoIdentity(t *testing.T) {
	engine := newTestEngine(t)
	creatorHeaders := map[string]string{
		middleware.HeaderUserID:   "9001",
		middleware.HeaderTenantID: "501",
	}
	productID := createProductWithHeaders(t, engine, creatorHeaders)
	t.Cleanup(func() { cleanupProduct(t, productID) })

	afterCreate := queryAudit(t, productID)
	assert.EqualValues(t, 9001, afterCreate.CreatedBy, "DemoIdentity 下 created_by 应等于请求头操作人")
	assert.EqualValues(t, 9001, afterCreate.UpdatedBy, "创建时 updated_by 应初始化为创建人")
}

// TestProductDeleteForgedOperatorIgnored 客户端无法通过请求体伪造操作人（Operator 字段 json:"-"）。
func TestProductDeleteForgedOperatorIgnored(t *testing.T) {
	engine, issuer := newTokenEngine(t)
	tok := issuer.sign(t, "itest-forge-user", "itest-forge-person")
	productID := createProductWithHeaders(t, engine, bearer(tok))
	t.Cleanup(func() { cleanupProduct(t, productID) })
	t.Cleanup(func() {
		dbclient.DemoDB(context.Background()).Exec(`DELETE FROM biz_iam_user_map WHERE iam_user_id = ?`, "itest-forge-user")
	})

	del := doJSON(t, engine, http.MethodDelete, productPath(productID), bearer(tok), map[string]any{
		"operator": map[string]any{"userID": 12345},
		"Operator": map[string]any{"userID": 12345},
	})
	assert.Equal(t, 0, del.Code, del.Msg)
	// deleted_by 应是令牌身份映射出的 local_uid，而不是请求体伪造的 12345。
	assert.NotEqualValues(t, 12345, queryAudit(t, productID).DeletedBy, "请求体不能伪造操作人")
}
