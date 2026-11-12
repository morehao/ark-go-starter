package ctrexample

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/api/internal/service/svcexample"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// FormatCtr 是响应格式化示例控制器接口。
type FormatCtr interface {
	FormatResponse(ctx *gin.Context)
}

type formatCtr struct {
	exampleSvc svcexample.FormatSvc
}

var _ FormatCtr = (*formatCtr)(nil)

// NewFormatCtr 构造 FormatCtr。
func NewFormatCtr() FormatCtr {
	return &formatCtr{
		exampleSvc: svcexample.NewFormatSvc(),
	}
}

func (ctr *formatCtr) FormatResponse(ctx *gin.Context) {
	res := ctr.exampleSvc.FormatResponse(ctx.Request.Context())

	gincontext.SuccessWithFormat(ctx, res)
}
