package middleware

import "github.com/gin-gonic/gin"

// Example 是示例中间件，调用 Next 后直接放行。
func Example() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
	}
}
