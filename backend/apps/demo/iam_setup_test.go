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

// newEngineWithMode 以给定 iam.mode 装配一个只挂 IAM 的引擎（不初始化 DB/Redis）。
//
// 只用于验证 SetupIAM 的三态契约：disabled 零副作用不挂路由；非法 mode 报错。
func newEngineWithMode(t *testing.T, mode string) *gin.Engine {
	t.Helper()
	prev := config.Conf
	t.Cleanup(func() { config.Conf = prev })

	config.Conf = &config.Config{
		Server: config.Server{Name: "demo", Port: "8099", Env: "dev"},
		IAM:    config.IAMConfig{Mode: mode, Prefix: testIAMPrefix},
	}

	gin.SetMode(gin.TestMode)
	return gin.New()
}

// TestSetupIAMDisabledHasNoSideEffects disabled：返回 nil、不挂 /iam/* 路由、无 IAM 运行时。
func TestSetupIAMDisabledHasNoSideEffects(t *testing.T) {
	for _, mode := range []string{config.ModeDisabled, ""} {
		t.Run("mode="+mode, func(t *testing.T) {
			engine := newEngineWithMode(t, mode)

			closer, err := demo.SetupIAM(engine)
			require.NoError(t, err, "disabled 不应报错")
			assert.Nil(t, closer, "disabled 应返回 nil closer（零副作用）")
			assert.Nil(t, demo.CurrentIAM(), "disabled 不应有 IAM 运行时")

			// /iam/install、/iam/config.json、/iam/oidc 等 IAM 路由均不应存在（404）。
			for _, path := range []string{"/iam/install", "/iam/config.json", "/iam/oidc"} {
				rec := httptest.NewRecorder()
				engine.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
				assert.Equal(t, http.StatusNotFound, rec.Code, "disabled 时 %s 应 404", path)
			}
		})
	}
}

// TestSetupIAMInvalidModeReturnsError 非法 mode：SetupIAM 返回 error（不静默回落）。
func TestSetupIAMInvalidModeReturnsError(t *testing.T) {
	engine := newEngineWithMode(t, "bogus")

	closer, err := demo.SetupIAM(engine)
	require.Error(t, err, "非法 mode 应报错")
	assert.Nil(t, closer, "非法 mode 不应返回 closer")
	assert.Nil(t, demo.CurrentIAM(), "非法 mode 不应有 IAM 运行时")
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
