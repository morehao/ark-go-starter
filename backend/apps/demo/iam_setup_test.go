package demo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/demo"
	"github.com/morehao/ark-go-starter/demo/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testIAMPrefix 是测试用 IAM 路由前缀。
const testIAMPrefix = "/iam"

// newEngineWithIAM 以给定 iam.enable 装配一个只挂 IAM 的引擎（不初始化 DB/Redis）。
//
// 只用于验证 SetupIAM 的「开关零副作用」契约：Enable=false 时不得触碰 DB/Redis、不得挂路由。
func newEngineWithIAM(t *testing.T, enable bool) *gin.Engine {
	t.Helper()
	prev := config.Conf
	t.Cleanup(func() { config.Conf = prev })

	config.Conf = &config.Config{
		Server: config.Server{Name: "demo", Port: "8099", Env: "dev"},
		IAM:    config.IAMConfig{Enable: enable, Prefix: testIAMPrefix},
	}

	gin.SetMode(gin.TestMode)
	return gin.New()
}

// TestSetupIAMDisabledHasNoSideEffects Enable=false：返回 nil、不挂 /iam/* 路由、无 IAM 运行时。
func TestSetupIAMDisabledHasNoSideEffects(t *testing.T) {
	engine := newEngineWithIAM(t, false)

	closer, err := demo.SetupIAM(engine)
	require.NoError(t, err, "Enable=false 不应报错")
	assert.Nil(t, closer, "Enable=false 应返回 nil closer（零副作用）")
	assert.Nil(t, demo.CurrentIAM(), "Enable=false 不应有 IAM 运行时")

	// /iam/install、/iam/config.json 等 IAM 路由均不应存在（404，非 true 路径的 200）。
	for _, path := range []string{"/iam/install", "/iam/config.json", "/iam/oidc"} {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
		assert.Equal(t, http.StatusNotFound, rec.Code, "Enable=false 时 %s 应 404", path)
	}
}

// TestIAMConfigGetPrefix 验证前缀归一化（保证以 / 开头、不以 / 结尾）。
func TestIAMConfigGetPrefix(t *testing.T) {
	cases := map[string]string{
		"":            testIAMPrefix,
		testIAMPrefix: testIAMPrefix,
		"iam":         testIAMPrefix,
		"/iam/":       testIAMPrefix,
		"/x/iam":      "/x/iam",
	}
	for in, want := range cases {
		got := config.IAMConfig{Prefix: in}.GetPrefix()
		assert.Equal(t, want, got, "prefix %q", in)
	}
}
