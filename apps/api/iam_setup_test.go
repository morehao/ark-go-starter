package demo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	demo "github.com/morehao/ark-go-starter/apps/api"
	"github.com/morehao/ark-go-starter/apps/api/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newEngineWithMode 以给定 iam.mode 装配一个引擎（不初始化 DB/Redis）。
//
// 只用于验证 SetupIAM 的两种契约：disabled 零副作用不挂路由；非法 mode 报错。
func newEngineWithMode(t *testing.T, mode string) *gin.Engine {
	t.Helper()
	prev := config.Conf
	t.Cleanup(func() { config.Conf = prev })

	config.Conf = &config.Config{
		Server: config.Server{Name: "demo", Port: "8099", Env: "dev"},
		IAM:    config.IAMConfig{Mode: mode},
	}

	gin.SetMode(gin.TestMode)
	return gin.New()
}

// TestSetupIAMDisabledHasNoSideEffects disabled：返回 nil、不挂任何 IAM 路由、无 IAM 运行时。
func TestSetupIAMDisabledHasNoSideEffects(t *testing.T) {
	for _, mode := range []string{config.ModeDisabled, ""} {
		t.Run("mode="+mode, func(t *testing.T) {
			engine := newEngineWithMode(t, mode)

			closer, err := demo.SetupIAM()
			require.NoError(t, err, "disabled 不应报错")
			assert.Nil(t, closer, "disabled 应返回 nil closer（零副作用）")
			assert.Nil(t, demo.CurrentIAM(), "disabled 不应有 IAM 运行时")

			// disabled 不挂任何 IAM 路由（代表性路径应 404）。
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/iam/install/status", nil))
			assert.Equal(t, http.StatusNotFound, rec.Code, "disabled 时不应挂载任何 IAM 路由")
		})
	}
}

// TestSetupIAMInvalidModeReturnsError 非法 mode：SetupIAM 返回 error（不静默回落）。
func TestSetupIAMInvalidModeReturnsError(t *testing.T) {
	newEngineWithMode(t, "bogus")

	closer, err := demo.SetupIAM()
	require.Error(t, err, "非法 mode 应报错")
	assert.Nil(t, closer, "非法 mode 不应返回 closer")
	assert.Nil(t, demo.CurrentIAM(), "非法 mode 不应有 IAM 运行时")
}
