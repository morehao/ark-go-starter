package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/morehao/ark-go-starter/demo/internal/middleware"
	"github.com/morehao/ark-iam/sdk/contract"
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
)

// fakeMapper 是 LocalUserEnsurer 的假实现：按 iamUserID 映射到固定的本地 uint。
type fakeMapper struct {
	// mapping 是 iamUserID → localUID 的映射；未命中返回错误。
	mapping map[string]uint
	// err 非空时所有调用返回该错误。
	err error
	// gotPersonID 记录最近一次调用传入的 personID，供断言。
	gotPersonID string
}

// EnsureLocalUser 实现 middleware.LocalUserEnsurer。
func (m *fakeMapper) EnsureLocalUser(_ context.Context, iamUserID, personID string) (uint, error) {
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

// deriveKID 复刻 IAM 的 kid 推导（公钥模数 SHA-256 前 16 字节 RawURL 编码）。
func deriveKID(pub *rsa.PublicKey) string {
	sum := sha256.Sum256(pub.N.Bytes())
	return base64.RawURLEncoding.EncodeToString(sum[:16])
}

// signToken 用私钥签发一枚 RS256 access token（自然人令牌）。
func signToken(t *testing.T, priv *rsa.PrivateKey, kid, issuer, userID, personID string, exp time.Time) string {
	t.Helper()
	claims := contract.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   contract.BuildPersonSubject(personID),
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:        userID,
		PersonIDClaim: personID,
		ClientID:      "demo_web",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	raw, err := tok.SignedString(priv)
	require.NoError(t, err)
	return raw
}

// newTestRouter 构造一个挂了 OIDCAuth 中间件的最小路由，返回 engine 与 keySource 的工具。
func newTestRouter(t *testing.T, mapper middleware.LocalUserEnsurer, issuer string) (*gin.Engine, *rsa.PrivateKey, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	kid := deriveKID(&priv.PublicKey)
	ks := rp.NewKeysFromSet(map[string]*rsa.PublicKey{kid: &priv.PublicKey})
	t.Cleanup(ks.Close)

	opts := []middleware.OIDCAuthOption{
		middleware.WithOIDCKeySource(ks),
		middleware.WithLocalUserMapper(mapper),
		middleware.WithOIDCSkipPaths("/v1/demo/health", "/demo/docs"),
	}
	if issuer != "" {
		opts = append(opts, middleware.WithOIDCIssuer(issuer))
	}

	engine := gin.New()
	grp := engine.Group("/v1/demo")
	grp.Use(middleware.OIDCAuth(opts...))
	// 回显身份：把 gincontext 解析出的 uint UserID/PersonID 输出，验证映射链路。
	grp.GET("/whoami", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"userID":    gincontext.GetUserID(ctx),
			"personID":  gincontext.GetPersonID(ctx),
			"authToken": ctx.GetString(gcontext.KeyAuthToken),
		})
	})
	grp.GET("/health", func(ctx *gin.Context) { ctx.JSON(http.StatusOK, gin.H{"ok": true}) })
	return engine, priv, kid
}

// doGet 发起 GET 请求（带可选 Authorization）。
func doGet(engine *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

// TestOIDCAuthRejectsMissingToken 无 token 访问受保护接口应 401。
func TestOIDCAuthRejectsMissingToken(t *testing.T) {
	mapper := &fakeMapper{mapping: map[string]uint{testUserID: 42}}
	engine, _, _ := newTestRouter(t, mapper, testIssuer)

	rec := doGet(engine, "/v1/demo/whoami", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestOIDCAuthMapsIdentityToLocalUID 有效 token → 映射为本地 uint 身份写入 gin 上下文。
func TestOIDCAuthMapsIdentityToLocalUID(t *testing.T) {
	mapper := &fakeMapper{mapping: map[string]uint{testUserID: 42}}
	engine, priv, kid := newTestRouter(t, mapper, testIssuer)

	token := signToken(t, priv, kid, testIssuer, testUserID, testPerson, time.Now().Add(time.Hour))
	rec := doGet(engine, "/v1/demo/whoami", token)

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var out struct {
		UserID    uint   `json:"userID"`
		PersonID  uint   `json:"personID"`
		AuthToken string `json:"authToken"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.EqualValues(t, 42, out.UserID, "IAM user-1 应映射为本地 uint 42")
	assert.EqualValues(t, 42, out.PersonID, "personID 应投影为同一 local_uid")
	assert.Equal(t, token, out.AuthToken, "authToken 应透传原令牌")
	assert.Equal(t, testPerson, mapper.gotPersonID, "应把 IAM personID 传给映射器")
}

// TestOIDCAuthRejectsExpiredToken 过期令牌应 401。
func TestOIDCAuthRejectsExpiredToken(t *testing.T) {
	mapper := &fakeMapper{mapping: map[string]uint{testUserID: 42}}
	engine, priv, kid := newTestRouter(t, mapper, testIssuer)

	token := signToken(t, priv, kid, testIssuer, testUserID, testPerson, time.Now().Add(-time.Hour))
	rec := doGet(engine, "/v1/demo/whoami", token)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestOIDCAuthRejectsWrongIssuer issuer 不匹配应 401（配置了 issuer 时强制校验）。
func TestOIDCAuthRejectsWrongIssuer(t *testing.T) {
	mapper := &fakeMapper{mapping: map[string]uint{testUserID: 42}}
	engine, priv, kid := newTestRouter(t, mapper, testIssuer)

	token := signToken(t, priv, kid, "http://evil/oidc", testUserID, testPerson, time.Now().Add(time.Hour))
	rec := doGet(engine, "/v1/demo/whoami", token)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestOIDCAuthSkipPath 白名单路径（health）无需 token 直接放行。
func TestOIDCAuthSkipPath(t *testing.T) {
	mapper := &fakeMapper{mapping: map[string]uint{testUserID: 42}}
	engine, _, _ := newTestRouter(t, mapper, testIssuer)

	rec := doGet(engine, "/v1/demo/health", "")
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestOIDCAuthMapperErrorAborts 映射失败（如 DB 异常）应 401 fail-closed。
func TestOIDCAuthMapperErrorAborts(t *testing.T) {
	mapper := &fakeMapper{err: assert.AnError}
	engine, priv, kid := newTestRouter(t, mapper, testIssuer)

	token := signToken(t, priv, kid, testIssuer, testUserID, testPerson, time.Now().Add(time.Hour))
	rec := doGet(engine, "/v1/demo/whoami", token)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestOIDCAuthNoKeySourceFailsClosed 缺少 KeySource 时构造的中间件一律 401。
func TestOIDCAuthNoKeySourceFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	grp := engine.Group("/v1/demo")
	grp.Use(middleware.OIDCAuth(middleware.WithLocalUserMapper(&fakeMapper{mapping: map[string]uint{}})))
	grp.GET("/whoami", func(ctx *gin.Context) { ctx.JSON(http.StatusOK, gin.H{"ok": true}) })

	rec := doGet(engine, "/v1/demo/whoami", "anything")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
