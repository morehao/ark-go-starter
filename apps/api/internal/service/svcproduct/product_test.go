//go:build integration

// 依赖本地 PostgreSQL/Redis，属集成测试：默认 `go test ./...` 不运行，
// 需 `go test -tags=integration ./...`。

package svcproduct

import (
	"testing"

	"github.com/morehao/ark-go-starter/apps/api/internal/dto/dtoproduct"
	"github.com/morehao/ark-go-starter/libs/go/testsetup"
	"github.com/morehao/golib/gutil"
	"github.com/stretchr/testify/assert"
)

func TestProductList(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameDemo)
	defer testsetup.Close(testsetup.AppNameDemo)

	ctx := testsetup.NewContext().Request.Context()
	svc := NewProductSvc()
	res, err := svc.PageList(ctx, &dtoproduct.ProductPageListReq{})
	assert.Nil(t, err)
	t.Logf("res: %s", gutil.ToJsonString(res))
}
