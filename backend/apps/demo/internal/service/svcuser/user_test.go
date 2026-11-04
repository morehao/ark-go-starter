//go:build integration

// 依赖本地 PostgreSQL/Redis，属集成测试：默认 `go test ./...` 不运行，
// 需 `go test -tags=integration ./...`。

package svcuser

import (
	"testing"

	"github.com/morehao/ark-go-starter/demo/internal/dto/dtouser"
	"github.com/morehao/ark-go-starter/pkg/testsetup"
	"github.com/morehao/golib/gutil"
	"github.com/stretchr/testify/assert"
)

func TestUserList(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameDemo)
	defer testsetup.Close(testsetup.AppNameDemo)

	ctx := testsetup.NewContext().Request.Context()
	svc := NewUserSvc()
	res, err := svc.PageList(ctx, &dtouser.UserPageListReq{})
	assert.Nil(t, err)
	t.Logf("res: %s", gutil.ToJsonString(res))
}
