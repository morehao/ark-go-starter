//go:build integration

// 依赖 testsetup 初始化环境并访问外部 HTTP 服务，属集成测试：
// 默认 `go test ./...` 不运行，需 `go test -tags=integration ./...`。

package httpbingo

import (
	"testing"

	"github.com/morehao/ark-go-starter/apps/api/config"
	"github.com/morehao/ark-go-starter/libs/go/testsetup"
	"github.com/morehao/golib/gutil"
	"github.com/stretchr/testify/assert"
)

func findConfigPath(t *testing.T) string {
	t.Helper()
	return testsetup.TestConfigPath(testsetup.AppNameDemo)
}

func TestGet(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameDemo)
	config.LoadConfig(findConfigPath(t))

	ctx := testsetup.NewContext()
	res, err := Get(ctx, &GetRequest{
		ID: 1,
	})
	assert.Nil(t, err)
	t.Log(gutil.ToJsonString(res))
}
