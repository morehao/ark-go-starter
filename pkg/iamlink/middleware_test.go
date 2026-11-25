package iamlink_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/pkg/iamlink"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer = "http://localhost:8099/iam/oidc"
	testUserID = "user-1"
	testPerson = "person-9"
	testTenant = "tenant-7"
)

// fakeEnsurer 是 LocalUserEnsurer 的假实现：按 iamUserID 映射到固定的本地 uint。
type fakeEnsurer struct {
	mapping     map[string]uint
	err         error
	gotIssuer   string
	gotPersonID string
}

// EnsureLocalUser 实现 iamlink.LocalUserEnsurer。
func (m *fakeEnsurer) EnsureLocalUser(_ context.Context, issuer, iamUserID, personID string) (uint, error) {
	m.gotIssuer = issuer
	m.gotPersonID = personID
	if m.err != nil {
		return 0, m.err
	}
	uid, ok := m.mapping[iamUserID]
	if !ok {
		return 0, assert.AnError
	}
	return uid, nil
}

// newRouter 构造一个先注入 rp.Identity、再跑 iamlink.Middleware 的最小路由。
//
// inject 为 nil 时不注入身份（模拟上游 oidcmw 未生效）。
func newRouter(t *testing.T, ensurer iamlink.LocalUserEnsurer, inject *rp.Identity) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	grp := engine.Group("/v1/demo")
	grp.Use(func(ctx *gin.Context) {
		if inject != nil {
			ctx.Request = ctx.Request.WithContext(rp.WithIdentity(ctx.Request.Context(), inject))
		}
		ctx.Next()
	})
	grp.Use(iamlink.Middleware(testIssuer, iamlink.WithLocalUserEnsurer(ensurer)))
	grp.GET("/whoami", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"userID":   gincontext.GetUserID(ctx),
			"personID": gincontext.GetPersonID(ctx),
		})
	})
	return engine
}

// doGet 发起带可选 Bearer token 的 GET。
func doGet(engine *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

// TestMapsIdentityToLocalUID 有 identity → 映射为本地 uint 写入 gcontext.KeyUserID/KeyPersonID。
func TestMapsIdentityToLocalUID(t *testing.T) {
	ensurer := &fakeEnsurer{mapping: map[string]uint{testUserID: 42}}
	engine := newRouter(t, ensurer, &rp.Identity{UserID: testUserID, PersonID: testPerson})

	rec := doGet(engine, "/v1/demo/whoami", "rawtoken")
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"userID":42`)
	assert.Contains(t, rec.Body.String(), `"personID":42`)
	assert.Equal(t, testIssuer, ensurer.gotIssuer, "应把装配层 issuer 传入映射器")
	assert.Equal(t, testPerson, ensurer.gotPersonID, "应把 IAM personID 传入映射器")
}

// TestMissingIdentityAborts 上游未注入 identity → 401 fail-closed。
func TestMissingIdentityAborts(t *testing.T) {
	ensurer := &fakeEnsurer{mapping: map[string]uint{testUserID: 42}}
	engine := newRouter(t, ensurer, nil)

	assert.Equal(t, http.StatusUnauthorized, doGet(engine, "/v1/demo/whoami", "x").Code)
}

// TestEmptyUserIDAborts identity 缺 user_id → 401。
func TestEmptyUserIDAborts(t *testing.T) {
	ensurer := &fakeEnsurer{mapping: map[string]uint{}}
	engine := newRouter(t, ensurer, &rp.Identity{UserID: ""})

	assert.Equal(t, http.StatusUnauthorized, doGet(engine, "/v1/demo/whoami", "x").Code)
}

// TestMapperErrorAborts 映射失败 → 401 fail-closed。
func TestMapperErrorAborts(t *testing.T) {
	ensurer := &fakeEnsurer{err: assert.AnError}
	engine := newRouter(t, ensurer, &rp.Identity{UserID: testUserID, PersonID: testPerson})

	assert.Equal(t, http.StatusUnauthorized, doGet(engine, "/v1/demo/whoami", "x").Code)
}

// TestAuthTokenAndTenantPropagated 令牌原样透传、tenant 写入上下文。
func TestAuthTokenAndTenantPropagated(t *testing.T) {
	ensurer := &fakeEnsurer{mapping: map[string]uint{testUserID: 7}}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	grp := engine.Group("/v1/demo")
	grp.Use(func(ctx *gin.Context) {
		ctx.Request = ctx.Request.WithContext(rp.WithIdentity(ctx.Request.Context(), &rp.Identity{
			UserID: testUserID, PersonID: testPerson, TenantID: testTenant,
		}))
		ctx.Next()
	})
	grp.Use(iamlink.Middleware(testIssuer, iamlink.WithLocalUserEnsurer(ensurer)))
	grp.GET("/echo", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"authToken": ctx.GetString(gcontext.KeyAuthToken),
			"tenantID":  ctx.GetString(gcontext.KeyTenantID),
		})
	})

	rec := doGet(engine, "/v1/demo/echo", "raw-abc")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "raw-abc")
	assert.Contains(t, rec.Body.String(), testTenant)
}
