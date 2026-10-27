package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/biz/gcontext"
)

// HeaderUserID 演示用身份请求头。
const HeaderUserID = "X-User-ID"

// DemoIdentity 演示用的身份中间件：从 HeaderUserID 读取用户 ID 写入 gin Keys。
//
// 这是接入真实鉴权之前的**占位实现**，只应在 dev 环境挂载（见 demo.Routers）：
// 它信任客户端传入的请求头，任何客户端都能伪造操作人，绝不可用于生产。
// 接入真实鉴权后应删除本中间件，改由鉴权中间件写入 gcontext.KeyUserID。
func DemoIdentity() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if userID, err := strconv.ParseUint(ctx.GetHeader(HeaderUserID), 10, 64); err == nil {
			ctx.Set(gcontext.KeyUserID, strconv.FormatUint(userID, 10))
		}
		ctx.Next()
	}
}
