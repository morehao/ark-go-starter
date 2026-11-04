// Package dtoexample 定义示例接口的请求与响应 DTO。
package dtoexample

// GetUserReq 是示例用户查询请求。
type GetUserReq struct {
	ID uint `json:"id" form:"id"` // 用户ID
}
