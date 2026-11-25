package oidcmw_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/morehao/ark-go-starter/pkg/oidcmw"
	"github.com/morehao/ark-iam/contract"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer  = "http://localhost:8099/iam/oidc"
	testUserID  = "user-1"
	testPerson  = "person-9"
	testAud     = "demo_web"
	otherAud    = "another_app"
	healthPath  = "/v1/demo/health"
	whoamiPath  = "/v1/demo/whoami"
	skipDocPath = "/demo/docs"
)

// deriveKID 复刻 IAM 的 kid 推导（公钥模数 SHA-256 前 16 字节 RawURL 编码）。
func deriveKID(pub *rsa.PublicKey) string {
	sum := sha256.Sum256(pub.N.Bytes())
	return base64.RawURLEncoding.EncodeToString(sum[:16])
}

// signToken 用私钥签发一枚 RS256 access token。
func signToken(t *testing.T, priv *rsa.PrivateKey, kid, issuer, userID, personID string, exp time.Time, audiences ...string) string {
	t.Helper()
	claims := contract.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   contract.BuildPersonSubject(personID),
			Audience:  jwt.ClaimStrings(audiences),
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

// newKeySource 生成一对 RSA 密钥并返回 rp 内存 KeySource、私钥与 kid。
func newKeySource(t *testing.T) (contract.KeySource, *rsa.PrivateKey, string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	kid := deriveKID(&priv.PublicKey)
	ks := rp.NewKeysFromSet(map[string]*rsa.PublicKey{kid: &priv.PublicKey})
	t.Cleanup(ks.Close)
	return ks, priv, kid
}

// newRouter 构造挂了 OIDCAuth 的最小路由，回显 rp.IdentityFrom 取到的身份。
func newRouter(t *testing.T, verifier *rp.Verifier, opts ...oidcmw.Option) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	grp := engine.Group("/v1/demo")
	grp.Use(oidcmw.OIDCAuth(verifier, opts...))
	grp.GET("/whoami", func(ctx *gin.Context) {
		ident := rp.IdentityFrom(ctx.Request.Context())
		if ident == nil {
			ctx.JSON(http.StatusOK, gin.H{"identity": nil})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"userID": ident.UserID, "personID": ident.PersonID})
	})
	grp.GET("/health", func(ctx *gin.Context) { ctx.JSON(http.StatusOK, gin.H{"ok": true}) })
	return engine
}

// doGet 发起 GET 请求（带可选 Bearer token）。
func doGet(engine *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

// TestRejectsMissingToken 无 token 访问受保护接口应 401。
func TestRejectsMissingToken(t *testing.T) {
	ks, _, _ := newKeySource(t)
	verifier := rp.NewVerifier(ks, rp.WithIssuer(testIssuer))
	engine := newRouter(t, verifier, oidcmw.WithSkipPaths(healthPath))

	assert.Equal(t, http.StatusUnauthorized, doGet(engine, whoamiPath, "").Code)
}

// TestInjectsIdentity 有效 token → rp.WithIdentity 注入，下游可用 IdentityFrom 取回。
func TestInjectsIdentity(t *testing.T) {
	ks, priv, kid := newKeySource(t)
	verifier := rp.NewVerifier(ks, rp.WithIssuer(testIssuer))
	engine := newRouter(t, verifier, oidcmw.WithSkipPaths(healthPath))

	token := signToken(t, priv, kid, testIssuer, testUserID, testPerson, time.Now().Add(time.Hour))
	rec := doGet(engine, whoamiPath, token)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), testUserID)
}

// TestRejectsExpiredToken 过期令牌应 401。
func TestRejectsExpiredToken(t *testing.T) {
	ks, priv, kid := newKeySource(t)
	verifier := rp.NewVerifier(ks, rp.WithIssuer(testIssuer))
	engine := newRouter(t, verifier)

	token := signToken(t, priv, kid, testIssuer, testUserID, testPerson, time.Now().Add(-time.Hour))
	assert.Equal(t, http.StatusUnauthorized, doGet(engine, whoamiPath, token).Code)
}

// TestRejectsWrongIssuer issuer 不匹配应 401。
func TestRejectsWrongIssuer(t *testing.T) {
	ks, priv, kid := newKeySource(t)
	verifier := rp.NewVerifier(ks, rp.WithIssuer(testIssuer))
	engine := newRouter(t, verifier)

	token := signToken(t, priv, kid, "http://evil/oidc", testUserID, testPerson, time.Now().Add(time.Hour))
	assert.Equal(t, http.StatusUnauthorized, doGet(engine, whoamiPath, token).Code)
}

// TestSkipPath 白名单路径无需 token 直接放行。
func TestSkipPath(t *testing.T) {
	ks, _, _ := newKeySource(t)
	verifier := rp.NewVerifier(ks, rp.WithIssuer(testIssuer))
	engine := newRouter(t, verifier, oidcmw.WithSkipPaths(healthPath))

	assert.Equal(t, http.StatusOK, doGet(engine, healthPath, "").Code)
}

// TestNilVerifierFailsClosed 缺少验签器时一律 401。
func TestNilVerifierFailsClosed(t *testing.T) {
	engine := newRouter(t, nil)
	assert.Equal(t, http.StatusUnauthorized, doGet(engine, whoamiPath, "anything").Code)
}

// errKeySource 是 contract.KeySource 的假实现：始终返回密钥来源不可用。
type errKeySource struct{ err error }

// PublicKey 实现 contract.KeySource。
func (s errKeySource) PublicKey(_ context.Context, _ string) (*rsa.PublicKey, error) {
	return nil, s.err
}

// TestKeySourceUnavailableFailsClosed 密钥来源不可用时应 401（fail-closed）。
func TestKeySourceUnavailableFailsClosed(t *testing.T) {
	verifier := rp.NewVerifier(errKeySource{err: contract.ErrKeySourceUnavailable}, rp.WithIssuer(testIssuer))
	engine := newRouter(t, verifier)
	assert.Equal(t, http.StatusUnauthorized, doGet(engine, whoamiPath, "some.jwt.token").Code)
}

// TestAudienceMatchPasses 配置 aud 校验且令牌 aud 命中 → 放行。
func TestAudienceMatchPasses(t *testing.T) {
	ks, priv, kid := newKeySource(t)
	verifier := rp.NewVerifier(ks, rp.WithIssuer(testIssuer), rp.WithAudiences(testAud))
	engine := newRouter(t, verifier)

	token := signToken(t, priv, kid, testIssuer, testUserID, testPerson, time.Now().Add(time.Hour), testAud)
	require.Equal(t, http.StatusOK, doGet(engine, whoamiPath, token).Code)
}

// TestAudienceMismatchRejects 令牌 aud 不匹配 → 401。
func TestAudienceMismatchRejects(t *testing.T) {
	ks, priv, kid := newKeySource(t)
	verifier := rp.NewVerifier(ks, rp.WithIssuer(testIssuer), rp.WithAudiences(testAud))
	engine := newRouter(t, verifier)

	token := signToken(t, priv, kid, testIssuer, testUserID, testPerson, time.Now().Add(time.Hour), otherAud)
	assert.Equal(t, http.StatusUnauthorized, doGet(engine, whoamiPath, token).Code)
}

// TestAudienceMissingRejects 配置 aud 校验但令牌不带 aud → 401。
func TestAudienceMissingRejects(t *testing.T) {
	ks, priv, kid := newKeySource(t)
	verifier := rp.NewVerifier(ks, rp.WithIssuer(testIssuer), rp.WithAudiences(testAud))
	engine := newRouter(t, verifier)

	token := signToken(t, priv, kid, testIssuer, testUserID, testPerson, time.Now().Add(time.Hour))
	assert.Equal(t, http.StatusUnauthorized, doGet(engine, whoamiPath, token).Code)
}

// TestCustomUnauthorizedResponder 401 钩子生效（自定义响应体）。
func TestCustomUnauthorizedResponder(t *testing.T) {
	ks, _, _ := newKeySource(t)
	verifier := rp.NewVerifier(ks, rp.WithIssuer(testIssuer))
	called := false
	engine := newRouter(t, verifier, oidcmw.WithUnauthorizedResponder(func(ctx *gin.Context, msg string) {
		called = true
		ctx.AbortWithStatusJSON(http.StatusTeapot, gin.H{"msg": msg})
	}))

	rec := doGet(engine, whoamiPath, "")
	assert.True(t, called)
	assert.Equal(t, http.StatusTeapot, rec.Code)
}
